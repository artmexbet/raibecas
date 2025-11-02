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

// Mock repositories for testing
type mockUserRepository struct {
	users map[uuid.UUID]*domain.User
}

func (m *mockUserRepository) Create(ctx context.Context, user *domain.User) error {
	m.users[user.ID] = user
	return nil
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	user, ok := m.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return user, nil
}

func (m *mockUserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	for _, user := range m.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (m *mockUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	for _, user := range m.users {
		if user.Email == email {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	for _, user := range m.users {
		if user.Username == username {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockUserRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	user, ok := m.users[userID]
	if !ok {
		return domain.ErrUserNotFound
	}
	user.PasswordHash = passwordHash
	return nil
}

type mockTokenStore struct {
	tokens map[uuid.UUID]*domain.RefreshToken
}

func (m *mockTokenStore) StoreRefreshToken(ctx context.Context, token *domain.RefreshToken, ttl time.Duration) error {
	m.tokens[token.UserID] = token
	return nil
}

func (m *mockTokenStore) GetRefreshToken(ctx context.Context, userID uuid.UUID) (*domain.RefreshToken, error) {
	token, ok := m.tokens[userID]
	if !ok {
		return nil, domain.ErrTokenNotFound
	}
	return token, nil
}

func (m *mockTokenStore) DeleteRefreshToken(ctx context.Context, userID uuid.UUID) error {
	delete(m.tokens, userID)
	return nil
}

func (m *mockTokenStore) DeleteAllRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	delete(m.tokens, userID)
	return nil
}

func TestAuthService_Login_Success(t *testing.T) {
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

	// Test
	req := LoginRequest{
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

	// Verify refresh token was stored
	if _, ok := mockTokens.tokens[userID]; !ok {
		t.Error("Refresh token was not stored")
	}
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
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

	// Test
	req := LoginRequest{
		Email:    email,
		Password: "wrongpassword",
	}

	_, _, err := authService.Login(context.Background(), req)

	// Assert
	if err != domain.ErrInvalidCredentials {
		t.Errorf("Expected ErrInvalidCredentials, got: %v", err)
	}
}

func TestAuthService_Login_UserNotActive(t *testing.T) {
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
				IsActive:     false, // User is not active
			},
		},
	}

	mockTokens := &mockTokenStore{
		tokens: make(map[uuid.UUID]*domain.RefreshToken),
	}

	jwtManager := jwt.NewManager("test-secret", "test-issuer", 15*time.Minute, 7*24*time.Hour)
	authService := NewAuthService(mockUsers, mockTokens, jwtManager)

	// Test
	req := LoginRequest{
		Email:    email,
		Password: password,
	}

	_, _, err := authService.Login(context.Background(), req)

	// Assert
	if err != domain.ErrUserNotActive {
		t.Errorf("Expected ErrUserNotActive, got: %v", err)
	}
}

func TestAuthService_Logout(t *testing.T) {
	// Setup
	userID := uuid.New()
	mockUsers := &mockUserRepository{
		users: make(map[uuid.UUID]*domain.User),
	}

	mockTokens := &mockTokenStore{
		tokens: map[uuid.UUID]*domain.RefreshToken{
			userID: {
				UserID: userID,
				Token:  "test-token",
			},
		},
	}

	jwtManager := jwt.NewManager("test-secret", "test-issuer", 15*time.Minute, 7*24*time.Hour)
	authService := NewAuthService(mockUsers, mockTokens, jwtManager)

	// Test
	err := authService.Logout(context.Background(), userID)

	// Assert
	if err != nil {
		t.Fatalf("Expected successful logout, got error: %v", err)
	}

	// Verify token was removed
	if _, ok := mockTokens.tokens[userID]; ok {
		t.Error("Refresh token was not removed")
	}
}
