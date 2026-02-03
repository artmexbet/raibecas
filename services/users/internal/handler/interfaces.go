package handler

import (
	"context"

	"github.com/google/uuid"

	"github.com/artmexbet/raibecas/services/users/internal/domain"
	"github.com/artmexbet/raibecas/services/users/internal/postgres"
)

// ServiceInterface defines the interface for user service
type ServiceInterface interface {
	// Users

	ListUsers(ctx context.Context, params postgres.ListUsersParams) ([]domain.User, int, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	UpdateUser(ctx context.Context, params postgres.UpdateUserParams) (*domain.User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error

	// Registration Requests

	CreateRegistrationRequest(ctx context.Context, req *domain.RegistrationRequest) (*domain.RegistrationRequest, error)
	ListRegistrationRequests(ctx context.Context, status domain.RegistrationStatus, limit, offset int) ([]domain.RegistrationRequest, int, error)
	ApproveRegistrationRequest(ctx context.Context, requestID uuid.UUID, approverID uuid.UUID, role string) (*domain.User, error)
	RejectRegistrationRequest(ctx context.Context, requestID uuid.UUID, approverID uuid.UUID, reason string) error
}
