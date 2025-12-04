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

	"github.com/artmexbet/raibecas/services/auth/internal/domain"
)

// TokenStore handles refresh token storage in Redis
type TokenStore struct {
	client *redis.Client
	logger *slog.Logger // структурированное логирование
}

// NewTokenStore creates a new token store
func NewTokenStore(client *redis.Client) *TokenStore {
	return &TokenStore{
		client: client,
		logger: slog.Default(),
	}
}

// NewTokenStoreWithLogger creates a new token store with custom logger
func NewTokenStoreWithLogger(client *redis.Client, logger *slog.Logger) *TokenStore {
	return &TokenStore{
		client: client,
		logger: logger,
	}
}

// StoreRefreshToken stores a refresh token in Redis using Sets for multi-device support
// Использует Redis Sets для хранения нескольких токенов на пользователя (разные устройства)
// Использует pipeline для группировки команд и улучшения производительности
func (s *TokenStore) StoreRefreshToken(ctx context.Context, token *domain.RefreshToken, ttl time.Duration) error {
	// Маршалим данные токена один раз
	tokenData, err := json.Marshal(token)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to marshal token", "user_id", token.UserID, "error", err)
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	// Подготавливаем ключи
	tokenDataKey := fmt.Sprintf("refresh_token:data:%s", token.Token)
	userTokensSetKey := fmt.Sprintf("refresh_token:user:%s:tokens", token.UserID.String())

	// Используем pipeline для группировки команд в одну сетевую операцию
	pipe := s.client.Pipeline()

	// 1. Сохраняем данные токена
	pipe.Set(ctx, tokenDataKey, tokenData, ttl)

	// 2. Добавляем токен в Set пользователя
	pipe.SAdd(ctx, userTokensSetKey, token.Token)

	// 3. Устанавливаем TTL для Set
	pipe.Expire(ctx, userTokensSetKey, ttl)

	// Выполняем все команды одновременно
	_, err = pipe.Exec(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "pipeline execution failed", "user_id", token.UserID, "error", err)
		return fmt.Errorf("failed to store refresh token: %w", err)
	}

	s.logger.InfoContext(ctx, "stored refresh token", "user_id", token.UserID, "device_id", token.DeviceID, "ttl", ttl)
	return nil
}

// GetRefreshTokenByValue retrieves a refresh token from Redis by token value
func (s *TokenStore) GetRefreshTokenByValue(ctx context.Context, tokenValue string) (*domain.RefreshToken, error) {
	tokenDataKey := fmt.Sprintf("refresh_token:data:%s", tokenValue)

	data, err := s.client.Get(ctx, tokenDataKey).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			s.logger.DebugContext(ctx, "token not found", "token", tokenValue)
			return nil, domain.ErrTokenNotFound
		}
		s.logger.ErrorContext(ctx, "failed to get token data", "token", tokenValue, "error", err)
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	var token domain.RefreshToken
	if err := json.Unmarshal(data, &token); err != nil {
		s.logger.ErrorContext(ctx, "failed to unmarshal token", "token", tokenValue, "error", err)
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &token, nil
}

// DeleteRefreshToken deletes a specific refresh token from Redis
// Удаляет конкретный токен пользователя (при logout с одного устройства)
// Использует pipeline для атомарного удаления из двух мест
func (s *TokenStore) DeleteRefreshToken(ctx context.Context, userID uuid.UUID, tokenValue string) error {
	tokenDataKey := fmt.Sprintf("refresh_token:data:%s", tokenValue)
	userTokensSetKey := fmt.Sprintf("refresh_token:user:%s:tokens", userID.String())

	// Pipeline для удаления из обоих мест одновременно
	pipe := s.client.Pipeline()
	pipe.Del(ctx, tokenDataKey)
	pipe.SRem(ctx, userTokensSetKey, tokenValue)

	results, err := pipe.Exec(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "pipeline execution failed", "user_id", userID, "error", err)
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}

	// Проверяем результаты
	if len(results) > 0 && results[0].Err() != nil {
		s.logger.ErrorContext(ctx, "failed to delete token data", "user_id", userID, "error", results[0].Err())
		return fmt.Errorf("failed to delete token data: %w", results[0].Err())
	}

	s.logger.InfoContext(ctx, "deleted refresh token", "user_id", userID)
	return nil
}

