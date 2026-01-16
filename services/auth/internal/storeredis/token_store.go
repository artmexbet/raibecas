package storeredis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/artmexbet/raibecas/services/auth/pkg/jwt"
)

// TokenStoreRedis implements jwt.TokenStore interface with Redis backend
type TokenStoreRedis struct {
	client *redis.Client
	logger *slog.Logger
}

// NewTokenStoreRedis creates a new Redis-backed token store
func NewTokenStoreRedis(client *redis.Client, logger *slog.Logger) *TokenStoreRedis {
	if logger == nil {
		logger = slog.Default()
	}
	return &TokenStoreRedis{
		client: client,
		logger: logger,
	}
}

// Redis key patterns
const (
	// Refresh tokens
	keyRefreshToken       = "auth:refresh:%s"                // auth:refresh:{tokenID}
	keyUserRefreshTokens  = "auth:user:%s:refresh"           // auth:user:{userID}:refresh (Set)
	keyDeviceRefreshToken = "auth:user:%s:device:%s:refresh" // auth:user:{userID}:device:{deviceID}:refresh
	keyTokenFamily        = "auth:family:%s"                 // auth:family:{familyID} (Set)

	// Access token blacklist
	keyAccessBlacklist = "auth:blacklist:%s" // auth:blacklist:{jti}

	// Indexes for fast lookups
	keyAllFamilies = "auth:families" // Set всех семей токенов
)

// StoreRefreshToken stores a refresh token with all metadata
func (s *TokenStoreRedis) StoreRefreshToken(ctx context.Context, metadata *jwt.RefreshTokenMetadata, ttl time.Duration) error {
	tokenKey := fmt.Sprintf(keyRefreshToken, metadata.TokenID)
	userTokensKey := fmt.Sprintf(keyUserRefreshTokens, metadata.UserID.String())
	deviceTokenKey := fmt.Sprintf(keyDeviceRefreshToken, metadata.UserID.String(), metadata.DeviceID)
	familyKey := fmt.Sprintf(keyTokenFamily, metadata.TokenFamily)

	// Сериализуем метаданные
	data, err := json.Marshal(metadata)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to marshal token metadata",
			"token_id", metadata.TokenID,
			"user_id", metadata.UserID,
			"error", err)
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Используем pipeline для атомарности
	pipe := s.client.Pipeline()

	// 1. Сохраняем полные метаданные токена
	pipe.Set(ctx, tokenKey, data, ttl)

	// 2. Добавляем токен в индекс пользователя
	pipe.SAdd(ctx, userTokensKey, metadata.TokenID)
	pipe.Expire(ctx, userTokensKey, ttl)

	// 3. Сохраняем привязку к устройству
	pipe.Set(ctx, deviceTokenKey, metadata.TokenID, ttl)

	// 4. Добавляем токен в семью (для отслеживания ротации)
	pipe.SAdd(ctx, familyKey, metadata.TokenID)
	pipe.Expire(ctx, familyKey, ttl)

	// 5. Добавляем семью в глобальный индекс
	pipe.SAdd(ctx, keyAllFamilies, metadata.TokenFamily)

	// Выполняем все команды
	if _, err := pipe.Exec(ctx); err != nil {
		s.logger.ErrorContext(ctx, "failed to store refresh token",
			"token_id", metadata.TokenID,
			"user_id", metadata.UserID,
			"device_id", metadata.DeviceID,
			"error", err)
		return fmt.Errorf("failed to store refresh token: %w", err)
	}

	s.logger.InfoContext(ctx, "stored refresh token",
		"token_id", metadata.TokenID,
		"user_id", metadata.UserID,
		"device_id", metadata.DeviceID,
		"family", metadata.TokenFamily,
		"ttl", ttl)

	return nil
}

// GetRefreshToken retrieves refresh token metadata
func (s *TokenStoreRedis) GetRefreshToken(ctx context.Context, tokenID string) (*jwt.RefreshTokenMetadata, error) {
	tokenKey := fmt.Sprintf(keyRefreshToken, tokenID)

	data, err := s.client.Get(ctx, tokenKey).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			s.logger.DebugContext(ctx, "refresh token not found", "token_id", tokenID)
			return nil, fmt.Errorf("token not found")
		}
		s.logger.ErrorContext(ctx, "failed to get refresh token", "token_id", tokenID, "error", err)
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	var metadata jwt.RefreshTokenMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal token metadata", "token_id", tokenID, "error", err)
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &metadata, nil
}

