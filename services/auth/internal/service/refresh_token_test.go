package service

import (
	"context"
	"testing"
	"time"

	"auth/internal/domain"
	"auth/pkg/jwt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthService_RefreshTokens_Success(t *testing.T) {
	// Setup
	userID := uuid.New()
	email := "test@example.com"
	password := "password123"
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte(password), 12)

	mockUsers := &mockUserRepository{
		users: map[uuid.UUID]*domain.User{
			userID: {
				ID:           userID,
				Email:        email,
				PasswordHash: string(passwordHash),
				Role:         domain.RoleUser,
				IsActive:     true,
			},
		},
	}

	mockTokens := &mockTokenStore{
		tokens: make(map[uuid.UUID]*domain.RefreshToken),
	}

	jwtManager := jwt.NewManager("test-secret", "test-issuer", 15*time.Minute, 7*24*time.Hour)
	authService := NewAuthService(mockUsers, mockTokens, jwtManager)

	// First, login to get initial tokens
	loginReq := LoginRequest{
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

	// Now refresh tokens
	refreshReq := RefreshRequest{
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
	_, _, err = authService.RefreshTokens(context.Background(), refreshReq)
	if err == nil {
		t.Error("Expected error when using old refresh token, got nil")
	}
}

func TestAuthService_RefreshTokens_InvalidToken(t *testing.T) {
	// Setup
	mockUsers := &mockUserRepository{
		users: make(map[uuid.UUID]*domain.User),
	}

	mockTokens := &mockTokenStore{
		tokens: make(map[uuid.UUID]*domain.RefreshToken),
	}

	jwtManager := jwt.NewManager("test-secret", "test-issuer", 15*time.Minute, 7*24*time.Hour)
	authService := NewAuthService(mockUsers, mockTokens, jwtManager)

	// Test with invalid token
	refreshReq := RefreshRequest{
		RefreshToken: "invalid-token",
	}

	_, _, err := authService.RefreshTokens(context.Background(), refreshReq)

	// Assert
	if err != domain.ErrInvalidToken {
		t.Errorf("Expected ErrInvalidToken, got: %v", err)
	}
}

func TestAuthService_RefreshTokens_UserNotActive(t *testing.T) {
	// Setup
	userID := uuid.New()
	email := "test@example.com"
	password := "password123"
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte(password), 12)

	mockUsers := &mockUserRepository{
		users: map[uuid.UUID]*domain.User{
			userID: {
				ID:           userID,
				Email:        email,
				PasswordHash: string(passwordHash),
				Role:         domain.RoleUser,
				IsActive:     true, // Initially active
			},
		},
	}

	mockTokens := &mockTokenStore{
		tokens: make(map[uuid.UUID]*domain.RefreshToken),
	}

	jwtManager := jwt.NewManager("test-secret", "test-issuer", 15*time.Minute, 7*24*time.Hour)
	authService := NewAuthService(mockUsers, mockTokens, jwtManager)

	// Login to get tokens
	loginReq := LoginRequest{
		Email:    email,
		Password: password,
	}

	initialTokens, _, err := authService.Login(context.Background(), loginReq)
	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}

	// Deactivate user
	mockUsers.users[userID].IsActive = false

	// Try to refresh
	refreshReq := RefreshRequest{
		RefreshToken: initialTokens.RefreshToken,
	}

	_, _, err = authService.RefreshTokens(context.Background(), refreshReq)

	// Assert
	if err != domain.ErrUserNotActive {
		t.Errorf("Expected ErrUserNotActive, got: %v", err)
	}
}
