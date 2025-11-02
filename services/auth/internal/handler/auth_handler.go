package handler

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"auth/internal/domain"
	"auth/pkg/jwt"

	"github.com/google/uuid"
	natspkg "github.com/nats-io/nats.go"
)

type IAuthService interface {
	ValidateAccessToken(context.Context, string) (*jwt.Claims, error)
	Login(context.Context, domain.LoginRequest) (*domain.TokenPair, uuid.UUID, error)
	RefreshTokens(context.Context, domain.RefreshRequest) (*domain.TokenPair, uuid.UUID, error)
	Logout(ctx context.Context, userID uuid.UUID, token string) error
	LogoutAll(ctx context.Context, userID uuid.UUID) error
	ChangePassword(ctx context.Context, req domain.ChangePasswordRequest) error
}

type IEventPublisher interface {
	PublishUserLogin(domain.UserLoginEvent) error
	PublishUserLogout(domain.UserLogoutEvent) error
	PublishPasswordReset(domain.PasswordResetEvent) error
	PublishRegistrationRequested(domain.RegistrationRequestedEvent) error
	PublishUserRegistered(domain.UserRegisteredEvent) error
}

// AuthHandler handles authentication NATS requests
type AuthHandler struct {
	authService IAuthService
	publisher   IEventPublisher
}

// NewAuthHandler creates a new NATS auth handler
func NewAuthHandler(authService IAuthService, publisher IEventPublisher) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		publisher:   publisher,
	}
}

// HandleLogin handles login requests via NATS
func (h *AuthHandler) HandleLogin(msg *natspkg.Msg) {
	var req LoginRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		h.respondError(msg, "Invalid request format")
		return
	}

	ctx := context.Background()
	loginReq := req.ToDomain()

	tokens, userID, err := h.authService.Login(ctx, loginReq)
	if err != nil {
		h.respondError(msg, "Invalid credentials")
		return
	}

	// Publish login event
	_ = h.publisher.PublishUserLogin(domain.UserLoginEvent{
		UserID:    userID,
		DeviceID:  req.DeviceID,
		UserAgent: req.UserAgent,
		IPAddress: req.IPAddress,
		Timestamp: time.Now(),
	})

	response := LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    900, // 15 minutes
	}

	h.respond(msg, response, nil)
}

// HandleValidate handles token validation requests via NATS
func (h *AuthHandler) HandleValidate(msg *natspkg.Msg) {
	var req ValidateRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		h.respondError(msg, "Invalid request format")
		return
	}

	ctx := context.Background()
	claims, err := h.authService.ValidateAccessToken(ctx, req.Token)
	if err != nil {
		response := ValidateResponse{Valid: false}
		h.respond(msg, response, nil)
		return
	}

	response := ValidateResponse{
		Valid:  true,
		UserID: claims.UserID,
		Role:   claims.Role,
	}

	h.respond(msg, response, nil)
}

// HandleRefresh handles token refresh requests via NATS
func (h *AuthHandler) HandleRefresh(msg *natspkg.Msg) {
	var req RefreshRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		h.respondError(msg, "Invalid request format")
		return
	}

	ctx := context.Background()
	refreshReq := req.ToDomain()

	tokens, _, err := h.authService.RefreshTokens(ctx, refreshReq)
	if err != nil {
		h.respondError(msg, "Invalid or expired refresh token")
		return
	}

	response := LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    900,
	}

	h.respond(msg, response, nil)
}

// HandleLogout handles logout requests via NATS
func (h *AuthHandler) HandleLogout(msg *natspkg.Msg) {
	var req LogoutRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		h.respondError(msg, "Invalid request format")
		return
	}

	// Validate token first
	ctx := context.Background()
	claims, err := h.authService.ValidateAccessToken(ctx, req.Token)
	if err != nil || claims.UserID != req.UserID {
		h.respondError(msg, "Unauthorized")
		return
	}

	if err := h.authService.Logout(ctx, req.UserID, req.Token); err != nil {
		h.respondError(msg, "Failed to logout")
		return
	}

	// Publish logout event
	_ = h.publisher.PublishUserLogout(domain.UserLogoutEvent{
		UserID:    req.UserID,
		Timestamp: time.Now(),
	})

	response := SuccessResponse{Message: "Logged out successfully"}
	h.respond(msg, response, nil)
}

// HandleLogoutAll handles logout all requests via NATS
func (h *AuthHandler) HandleLogoutAll(msg *natspkg.Msg) {
	var req LogoutAllRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		h.respondError(msg, "Invalid request format")
		return
	}

	// Validate token first
	ctx := context.Background()
	claims, err := h.authService.ValidateAccessToken(ctx, req.Token)
	if err != nil || claims.UserID != req.UserID {
		h.respondError(msg, "Unauthorized")
		return
	}

	if err := h.authService.LogoutAll(ctx, req.UserID); err != nil {
		h.respondError(msg, "Failed to logout from all devices")
		return
	}

	response := SuccessResponse{Message: "Logged out from all devices successfully"}
	h.respond(msg, response, nil)
}

// HandleChangePassword handles password change requests via NATS
func (h *AuthHandler) HandleChangePassword(msg *natspkg.Msg) {
	var req ChangePasswordRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		h.respondError(msg, "Invalid request format")
		return
	}

	// Validate token first
	ctx := context.Background()
	claims, err := h.authService.ValidateAccessToken(ctx, req.Token)
	if err != nil || claims.UserID != req.UserID {
		h.respondError(msg, "Unauthorized")
		return
	}

	changeReq := req.ToDomain()

	if err := h.authService.ChangePassword(ctx, changeReq); err != nil {
		h.respondError(msg, err.Error())
		return
	}

	// Publish password reset event
	_ = h.publisher.PublishPasswordReset(domain.PasswordResetEvent{
		UserID:    req.UserID,
		Method:    "self-service",
		Timestamp: time.Now(),
	})

	response := SuccessResponse{Message: "Password changed successfully"}
	h.respond(msg, response, nil)
}

// Helper methods
func (h *AuthHandler) respond(msg *natspkg.Msg, data any, err error) {
	resp := Response{
		Success: err == nil,
		Data:    data,
	}
	if err != nil {
		resp.Error = err.Error()
	}

	response, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		return
	}

	if err := msg.Respond(response); err != nil {
		log.Printf("Failed to send response: %v", err)
	}
}

func (h *AuthHandler) respondError(msg *natspkg.Msg, errorMsg string) {
	resp := Response{
		Success: false,
		Error:   errorMsg,
	}

	response, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Failed to marshal error response: %v", err)
		return
	}

	if err := msg.Respond(response); err != nil {
		log.Printf("Failed to send error response: %v", err)
	}
}
