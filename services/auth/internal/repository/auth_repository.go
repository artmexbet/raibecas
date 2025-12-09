package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/artmexbet/raibecas/services/auth/internal/domain"
)

type IAuthStorage interface {
	CreateUser(ctx context.Context, user *domain.User) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	GetUserByUsername(ctx context.Context, username string) (*domain.User, error)
	ExistsUserByEmail(ctx context.Context, email string) (bool, error)
	ExistsUserByUsername(ctx context.Context, username string) (bool, error)
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
}

type AuthRepository struct {
	storage IAuthStorage
}

func (r *AuthRepository) Create(ctx context.Context, user *domain.User) error {
	return r.storage.CreateUser(ctx, user)
}

func (r *AuthRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return r.storage.GetUserByID(ctx, id)
}

func (r *AuthRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.storage.GetUserByEmail(ctx, email)
}

func (r *AuthRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	return r.storage.GetUserByUsername(ctx, username)
}

func (r *AuthRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return r.storage.ExistsUserByUsername(ctx, email)
}

func (r *AuthRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	return r.storage.ExistsUserByUsername(ctx, username)
}

func (r *AuthRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	return r.storage.UpdatePassword(ctx, userID, passwordHash)
}

func NewAuthRepository(storage IAuthStorage) *AuthRepository {
	return &AuthRepository{storage: storage}
}
