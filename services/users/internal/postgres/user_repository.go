package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/artmexbet/raibecas/services/users/internal/domain"
	"github.com/artmexbet/raibecas/services/users/internal/postgres/queries"
)

type ListUsersParams struct {
	Limit    int
	Offset   int
	Search   string
	IsActive *bool
}

func (p *Postgres) ListUsers(ctx context.Context, params ListUsersParams) ([]domain.User, int, error) {
	// Count
	total, err := p.q.CountUsers(ctx, queries.CountUsersParams{
		Search:         params.Search,
		IsActiveFilter: params.IsActive,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// List
	dbUsers, err := p.q.ListUsers(ctx, queries.ListUsersParams{
		Search:         params.Search,
		IsActiveFilter: params.IsActive,
		Limit:          int32(params.Limit),
		Offset:         int32(params.Offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	users := make([]domain.User, len(dbUsers))
	for i, u := range dbUsers {
		var fullName string
		if u.FullName != nil {
			fullName = *u.FullName
		}
		users[i] = domain.User{
			ID:          u.ID,
			Email:       u.Email,
			Username:    u.Username,
			FullName:    fullName,
			Role:        string(u.Role),
			IsActive:    u.IsActive,
			CreatedAt:   u.CreatedAt,
			LastLoginAt: u.LastLoginAt,
			UpdatedAt:   u.UpdatedAt,
		}
	}

	return users, int(total), nil
}

func (p *Postgres) CountTotalUsers(ctx context.Context) (int64, error) {
	total, err := p.q.CountUsers(ctx, queries.CountUsersParams{})
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}
	return total, nil
}

func (p *Postgres) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, err := p.q.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	var fullName string
	if u.FullName != nil {
		fullName = *u.FullName
	}
	return &domain.User{
		ID:          u.ID,
		Email:       u.Email,
		Username:    u.Username,
		FullName:    fullName,
		Role:        string(u.Role),
		IsActive:    u.IsActive,
		CreatedAt:   u.CreatedAt,
		LastLoginAt: u.LastLoginAt,
		UpdatedAt:   u.UpdatedAt,
	}, nil
}

type UpdateUserParams struct {
	ID       uuid.UUID
	Email    *string
	Username *string
	FullName *string
	Role     *string
	IsActive *bool
}

func (p *Postgres) UpdateUser(ctx context.Context, params UpdateUserParams) (*domain.User, error) {
	// Start transaction
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	qtx := p.q.WithTx(tx)

	// Convert string role to NullRoleEnum if provided
	var roleEnum queries.NullRoleEnum
	if params.Role != nil && *params.Role != "" {
		roleEnum = queries.NullRoleEnum{
			RoleEnum: queries.RoleEnum(*params.Role),
			Valid:    true,
		}
	}

	u, err := qtx.UpdateUser(ctx, queries.UpdateUserParams{
		ID:       params.ID,
		Email:    params.Email,
		Username: params.Username,
		FullName: params.FullName,
		Role:     roleEnum,
		IsActive: params.IsActive,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Create outbox event for user update
	outboxEvent := &domain.OutboxEvent{
		ID:            uuid.New(),
		AggregateID:   u.ID,
		AggregateType: domain.AggregateTypeUser,
		EventType:     domain.EventTypeUserUpdated,
		Payload: map[string]interface{}{
			"user_id":   u.ID.String(),
			"username":  u.Username,
			"email":     u.Email,
			"role":      string(u.Role),
			"is_active": u.IsActive,
		},
		CreatedAt:  u.UpdatedAt,
		RetryCount: 0,
	}

	if err := p.CreateOutboxEvent(ctx, tx, outboxEvent); err != nil {
		return nil, fmt.Errorf("failed to create outbox event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	var fullName string
	if u.FullName != nil {
		fullName = *u.FullName
	}
	return &domain.User{
		ID:          u.ID,
		Email:       u.Email,
		Username:    u.Username,
		FullName:    fullName,
		Role:        string(u.Role),
		IsActive:    u.IsActive,
		CreatedAt:   u.CreatedAt,
		LastLoginAt: u.LastLoginAt,
		UpdatedAt:   u.UpdatedAt,
	}, nil
}

func (p *Postgres) DeleteUser(ctx context.Context, id uuid.UUID) error {
	tag, err := p.q.DeleteUser(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (p *Postgres) CreateUser(ctx context.Context, user *domain.User) error {
	u, err := p.q.CreateUser(ctx, queries.CreateUserParams{
		Username:     user.Username,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		FullName:     &user.FullName,
		Role:         queries.RoleEnum(user.Role),
		IsActive:     user.IsActive,
	})
	if err != nil {
		return err
	}
	user.ID = u.ID
	user.CreatedAt = u.CreatedAt
	user.UpdatedAt = u.UpdatedAt
	return nil
}
