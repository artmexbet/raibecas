package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/artmexbet/raibecas/libs/dto"
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
	tracer      trace.Tracer
}

// NewAuthHandler creates a new NATS auth handler
func NewAuthHandler(authService AuthService, publisher EventPublisher, tracer trace.Tracer) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		publisher:   publisher,
		tracer:      tracer,
	}
}

// HandleLogin handles login requests via NATS
func (h *AuthHandler) HandleLogin(msg *natsw.Message) error {
	ctx, span := h.tracer.Start(msg.Ctx, "auth.handler.login")
	defer span.End()

	var req LoginRequest
	if err := msg.UnmarshalData(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request format")
		slog.ErrorContext(ctx, "invalid login request format", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	span.SetAttributes(attribute.String("auth.email", req.Email))

	result, err := h.authService.Login(ctx, req.ToDomain())
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "login failed")
		slog.ErrorContext(ctx, "login failed", "email", req.Email, "error", err)
		return h.respondError(msg, mapDomainErrorToCode(err))
	}

	span.SetAttributes(attribute.String("auth.user_id", result.User.ID.String()))

	// Publish login event asynchronously
	go func() {
		if pubErr := h.publisher.PublishUserLogin(ctx, domain.UserLoginEvent{
			User:      result.User,
			DeviceID:  req.DeviceID,
			UserAgent: req.UserAgent,
			IPAddress: req.IPAddress,
			Timestamp: time.Now(),
		}); pubErr != nil {
			slog.ErrorContext(ctx, "failed to publish login event", "user_id", result.User.ID, "error", pubErr)
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
	ctx, span := h.tracer.Start(msg.Ctx, "auth.handler.validate")
	defer span.End()

	var req ValidateRequest
	if err := msg.UnmarshalData(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request format")
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	// Проверяем наличие fingerprint (кроме WS-соединений, где браузер не может его передать)
	if !req.SkipFingerprint && req.Fingerprint == "" {
		span.SetAttributes(attribute.Bool("auth.valid", false))
		response := ValidateResponse{Valid: false}
		return h.respond(msg, response)
	}

	claims, err := h.authService.ValidateAccessToken(ctx, req.Token, req.Fingerprint)
	if err != nil {
		span.SetAttributes(attribute.Bool("auth.valid", false))
		response := ValidateResponse{Valid: false}
		return h.respond(msg, response)
	}

	span.SetAttributes(
		attribute.Bool("auth.valid", true),
		attribute.String("auth.user_id", claims.UserID.String()),
		attribute.String("auth.role", claims.Role),
	)

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
	ctx, span := h.tracer.Start(msg.Ctx, "auth.handler.refresh")
	defer span.End()

	var req RefreshRequest
	if err := msg.UnmarshalData(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request format")
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	// Проверяем наличие fingerprint
	if req.Fingerprint == "" {
		span.SetStatus(codes.Error, "fingerprint required")
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	result, err := h.authService.RefreshTokens(ctx, req.ToDomain(), req.Fingerprint)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "refresh failed")
		slog.WarnContext(ctx, "token refresh failed", "error", err)
		return h.respondError(msg, mapDomainErrorToCode(err))
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
	ctx, span := h.tracer.Start(msg.Ctx, "auth.handler.logout")
	defer span.End()

	var req LogoutRequest
	if err := msg.UnmarshalData(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request format")
		slog.ErrorContext(ctx, "invalid logout request format", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	span.SetAttributes(attribute.String("auth.user_id", req.UserID.String()))

	if err := h.authService.Logout(ctx, req.TokenID, req.AccessTokenJTI); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "logout failed")
		slog.ErrorContext(ctx, "logout failed", "user_id", req.UserID, "error", err)
		return h.respondError(msg, mapDomainErrorToCode(err))
	}

	// Publish logout event asynchronously
	go func() {
		if pubErr := h.publisher.PublishUserLogout(ctx, domain.UserLogoutEvent{
			UserID:    req.UserID,
			Timestamp: time.Now(),
		}); pubErr != nil {
			slog.ErrorContext(ctx, "failed to publish logout event", "user_id", req.UserID, "error", pubErr)
		}
	}()

	response := SuccessResponse{Message: "Logged out successfully"}
	return h.respond(msg, response)
}

// HandleLogoutAll handles logout all requests via NATS
func (h *AuthHandler) HandleLogoutAll(msg *natsw.Message) error {
	ctx, span := h.tracer.Start(msg.Ctx, "auth.handler.logout_all")
	defer span.End()

	var req LogoutAllRequest
	if err := msg.UnmarshalData(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request format")
		slog.ErrorContext(ctx, "invalid logout_all request format", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	span.SetAttributes(attribute.String("auth.user_id", req.UserID.String()))

	if err := h.authService.LogoutAll(ctx, req.UserID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "logout_all failed")
		slog.ErrorContext(ctx, "logout_all failed", "user_id", req.UserID, "error", err)
		return h.respondError(msg, mapDomainErrorToCode(err))
	}

	// Publish logout event asynchronously
	go func() {
		if pubErr := h.publisher.PublishUserLogout(ctx, domain.UserLogoutEvent{
			UserID:    req.UserID,
			Timestamp: time.Now(),
		}); pubErr != nil {
			slog.ErrorContext(ctx, "failed to publish logout event", "user_id", req.UserID, "error", pubErr)
		}
	}()

	response := SuccessResponse{Message: "Logged out from all devices successfully"}
	return h.respond(msg, response)
}

// HandleChangePassword handles password change requests via NATS
func (h *AuthHandler) HandleChangePassword(msg *natsw.Message) error {
	ctx, span := h.tracer.Start(msg.Ctx, "auth.handler.change_password")
	defer span.End()

	var req ChangePasswordRequest
	if err := msg.UnmarshalData(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request format")
		slog.ErrorContext(ctx, "invalid change password request format", "error", err)
		return h.respondError(msg, dto.ErrCodeInvalidRequest)
	}

	span.SetAttributes(attribute.String("auth.user_id", req.UserID.String()))

	if err := h.authService.ChangePassword(ctx, req.ToDomain()); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "change password failed")
		slog.ErrorContext(ctx, "change password failed", "user_id", req.UserID, "error", err)
		return h.respondError(msg, mapDomainErrorToCode(err))
	}

	// Publish password reset event asynchronously
	go func() {
		if pubErr := h.publisher.PublishPasswordReset(ctx, domain.PasswordResetEvent{
			UserID:    req.UserID,
			Method:    "self-service",
			Timestamp: time.Now(),
		}); pubErr != nil {
			slog.ErrorContext(ctx, "failed to publish password reset event", "user_id", req.UserID, "error", pubErr)
		}
	}()

	response := SuccessResponse{Message: "Password changed successfully"}
	return h.respond(msg, response)
}

// mapDomainErrorToCode maps domain errors to standard dto error codes.
func mapDomainErrorToCode(err error) dto.ErrorCode {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials):
		return dto.ErrCodeUnauthorized
	case errors.Is(err, domain.ErrUserNotFound):
		return dto.ErrCodeNotFound
	case errors.Is(err, domain.ErrUserNotActive):
		return dto.ErrCodeForbidden
	case errors.Is(err, domain.ErrInvalidToken),
		errors.Is(err, domain.ErrExpiredToken),
		errors.Is(err, domain.ErrTokenNotFound):
		return dto.ErrCodeUnauthorized
	case errors.Is(err, domain.ErrUsernameAlreadyExists),
		errors.Is(err, domain.ErrEmailAlreadyExists),
		errors.Is(err, domain.ErrInvalidEmail),
		errors.Is(err, domain.ErrInvalidPassword),
		errors.Is(err, domain.ErrRegistrationNotPending):
		return dto.ErrCodeInvalidRequest
	case errors.Is(err, domain.ErrRegistrationNotFound):
		return dto.ErrCodeNotFound
	default:
		return dto.ErrCodeInternal
	}
}

// Helper methods
func (h *AuthHandler) respond(msg *natsw.Message, data any) error {
	resp := Response{
		Success: true,
		Data:    data,
	}

	return msg.RespondJSON(resp)
}

func (h *AuthHandler) respondError(msg *natsw.Message, errCode dto.ErrorCode) error {
	resp := Response{
		Success: false,
		Error:   string(errCode),
	}

	if err := msg.RespondJSON(resp); err != nil {
		slog.ErrorContext(msg.Ctx, "Failed to send error response", "error", err)
		return err
	}
	return fmt.Errorf("%s", string(errCode))
}
