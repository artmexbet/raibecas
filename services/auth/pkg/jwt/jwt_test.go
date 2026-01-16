package jwt

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Простой mock для базовых тестов
type simpleTokenStore struct{}

func (s *simpleTokenStore) StoreRefreshToken(ctx context.Context, metadata *RefreshTokenMetadata, ttl time.Duration) error {
	return nil
}

func (s *simpleTokenStore) GetRefreshToken(ctx context.Context, tokenID string) (*RefreshTokenMetadata, error) {
	return nil, nil
}

func (s *simpleTokenStore) RevokeRefreshToken(ctx context.Context, tokenID string) error {
	return nil
}

func (s *simpleTokenStore) RevokeRefreshTokenFamily(ctx context.Context, tokenFamily string) error {
	return nil
}

func (s *simpleTokenStore) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	return nil
}

func (s *simpleTokenStore) BlacklistAccessToken(ctx context.Context, jti string, ttl time.Duration) error {
	return nil
}

func (s *simpleTokenStore) IsAccessTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	return false, nil
}

func (s *simpleTokenStore) GetTokenFamily(ctx context.Context, tokenFamily string) ([]*RefreshTokenMetadata, error) {
	return nil, nil
}

func (s *simpleTokenStore) GetUserDevices(ctx context.Context, userID uuid.UUID) ([]string, error) {
	return nil, nil
}

func (s *simpleTokenStore) RevokeDeviceTokens(ctx context.Context, userID uuid.UUID, deviceID string) error {
	return nil
}

func TestJWTManager_GenerateAccessToken(t *testing.T) {
	manager := NewManager("test-secret", "test-issuer", 15*time.Minute, 7*24*time.Hour, &simpleTokenStore{})

	userID := uuid.New()
	role := "user"

	metadata := &TokenMetadata{
		UserID:      userID,
		Role:        role,
		DeviceID:    "test-device",
		Fingerprint: "test-fingerprint",
	}

	token, claims, err := manager.GenerateAccessToken(metadata)
	if err != nil {
		t.Fatalf("Failed to generate access token: %v", err)
	}

	if token == "" {
		t.Fatal("Generated token is empty")
	}

	if claims.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, claims.UserID)
	}

	if claims.Role != role {
		t.Errorf("Expected role %s, got %s", role, claims.Role)
	}
}

func TestJWTManager_ValidateAccessToken(t *testing.T) {
	manager := NewManager("test-secret", "test-issuer", 15*time.Minute, 7*24*time.Hour, &simpleTokenStore{})

	userID := uuid.New()
	role := "admin"
	fingerprint := "test-fingerprint"

	metadata := &TokenMetadata{
		UserID:      userID,
		Role:        role,
		DeviceID:    "test-device",
		Fingerprint: fingerprint,
	}

	// Generate token
	token, _, err := manager.GenerateAccessToken(metadata)
	if err != nil {
		t.Fatalf("Failed to generate access token: %v", err)
	}

	// Validate token
	result, err := manager.ValidateAccessToken(context.Background(), token, fingerprint)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if !result.Valid {
		t.Fatal("Token should be valid")
	}

	if result.Claims.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, result.Claims.UserID)
	}

	if result.Claims.Role != role {
		t.Errorf("Expected role %s, got %s", role, result.Claims.Role)
	}

	if result.Claims.Issuer != "test-issuer" {
		t.Errorf("Expected issuer %s, got %s", "test-issuer", result.Claims.Issuer)
	}
}

func TestJWTManager_ValidateAccessToken_InvalidToken(t *testing.T) {
	manager := NewManager("test-secret", "test-issuer", 15*time.Minute, 7*24*time.Hour, &simpleTokenStore{})

	_, err := manager.ValidateAccessToken(context.Background(), "invalid-token", "test-fp")
	if err == nil {
		t.Fatal("Expected error for invalid token, got nil")
	}
}

func TestJWTManager_ValidateAccessToken_WrongSecret(t *testing.T) {
	manager1 := NewManager("secret1", "test-issuer", 15*time.Minute, 7*24*time.Hour, &simpleTokenStore{})
	manager2 := NewManager("secret2", "test-issuer", 15*time.Minute, 7*24*time.Hour, &simpleTokenStore{})

	userID := uuid.New()
	fingerprint := "test-fp"

	metadata := &TokenMetadata{
		UserID:      userID,
		Role:        "user",
		DeviceID:    "test-device",
		Fingerprint: fingerprint,
	}

	token, _, err := manager1.GenerateAccessToken(metadata)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Try to validate with different secret
	_, err = manager2.ValidateAccessToken(context.Background(), token, fingerprint)
	if err == nil {
		t.Fatal("Expected error for token signed with different secret, got nil")
	}
}

func TestJWTManager_GenerateRefreshToken(t *testing.T) {
	manager := NewManager("test-secret", "test-issuer", 15*time.Minute, 7*24*time.Hour, &simpleTokenStore{})

	metadata := &TokenMetadata{
		UserID:      uuid.New(),
		Role:        "user",
		DeviceID:    "test-device",
		Fingerprint: "test-fp",
	}

	token, refreshMeta, err := manager.GenerateRefreshToken(metadata)
	if err != nil {
		t.Fatalf("Failed to generate refresh token: %v", err)
	}

	if token == "" {
		t.Fatal("Generated refresh token is empty")
	}

	if refreshMeta.TokenID == "" {
		t.Fatal("Token ID is empty")
	}

	if refreshMeta.TokenFamily == "" {
		t.Fatal("Token family is empty")
	}
}
