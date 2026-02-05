package postgres

import (
	"context"
	"fmt"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"

	"github.com/artmexbet/raibecas/services/documents/internal/config"
	"github.com/artmexbet/raibecas/services/documents/internal/postgres/queries"
)

//go:generate sqlc generate

// NewQueries creates a new PostgreSQL connection pool and returns Queries instance
func NewQueries(ctx context.Context, cfg config.DatabaseConfig) (*queries.Queries, *pgxpool.Pool, error) {
	pgCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse postgres config: %w", err)
	}

	pgCfg.MaxConns = int32(cfg.MaxConns)
	pgCfg.MinConns = int32(cfg.MinConns)

	// Add OpenTelemetry tracing
	pgCfg.ConnConfig.Tracer = otelpgx.NewTracer(
		otelpgx.WithTracerProvider(otel.GetTracerProvider()),
		otelpgx.WithTrimSQLInSpanName(),
	)

	pool, err := pgxpool.NewWithConfig(ctx, pgCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection and close pool if ping fails
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return queries.New(pool), pool, nil
}
