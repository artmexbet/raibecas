package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/artmexbet/raibecas/services/users/internal/domain"
	"github.com/artmexbet/raibecas/services/users/internal/postgres"
)

// UserRepository defines the interface for user data access
type UserRepository interface {
	ListUsers(ctx context.Context, params postgres.ListUsersParams) ([]domain.User, int, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	UpdateUser(ctx context.Context, params postgres.UpdateUserParams) (*domain.User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
	CreateUser(ctx context.Context, user *domain.User) error
	CountTotalUsers(ctx context.Context) (int64, error)
}

// RegistrationRepository defines the interface for registration request data access
type RegistrationRepository interface {
	CreateRegistrationRequest(ctx context.Context, req *domain.RegistrationRequest) error
	ListRegistrationRequests(ctx context.Context, status domain.RegistrationStatus, limit, offset int) ([]domain.RegistrationRequest, int, error)
	GetRegistrationRequestByID(ctx context.Context, id uuid.UUID) (*domain.RegistrationRequest, error)
	RejectRegistrationRequest(ctx context.Context, id uuid.UUID, approverID uuid.UUID, reason string) error
	ApproveRegistrationRequest(ctx context.Context, requestID uuid.UUID, approverID uuid.UUID, role string) (*domain.User, error)
}

// OutboxRepository defines the interface for outbox event data access
type OutboxRepository interface {
	GetUnprocessedEventsTx(ctx context.Context, limit int) (pgx.Tx, []domain.OutboxEvent, error)
	MarkEventAsProcessed(ctx context.Context, tx pgx.Tx, eventID uuid.UUID) error
	MarkEventAsFailed(ctx context.Context, tx pgx.Tx, eventID uuid.UUID, errorMsg string) error
	CleanupStaleLocks(ctx context.Context, timeout time.Duration) error
}

// Metrics defines the interface for business metrics collection
type Metrics interface {
	IncRegisteredUsers()
}