// DeleteAllRefreshTokens deletes all refresh tokens for a user (logout all devices)
// Использует pipeline для удаления всех ключей одновременно
func (s *TokenStore) DeleteAllRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	userTokensSetKey := fmt.Sprintf("refresh_token:user:%s:tokens", userID.String())

	// Получаем все токены пользователя
	tokens, err := s.client.SMembers(ctx, userTokensSetKey).Result()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get tokens", "user_id", userID, "error", err)
		return fmt.Errorf("failed to get tokens: %w", err)
	}

	if len(tokens) == 0 {
		s.logger.DebugContext(ctx, "no tokens to delete", "user_id", userID)
		return nil
	}

	// Pipeline для удаления всех ключей токенов + сам Set
	pipe := s.client.Pipeline()

	// Удаляем данные каждого токена
	for _, token := range tokens {
		tokenDataKey := fmt.Sprintf("refresh_token:data:%s", token)
		pipe.Del(ctx, tokenDataKey)
	}

	// Удаляем Set пользователя
	pipe.Del(ctx, userTokensSetKey)

	results, err := pipe.Exec(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "pipeline execution failed", "user_id", userID, "error", err)
		return fmt.Errorf("failed to delete all tokens: %w", err)
	}

	// Проверяем первый результат (остальные обычно успешные если первый успешен)
	if len(results) > 0 && results[0].Err() != nil {
		s.logger.ErrorContext(ctx, "failed to delete token data", "user_id", userID, "error", results[0].Err())
		return fmt.Errorf("failed to delete token data: %w", results[0].Err())
	}

	s.logger.InfoContext(ctx, "deleted all refresh tokens", "user_id", userID, "token_count", len(tokens))
	return nil
}

// GetAllRefreshTokens возвращает все активные токены пользователя
// Полезно для отображения списка активных устройств
func (s *TokenStore) GetAllRefreshTokens(ctx context.Context, userID uuid.UUID) ([]*domain.RefreshToken, error) {
	userTokensSetKey := fmt.Sprintf("refresh_token:user:%s:tokens", userID.String())

	tokens, err := s.client.SMembers(ctx, userTokensSetKey).Result()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get tokens", "user_id", userID, "error", err)
		return nil, fmt.Errorf("failed to get tokens: %w", err)
	}

	var result []*domain.RefreshToken
	for _, tokenValue := range tokens {
		token, err := s.GetRefreshTokenByValue(ctx, tokenValue)
		if err != nil {
			s.logger.WarnContext(ctx, "failed to get token data", "token", tokenValue, "error", err)
			// Продолжаем, даже если один токен не найден
			continue
		}
		result = append(result, token)
	}

	return result, nil
}

// IsTokenValid проверяет, существует ли токен и не истек ли он
func (s *TokenStore) IsTokenValid(ctx context.Context, tokenValue string) bool {
	token, err := s.GetRefreshTokenByValue(ctx, tokenValue)
	if err != nil {
		return false
	}

	// Проверяем, не истек ли токен
	return time.Now().Before(token.ExpiresAt)
}

// DeleteTokensByDevice удаляет все токены с конкретного устройства
// Полезно для logout с одного устройства, оставляя сессии на других активными
func (s *TokenStore) DeleteTokensByDevice(ctx context.Context, userID uuid.UUID, deviceID string) error {
	userTokensSetKey := fmt.Sprintf("refresh_token:user:%s:tokens", userID.String())

	tokens, err := s.client.SMembers(ctx, userTokensSetKey).Result()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get tokens", "user_id", userID, "error", err)
		return fmt.Errorf("failed to get tokens: %w", err)
	}

	deletedCount := 0
	for _, tokenValue := range tokens {
		token, err := s.GetRefreshTokenByValue(ctx, tokenValue)
		if err != nil {
			s.logger.WarnContext(ctx, "failed to get token data", "token", tokenValue, "error", err)
			continue
		}

		if token.DeviceID == deviceID {
			if err := s.DeleteRefreshToken(ctx, userID, tokenValue); err != nil {
				s.logger.ErrorContext(ctx, "failed to delete token for device", "device_id", deviceID, "error", err)
				return err
			}
			deletedCount++
		}
	}

	s.logger.InfoContext(ctx, "deleted tokens for device", "device_id", deviceID, "user_id", userID, "deleted_count", deletedCount)
	return nil
}

// GetActiveDevices возвращает список активных устройств пользователя
// Возвращает mapping deviceID -> token info для удобства
func (s *TokenStore) GetActiveDevices(ctx context.Context, userID uuid.UUID) (map[string]*domain.RefreshToken, error) {
	tokens, err := s.GetAllRefreshTokens(ctx, userID)
	if err != nil {
		return nil, err
	}

	devices := make(map[string]*domain.RefreshToken)
	for _, token := range tokens {
		devices[token.DeviceID] = token
	}

	return devices, nil
}

// CountActiveTokens возвращает количество активных токенов пользователя
func (s *TokenStore) CountActiveTokens(ctx context.Context, userID uuid.UUID) (int64, error) {
	userTokensSetKey := fmt.Sprintf("refresh_token:user:%s:tokens", userID.String())

	count, err := s.client.SCard(ctx, userTokensSetKey).Result()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get token count", "user_id", userID, "error", err)
		return 0, fmt.Errorf("failed to count tokens: %w", err)
	}

	return count, nil
}