// RevokeRefreshToken отзывает конкретный refresh token
func (s *TokenStoreRedis) RevokeRefreshToken(ctx context.Context, tokenID string) error {
	// Получаем метаданные для очистки всех связанных ключей
	metadata, err := s.GetRefreshToken(ctx, tokenID)
	if err != nil {
		return err
	}

	// Обновляем статус отзыва
	metadata.IsRevoked = true
	now := time.Now()
	metadata.RevokedAt = &now

	// Сохраняем обновлённые метаданные
	tokenKey := fmt.Sprintf(keyRefreshToken, tokenID)
	data, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Используем оставшееся TTL
	ttl, err := s.client.TTL(ctx, tokenKey).Result()
	if err != nil {
		ttl = time.Hour // fallback
	}

	if err := s.client.Set(ctx, tokenKey, data, ttl).Err(); err != nil {
		s.logger.ErrorContext(ctx, "failed to revoke token", "token_id", tokenID, "error", err)
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	s.logger.InfoContext(ctx, "revoked refresh token",
		"token_id", tokenID,
		"user_id", metadata.UserID,
		"family", metadata.TokenFamily)

	return nil
}

// RevokeRefreshTokenFamily отзывает всю семью токенов (при обнаружении кражи)
func (s *TokenStoreRedis) RevokeRefreshTokenFamily(ctx context.Context, tokenFamily string) error {
	familyKey := fmt.Sprintf(keyTokenFamily, tokenFamily)

	// Получаем все токены в семье
	tokenIDs, err := s.client.SMembers(ctx, familyKey).Result()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get token family", "family", tokenFamily, "error", err)
		return fmt.Errorf("failed to get token family: %w", err)
	}

	if len(tokenIDs) == 0 {
		s.logger.DebugContext(ctx, "token family is empty", "family", tokenFamily)
		return nil
	}

	// Отзываем каждый токен в семье
	var revokeErrors []error
	for _, tokenID := range tokenIDs {
		if err := s.RevokeRefreshToken(ctx, tokenID); err != nil {
			revokeErrors = append(revokeErrors, err)
			s.logger.WarnContext(ctx, "failed to revoke token in family",
				"token_id", tokenID,
				"family", tokenFamily,
				"error", err)
		}
	}

	if len(revokeErrors) > 0 {
		s.logger.ErrorContext(ctx, "partially revoked token family",
			"family", tokenFamily,
			"total", len(tokenIDs),
			"errors", len(revokeErrors))
		return fmt.Errorf("failed to revoke %d tokens in family", len(revokeErrors))
	}

	s.logger.WarnContext(ctx, "revoked entire token family (possible theft detected)",
		"family", tokenFamily,
		"revoked_count", len(tokenIDs))

	return nil
}

// RevokeAllUserTokens отзывает все токены пользователя
func (s *TokenStoreRedis) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	userTokensKey := fmt.Sprintf(keyUserRefreshTokens, userID.String())

	// Получаем все токены пользователя
	tokenIDs, err := s.client.SMembers(ctx, userTokensKey).Result()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get user tokens", "user_id", userID, "error", err)
		return fmt.Errorf("failed to get user tokens: %w", err)
	}

	if len(tokenIDs) == 0 {
		s.logger.DebugContext(ctx, "user has no tokens", "user_id", userID)
		return nil
	}

	// Отзываем каждый токен
	pipe := s.client.Pipeline()
	for _, tokenID := range tokenIDs {
		tokenKey := fmt.Sprintf(keyRefreshToken, tokenID)
		pipe.Del(ctx, tokenKey)
	}

	// Очищаем индексы пользователя
	pipe.Del(ctx, userTokensKey)

	if _, err := pipe.Exec(ctx); err != nil {
		s.logger.ErrorContext(ctx, "failed to revoke user tokens", "user_id", userID, "error", err)
		return fmt.Errorf("failed to revoke user tokens: %w", err)
	}

	s.logger.InfoContext(ctx, "revoked all user tokens",
		"user_id", userID,
		"revoked_count", len(tokenIDs))

	return nil
}

