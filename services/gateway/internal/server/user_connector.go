package server

import (
	"context"

	"github.com/google/uuid"

	"github.com/artmexbet/raibecas/services/gateway/internal/domain"
)

// UserServiceConnector defines the interface for communicating with the users service
type UserServiceConnector interface {
	// ListUsers retrieves a list of users with filtering and pagination
	ListUsers(ctx context.Context, query domain.ListUsersQuery) (*domain.ListUsersResponse, error)

	// GetUser retrieves a single user by ID
	GetUser(ctx context.Context, id uuid.UUID) (*domain.GetUserResponse, error)

	// UpdateUser updates an existing user
	UpdateUser(ctx context.Context, id uuid.UUID, req domain.UpdateUserRequest) (*domain.UpdateUserResponse, error)

	// DeleteUser deletes a user by ID
	DeleteUser(ctx context.Context, id uuid.UUID) error
}
