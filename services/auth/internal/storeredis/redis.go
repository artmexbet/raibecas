package storeredis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"auth/internal/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// TokenStore handles refresh token storage in Redis
type TokenStore struct {
	client *redis.Client
}

// NewTokenStore creates a new token store
func NewTokenStore(client *redis.Client) *TokenStore {
	return &TokenStore{
		client: client,
	}
}

// StoreRefreshToken stores a refresh token in Redis
func (s *TokenStore) StoreRefreshToken(ctx context.Context, token *domain.RefreshToken, ttl time.Duration) error {
	key := fmt.Sprintf("refresh_token:%s", token.UserID.String())

	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	err = s.client.Set(ctx, key, data, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to store refresh token: %w", err)
	}

	return nil
}

// GetRefreshToken retrieves a refresh token from Redis
func (s *TokenStore) GetRefreshToken(ctx context.Context, userID uuid.UUID) (*domain.RefreshToken, error) {
	key := fmt.Sprintf("refresh_token:%s", userID.String())

	data, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, domain.ErrTokenNotFound
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	var token domain.RefreshToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &token, nil
}

// DeleteRefreshToken deletes a refresh token from Redis
func (s *TokenStore) DeleteRefreshToken(ctx context.Context, userID uuid.UUID) error {
	key := fmt.Sprintf("refresh_token:%s", userID.String())

	err := s.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}

	return nil
}

// DeleteAllRefreshTokens deletes all refresh tokens for a user (logout all devices)
func (s *TokenStore) DeleteAllRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	pattern := fmt.Sprintf("refresh_token:%s*", userID.String())

	iter := s.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := s.client.Del(ctx, iter.Val()).Err(); err != nil {
			return fmt.Errorf("failed to delete token: %w", err)
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	return nil
}
