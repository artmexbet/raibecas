package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
}

// RegistrationRepository defines the interface for registration request data access
type RegistrationRepository interface {
	Create(ctx context.Context, req *RegistrationRequest) error
	GetByID(ctx context.Context, id uuid.UUID) (*RegistrationRequest, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status RegistrationStatus, approvedBy *uuid.UUID) error
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ExistsByUsername(ctx context.Context, username string) (bool, error)
}

// TokenStore defines the interface for token storage
type TokenStore interface {
	StoreRefreshToken(ctx context.Context, token *RefreshToken, ttl time.Duration) error
	GetRefreshToken(ctx context.Context, userID uuid.UUID) (*RefreshToken, error)
	GetRefreshTokenByValue(ctx context.Context, tokenValue string) (*RefreshToken, error)
	DeleteRefreshToken(ctx context.Context, userID uuid.UUID) error
	DeleteAllRefreshTokens(ctx context.Context, userID uuid.UUID) error
}
