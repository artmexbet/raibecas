package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/artmexbet/raibecas/services/auth/internal/domain"
	"github.com/artmexbet/raibecas/services/auth/internal/postgres/queries"
)

func (p *Postgres) CreateUser(ctx context.Context, user *domain.User) error {
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // safe to ignore

	q := p.q.WithTx(tx)
	_, err = q.CreateUser(ctx, queries.CreateUserParams{
		Username:     user.Username,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
	})
	if err != nil {
		return err
	}
	tx.Commit(ctx) //nolint:errcheck // safe to ignore
	return nil
}

func (p *Postgres) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, err := p.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("error getting user by id: %v", err)
	}
	user, err := u.ToDomain()
	if err != nil {
		return nil, fmt.Errorf("error converting user to domain: %v", err)
	}
	return &user, nil
}

func (p *Postgres) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, err := p.q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("error getting user by email: %v", err)
	}
	user, err := u.ToDomain()
	if err != nil {
		return nil, fmt.Errorf("error converting user to domain: %v", err)
	}
	return &user, nil
}

func (p *Postgres) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	u, err := p.q.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("error getting user by username: %v", err)
	}
	user, err := u.ToDomain()
	if err != nil {
		return nil, fmt.Errorf("error converting user to domain: %v", err)
	}
	return &user, nil
}

func (p *Postgres) ExistsUserByEmail(ctx context.Context, email string) (bool, error) {
	exists, err := p.q.UserExistsByEmail(ctx, email)
	if err != nil {
		return false, fmt.Errorf("error checking if user exists by email: %v", err)
	}
	return exists, nil
}

func (p *Postgres) ExistsUserByUsername(ctx context.Context, username string) (bool, error) {
	exists, err := p.q.UserExistsByUsername(ctx, username)
	if err != nil {
		return false, fmt.Errorf("error checking if user exists by username: %v", err)
	}
	return exists, nil
}

func (p *Postgres) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // safe to ignore

	q := p.q.WithTx(tx)

	err = q.UpdateUserPassword(ctx, queries.UpdateUserPasswordParams{
		ID:           userID,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return fmt.Errorf("error updating user password: %v", err)
	}
	tx.Commit(ctx) //nolint:errcheck // safe to ignore
	return nil
}
