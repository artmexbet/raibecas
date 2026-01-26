package jwt

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// TokenStore определяет интерфейс для хранения токенов (инверсия зависимостей)
type TokenStore interface {
	// Refresh tokens
	StoreRefreshToken(ctx context.Context, metadata *RefreshTokenMetadata, ttl time.Duration) error
	GetRefreshToken(ctx context.Context, tokenID string) (*RefreshTokenMetadata, error)
	RevokeRefreshToken(ctx context.Context, tokenID string) error
	RevokeRefreshTokenFamily(ctx context.Context, tokenFamily string) error
	RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error

	// Access token blacklist
	BlacklistAccessToken(ctx context.Context, jti string, ttl time.Duration) error
	IsAccessTokenBlacklisted(ctx context.Context, jti string) (bool, error)

	// Token families (для обнаружения кражи токенов)
	GetTokenFamily(ctx context.Context, tokenFamily string) ([]*RefreshTokenMetadata, error)

	// Device management
	GetUserDevices(ctx context.Context, userID uuid.UUID) ([]string, error)
	RevokeDeviceTokens(ctx context.Context, userID uuid.UUID, deviceID string) error
}

// TokenGenerator определяет интерфейс для генерации токенов
type TokenGenerator interface {
	GenerateAccessToken(metadata *TokenMetadata) (string, *AccessTokenClaims, error)
	GenerateRefreshToken(metadata *TokenMetadata) (string, *RefreshTokenMetadata, error)

	// Store operations
	StoreRefreshToken(ctx context.Context, metadata *RefreshTokenMetadata) error
	GetRefreshTokenTTL() time.Duration
}

// TokenValidator определяет интерфейс для валидации токенов
type TokenValidator interface {
	ValidateAccessToken(ctx context.Context, token string, expectedFingerprint string) (*ValidationResult, error)
	ValidateRefreshToken(ctx context.Context, tokenID string, expectedFingerprint string) (*RefreshTokenMetadata, error)
}

// TokenManager объединяет все операции с токенами
type TokenManager interface {
	TokenGenerator
	TokenValidator

	// Token rotation
	RotateRefreshToken(ctx context.Context, oldTokenID string, metadata *TokenMetadata) (accessToken string, refreshToken string, err error)

	// Revocation
	RevokeAccessToken(ctx context.Context, jti string) error
	RevokeRefreshToken(ctx context.Context, tokenID string) error
	RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error

	// Introspection
	IntrospectAccessToken(ctx context.Context, token string) (*AccessTokenClaims, error)
}
