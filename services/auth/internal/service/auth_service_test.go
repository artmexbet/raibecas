package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/artmexbet/raibecas/services/auth/internal/domain"
	"github.com/artmexbet/raibecas/services/auth/pkg/jwt"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthService_Login_Success(t *testing.T) {
	// Setup
	userID := uuid.New()
	email := "test@example.com"
	password := "password123"
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte(password), 12)

	// Use generated mocks
	mockUsers := NewMockIUserRepository(t)
	mockTokens := NewMockITokenStore(t)

	// Expectations
	expectedUser := &domain.User{
		ID:           userID,
		Email:        email,
		PasswordHash: string(passwordHash),
		Role:         domain.RoleUser,
		IsActive:     true,
	}
	mockUsers.EXPECT().GetByEmail(mock.Anything, email).Return(expectedUser, nil)
	mockTokens.EXPECT().StoreRefreshToken(mock.Anything, mock.Anything, mock.Anything).Return(nil)

	jwtManager := jwt.NewManager("test-secret", "test-issuer", 15*time.Minute, 7*24*time.Hour)
	authService := NewAuthService(mockUsers, mockTokens, jwtManager)

	// Test
	req := domain.LoginRequest{
		Email:     email,
		Password:  password,
		DeviceID:  "test-device",
		UserAgent: "test-agent",
		IPAddress: "127.0.0.1",
	}

	tokens, returnedUserID, err := authService.Login(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("Expected successful login, got error: %v", err)
	}

	if tokens == nil {
		t.Fatal("Expected tokens, got nil")
	}

	if tokens.AccessToken == "" {
		t.Error("Access token is empty")
	}

	if tokens.RefreshToken == "" {
		t.Error("Refresh token is empty")
	}

	if returnedUserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, returnedUserID)
	}

	// Expectations asserted by mock cleanup
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
	// Setup
	userID := uuid.New()
	email := "test@example.com"
	password := "password123"
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte(password), 12)

	mockUsers := NewMockIUserRepository(t)
	mockTokens := NewMockITokenStore(t)

	mockUsers.EXPECT().GetByEmail(mock.Anything, email).Return(&domain.User{
		ID:           userID,
		Email:        email,
		PasswordHash: string(passwordHash),
		Role:         domain.RoleUser,
		IsActive:     true,
	}, nil)

	jwtManager := jwt.NewManager("test-secret", "test-issuer", 15*time.Minute, 7*24*time.Hour)
	authService := NewAuthService(mockUsers, mockTokens, jwtManager)

	// Test
	req := domain.LoginRequest{
		Email:    email,
		Password: "wrongpassword",
	}

	_, _, err := authService.Login(context.Background(), req)

	// Assert
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("Expected ErrInvalidCredentials, got: %v", err)
	}
}

func TestAuthService_Login_UserNotActive(t *testing.T) {
	// Setup
	userID := uuid.New()
	email := "test@example.com"
	password := "password123"
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte(password), 12)

	mockUsers := NewMockIUserRepository(t)
	mockTokens := NewMockITokenStore(t)

	mockUsers.EXPECT().GetByEmail(mock.Anything, email).Return(&domain.User{
		ID:           userID,
		Email:        email,
		PasswordHash: string(passwordHash),
		Role:         domain.RoleUser,
		IsActive:     false, // User is not active
	}, nil)

	jwtManager := jwt.NewManager("test-secret", "test-issuer", 15*time.Minute, 7*24*time.Hour)
	authService := NewAuthService(mockUsers, mockTokens, jwtManager)

	// Test
	req := domain.LoginRequest{
		Email:    email,
		Password: password,
	}

	_, _, err := authService.Login(context.Background(), req)

	// Assert
	if !errors.Is(err, domain.ErrUserNotActive) {
		t.Errorf("Expected ErrUserNotActive, got: %v", err)
	}
}

func TestAuthService_Logout(t *testing.T) {
	// Setup
	userID := uuid.New()
	mockUsers := NewMockIUserRepository(t)
	mockTokens := NewMockITokenStore(t)

	// Expect DeleteRefreshToken to be called with provided token
	mockTokens.EXPECT().DeleteRefreshToken(mock.Anything, userID, "test-token").Return(nil)

	jwtManager := jwt.NewManager("test-secret", "test-issuer", 15*time.Minute, 7*24*time.Hour)
	authService := NewAuthService(mockUsers, mockTokens, jwtManager)

	// Test
	err := authService.Logout(context.Background(), userID, "test-token")

	// Assert
	if err != nil {
		t.Fatalf("Expected successful logout, got error: %v", err)
	}

	// Expectations asserted by mock cleanup
}
