package jwt

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims represents JWT claims (deprecated - use AccessTokenClaims)
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Role   string    `json:"role"`
	jwt.RegisteredClaims
}

// Manager handles JWT operations with modern security practices
type Manager struct {
	secret          []byte
	issuer          string
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	store           TokenStore
}

// NewManager creates a new JWT manager with token store
func NewManager(secret, issuer string, accessTTL, refreshTTL time.Duration, store TokenStore) *Manager {
	return &Manager{
		secret:          []byte(secret),
		issuer:          issuer,
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
		store:           store,
	}
}

// GenerateAccessToken generates an access token with enhanced security
func (m *Manager) GenerateAccessToken(metadata *TokenMetadata) (string, *AccessTokenClaims, error) {
	now := time.Now()
	jti := uuid.New().String()

	claims := &AccessTokenClaims{
		UserID:      metadata.UserID,
		Role:        metadata.Role,
		TokenType:   string(TokenTypeAccess),
		JTI:         jti,
		Fingerprint: metadata.Fingerprint,
		DeviceID:    metadata.DeviceID,
		IssuedAt:    now,
		ExpiresAt:   now.Add(m.accessTokenTTL),
		NotBefore:   now,
		Issuer:      m.issuer,
		Subject:     metadata.UserID.String(),
	}

	// Создаём JWT с использованием стандартных claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":     claims.UserID.String(),
		"role":        claims.Role,
		"token_type":  claims.TokenType,
		"jti":         claims.JTI,
		"fingerprint": claims.Fingerprint,
		"device_id":   claims.DeviceID,
		"iat":         claims.IssuedAt.Unix(),
		"exp":         claims.ExpiresAt.Unix(),
		"nbf":         claims.NotBefore.Unix(),
		"iss":         claims.Issuer,
		"sub":         claims.Subject,
	})

	signedToken, err := token.SignedString(m.secret)
	if err != nil {
		return "", nil, fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, claims, nil
}

// GenerateRefreshToken generates a cryptographically secure refresh token
func (m *Manager) GenerateRefreshToken(metadata *TokenMetadata) (string, *RefreshTokenMetadata, error) {
	now := time.Now()
	tokenID := uuid.New().String()

	// Генерируем криптографически стойкий токен
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", nil, fmt.Errorf("failed to generate random token: %w", err)
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Создаём rotation hash для защиты от реплей атак
	rotationHash := m.generateRotationHash(tokenID, metadata.UserID.String())

	// Если tokenFamily не указан, создаём новый
	tokenFamily := metadata.TokenFamily
	if tokenFamily == "" {
		tokenFamily = uuid.New().String()
	}

	refreshMetadata := &RefreshTokenMetadata{
		TokenID:      tokenID,
		UserID:       metadata.UserID,
		DeviceID:     metadata.DeviceID,
		UserAgent:    metadata.UserAgent,
		IPAddress:    metadata.IPAddress,
		Fingerprint:  metadata.Fingerprint,
		TokenFamily:  tokenFamily,
		RotationHash: rotationHash,
		CreatedAt:    now,
		ExpiresAt:    now.Add(m.refreshTokenTTL),
		IsRevoked:    false,
	}

	return token, refreshMetadata, nil
}

// generateRotationHash создаёт хеш для проверки ротации токенов
func (m *Manager) generateRotationHash(tokenID, userID string) string {
	h := hmac.New(sha256.New, m.secret)
	h.Write([]byte(tokenID + userID))
	return hex.EncodeToString(h.Sum(nil))
}

// GenerateFingerprint генерирует fingerprint для защиты от XSS
func GenerateFingerprint() (string, error) {
	fingerprintBytes := make([]byte, 32)
	if _, err := rand.Read(fingerprintBytes); err != nil {
		return "", fmt.Errorf("failed to generate fingerprint: %w", err)
	}
	return base64.URLEncoding.EncodeToString(fingerprintBytes), nil
}

