package redis

import (
	"github.com/artmexbet/raibecas/services/chat/internal/config"
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
