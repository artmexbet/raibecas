package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/artmexbet/raibecas/services/chat/internal/config"
	"github.com/artmexbet/raibecas/services/chat/internal/domain"
)

type Redis struct {
	cfg    *config.Redis
	client *redis.Client
}

// New creates a new Redis instance with the provided configuration and Redis client.
// cfg is the Redis configuration, and client is the Redis client to use for operations.
func New(cfg *config.Redis, client *redis.Client) *Redis {
	return &Redis{
		cfg:    cfg,
		client: client,
	}
}

// getChatHistoryKey returns the key for storing chat history for a user
func (r *Redis) getChatHistoryKey(userID string) string {
	return fmt.Sprintf("chat:history:%s", userID)
}

// getTemporaryMessageKey returns the key for temporary message chunks
func (r *Redis) getTemporaryMessageKey(userID string) string {
	return fmt.Sprintf("chat:temp_msg:%s", userID)
}

// RetrieveChatHistory retrieves chat history for a given user ID from Redis.
// May be expanded in the future to include more complex retrieval logic.
func (r *Redis) RetrieveChatHistory(ctx context.Context, userID string) ([]domain.Message, error) {
	key := r.getChatHistoryKey(userID)
	val, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return []domain.Message{}, nil
		}
		return nil, fmt.Errorf("could not retrieve history: %w", err)
	}
	var result []domain.Message
	err = json.Unmarshal(val, &result)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal history: %w", err)
	}
	return result, nil
}

// SaveMessage saves a message to chat history and sets TTL
func (r *Redis) SaveMessage(ctx context.Context, userID string, message domain.Message) error {
	key := r.getChatHistoryKey(userID)

	// Get existing history
	history, err := r.RetrieveChatHistory(ctx, userID)
	if err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("could not retrieve existing history: %w", err)
	} else if errors.Is(err, redis.Nil) {
		history = []domain.Message{}
	}

	// Append new message
	history = append(history, message)

	// Marshal updated history
	data, err := json.Marshal(history)
	if err != nil {
		return fmt.Errorf("could not marshal history: %w", err)
	}

	// Save with TTL
	ttl := r.cfg.ChatTTL
	err = r.client.Set(ctx, key, data, ttl).Err()
	if err != nil {
		return fmt.Errorf("could not save message: %w", err)
	}

	return nil
}

// ClearChatHistory clears all messages for a user
func (r *Redis) ClearChatHistory(ctx context.Context, userID string) error {
	key := r.getChatHistoryKey(userID)
	tempKey := r.getTemporaryMessageKey(userID)

	// Delete both history and temporary message
	err := r.client.Del(ctx, key, tempKey).Err()
	if err != nil {
		return fmt.Errorf("could not clear chat history: %w", err)
	}

	return nil
}

// GetChatSize returns the number of messages in the chat history
func (r *Redis) GetChatSize(ctx context.Context, userID string) (int, error) {
	history, err := r.RetrieveChatHistory(ctx, userID)
	if err != nil {
		return 0, err
	}
	return len(history), nil
}
