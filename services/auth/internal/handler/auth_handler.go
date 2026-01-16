package handler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/artmexbet/raibecas/libs/natsw"

	"github.com/artmexbet/raibecas/services/auth/internal/domain"
	"github.com/artmexbet/raibecas/services/auth/pkg/jwt"
)

type AuthService interface {
	ValidateAccessToken(context.Context, string) (*jwt.Claims, error)
	Login(context.Context, domain.LoginRequest) (*domain.TokenPair, uuid.UUID, error)
	RefreshTokens(context.Context, domain.RefreshRequest) (*domain.TokenPair, uuid.UUID, error)
	Logout(ctx context.Context, userID uuid.UUID, token string) error
	LogoutAll(ctx context.Context, userID uuid.UUID) error
	ChangePassword(ctx context.Context, req domain.ChangePasswordRequest) error
}

type EventPublisher interface {
	PublishUserLogin(ctx context.Context, event domain.UserLoginEvent) error
	PublishUserLogout(ctx context.Context, event domain.UserLogoutEvent) error
	PublishPasswordReset(ctx context.Context, event domain.PasswordResetEvent) error
	PublishRegistrationRequested(ctx context.Context, event domain.RegistrationRequestedEvent) error
	PublishUserRegistered(ctx context.Context, event domain.UserRegisteredEvent) error
}

// AuthHandler handles authentication NATS requests
type AuthHandler struct {
	authService AuthService
	publisher   EventPublisher
}

// NewAuthHandler creates a new NATS auth handler
func NewAuthHandler(authService AuthService, publisher EventPublisher) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		publisher:   publisher,
	}
}

// HandleLogin handles login requests via NATS
func (h *AuthHandler) HandleLogin(msg *natsw.Message) error {
	var req LoginRequest
	if err := msg.UnmarshalData(&req); err != nil {
		return h.respondError(msg, "Invalid request format")
	}

	ctx := msg.Ctx
	loginReq := req.ToDomain()

	tokens, userID, err := h.authService.Login(ctx, loginReq)
	if err != nil {
		return h.respondError(msg, fmt.Sprintf("invalid credentials: %v", err))
	}

	// Publish login event
	_ = h.publisher.PublishUserLogin(ctx, domain.UserLoginEvent{
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

	return h.respond(msg, response)
}

// HandleValidate handles token validation requests via NATS
func (h *AuthHandler) HandleValidate(msg *natsw.Message) error {
	var req ValidateRequest
	if err := msg.UnmarshalData(&req); err != nil {
		return h.respondError(msg, "Invalid request format")
	}

	ctx := msg.Ctx
	claims, err := h.authService.ValidateAccessToken(ctx, req.Token)
	if err != nil {
		response := ValidateResponse{Valid: false}
		return h.respond(msg, response)
	}

	response := ValidateResponse{
		Valid:  true,
		UserID: claims.UserID,
		Role:   claims.Role,
	}

	return h.respond(msg, response)
}

// HandleRefresh handles token refresh requests via NATS
func (h *AuthHandler) HandleRefresh(msg *natsw.Message) error {
	var req RefreshRequest
	if err := msg.UnmarshalData(&req); err != nil {
		return h.respondError(msg, "Invalid request format")
	}

	ctx := msg.Ctx
	refreshReq := req.ToDomain()

	tokens, _, err := h.authService.RefreshTokens(ctx, refreshReq)
	if err != nil {
		return h.respondError(msg, "Invalid or expired refresh token")
	}

	response := LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    900,
	}

	return h.respond(msg, response)
}

// HandleLogout handles logout requests via NATS
func (h *AuthHandler) HandleLogout(msg *natsw.Message) error {
	var req LogoutRequest
	if err := msg.UnmarshalData(&req); err != nil {
		return h.respondError(msg, "Invalid request format")
	}

	ctx := msg.Ctx
	claims, err := h.authService.ValidateAccessToken(ctx, req.Token)
	if err != nil || claims.UserID != req.UserID {
		return h.respondError(msg, "Unauthorized")
	}

	if err := h.authService.Logout(ctx, req.UserID, req.Token); err != nil {
		return h.respondError(msg, "Failed to logout")
	}

	// Publish logout event
	_ = h.publisher.PublishUserLogout(ctx, domain.UserLogoutEvent{
		UserID:    req.UserID,
		Timestamp: time.Now(),
	})

	response := SuccessResponse{Message: "Logged out successfully"}
	return h.respond(msg, response)
}

// HandleLogoutAll handles logout all requests via NATS
func (h *AuthHandler) HandleLogoutAll(msg *natsw.Message) error {
	var req LogoutAllRequest
	if err := msg.UnmarshalData(&req); err != nil {
		return h.respondError(msg, "Invalid request format")
	}

	ctx := msg.Ctx
	claims, err := h.authService.ValidateAccessToken(ctx, req.Token)
	if err != nil || claims.UserID != req.UserID {
		return h.respondError(msg, "Unauthorized")
	}

	if err := h.authService.LogoutAll(ctx, req.UserID); err != nil {
		return h.respondError(msg, "Failed to logout from all devices")
	}

	response := SuccessResponse{Message: "Logged out from all devices successfully"}
	return h.respond(msg, response)
}

// HandleChangePassword handles password change requests via NATS
func (h *AuthHandler) HandleChangePassword(msg *natsw.Message) error {
	var req ChangePasswordRequest
	if err := msg.UnmarshalData(&req); err != nil {
		return h.respondError(msg, "Invalid request format")
	}

	ctx := msg.Ctx
	claims, err := h.authService.ValidateAccessToken(ctx, req.Token)
	if err != nil || claims.UserID != req.UserID {
		return h.respondError(msg, "Unauthorized")
	}

	changeReq := req.ToDomain()

	if err := h.authService.ChangePassword(ctx, changeReq); err != nil {
		return h.respondError(msg, err.Error())
	}

	// Publish password reset event
	_ = h.publisher.PublishPasswordReset(ctx, domain.PasswordResetEvent{
		UserID:    req.UserID,
		Method:    "self-service",
		Timestamp: time.Now(),
	})

	response := SuccessResponse{Message: "Password changed successfully"}
	return h.respond(msg, response)
}

// Helper methods
func (h *AuthHandler) respond(msg *natsw.Message, data any) error {
	resp := Response{
		Success: true,
		Data:    data,
	}

	return msg.RespondJSON(resp)
}

func (h *AuthHandler) respondError(msg *natsw.Message, errorMsg string) error {
	resp := Response{
		Success: false,
		Error:   errorMsg,
	}

	if err := msg.RespondJSON(resp); err != nil {
		slog.ErrorContext(msg.Ctx, "Failed to send error response", "error", err)
		return err
	}
	return fmt.Errorf("%s", errorMsg)
}
