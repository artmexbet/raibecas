package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/artmexbet/raibecas/services/auth/internal/domain"
)

type IRegistrationStorage interface {
	CreateRegistrationRequest(ctx context.Context, req *domain.RegistrationRequest) (uuid.UUID, error)
	GetRegistrationRequestByID(ctx context.Context, id uuid.UUID) (*domain.RegistrationRequest, error)
	UpdateRegistrationRequestStatus(ctx context.Context, id uuid.UUID, status domain.RegistrationStatus, approvedBy *uuid.UUID) error
	ExistsPendingRegistrationByEmail(ctx context.Context, email string) (bool, error)
	ExistsPendingRegistrationByUsername(ctx context.Context, username string) (bool, error)
}

// RegistrationRepository handles registration request data access
type RegistrationRepository struct {
	storage IRegistrationStorage
}

// NewRegistrationRepository creates a new registration repository
func NewRegistrationRepository(storage IRegistrationStorage) *RegistrationRepository {
	return &RegistrationRepository{storage: storage}
}

// Create creates a new registration request
func (r *RegistrationRepository) Create(ctx context.Context, req *domain.RegistrationRequest) (uuid.UUID, error) {
	return r.storage.CreateRegistrationRequest(ctx, req)
}

// GetByID retrieves a registration request by ID
func (r *RegistrationRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.RegistrationRequest, error) {
	return r.storage.GetRegistrationRequestByID(ctx, id)
}

// UpdateStatus updates the status of a registration request
func (r *RegistrationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.RegistrationStatus, approvedBy *uuid.UUID) error {
	return r.storage.UpdateRegistrationRequestStatus(ctx, id, status, approvedBy)
}

// ExistsByEmail checks if a registration request with the given email exists
func (r *RegistrationRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return r.storage.ExistsPendingRegistrationByEmail(ctx, email)
}

// ExistsByUsername checks if a registration request with the given username exists
func (r *RegistrationRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	return r.storage.ExistsPendingRegistrationByUsername(ctx, username)
}
