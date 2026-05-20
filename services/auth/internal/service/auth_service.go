package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/bcrypt"

	"github.com/artmexbet/raibecas/services/auth/internal/domain"
	"github.com/artmexbet/raibecas/services/auth/pkg/jwt"
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	CreateUser(ctx context.Context, user *domain.User) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
	ExistsUserByEmail(ctx context.Context, email string) (bool, error)
	ExistsUserByUsername(ctx context.Context, username string) (bool, error)
}

// AuthService handles authentication business logic
type AuthService struct {
	userRepo   UserRepository
	jwtManager jwt.TokenManager
	bcryptCost int
	tracer     trace.Tracer
	logger     *slog.Logger
}

// NewAuthService creates a new auth service
func NewAuthService(
	userRepo UserRepository,
	jwtManager jwt.TokenManager,
	tracer trace.Tracer,
	logger *slog.Logger,
) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		jwtManager: jwtManager,
		bcryptCost: 12, // Default bcrypt cost
		tracer:     tracer,
		logger:     logger,
	}
}

// Login authenticates a user and returns tokens with enhanced security
func (s *AuthService) Login(ctx context.Context, req domain.LoginRequest) (*domain.LoginResult, error) {
	ctx, span := s.tracer.Start(ctx, "auth.service.login",
		trace.WithAttributes(attribute.String("auth.email", req.Email)),
	)
	defer span.End()

	// Get user by email
	user, err := s.userRepo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		span.SetStatus(codes.Error, "user not found")
		s.logger.WarnContext(ctx, "login attempt for non-existent user", "email", req.Email)
		return nil, domain.ErrUserNotFound
	}

	// Check if user is active
	if !user.IsActive {
		span.SetStatus(codes.Error, "user not active")
		s.logger.WarnContext(ctx, "login attempt for inactive user", "user_id", user.ID, "email", req.Email)
		return nil, domain.ErrUserNotActive
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		span.SetStatus(codes.Error, "invalid credentials")
		s.logger.WarnContext(ctx, "invalid credentials", "user_id", user.ID, "email", req.Email)
		return nil, domain.ErrInvalidCredentials
	}

	// Генерируем fingerprint для защиты от XSS
	fingerprint, err := jwt.GenerateFingerprint()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "fingerprint generation failed")
		s.logger.ErrorContext(ctx, "failed to generate fingerprint", "error", err)
		return nil, fmt.Errorf("failed to generate fingerprint: %w", err)
	}

	// Создаём метаданные для токенов
	metadata := &jwt.TokenMetadata{
		UserID:      user.ID,
		Role:        string(user.Role),
		DeviceID:    req.DeviceID,
		UserAgent:   req.UserAgent,
		IPAddress:   req.IPAddress,
		Fingerprint: fingerprint,
		// TokenFamily будет создан автоматически в GenerateRefreshToken
	}

	// Генерируем access token
	accessToken, _, err := s.jwtManager.GenerateAccessToken(metadata)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "access token generation failed")
		s.logger.ErrorContext(ctx, "failed to generate access token", "user_id", user.ID, "error", err)
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Генерируем refresh token
	refreshToken, refreshMetadata, err := s.jwtManager.GenerateRefreshToken(metadata)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "refresh token generation failed")
		s.logger.ErrorContext(ctx, "failed to generate refresh token", "user_id", user.ID, "error", err)
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Сохраняем refresh token в хранилище
	if err := s.jwtManager.StoreRefreshToken(ctx, refreshMetadata); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "refresh token storage failed")
		s.logger.ErrorContext(ctx, "failed to store refresh token", "user_id", user.ID, "error", err)
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	span.SetAttributes(attribute.String("auth.user_id", user.ID.String()))
	s.logger.InfoContext(ctx, "user logged in successfully", "user_id", user.ID, "email", req.Email)

	return &domain.LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenID:      refreshMetadata.TokenID,
		Fingerprint:  fingerprint,
		User:         user,
	}, nil
}

// Logout logs out a user by revoking their refresh token
func (s *AuthService) Logout(ctx context.Context, tokenID string, accessTokenJTI string) error {
	ctx, span := s.tracer.Start(ctx, "auth.service.logout",
		trace.WithAttributes(attribute.String("auth.token_id", tokenID)),
	)
	defer span.End()

	// Отзываем refresh token
	if err := s.jwtManager.RevokeRefreshToken(ctx, tokenID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "revoke refresh token failed")
		s.logger.ErrorContext(ctx, "failed to revoke refresh token", "token_id", tokenID, "error", err)
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	// Добавляем access token в blacklist
	if accessTokenJTI != "" {
		if err := s.jwtManager.RevokeAccessToken(ctx, accessTokenJTI); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "blacklist access token failed")
			s.logger.ErrorContext(ctx, "failed to blacklist access token", "jti", accessTokenJTI, "error", err)
			return fmt.Errorf("failed to blacklist access token: %w", err)
		}
	}

	s.logger.InfoContext(ctx, "user logged out", "token_id", tokenID)
	return nil
}

// LogoutAll logs out a user from all devices
func (s *AuthService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	ctx, span := s.tracer.Start(ctx, "auth.service.logout_all",
		trace.WithAttributes(attribute.String("auth.user_id", userID.String())),
	)
	defer span.End()

	if err := s.jwtManager.RevokeAllUserTokens(ctx, userID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "revoke all tokens failed")
		s.logger.ErrorContext(ctx, "failed to revoke all tokens", "user_id", userID, "error", err)
		return fmt.Errorf("failed to revoke all tokens: %w", err)
	}

	s.logger.InfoContext(ctx, "user logged out from all devices", "user_id", userID)
	return nil
}

