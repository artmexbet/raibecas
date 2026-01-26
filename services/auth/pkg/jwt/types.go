package jwt

import (
	"time"

	"github.com/google/uuid"
)

// TokenType represents the type of token
type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

// TokenMetadata contains metadata for token generation and validation
type TokenMetadata struct {
	UserID      uuid.UUID
	Role        string
	DeviceID    string
	UserAgent   string
	IPAddress   string
	Fingerprint string // Защита от XSS
	TokenFamily string // Для отслеживания цепочки refresh токенов
}

// AccessTokenClaims represents enhanced JWT claims for access tokens
type AccessTokenClaims struct {
	UserID      uuid.UUID `json:"user_id"`
	Role        string    `json:"role"`
	TokenType   string    `json:"token_type"`
	JTI         string    `json:"jti"`         // JWT ID - уникальный идентификатор токена
	Fingerprint string    `json:"fingerprint"` // Защита от XSS
	DeviceID    string    `json:"device_id"`
	IssuedAt    time.Time `json:"iat"`
	ExpiresAt   time.Time `json:"exp"`
	NotBefore   time.Time `json:"nbf"`
	Issuer      string    `json:"iss"`
	Subject     string    `json:"sub"`
}

// RefreshTokenMetadata represents metadata stored with refresh token in Redis
type RefreshTokenMetadata struct {
	TokenID      string     `json:"token_id"` // Уникальный ID токена
	UserID       uuid.UUID  `json:"user_id"`
	DeviceID     string     `json:"device_id"`
	UserAgent    string     `json:"user_agent"`
	IPAddress    string     `json:"ip_address"`
	Fingerprint  string     `json:"fingerprint"`   // Защита от XSS
	TokenFamily  string     `json:"token_family"`  // Для обнаружения кражи токенов
	PreviousJTI  string     `json:"previous_jti"`  // Предыдущий токен в цепочке
	RotationHash string     `json:"rotation_hash"` // Хеш для проверки ротации
	CreatedAt    time.Time  `json:"created_at"`
	ExpiresAt    time.Time  `json:"expires_at"`
	IsRevoked    bool       `json:"is_revoked"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
}

// ValidationResult contains the result of token validation
type ValidationResult struct {
	Valid               bool
	Claims              *AccessTokenClaims
	Error               error
	IsBlacklisted       bool
	IsExpired           bool
	FingerprintMismatch bool
}