// ValidateAccessToken validates an access token with enhanced security checks
func (m *Manager) ValidateAccessToken(ctx context.Context, tokenString string, expectedFingerprint string) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid: false,
	}

	// Парсим токен
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Проверяем алгоритм подписи
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})

	if err != nil {
		result.Error = fmt.Errorf("failed to parse token: %w", err)
		return result, result.Error
	}

	// Извлекаем claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		result.Error = fmt.Errorf("invalid token claims")
		return result, result.Error
	}

	// Проверяем expiration
	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			result.IsExpired = true
			result.Error = fmt.Errorf("token has expired")
			return result, result.Error
		}
	}

	// Извлекаем JTI
	jti, ok := claims["jti"].(string)
	if !ok {
		result.Error = fmt.Errorf("missing jti claim")
		return result, result.Error
	}

	// Проверяем blacklist
	isBlacklisted, err := m.store.IsAccessTokenBlacklisted(ctx, jti)
	if err != nil {
		result.Error = fmt.Errorf("failed to check blacklist: %w", err)
		return result, result.Error
	}

	if isBlacklisted {
		result.IsBlacklisted = true
		result.Error = fmt.Errorf("token is blacklisted")
		return result, result.Error
	}

	// Проверяем fingerprint для защиты от XSS.
	// Если expectedFingerprint пустой — проверка пропускается (WS-режим: браузер не может
	// передать HttpOnly cookie при WS upgrade, токен итак короткоживущий).
	tokenFingerprint, _ := claims["fingerprint"].(string)
	if expectedFingerprint != "" && tokenFingerprint != expectedFingerprint {
		result.FingerprintMismatch = true
		result.Error = fmt.Errorf("fingerprint mismatch")
		return result, result.Error
	}

	// Парсим claims
	userIDStr, _ := claims["user_id"].(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		result.Error = fmt.Errorf("invalid user_id: %w", err)
		return result, result.Error
	}

	role, _ := claims["role"].(string)
	tokenType, _ := claims["token_type"].(string)
	deviceID, _ := claims["device_id"].(string)
	issuer, _ := claims["iss"].(string)
	subject, _ := claims["sub"].(string)

	iat := time.Unix(int64(claims["iat"].(float64)), 0)
	exp := time.Unix(int64(claims["exp"].(float64)), 0)
	nbf := time.Unix(int64(claims["nbf"].(float64)), 0)

	result.Claims = &AccessTokenClaims{
		UserID:      userID,
		Role:        role,
		TokenType:   tokenType,
		JTI:         jti,
		Fingerprint: tokenFingerprint,
		DeviceID:    deviceID,
		IssuedAt:    iat,
		ExpiresAt:   exp,
		NotBefore:   nbf,
		Issuer:      issuer,
		Subject:     subject,
	}
	result.Valid = true

	return result, nil
}

// ValidateRefreshToken validates a refresh token with security checks
func (m *Manager) ValidateRefreshToken(ctx context.Context, tokenID string, expectedFingerprint string) (*RefreshTokenMetadata, error) {
	// Получаем метаданные токена из хранилища
	metadata, err := m.store.GetRefreshToken(ctx, tokenID)
	if err != nil {
		return nil, fmt.Errorf("token not found: %w", err)
	}

	// Проверяем, не отозван ли токен
	if metadata.IsRevoked {
		return nil, fmt.Errorf("token has been revoked")
	}

	// Проверяем expiration
	if time.Now().After(metadata.ExpiresAt) {
		return nil, fmt.Errorf("token has expired")
	}

	// Проверяем fingerprint
	if metadata.Fingerprint != expectedFingerprint {
		// Возможная кража токена - отзываем всю семью токенов
		_ = m.store.RevokeRefreshTokenFamily(ctx, metadata.TokenFamily)
		return nil, fmt.Errorf("fingerprint mismatch - possible token theft detected")
	}

	return metadata, nil
}

