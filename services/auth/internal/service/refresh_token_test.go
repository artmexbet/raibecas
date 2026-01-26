package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"

	"github.com/artmexbet/raibecas/services/auth/internal/domain"
	"github.com/artmexbet/raibecas/services/auth/pkg/jwt"
)

func TestAuthService_RefreshTokens_Success(t *testing.T) {
	// Setup
	userID := uuid.New()
	email := "test@example.com"
	password := "password123"
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte(password), 12)

	mockUsers := NewMockUserRepository(t)
	mockTokenStore := jwt.NewMockTokenStore(t)

	// Expectations for login
	mockUsers.EXPECT().GetUserByEmail(mock.Anything, email).Return(&domain.User{
		ID:           userID,
		Email:        email,
		PasswordHash: string(passwordHash),
		Role:         domain.RoleUser,
		IsActive:     true,
	}, nil)
	mockTokenStore.EXPECT().StoreRefreshToken(mock.Anything, mock.Anything, mock.Anything).Return(nil)

	jwtManager := jwt.NewManager("test-secret", "test-issuer", 15*time.Minute, 7*24*time.Hour, mockTokenStore)
	authService := NewAuthService(mockUsers, jwtManager)

	// First, login to get initial tokens
	loginReq := domain.LoginRequest{
		Email:     email,
		Password:  password,
		DeviceID:  "test-device",
		UserAgent: "test-agent",
		IPAddress: "127.0.0.1",
	}

	initialResult, err := authService.Login(context.Background(), loginReq)
	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}

	// Expectations for refresh
	tokenFamily := uuid.New().String()

	// Mock ValidateRefreshToken - first GetRefreshToken call
	mockTokenStore.EXPECT().GetRefreshToken(mock.Anything, initialResult.TokenID).Return(&jwt.RefreshTokenMetadata{
		TokenID:     initialResult.TokenID,
		UserID:      userID,
		TokenFamily: tokenFamily,
		DeviceID:    "test-device",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
		Fingerprint: initialResult.Fingerprint,
		IsRevoked:   false,
	}, nil).Once()

	expectedUser := &domain.User{
		ID:           userID,
		Email:        email,
		PasswordHash: string(passwordHash),
		Role:         domain.RoleUser,
		IsActive:     true,
	}

	// Mock GetByID to verify user is still active
	mockUsers.EXPECT().GetUserByID(mock.Anything, userID).Return(expectedUser, nil)

	// Mock RotateRefreshToken operations - second GetRefreshToken call
	mockTokenStore.EXPECT().GetRefreshToken(mock.Anything, initialResult.TokenID).Return(&jwt.RefreshTokenMetadata{
		TokenID:     initialResult.TokenID,
		UserID:      userID,
		TokenFamily: tokenFamily,
		DeviceID:    "test-device",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
		Fingerprint: initialResult.Fingerprint,
		IsRevoked:   false,
	}, nil).Once()

	mockTokenStore.EXPECT().RevokeRefreshToken(mock.Anything, initialResult.TokenID).Return(nil)
	mockTokenStore.EXPECT().StoreRefreshToken(mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Now refresh tokens
	refreshReq := domain.RefreshRequest{
		TokenID:   initialResult.TokenID,
		DeviceID:  "test-device",
		UserAgent: "test-agent",
		IPAddress: "127.0.0.1",
	}

	newResult, err := authService.RefreshTokens(context.Background(), refreshReq, initialResult.Fingerprint)

	// Assert
	if err != nil {
		t.Fatalf("Expected successful refresh, got error: %v", err)
	}

	if newResult == nil {
		t.Fatal("Expected result, got nil")
	}

	if newResult.AccessToken == "" {
		t.Error("New access token is empty")
	}

	if newResult.RefreshToken == "" {
		t.Error("New refresh token is empty")
	}

	if newResult.RefreshToken == initialResult.RefreshToken {
		t.Error("Refresh token should be rotated, but got same token")
	}

	if newResult.User != expectedUser {
		t.Errorf("Expected user ID %v, got %v", expectedUser, newResult.User)
	}
}

func TestAuthService_RefreshTokens_InvalidToken(t *testing.T) {
	// Setup
	mockUsers := NewMockUserRepository(t)
	mockTokenStore := jwt.NewMockTokenStore(t)

	jwtManager := jwt.NewManager("test-secret", "test-issuer", 15*time.Minute, 7*24*time.Hour, mockTokenStore)
	authService := NewAuthService(mockUsers, jwtManager)

	// Test with invalid token
	tokenID := uuid.New().String()
	refreshReq := domain.RefreshRequest{
		TokenID: tokenID,
	}

	mockTokenStore.EXPECT().GetRefreshToken(mock.Anything, tokenID).Return(nil, domain.ErrTokenNotFound)

	_, err := authService.RefreshTokens(context.Background(), refreshReq, "test-fingerprint")

	// Assert
	if !errors.Is(err, domain.ErrInvalidToken) {
		t.Errorf("Expected ErrInvalidToken, got: %v", err)
	}
}

func TestAuthService_RefreshTokens_UserNotActive(t *testing.T) {
	// Setup
	userID := uuid.New()
	email := "test@example.com"
	password := "password123"
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte(password), 12)

	mockUsers := NewMockUserRepository(t)
	mockTokenStore := jwt.NewMockTokenStore(t)

	// Expectations for login
	mockUsers.EXPECT().GetUserByEmail(mock.Anything, email).Return(&domain.User{
		ID:           userID,
		Email:        email,
		PasswordHash: string(passwordHash),
		Role:         domain.RoleUser,
		IsActive:     true, // Initially active
	}, nil)
	mockTokenStore.EXPECT().StoreRefreshToken(mock.Anything, mock.Anything, mock.Anything).Return(nil)

	jwtManager := jwt.NewManager("test-secret", "test-issuer", 15*time.Minute, 7*24*time.Hour, mockTokenStore)
	authService := NewAuthService(mockUsers, jwtManager)

	// Login to get tokens
	loginReq := domain.LoginRequest{
		Email:    email,
		Password: password,
	}

	initialResult, err := authService.Login(context.Background(), loginReq)
	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}

	// Mock ValidateRefreshToken
	tokenFamily := uuid.New().String()
	mockTokenStore.EXPECT().GetRefreshToken(mock.Anything, initialResult.TokenID).Return(&jwt.RefreshTokenMetadata{
		TokenID:     initialResult.TokenID,
		UserID:      userID,
		TokenFamily: tokenFamily,
		ExpiresAt:   time.Now().Add(1 * time.Hour),
		Fingerprint: initialResult.Fingerprint,
		IsRevoked:   false,
	}, nil)

	// Now set user as not active for subsequent GetByID call
	mockUsers.EXPECT().GetUserByID(mock.Anything, userID).Return(&domain.User{
		ID:           userID,
		Email:        email,
		PasswordHash: string(passwordHash),
		Role:         domain.RoleUser,
		IsActive:     false,
	}, nil)

	// Try to refresh
	refreshReq := domain.RefreshRequest{
		TokenID: initialResult.TokenID,
	}

	_, err = authService.RefreshTokens(context.Background(), refreshReq, initialResult.Fingerprint)

	// Assert
	if !errors.Is(err, domain.ErrUserNotActive) {
		t.Errorf("Expected ErrUserNotActive, got: %v", err)
	}
}
