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
	ValidateAccessToken(ctx context.Context, token string, fingerprint string) (*jwt.AccessTokenClaims, error)
	Login(ctx context.Context, req domain.LoginRequest) (*domain.LoginResult, error)
	RefreshTokens(ctx context.Context, req domain.RefreshRequest, fingerprint string) (*domain.LoginResult, error)
	Logout(ctx context.Context, tokenID string, accessTokenJTI string) error
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
		slog.ErrorContext(msg.Ctx, "invalid login request format", "error", err)
		return h.respondError(msg, "Invalid request format")
	}

	ctx := msg.Ctx
	loginReq := req.ToDomain()

	result, err := h.authService.Login(ctx, loginReq)
	if err != nil {
		slog.ErrorContext(ctx, "login failed", "email", req.Email, "error", err)
		return h.respondError(msg, fmt.Sprintf("cannot login: %v", err))
	}

	// Publish login event asynchronously
	go func() {
		if err := h.publisher.PublishUserLogin(ctx, domain.UserLoginEvent{
			User:      result.User,
			DeviceID:  req.DeviceID,
			UserAgent: req.UserAgent,
			IPAddress: req.IPAddress,
			Timestamp: time.Now(),
		}); err != nil {
			slog.ErrorContext(ctx, "failed to publish login event", "user_id", result.User.ID, "error", err)
		}
	}()

	response := LoginResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		TokenID:      result.TokenID,
		Fingerprint:  result.Fingerprint,
		ExpiresIn:    900, // 15 minutes
		User:         result.User,
	}

	return h.respond(msg, response)
}

// HandleValidate handles token validation requests via NATS
func (h *AuthHandler) HandleValidate(msg *natsw.Message) error {
	var req ValidateRequest
	if err := msg.UnmarshalData(&req); err != nil {
		return h.respondError(msg, "Invalid request format")
	}

	// Проверяем наличие fingerprint
	if req.Fingerprint == "" {
		response := ValidateResponse{Valid: false}
		return h.respond(msg, response)
	}

	ctx := msg.Ctx
	claims, err := h.authService.ValidateAccessToken(ctx, req.Token, req.Fingerprint)
	if err != nil {
		response := ValidateResponse{Valid: false}
		return h.respond(msg, response)
	}

	response := ValidateResponse{
		Valid:  true,
		UserID: claims.UserID,
		Role:   claims.Role,
		JTI:    claims.JTI,
	}

	return h.respond(msg, response)
}

// HandleRefresh handles token refresh requests via NATS
func (h *AuthHandler) HandleRefresh(msg *natsw.Message) error {
	var req RefreshRequest
	if err := msg.UnmarshalData(&req); err != nil {
		return h.respondError(msg, "Invalid request format")
	}

	// Проверяем наличие fingerprint
	if req.Fingerprint == "" {
		return h.respondError(msg, "Fingerprint is required")
	}

	ctx := msg.Ctx
	refreshReq := req.ToDomain()

	result, err := h.authService.RefreshTokens(ctx, refreshReq, req.Fingerprint)
	if err != nil {
		return h.respondError(msg, "Invalid or expired refresh token")
	}

	response := LoginResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		TokenID:      result.TokenID,
		Fingerprint:  result.Fingerprint,
		ExpiresIn:    900,
		User:         result.User,
	}

	return h.respond(msg, response)
}

// HandleLogout handles logout requests via NATS
func (h *AuthHandler) HandleLogout(msg *natsw.Message) error {
	var req LogoutRequest
	if err := msg.UnmarshalData(&req); err != nil {
		slog.ErrorContext(msg.Ctx, "invalid logout request format", "error", err)
		return h.respondError(msg, "Invalid request format")
	}

	ctx := msg.Ctx

	if err := h.authService.Logout(ctx, req.TokenID, req.AccessTokenJTI); err != nil {
		slog.ErrorContext(ctx, "logout failed", "user_id", req.UserID, "error", err)
		return h.respondError(msg, fmt.Sprintf("Failed to logout: %v", err))
	}

	// Publish logout event asynchronously
	go func() {
		if err := h.publisher.PublishUserLogout(ctx, domain.UserLogoutEvent{
			UserID:    req.UserID,
			Timestamp: time.Now(),
		}); err != nil {
			slog.ErrorContext(ctx, "failed to publish logout event", "user_id", req.UserID, "error", err)
		}
	}()

	response := SuccessResponse{Message: "Logged out successfully"}
	return h.respond(msg, response)
}

// HandleLogoutAll handles logout all requests via NATS
func (h *AuthHandler) HandleLogoutAll(msg *natsw.Message) error {
	var req LogoutAllRequest
	if err := msg.UnmarshalData(&req); err != nil {
		slog.ErrorContext(msg.Ctx, "invalid logout_all request format", "error", err)
		return h.respondError(msg, "Invalid request format")
	}

	ctx := msg.Ctx

	if err := h.authService.LogoutAll(ctx, req.UserID); err != nil {
		slog.ErrorContext(ctx, "logout_all failed", "user_id", req.UserID, "error", err)
		return h.respondError(msg, "Failed to logout from all devices")
	}

	// Publish logout event asynchronously
	go func() {
		if err := h.publisher.PublishUserLogout(ctx, domain.UserLogoutEvent{
			UserID:    req.UserID,
			Timestamp: time.Now(),
		}); err != nil {
			slog.ErrorContext(ctx, "failed to publish logout event", "user_id", req.UserID, "error", err)
		}
	}()

	response := SuccessResponse{Message: "Logged out from all devices successfully"}
	return h.respond(msg, response)
}

// HandleChangePassword handles password change requests via NATS
func (h *AuthHandler) HandleChangePassword(msg *natsw.Message) error {
	var req ChangePasswordRequest
	if err := msg.UnmarshalData(&req); err != nil {
		slog.ErrorContext(msg.Ctx, "invalid change password request format", "error", err)
		return h.respondError(msg, "Invalid request format")
	}

	ctx := msg.Ctx
	changeReq := req.ToDomain()

	if err := h.authService.ChangePassword(ctx, changeReq); err != nil {
		slog.ErrorContext(ctx, "change password failed", "user_id", req.UserID, "error", err)
		return h.respondError(msg, err.Error())
	}

	// Publish password reset event asynchronously
	go func() {
		if err := h.publisher.PublishPasswordReset(ctx, domain.PasswordResetEvent{
			UserID:    req.UserID,
			Method:    "self-service",
			Timestamp: time.Now(),
		}); err != nil {
			slog.ErrorContext(ctx, "failed to publish password reset event", "user_id", req.UserID, "error", err)
		}
	}()

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
