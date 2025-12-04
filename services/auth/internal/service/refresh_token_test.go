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

	mockUsers := NewMockIUserRepository(t)
	mockTokens := NewMockITokenStore(t)

	// Expectations for login
	mockUsers.EXPECT().GetByEmail(mock.Anything, email).Return(&domain.User{
		ID:           userID,
		Email:        email,
		PasswordHash: string(passwordHash),
		Role:         domain.RoleUser,
		IsActive:     true,
	}, nil)
	mockTokens.EXPECT().StoreRefreshToken(mock.Anything, mock.Anything, mock.Anything).Return(nil)

	jwtManager := jwt.NewManager("test-secret", "test-issuer", 15*time.Minute, 7*24*time.Hour)
	authService := NewAuthService(mockUsers, mockTokens, jwtManager)

	// First, login to get initial tokens
	loginReq := domain.LoginRequest{
		Email:     email,
		Password:  password,
		DeviceID:  "test-device",
		UserAgent: "test-agent",
		IPAddress: "127.0.0.1",
	}

	initialTokens, _, err := authService.Login(context.Background(), loginReq)
	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}

	// Expectations for refresh: GetRefreshTokenByValue -> return stored token
	// make it once so subsequent calls can return not found
	mockTokens.EXPECT().GetRefreshTokenByValue(mock.Anything, initialTokens.RefreshToken).Return(&domain.RefreshToken{
		Token:     initialTokens.RefreshToken,
		UserID:    userID,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}, nil).Once()
	// Expect DeleteRefreshToken for old token and StoreRefreshToken for new one
	mockTokens.EXPECT().DeleteRefreshToken(mock.Anything, userID, initialTokens.RefreshToken).Return(nil)
	mockTokens.EXPECT().StoreRefreshToken(mock.Anything, mock.Anything, mock.Anything).Return(nil)
	// Expect GetByID call to verify user
	mockUsers.EXPECT().GetByID(mock.Anything, userID).Return(&domain.User{ID: userID, Email: email, PasswordHash: string(passwordHash), Role: domain.RoleUser, IsActive: true}, nil)

	// Now refresh tokens
	refreshReq := domain.RefreshRequest{
		RefreshToken: initialTokens.RefreshToken,
		DeviceID:     "test-device",
		UserAgent:    "test-agent",
		IPAddress:    "127.0.0.1",
	}

	newTokens, returnedUserID, err := authService.RefreshTokens(context.Background(), refreshReq)

	// Assert
	if err != nil {
		t.Fatalf("Expected successful refresh, got error: %v", err)
	}

	if newTokens == nil {
		t.Fatal("Expected tokens, got nil")
	}

	if newTokens.AccessToken == "" {
		t.Error("New access token is empty")
	}

	if newTokens.RefreshToken == "" {
		t.Error("New refresh token is empty")
	}

	if newTokens.RefreshToken == initialTokens.RefreshToken {
		t.Error("Refresh token should be rotated, but got same token")
	}

	if returnedUserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, returnedUserID)
	}

	// Old refresh token should not work
	mockTokens.EXPECT().GetRefreshTokenByValue(mock.Anything, initialTokens.RefreshToken).Return(nil, domain.ErrTokenNotFound)
	_, _, err = authService.RefreshTokens(context.Background(), refreshReq)
	if err == nil {
		t.Error("Expected error when using old refresh token, got nil")
	}
}

func TestAuthService_RefreshTokens_InvalidToken(t *testing.T) {
	// Setup
	mockUsers := NewMockIUserRepository(t)
	mockTokens := NewMockITokenStore(t)

	jwtManager := jwt.NewManager("test-secret", "test-issuer", 15*time.Minute, 7*24*time.Hour)
	authService := NewAuthService(mockUsers, mockTokens, jwtManager)

	// Test with invalid token
	refreshReq := domain.RefreshRequest{
		RefreshToken: "invalid-token",
	}

	mockTokens.EXPECT().GetRefreshTokenByValue(mock.Anything, "invalid-token").Return(nil, domain.ErrTokenNotFound)

	_, _, err := authService.RefreshTokens(context.Background(), refreshReq)

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

	mockUsers := NewMockIUserRepository(t)
	mockTokens := NewMockITokenStore(t)

	// Expectations for login
	mockUsers.EXPECT().GetByEmail(mock.Anything, email).Return(&domain.User{
		ID:           userID,
		Email:        email,
		PasswordHash: string(passwordHash),
		Role:         domain.RoleUser,
		IsActive:     true, // Initially active
	}, nil)
	mockTokens.EXPECT().StoreRefreshToken(mock.Anything, mock.Anything, mock.Anything).Return(nil)

	jwtManager := jwt.NewManager("test-secret", "test-issuer", 15*time.Minute, 7*24*time.Hour)
	authService := NewAuthService(mockUsers, mockTokens, jwtManager)

	// Login to get tokens
	loginReq := domain.LoginRequest{
		Email:    email,
		Password: password,
	}

	initialTokens, _, err := authService.Login(context.Background(), loginReq)
	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}

	// Now set user as not active for subsequent GetByID call
	mockUsers.EXPECT().GetByID(mock.Anything, userID).Return(&domain.User{ID: userID, Email: email, PasswordHash: string(passwordHash), Role: domain.RoleUser, IsActive: false}, nil)
	mockTokens.EXPECT().GetRefreshTokenByValue(mock.Anything, initialTokens.RefreshToken).Return(&domain.RefreshToken{Token: initialTokens.RefreshToken, UserID: userID, ExpiresAt: time.Now().Add(1 * time.Hour)}, nil)

	// Try to refresh
	refreshReq := domain.RefreshRequest{
		RefreshToken: initialTokens.RefreshToken,
	}

	_, _, err = authService.RefreshTokens(context.Background(), refreshReq)

	// Assert
	if !errors.Is(err, domain.ErrUserNotActive) {
		t.Errorf("Expected ErrUserNotActive, got: %v", err)
	}
}
