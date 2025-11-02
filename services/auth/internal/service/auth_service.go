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

// AuthService handles authentication business logic
type AuthService struct {
	userRepo   domain.UserRepository
	tokenStore domain.TokenStore
	jwtManager *jwt.Manager
	bcryptCost int
}

// NewAuthService creates a new auth service
func NewAuthService(
	userRepo domain.UserRepository,
	tokenStore domain.TokenStore,
	jwtManager *jwt.Manager,
) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		tokenStore: tokenStore,
		jwtManager: jwtManager,
		bcryptCost: 12, // Default bcrypt cost
	}
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email     string
	Password  string
	DeviceID  string
	UserAgent string
	IPAddress string
}

// TokenPair represents access and refresh tokens
type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*TokenPair, uuid.UUID, error) {
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

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, user.ID, nil
}

// Logout logs out a user by revoking their refresh token
func (s *AuthService) Logout(ctx context.Context, userID uuid.UUID) error {
	if err := s.tokenStore.DeleteRefreshToken(ctx, userID); err != nil {
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

// RefreshRequest represents a token refresh request
type RefreshRequest struct {
	RefreshToken string
	DeviceID     string
	UserAgent    string
	IPAddress    string
}

// RefreshTokens refreshes access and refresh tokens
func (s *AuthService) RefreshTokens(ctx context.Context, req RefreshRequest) (*TokenPair, uuid.UUID, error) {
	// Get refresh token from storage by token value
	storedToken, err := s.tokenStore.GetRefreshTokenByValue(ctx, req.RefreshToken)
	if err != nil {
		return nil, uuid.Nil, domain.ErrInvalidToken
	}

	// Check if token has expired
	if time.Now().After(storedToken.ExpiresAt) {
		// Clean up expired token
		_ = s.tokenStore.DeleteRefreshToken(ctx, storedToken.UserID)
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
	if err := s.tokenStore.DeleteRefreshToken(ctx, storedToken.UserID); err != nil {
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

	return &TokenPair{
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

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	UserID      uuid.UUID
	OldPassword string
	NewPassword string
}

// ChangePassword changes a user's password
func (s *AuthService) ChangePassword(ctx context.Context, req ChangePasswordRequest) error {
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
