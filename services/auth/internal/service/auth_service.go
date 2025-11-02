package service

import (
	"context"
	"fmt"
	"time"

	"auth/internal/domain"
	"auth/pkg/jwt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// IUserRepository defines the interface for user data access
type IUserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
}

// ITokenStore defines the interface for token storage
type ITokenStore interface {
	StoreRefreshToken(ctx context.Context, token *domain.RefreshToken, ttl time.Duration) error
	GetRefreshTokenByValue(ctx context.Context, tokenValue string) (*domain.RefreshToken, error)
	DeleteRefreshToken(ctx context.Context, userID uuid.UUID, tokenValue string) error
	DeleteAllRefreshTokens(ctx context.Context, userID uuid.UUID) error
}

// AuthService handles authentication business logic
type AuthService struct {
	userRepo   IUserRepository
	tokenStore ITokenStore
	jwtManager *jwt.Manager
	bcryptCost int
}

// NewAuthService creates a new auth service
func NewAuthService(
	userRepo IUserRepository,
	tokenStore ITokenStore,
	jwtManager *jwt.Manager,
) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		tokenStore: tokenStore,
		jwtManager: jwtManager,
		bcryptCost: 12, // Default bcrypt cost
	}
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(ctx context.Context, req domain.LoginRequest) (*domain.TokenPair, uuid.UUID, error) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, uuid.Nil, domain.ErrInvalidCredentials
	}

	// Check if user is active
	if !user.IsActive {
		return nil, uuid.Nil, domain.ErrUserNotActive
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, uuid.Nil, domain.ErrInvalidCredentials
	}

	// Generate tokens
	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, string(user.Role))
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken()
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store refresh token in Redis
	refreshTokenData := &domain.RefreshToken{
		Token:     refreshToken,
		UserID:    user.ID,
		DeviceID:  req.DeviceID,
		UserAgent: req.UserAgent,
		IPAddress: req.IPAddress,
		ExpiresAt: time.Now().Add(s.jwtManager.GetRefreshTokenTTL()),
		CreatedAt: time.Now(),
	}

	if err := s.tokenStore.StoreRefreshToken(ctx, refreshTokenData, s.jwtManager.GetRefreshTokenTTL()); err != nil {
		return nil, uuid.Nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, user.ID, nil
}

// Logout logs out a user by revoking their refresh token
func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID, token string) error {
	if err := s.tokenStore.DeleteRefreshToken(ctx, userID, token); err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}
	return nil
}

// LogoutAll logs out a user from all devices
func (s *AuthService) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	if err := s.tokenStore.DeleteAllRefreshTokens(ctx, userID); err != nil {
		return fmt.Errorf("failed to delete all refresh tokens: %w", err)
	}
	return nil
}

// RefreshTokens refreshes access and refresh tokens
func (s *AuthService) RefreshTokens(ctx context.Context, req domain.RefreshRequest) (*domain.TokenPair, uuid.UUID, error) {
	// Get refresh token from storage by token value
	storedToken, err := s.tokenStore.GetRefreshTokenByValue(ctx, req.RefreshToken)
	if err != nil {
		return nil, uuid.Nil, domain.ErrInvalidToken
	}

	// Check if token has expired
	if time.Now().After(storedToken.ExpiresAt) {
		// Clean up expired token
		_ = s.tokenStore.DeleteRefreshToken(ctx, storedToken.UserID, storedToken.Token)
		return nil, uuid.Nil, domain.ErrExpiredToken
	}

	// Verify user still exists and is active
	user, err := s.userRepo.GetByID(ctx, storedToken.UserID)
	if err != nil {
		return nil, uuid.Nil, domain.ErrUserNotFound
	}

	if !user.IsActive {
		return nil, uuid.Nil, domain.ErrUserNotActive
	}

	// Generate new access token
	accessToken, err := s.jwtManager.GenerateAccessToken(user.ID, string(user.Role))
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate new refresh token
	newRefreshToken, err := s.jwtManager.GenerateRefreshToken()
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Delete old refresh token
	if err := s.tokenStore.DeleteRefreshToken(ctx, storedToken.UserID, storedToken.Token); err != nil {
		return nil, uuid.Nil, fmt.Errorf("failed to delete old refresh token: %w", err)
	}

	// Store new refresh token
	newRefreshTokenData := &domain.RefreshToken{
		Token:     newRefreshToken,
		UserID:    user.ID,
		DeviceID:  req.DeviceID,
		UserAgent: req.UserAgent,
		IPAddress: req.IPAddress,
		ExpiresAt: time.Now().Add(s.jwtManager.GetRefreshTokenTTL()),
		CreatedAt: time.Now(),
	}

	if err := s.tokenStore.StoreRefreshToken(ctx, newRefreshTokenData, s.jwtManager.GetRefreshTokenTTL()); err != nil {
		return nil, uuid.Nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, user.ID, nil
}

// ValidateAccessToken validates an access token and returns user info
func (s *AuthService) ValidateAccessToken(ctx context.Context, token string) (*jwt.Claims, error) {
	claims, err := s.jwtManager.ValidateAccessToken(token)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}

	// Optionally verify user still exists and is active
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, domain.ErrUserNotFound
	}

	if !user.IsActive {
		return nil, domain.ErrUserNotActive
	}

	return claims, nil
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