// BlacklistAccessToken добавляет access token в blacklist
func (s *TokenStoreRedis) BlacklistAccessToken(ctx context.Context, jti string, ttl time.Duration) error {
	blacklistKey := fmt.Sprintf(keyAccessBlacklist, jti)

	if err := s.client.Set(ctx, blacklistKey, "1", ttl).Err(); err != nil {
		s.logger.ErrorContext(ctx, "failed to blacklist access token", "jti", jti, "error", err)
		return fmt.Errorf("failed to blacklist token: %w", err)
	}

	s.logger.InfoContext(ctx, "blacklisted access token", "jti", jti, "ttl", ttl)
	return nil
}

// IsAccessTokenBlacklisted проверяет, находится ли токен в blacklist
func (s *TokenStoreRedis) IsAccessTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	blacklistKey := fmt.Sprintf(keyAccessBlacklist, jti)

	exists, err := s.client.Exists(ctx, blacklistKey).Result()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to check token blacklist", "jti", jti, "error", err)
		return false, fmt.Errorf("failed to check blacklist: %w", err)
	}

	return exists > 0, nil
}

// GetTokenFamily возвращает все токены в семье
func (s *TokenStoreRedis) GetTokenFamily(ctx context.Context, tokenFamily string) ([]*jwt.RefreshTokenMetadata, error) {
	familyKey := fmt.Sprintf(keyTokenFamily, tokenFamily)

	tokenIDs, err := s.client.SMembers(ctx, familyKey).Result()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get token family", "family", tokenFamily, "error", err)
		return nil, fmt.Errorf("failed to get token family: %w", err)
	}

	var tokens []*jwt.RefreshTokenMetadata
	for _, tokenID := range tokenIDs {
		metadata, err := s.GetRefreshToken(ctx, tokenID)
		if err != nil {
			s.logger.WarnContext(ctx, "failed to get token in family",
				"token_id", tokenID,
				"family", tokenFamily,
				"error", err)
			continue
		}
		tokens = append(tokens, metadata)
	}

	return tokens, nil
}

// GetUserDevices возвращает список устройств пользователя
func (s *TokenStoreRedis) GetUserDevices(ctx context.Context, userID uuid.UUID) ([]string, error) {
	userTokensKey := fmt.Sprintf(keyUserRefreshTokens, userID.String())

	tokenIDs, err := s.client.SMembers(ctx, userTokensKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user tokens: %w", err)
	}

	deviceMap := make(map[string]bool)
	for _, tokenID := range tokenIDs {
		metadata, err := s.GetRefreshToken(ctx, tokenID)
		if err != nil {
			continue
		}
		deviceMap[metadata.DeviceID] = true
	}

	devices := make([]string, 0, len(deviceMap))
	for device := range deviceMap {
		devices = append(devices, device)
	}

	return devices, nil
}

// RevokeDeviceTokens отзывает все токены конкретного устройства
func (s *TokenStoreRedis) RevokeDeviceTokens(ctx context.Context, userID uuid.UUID, deviceID string) error {
	userTokensKey := fmt.Sprintf(keyUserRefreshTokens, userID.String())

	// Получаем все токены пользователя
	tokenIDs, err := s.client.SMembers(ctx, userTokensKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get user tokens: %w", err)
	}

	revokedCount := 0
	for _, tokenID := range tokenIDs {
		metadata, err := s.GetRefreshToken(ctx, tokenID)
		if err != nil {
			continue
		}

		if metadata.DeviceID == deviceID {
			if err := s.RevokeRefreshToken(ctx, tokenID); err != nil {
				s.logger.WarnContext(ctx, "failed to revoke device token",
					"token_id", tokenID,
					"device_id", deviceID,
					"error", err)
				continue
			}
			revokedCount++
		}
	}

	s.logger.InfoContext(ctx, "revoked device tokens",
		"user_id", userID,
		"device_id", deviceID,
		"revoked_count", revokedCount)

	return nil
}