// RefreshTokens refreshes access and refresh tokens with rotation
func (s *AuthService) RefreshTokens(ctx context.Context, req domain.RefreshRequest, fingerprint string) (*domain.LoginResult, error) {
	ctx, span := s.tracer.Start(ctx, "auth.service.refresh_tokens")
	defer span.End()

	// Получаем refresh token metadata через валидацию
	oldMetadata, err := s.jwtManager.ValidateRefreshToken(ctx, req.TokenID, fingerprint)
	if err != nil {
		span.SetStatus(codes.Error, "invalid refresh token")
		s.logger.WarnContext(ctx, "refresh token validation failed", "token_id", req.TokenID, "error", err)
		return nil, domain.ErrInvalidToken
	}

	span.SetAttributes(attribute.String("auth.user_id", oldMetadata.UserID.String()))

	// Проверяем, что пользователь всё ещё активен
	user, err := s.userRepo.GetUserByID(ctx, oldMetadata.UserID)
	if err != nil {
		span.SetStatus(codes.Error, "user not found")
		s.logger.WarnContext(ctx, "user not found during token refresh", "user_id", oldMetadata.UserID)
		return nil, domain.ErrUserNotFound
	}

	if !user.IsActive {
		span.SetStatus(codes.Error, "user not active")
		s.logger.WarnContext(ctx, "token refresh for inactive user", "user_id", user.ID)
		return nil, domain.ErrUserNotActive
	}

	// Создаём метаданные для новых токенов
	metadata := &jwt.TokenMetadata{
		UserID:      user.ID,
		Role:        string(user.Role),
		DeviceID:    req.DeviceID,
		UserAgent:   req.UserAgent,
		IPAddress:   req.IPAddress,
		Fingerprint: fingerprint,
		TokenFamily: oldMetadata.TokenFamily, // Сохраняем семью
	}

	// Выполняем ротацию токенов
	accessToken, refreshToken, err := s.jwtManager.RotateRefreshToken(ctx, req.TokenID, metadata)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "token rotation failed")
		s.logger.ErrorContext(ctx, "failed to rotate tokens", "user_id", user.ID, "error", err)
		return nil, fmt.Errorf("failed to rotate tokens: %w", err)
	}

	s.logger.InfoContext(ctx, "tokens refreshed successfully", "user_id", user.ID)

	return &domain.LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenID:      oldMetadata.TokenID, // Новый ID будет в refresh token
		Fingerprint:  fingerprint,
		User:         user,
	}, nil
}

// ValidateAccessToken validates an access token and returns user info
func (s *AuthService) ValidateAccessToken(ctx context.Context, token string, fingerprint string) (*jwt.AccessTokenClaims, error) {
	ctx, span := s.tracer.Start(ctx, "auth.service.validate_token")
	defer span.End()

	result, err := s.jwtManager.ValidateAccessToken(ctx, token, fingerprint)
	if err != nil {
		span.SetStatus(codes.Error, "invalid token")
		return nil, domain.ErrInvalidToken
	}

	if !result.Valid {
		span.SetStatus(codes.Error, "token not valid")
		return nil, domain.ErrInvalidToken
	}

	// Optionally verify user still exists and is active
	user, err := s.userRepo.GetUserByID(ctx, result.Claims.UserID)
	if err != nil {
		span.SetStatus(codes.Error, "user not found")
		return nil, domain.ErrUserNotFound
	}

	if !user.IsActive {
		span.SetStatus(codes.Error, "user not active")
		return nil, domain.ErrUserNotActive
	}

	span.SetAttributes(
		attribute.String("auth.user_id", result.Claims.UserID.String()),
		attribute.String("auth.role", result.Claims.Role),
	)

	return result.Claims, nil
}

// ChangePassword changes a user's password
func (s *AuthService) ChangePassword(ctx context.Context, req domain.ChangePasswordRequest) error {
	ctx, span := s.tracer.Start(ctx, "auth.service.change_password",
		trace.WithAttributes(attribute.String("auth.user_id", req.UserID.String())),
	)
	defer span.End()

	// Get user
	user, err := s.userRepo.GetUserByID(ctx, req.UserID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "user not found")
		s.logger.ErrorContext(ctx, "failed to get user for password change", "user_id", req.UserID, "error", err)
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		span.SetStatus(codes.Error, "invalid old password")
		s.logger.WarnContext(ctx, "invalid old password during change", "user_id", req.UserID)
		return domain.ErrInvalidCredentials
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), s.bcryptCost)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "password hashing failed")
		s.logger.ErrorContext(ctx, "failed to hash new password", "user_id", req.UserID, "error", err)
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	if err := s.userRepo.UpdatePassword(ctx, req.UserID, string(hashedPassword)); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "password update failed")
		s.logger.ErrorContext(ctx, "failed to update password", "user_id", req.UserID, "error", err)
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Logout from all devices for security (don't block on error)
	if err := s.LogoutAll(ctx, req.UserID); err != nil {
		// Log but continue - password was successfully updated
		s.logger.WarnContext(ctx, "failed to logout all sessions after password change", "user_id", req.UserID, "error", err)
	}

	s.logger.InfoContext(ctx, "password changed successfully", "user_id", req.UserID)
	return nil
}
