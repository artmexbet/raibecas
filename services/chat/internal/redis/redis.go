package redis

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/artmexbet/raibecas/services/chat/internal/config"
	"github.com/artmexbet/raibecas/services/chat/internal/domain"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	cfg    *config.Redis
	client *redis.Client
}

func New(cfg *config.Redis, client *redis.Client) *Redis {
	return &Redis{
		cfg:    cfg,
		client: client,
	}
}

// RetrieveChatHistory retrieves chat history for a given user ID from Redis.
// May be expanded in the future to include more complex retrieval logic.
func (r *Redis) RetrieveChatHistory(ctx context.Context, userID string) ([]domain.Message, error) {
	val, err := r.client.Get(ctx, userID).Bytes()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve history: %w", err)
	}
	var result []domain.Message
	err = json.Unmarshal(val, &result)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal history: %w", err)
	}
	return result, nil
}
