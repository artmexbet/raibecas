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
			Role:        u.Role,
			IsActive:    u.IsActive,
			CreatedAt:   u.CreatedAt,
			LastLoginAt: u.LastLoginAt,
			UpdatedAt:   u.UpdatedAt,
		}
	}

	return users, int(total), nil
}

func (p *Postgres) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, err := p.q.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(pgx.ErrNoRows, err) {
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
		Role:        u.Role,
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
	IsActive *bool
}

func (p *Postgres) UpdateUser(ctx context.Context, params UpdateUserParams) (*domain.User, error) {
	u, err := p.q.UpdateUser(ctx, queries.UpdateUserParams{
		ID:       params.ID,
		Email:    params.Email,
		Username: params.Username,
		FullName: params.FullName,
		IsActive: params.IsActive,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to update user: %w", err)
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
		Role:        u.Role,
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
		Role:         user.Role,
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
