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
	// Store token data by user ID
	userKey := fmt.Sprintf("refresh_token:user:%s", token.UserID.String())
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}
	if err = s.client.Set(ctx, userKey, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to store refresh token: %w", err)
	}

	// Store user ID mapping by token value for reverse lookup
	tokenKey := fmt.Sprintf("refresh_token:value:%s", token.Token)
	if err = s.client.Set(ctx, tokenKey, token.UserID.String(), ttl).Err(); err != nil {
		return fmt.Errorf("failed to store refresh token mapping: %w", err)
	}

	return nil
}

// GetRefreshToken retrieves a refresh token from Redis by user ID
func (s *TokenStore) GetRefreshToken(ctx context.Context, userID uuid.UUID) (*domain.RefreshToken, error) {
	key := fmt.Sprintf("refresh_token:user:%s", userID.String())

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

// GetRefreshTokenByValue retrieves a refresh token from Redis by token value
func (s *TokenStore) GetRefreshTokenByValue(ctx context.Context, tokenValue string) (*domain.RefreshToken, error) {
	// Get user ID from token value mapping
	tokenKey := fmt.Sprintf("refresh_token:value:%s", tokenValue)
	userIDStr, err := s.client.Get(ctx, tokenKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, domain.ErrTokenNotFound
		}
		return nil, fmt.Errorf("failed to get token mapping: %w", err)
	}

	// Parse user ID
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Get token data by user ID
	return s.GetRefreshToken(ctx, userID)
}

// DeleteRefreshToken deletes a refresh token from Redis
func (s *TokenStore) DeleteRefreshToken(ctx context.Context, userID uuid.UUID) error {
	// Get token to find its value for deletion
	token, err := s.GetRefreshToken(ctx, userID)
	if err != nil && err != domain.ErrTokenNotFound {
		return fmt.Errorf("failed to get refresh token: %w", err)
	}

	// Delete user mapping
	userKey := fmt.Sprintf("refresh_token:user:%s", userID.String())
	if err := s.client.Del(ctx, userKey).Err(); err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}

	// Delete token value mapping if token exists
	if token != nil {
		tokenKey := fmt.Sprintf("refresh_token:value:%s", token.Token)
		if err := s.client.Del(ctx, tokenKey).Err(); err != nil {
			return fmt.Errorf("failed to delete token mapping: %w", err)
		}
	}

	return nil
}

// DeleteAllRefreshTokens deletes all refresh tokens for a user (logout all devices)
func (s *TokenStore) DeleteAllRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	// Get token to find its value for deletion
	token, err := s.GetRefreshToken(ctx, userID)
	if err != nil && err != domain.ErrTokenNotFound {
		return fmt.Errorf("failed to get refresh token: %w", err)
	}

	// Delete all user mappings
	pattern := fmt.Sprintf("refresh_token:user:%s*", userID.String())
	iter := s.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := s.client.Del(ctx, iter.Val()).Err(); err != nil {
			return fmt.Errorf("failed to delete token: %w", err)
		}
	}
	if err := iter.Err(); err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	// Delete token value mappings
	if token != nil {
		tokenKey := fmt.Sprintf("refresh_token:value:%s", token.Token)
		if err := s.client.Del(ctx, tokenKey).Err(); err != nil {
			return fmt.Errorf("failed to delete token mapping: %w", err)
		}
	}

	return nil
}
