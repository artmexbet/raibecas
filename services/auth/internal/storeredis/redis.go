package storeredis

type Config struct {
}

type RedisStorer struct {
	cfg Config
}

func NewRedisStorer(cfg Config) *RedisStorer {
	return &RedisStorer{
		cfg: cfg,
	}
}
