package nats

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"auth/internal/nats"
	"auth/internal/service"

	"github.com/google/uuid"
	natspkg "github.com/nats-io/nats.go"
)

// AuthHandler handles authentication NATS requests
type AuthHandler struct {
	authService *service.AuthService
	publisher   *nats.Publisher
}

// NewAuthHandler creates a new NATS auth handler
func NewAuthHandler(authService *service.AuthService, publisher *nats.Publisher) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		publisher:   publisher,
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	DeviceID  string `json:"device_id,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
	IPAddress string `json:"ip_address,omitempty"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// HandleLogin handles login requests via NATS
func (h *AuthHandler) HandleLogin(msg *natspkg.Msg) {
	var req LoginRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		h.respondError(msg, "Invalid request format")
		return
	}

	ctx := context.Background()
	loginReq := service.LoginRequest{
		Email:     req.Email,
		Password:  req.Password,
		DeviceID:  req.DeviceID,
		UserAgent: req.UserAgent,
		IPAddress: req.IPAddress,
	}

	tokens, userID, err := h.authService.Login(ctx, loginReq)
	if err != nil {
		h.respondError(msg, "Invalid credentials")
		return
	}

	// Publish login event
	_ = h.publisher.PublishUserLogin(nats.UserLoginEvent{
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

	h.respond(msg, response)
}

// ValidateRequest represents a token validation request
type ValidateRequest struct {
	Token string `json:"token"`
}

// ValidateResponse represents a token validation response
type ValidateResponse struct {
	Valid  bool      `json:"valid"`
	UserID uuid.UUID `json:"user_id,omitempty"`
	Role   string    `json:"role,omitempty"`
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
		h.respond(msg, response)
		return
	}

	response := ValidateResponse{
		Valid:  true,
		UserID: claims.UserID,
		Role:   claims.Role,
	}

	h.respond(msg, response)
}

// RefreshRequest represents a token refresh request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
	DeviceID     string `json:"device_id,omitempty"`
	UserAgent    string `json:"user_agent,omitempty"`
	IPAddress    string `json:"ip_address,omitempty"`
}

// HandleRefresh handles token refresh requests via NATS
func (h *AuthHandler) HandleRefresh(msg *natspkg.Msg) {
	var req RefreshRequest
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		h.respondError(msg, "Invalid request format")
		return
	}

	ctx := context.Background()
	refreshReq := service.RefreshRequest{
		RefreshToken: req.RefreshToken,
		DeviceID:     req.DeviceID,
		UserAgent:    req.UserAgent,
		IPAddress:    req.IPAddress,
	}

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

	h.respond(msg, response)
}

// LogoutRequest represents a logout request
type LogoutRequest struct {
	UserID uuid.UUID `json:"user_id"`
	Token  string    `json:"token"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Message string `json:"message"`
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

	if err := h.authService.Logout(ctx, req.UserID); err != nil {
		h.respondError(msg, "Failed to logout")
		return
	}

	// Publish logout event
	_ = h.publisher.PublishUserLogout(nats.UserLogoutEvent{
		UserID:    req.UserID,
		Timestamp: time.Now(),
	})

	response := SuccessResponse{Message: "Logged out successfully"}
	h.respond(msg, response)
}

// LogoutAllRequest represents a logout all request
type LogoutAllRequest struct {
	UserID uuid.UUID `json:"user_id"`
	Token  string    `json:"token"`
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
	h.respond(msg, response)
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	UserID      uuid.UUID `json:"user_id"`
	Token       string    `json:"token"`
	OldPassword string    `json:"old_password"`
	NewPassword string    `json:"new_password"`
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

	changeReq := service.ChangePasswordRequest{
		UserID:      req.UserID,
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	}

	if err := h.authService.ChangePassword(ctx, changeReq); err != nil {
		h.respondError(msg, err.Error())
		return
	}

	// Publish password reset event
	_ = h.publisher.PublishPasswordReset(nats.PasswordResetEvent{
		UserID:    req.UserID,
		Method:    "self-service",
		Timestamp: time.Now(),
	})

	response := SuccessResponse{Message: "Password changed successfully"}
	h.respond(msg, response)
}

// Helper methods
func (h *AuthHandler) respond(msg *natspkg.Msg, data interface{}) {
	response, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		return
	}

	if err := msg.Respond(response); err != nil {
		log.Printf("Failed to send response: %v", err)
	}
}

func (h *AuthHandler) respondError(msg *natspkg.Msg, errorMsg string) {
	h.respond(msg, ErrorResponse{Error: errorMsg})
}
