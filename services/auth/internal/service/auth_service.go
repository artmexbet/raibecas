package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/artmexbet/raibecas/services/auth/internal/domain"
	"github.com/artmexbet/raibecas/services/auth/pkg/jwt"
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
}

// AuthService handles authentication business logic
type AuthService struct {
	userRepo   UserRepository
	jwtManager jwt.TokenManager
	bcryptCost int
}

// NewAuthService creates a new auth service
func NewAuthService(
	userRepo UserRepository,
	jwtManager jwt.TokenManager,
) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		jwtManager: jwtManager,
		bcryptCost: 12, // Default bcrypt cost
	}
}

// Login authenticates a user and returns tokens with enhanced security
func (s *AuthService) Login(ctx context.Context, req domain.LoginRequest) (*domain.LoginResult, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, domain.ErrUserNotFound
	}

	// Check if user is active
	if !user.IsActive {
		return nil, domain.ErrUserNotActive
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	// Генерируем fingerprint для защиты от XSS
	fingerprint, err := jwt.GenerateFingerprint()
	if err != nil {
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
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Генерируем refresh token
	refreshToken, refreshMetadata, err := s.jwtManager.GenerateRefreshToken(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Сохраняем refresh token в хранилище
	if err := s.jwtManager.StoreRefreshToken(ctx, refreshMetadata); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &domain.LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenID:      refreshMetadata.TokenID,
		Fingerprint:  fingerprint,
		UserID:       user.ID,
	}, nil
}

// Logout logs out a user by revoking their refresh token
func (s *AuthService) Logout(ctx context.Context, tokenID string, accessTokenJTI string) error {
	// Отзываем refresh token
	if err := s.jwtManager.RevokeRefreshToken(ctx, tokenID); err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	// Добавляем access token в blacklist
	if accessTokenJTI != "" {
		if err := s.jwtManager.RevokeAccessToken(ctx, accessTokenJTI); err != nil {
			// Логируем ошибку, но не прерываем операцию
			// так как refresh token уже отозван
			return fmt.Errorf("failed to blacklist access token: %w", err)
		}
	}

	return nil
}

// LogoutAll logs out a user from all devices
func (s *AuthService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	if err := s.jwtManager.RevokeAllUserTokens(ctx, userID); err != nil {
		return fmt.Errorf("failed to revoke all tokens: %w", err)
	}
	return nil
}

// RefreshTokens refreshes access and refresh tokens with rotation
func (s *AuthService) RefreshTokens(ctx context.Context, req domain.RefreshRequest, fingerprint string) (*domain.LoginResult, error) {
	// Получаем refresh token metadata через валидацию
	oldMetadata, err := s.jwtManager.ValidateRefreshToken(ctx, req.TokenID, fingerprint)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	// Проверяем, что пользователь всё ещё активен
	user, err := s.userRepo.GetByID(ctx, oldMetadata.UserID)
	if err != nil {
		return nil, domain.ErrUserNotFound
	}

	if !user.IsActive {
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
		return nil, fmt.Errorf("failed to rotate tokens: %w", err)
	}

	return &domain.LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenID:      oldMetadata.TokenID, // Новый ID будет в refresh token
		Fingerprint:  fingerprint,
		UserID:       user.ID,
	}, nil
}

// ValidateAccessToken validates an access token and returns user info
func (s *AuthService) ValidateAccessToken(ctx context.Context, token string, fingerprint string) (*jwt.AccessTokenClaims, error) {
	result, err := s.jwtManager.ValidateAccessToken(ctx, token, fingerprint)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	if !result.Valid {
		return nil, domain.ErrInvalidToken
	}

	// Optionally verify user still exists and is active
	user, err := s.userRepo.GetByID(ctx, result.Claims.UserID)
	if err != nil {
		return nil, domain.ErrUserNotFound
	}

	if !user.IsActive {
		return nil, domain.ErrUserNotActive
	}

	return result.Claims, nil
}

// ChangePassword changes a user's password
func (s *AuthService) ChangePassword(ctx context.Context, req domain.ChangePasswordRequest) error {
	// Get user
	user, err := s.userRepo.GetByID(ctx, req.UserID)
	if err != nil {
		return err
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		return domain.ErrInvalidCredentials
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), s.bcryptCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	if err := s.userRepo.UpdatePassword(ctx, req.UserID, string(hashedPassword)); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Logout from all devices for security
	_ = s.LogoutAll(ctx, req.UserID)

	return nil
}
