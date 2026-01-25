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

	// CreateRegistrationRequest creates a new registration request
	CreateRegistrationRequest(ctx context.Context, req domain.CreateRegistrationRequestRequest) (*domain.CreateRegistrationRequestResponse, error)

	// ListRegistrationRequests retrieves a list of registration requests
	ListRegistrationRequests(ctx context.Context, query domain.ListRegistrationRequestsQuery) (*domain.ListRegistrationRequestsResponse, error)

	// ApproveRegistrationRequest approves a registration request
	ApproveRegistrationRequest(ctx context.Context, requestID, approverID uuid.UUID) (*domain.ApproveRegistrationRequestResponse, error)

	// RejectRegistrationRequest rejects a registration request
	RejectRegistrationRequest(ctx context.Context, requestID, approverID uuid.UUID, reason string) (*domain.RejectRegistrationRequestResponse, error)
}