// RotateRefreshToken выполняет ротацию refresh токена
func (m *Manager) RotateRefreshToken(ctx context.Context, oldTokenID string, metadata *TokenMetadata) (accessToken string, refreshToken string, err error) {
	// Получаем старый токен
	oldMetadata, err := m.store.GetRefreshToken(ctx, oldTokenID)
	if err != nil {
		return "", "", fmt.Errorf("old token not found: %w", err)
	}

	// Проверяем, не был ли токен использован (обнаружение replay атак)
	if oldMetadata.IsRevoked {
		// Токен уже был использован - возможная кража
		// Отзываем всю семью токенов
		_ = m.store.RevokeRefreshTokenFamily(ctx, oldMetadata.TokenFamily)
		return "", "", fmt.Errorf("token reuse detected - revoking token family")
	}

	// Отзываем старый токен
	if err := m.store.RevokeRefreshToken(ctx, oldTokenID); err != nil {
		return "", "", fmt.Errorf("failed to revoke old token: %w", err)
	}

	// Сохраняем token family для цепочки
	metadata.TokenFamily = oldMetadata.TokenFamily

	// Генерируем новую пару токенов
	accessToken, accessClaims, err := m.GenerateAccessToken(metadata)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, refreshMetadata, err := m.GenerateRefreshToken(metadata)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Сохраняем ссылку на предыдущий токен
	refreshMetadata.PreviousJTI = oldMetadata.TokenID

	// Сохраняем новый refresh токен
	if err := m.store.StoreRefreshToken(ctx, refreshMetadata, m.refreshTokenTTL); err != nil {
		return "", "", fmt.Errorf("failed to store refresh token: %w", err)
	}

	// Возвращаем JTI вместо самого токена для refresh (более безопасно)
	_ = accessClaims // используем для логирования если нужно

	return accessToken, refreshToken, nil
}

// RevokeAccessToken добавляет access token в blacklist
func (m *Manager) RevokeAccessToken(ctx context.Context, jti string) error {
	// TTL должен быть равен оставшемуся времени жизни токена
	// Здесь используем полный TTL для простоты
	if err := m.store.BlacklistAccessToken(ctx, jti, m.accessTokenTTL); err != nil {
		return fmt.Errorf("failed to blacklist token: %w", err)
	}
	return nil
}

// RevokeRefreshToken отзывает refresh token
func (m *Manager) RevokeRefreshToken(ctx context.Context, tokenID string) error {
	if err := m.store.RevokeRefreshToken(ctx, tokenID); err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}
	return nil
}

// RevokeAllUserTokens отзывает все токены пользователя
func (m *Manager) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	if err := m.store.RevokeAllUserTokens(ctx, userID); err != nil {
		return fmt.Errorf("failed to revoke all tokens: %w", err)
	}
	return nil
}

// IntrospectAccessToken проверяет токен без валидации fingerprint (для admin API)
func (m *Manager) IntrospectAccessToken(ctx context.Context, tokenString string) (*AccessTokenClaims, error) {
	_ = ctx // используется для будущих расширений (audit log и т.д.)

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Парсим claims
	userIDStr, _ := claims["user_id"].(string)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}

	role, _ := claims["role"].(string)
	tokenType, _ := claims["token_type"].(string)
	jti, _ := claims["jti"].(string)
	fingerprint, _ := claims["fingerprint"].(string)
	deviceID, _ := claims["device_id"].(string)
	issuer, _ := claims["iss"].(string)
	subject, _ := claims["sub"].(string)

	iat := time.Unix(int64(claims["iat"].(float64)), 0)
	exp := time.Unix(int64(claims["exp"].(float64)), 0)
	nbf := time.Unix(int64(claims["nbf"].(float64)), 0)

	return &AccessTokenClaims{
		UserID:      userID,
		Role:        role,
		TokenType:   tokenType,
		JTI:         jti,
		Fingerprint: fingerprint,
		DeviceID:    deviceID,
		IssuedAt:    iat,
		ExpiresAt:   exp,
		NotBefore:   nbf,
		Issuer:      issuer,
		Subject:     subject,
	}, nil
}

// GetRefreshTokenTTL returns the refresh token TTL
func (m *Manager) GetRefreshTokenTTL() time.Duration {
	return m.refreshTokenTTL
}

// StoreRefreshToken сохраняет refresh token в хранилище
func (m *Manager) StoreRefreshToken(ctx context.Context, metadata *RefreshTokenMetadata) error {
	return m.store.StoreRefreshToken(ctx, metadata, m.refreshTokenTTL)
}
