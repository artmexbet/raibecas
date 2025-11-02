package jwt

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJWTManager_GenerateAccessToken(t *testing.T) {
	manager := NewManager("test-secret", "test-issuer", 15*time.Minute, 7*24*time.Hour)

	userID := uuid.New()
	role := "user"

	token, err := manager.GenerateAccessToken(userID, role)
	if err != nil {
		t.Fatalf("Failed to generate access token: %v", err)
	}

	if token == "" {
		t.Fatal("Generated token is empty")
	}
}

func TestJWTManager_ValidateAccessToken(t *testing.T) {
	manager := NewManager("test-secret", "test-issuer", 15*time.Minute, 7*24*time.Hour)

	userID := uuid.New()
	role := "admin"

	// Generate token
	token, err := manager.GenerateAccessToken(userID, role)
	if err != nil {
		t.Fatalf("Failed to generate access token: %v", err)
	}

	// Validate token
	claims, err := manager.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, claims.UserID)
	}

	if claims.Role != role {
		t.Errorf("Expected role %s, got %s", role, claims.Role)
	}

	if claims.Issuer != "test-issuer" {
		t.Errorf("Expected issuer %s, got %s", "test-issuer", claims.Issuer)
	}
}

func TestJWTManager_ValidateAccessToken_InvalidToken(t *testing.T) {
	manager := NewManager("test-secret", "test-issuer", 15*time.Minute, 7*24*time.Hour)

	_, err := manager.ValidateAccessToken("invalid-token")
	if err == nil {
		t.Fatal("Expected error for invalid token, got nil")
	}
}

func TestJWTManager_ValidateAccessToken_WrongSecret(t *testing.T) {
	manager1 := NewManager("secret1", "test-issuer", 15*time.Minute, 7*24*time.Hour)
	manager2 := NewManager("secret2", "test-issuer", 15*time.Minute, 7*24*time.Hour)

	userID := uuid.New()
	token, err := manager1.GenerateAccessToken(userID, "user")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Try to validate with different secret
	_, err = manager2.ValidateAccessToken(token)
	if err == nil {
		t.Fatal("Expected error for token signed with different secret, got nil")
	}
}

func TestJWTManager_GenerateRefreshToken(t *testing.T) {
	manager := NewManager("test-secret", "test-issuer", 15*time.Minute, 7*24*time.Hour)

	token, err := manager.GenerateRefreshToken()
	if err != nil {
		t.Fatalf("Failed to generate refresh token: %v", err)
	}

	if token == "" {
		t.Fatal("Generated refresh token is empty")
	}

	// Verify it's a valid UUID
	_, err = uuid.Parse(token)
	if err != nil {
		t.Errorf("Refresh token is not a valid UUID: %v", err)
	}
}
