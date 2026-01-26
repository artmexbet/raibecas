package jwt_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/artmexbet/raibecas/services/auth/pkg/jwt"
)

// MockTokenStore для тестирования
type MockTokenStore struct {
	tokens          map[string]*jwt.RefreshTokenMetadata
	blacklist       map[string]bool
	families        map[string][]*jwt.RefreshTokenMetadata
	userTokens      map[string][]string
	revokedFamilies map[string]bool
}

func NewMockTokenStore() *MockTokenStore {
	return &MockTokenStore{
		tokens:          make(map[string]*jwt.RefreshTokenMetadata),
		blacklist:       make(map[string]bool),
		families:        make(map[string][]*jwt.RefreshTokenMetadata),
		userTokens:      make(map[string][]string),
		revokedFamilies: make(map[string]bool),
	}
}

func (m *MockTokenStore) StoreRefreshToken(_ context.Context, metadata *jwt.RefreshTokenMetadata, _ time.Duration) error {
	m.tokens[metadata.TokenID] = metadata

	userKey := metadata.UserID.String()
	m.userTokens[userKey] = append(m.userTokens[userKey], metadata.TokenID)

	m.families[metadata.TokenFamily] = append(m.families[metadata.TokenFamily], metadata)

	return nil
}

func (m *MockTokenStore) GetRefreshToken(_ context.Context, tokenID string) (*jwt.RefreshTokenMetadata, error) {
	token, ok := m.tokens[tokenID]
	if !ok {
		return nil, errors.New("token not found")
	}
	return token, nil
}

func (m *MockTokenStore) RevokeRefreshToken(_ context.Context, tokenID string) error {
	if token, ok := m.tokens[tokenID]; ok {
		token.IsRevoked = true
		now := time.Now()
		token.RevokedAt = &now
	}
	return nil
}

func (m *MockTokenStore) RevokeRefreshTokenFamily(_ context.Context, tokenFamily string) error {
	m.revokedFamilies[tokenFamily] = true
	for _, token := range m.families[tokenFamily] {
		token.IsRevoked = true
		now := time.Now()
		token.RevokedAt = &now
	}
	return nil
}

func (m *MockTokenStore) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	userKey := userID.String()
	for _, tokenID := range m.userTokens[userKey] {
		_ = m.RevokeRefreshToken(ctx, tokenID)
	}
	delete(m.userTokens, userKey)
	return nil
}

func (m *MockTokenStore) BlacklistAccessToken(_ context.Context, jti string, _ time.Duration) error {
	m.blacklist[jti] = true
	return nil
}

func (m *MockTokenStore) IsAccessTokenBlacklisted(_ context.Context, jti string) (bool, error) {
	return m.blacklist[jti], nil
}

func (m *MockTokenStore) GetTokenFamily(_ context.Context, tokenFamily string) ([]*jwt.RefreshTokenMetadata, error) {
	return m.families[tokenFamily], nil
}

func (m *MockTokenStore) GetUserDevices(_ context.Context, userID uuid.UUID) ([]string, error) {
	devices := make(map[string]bool)
	userKey := userID.String()
	for _, tokenID := range m.userTokens[userKey] {
		if token, ok := m.tokens[tokenID]; ok {
			devices[token.DeviceID] = true
		}
	}

	result := make([]string, 0, len(devices))
	for device := range devices {
		result = append(result, device)
	}
	return result, nil
}

func (m *MockTokenStore) RevokeDeviceTokens(ctx context.Context, userID uuid.UUID, deviceID string) error {
	userKey := userID.String()
	for _, tokenID := range m.userTokens[userKey] {
		if token, ok := m.tokens[tokenID]; ok && token.DeviceID == deviceID {
			_ = m.RevokeRefreshToken(ctx, tokenID)
		}
	}
	return nil
}

// Tests

func TestAccessTokenGeneration(t *testing.T) {
	store := NewMockTokenStore()
	manager := jwt.NewManager(
		"test-secret",
		"test-issuer",
		15*time.Minute,
		7*24*time.Hour,
		store,
	)

	metadata := &jwt.TokenMetadata{
		UserID:      uuid.New(),
		Role:        "user",
		DeviceID:    "device-1",
		UserAgent:   "test-agent",
		IPAddress:   "127.0.0.1",
		Fingerprint: "test-fingerprint",
	}

	token, claims, err := manager.GenerateAccessToken(metadata)

	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t, metadata.UserID, claims.UserID)
	assert.Equal(t, metadata.Role, claims.Role)
	assert.NotEmpty(t, claims.JTI)
	assert.Equal(t, metadata.Fingerprint, claims.Fingerprint)
	assert.Equal(t, "access", claims.TokenType)
}

func TestAccessTokenValidation(t *testing.T) {
	store := NewMockTokenStore()
	manager := jwt.NewManager(
		"test-secret",
		"test-issuer",
		15*time.Minute,
		7*24*time.Hour,
		store,
	)

	fingerprint, err := jwt.GenerateFingerprint()
	require.NoError(t, err)

	metadata := &jwt.TokenMetadata{
		UserID:      uuid.New(),
		Role:        "user",
		DeviceID:    "device-1",
		Fingerprint: fingerprint,
	}

	token, _, err := manager.GenerateAccessToken(metadata)
	require.NoError(t, err)

	// Valid token
	result, err := manager.ValidateAccessToken(context.Background(), token, fingerprint)
	assert.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Equal(t, metadata.UserID, result.Claims.UserID)

	// Invalid fingerprint
	result, err = manager.ValidateAccessToken(context.Background(), token, "wrong-fingerprint")
	assert.Error(t, err)
	assert.False(t, result.Valid)
	assert.True(t, result.FingerprintMismatch)
}

func TestRefreshTokenRotation(t *testing.T) {
	store := NewMockTokenStore()
	manager := jwt.NewManager(
		"test-secret",
		"test-issuer",
		15*time.Minute,
		7*24*time.Hour,
		store,
	)

	fingerprint, _ := jwt.GenerateFingerprint()
	userID := uuid.New()

	metadata := &jwt.TokenMetadata{
		UserID:      userID,
		Role:        "user",
		DeviceID:    "device-1",
		UserAgent:   "test-agent",
		IPAddress:   "127.0.0.1",
		Fingerprint: fingerprint,
	}

	// Generate initial tokens
	_, refreshToken1, err := manager.GenerateRefreshToken(metadata)
	require.NoError(t, err)

	err = manager.StoreRefreshToken(context.Background(), refreshToken1)
	require.NoError(t, err)

	// Rotate tokens
	accessToken2, refreshToken2Str, err := manager.RotateRefreshToken(
		context.Background(),
		refreshToken1.TokenID,
		metadata,
	)

	require.NoError(t, err)
	assert.NotEmpty(t, accessToken2)
	assert.NotEmpty(t, refreshToken2Str)

	// Old token should be revoked
	oldToken, err := store.GetRefreshToken(context.Background(), refreshToken1.TokenID)
	require.NoError(t, err)
	assert.True(t, oldToken.IsRevoked)
}

// Benchmark tests

func BenchmarkGenerateAccessToken(b *testing.B) {
	store := NewMockTokenStore()
	manager := jwt.NewManager(
		"test-secret",
		"test-issuer",
		15*time.Minute,
		7*24*time.Hour,
		store,
	)

	metadata := &jwt.TokenMetadata{
		UserID:      uuid.New(),
		Role:        "user",
		DeviceID:    "device-1",
		Fingerprint: "test-fingerprint",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = manager.GenerateAccessToken(metadata)
	}
}
