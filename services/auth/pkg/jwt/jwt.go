package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims represents JWT claims
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Role   string    `json:"role"`
	jwt.RegisteredClaims
}

// Manager handles JWT operations
type Manager struct {
	secret          []byte
	issuer          string
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

// NewManager creates a new JWT manager
func NewManager(secret, issuer string, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{
		secret:          []byte(secret),
		issuer:          issuer,
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
	}
}

// GenerateAccessToken generates an access token for a user
func (m *Manager) GenerateAccessToken(userID uuid.UUID, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    m.issuer,
			Subject:   userID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// GenerateRefreshToken generates a refresh token
func (m *Manager) GenerateRefreshToken() (string, error) {
	return uuid.New().String(), nil
}

// ValidateAccessToken validates an access token and returns the claims
func (m *Manager) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// GetRefreshTokenTTL returns the refresh token TTL
func (m *Manager) GetRefreshTokenTTL() time.Duration {
	return m.refreshTokenTTL
}
