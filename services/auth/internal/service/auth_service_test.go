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

func TestAuthService_Login_Success(t *testing.T) {
	// Setup
	userID := uuid.New()
	email := "test@example.com"
	password := "password123"
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte(password), 12)

	// Use generated mocks
	mockUsers := NewMockUserRepository(t)
	mockTokenStore := jwt.NewMockTokenStore(t)

	// Expectations
	expectedUser := &domain.User{
		ID:           userID,
		Email:        email,
		PasswordHash: string(passwordHash),
		Role:         domain.RoleUser,
		IsActive:     true,
	}
	mockUsers.EXPECT().GetUserByEmail(mock.Anything, email).Return(expectedUser, nil)
	mockTokenStore.EXPECT().StoreRefreshToken(mock.Anything, mock.Anything, mock.Anything).Return(nil)

	jwtManager := jwt.NewManager("test-secret", "test-issuer", 15*time.Minute, 7*24*time.Hour, mockTokenStore)
	authService := NewAuthService(mockUsers, jwtManager)

	// Test
	req := domain.LoginRequest{
		Email:     email,
		Password:  password,
		DeviceID:  "test-device",
		UserAgent: "test-agent",
		IPAddress: "127.0.0.1",
	}

	result, err := authService.Login(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("Expected successful login, got error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result, got nil")
	}

	if result.AccessToken == "" {
		t.Error("Access token is empty")
	}

	if result.RefreshToken == "" {
		t.Error("Refresh token is empty")
	}

	if result.User != expectedUser {
		t.Errorf("Expected user ID %v, got %v", expectedUser, result.User)
	}

	// Expectations asserted by mock cleanup
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
	// Setup
	userID := uuid.New()
	email := "test@example.com"
	password := "password123"
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte(password), 12)

	mockUsers := NewMockUserRepository(t)
	mockTokenStore := jwt.NewMockTokenStore(t)

	mockUsers.EXPECT().GetUserByEmail(mock.Anything, email).Return(&domain.User{
		ID:           userID,
		Email:        email,
		PasswordHash: string(passwordHash),
		Role:         domain.RoleUser,
		IsActive:     true,
	}, nil)

	jwtManager := jwt.NewManager("test-secret", "test-issuer", 15*time.Minute, 7*24*time.Hour, mockTokenStore)
	authService := NewAuthService(mockUsers, jwtManager)

	// Test
	req := domain.LoginRequest{
		Email:    email,
		Password: "wrongpassword",
	}

	_, err := authService.Login(context.Background(), req)

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

	mockUsers := NewMockUserRepository(t)
	mockTokenStore := jwt.NewMockTokenStore(t)

	mockUsers.EXPECT().GetUserByEmail(mock.Anything, email).Return(&domain.User{
		ID:           userID,
		Email:        email,
		PasswordHash: string(passwordHash),
		Role:         domain.RoleUser,
		IsActive:     false, // User is not active
	}, nil)

	jwtManager := jwt.NewManager("test-secret", "test-issuer", 15*time.Minute, 7*24*time.Hour, mockTokenStore)
	authService := NewAuthService(mockUsers, jwtManager)

	// Test
	req := domain.LoginRequest{
		Email:    email,
		Password: password,
	}

	_, err := authService.Login(context.Background(), req)

	// Assert
	if !errors.Is(err, domain.ErrUserNotActive) {
		t.Errorf("Expected ErrUserNotActive, got: %v", err)
	}
}

func TestAuthService_Logout(t *testing.T) {
	// Setup
	tokenID := uuid.New().String()
	accessTokenJTI := "test-jti"

	mockUsers := NewMockUserRepository(t)
	mockTokenStore := jwt.NewMockTokenStore(t)

	// Expect RevokeRefreshToken to be called with provided tokenID
	mockTokenStore.EXPECT().RevokeRefreshToken(mock.Anything, tokenID).Return(nil)
	// Expect BlacklistAccessToken to be called with provided JTI
	mockTokenStore.EXPECT().BlacklistAccessToken(mock.Anything, accessTokenJTI, mock.Anything).Return(nil)

	jwtManager := jwt.NewManager("test-secret", "test-issuer", 15*time.Minute, 7*24*time.Hour, mockTokenStore)
	authService := NewAuthService(mockUsers, jwtManager)

	// Test
	err := authService.Logout(context.Background(), tokenID, accessTokenJTI)

	// Assert
	if err != nil {
		t.Fatalf("Expected successful logout, got error: %v", err)
	}

	// Expectations asserted by mock cleanup
}
