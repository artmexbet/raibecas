package postgres

import (
	"auth/internal/postgres/queries"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:generate sqlc generate

type Postgres struct {
	pool *pgxpool.Pool

	q *queries.Queries
}

func New(pool *pgxpool.Pool) *Postgres {
	return &Postgres{
		pool: pool,
		q:    queries.New(pool),
	}
}

func (p *Postgres) Close() {
	p.pool.Close()
}
