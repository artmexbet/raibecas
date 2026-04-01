package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/artmexbet/raibecas/services/chat/internal/config"
)

//go:generate sqlc generate

// Store holds the pgxpool connection pool.
type Store struct {
	pool *pgxpool.Pool
}

// New creates a new Store, connecting to PostgreSQL using the provided config.
func New(ctx context.Context, cfg *config.Database) (*Store, error) {
	pool, err := pgxpool.New(ctx, cfg.GetDSN())
	if err != nil {
		return nil, fmt.Errorf("pgxpool.New: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}

	return &Store{pool: pool}, nil
}

// Close closes the connection pool.
func (s *Store) Close() {
	s.pool.Close()
}
