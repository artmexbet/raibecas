package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/artmexbet/raibecas/services/users/internal/domain"
	"github.com/artmexbet/raibecas/services/users/internal/postgres"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrInvalidStatus = errors.New("invalid status")
)

// Metrics defines the interface for business metrics collection
type Metrics interface {
	IncRegisteredUsers()
}

type Service struct {
	repo    *postgres.Postgres
	metrics Metrics
}

func New(repo *postgres.Postgres, metrics Metrics) *Service {
	return &Service{
		repo:    repo,
		metrics: metrics,
	}
}

// User methods

func (s *Service) ListUsers(ctx context.Context, params postgres.ListUsersParams) ([]domain.User, int, error) {
	if params.Limit > 100 {
		params.Limit = 100
	}
	if params.Limit <= 0 {
		params.Limit = 10
	}
	return s.repo.ListUsers(ctx, params)
}

func (s *Service) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrNotFound
	}
	return u, nil
}

func (s *Service) UpdateUser(ctx context.Context, params postgres.UpdateUserParams) (*domain.User, error) {
	u, err := s.repo.UpdateUser(ctx, params)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, ErrNotFound
	}
	return u, nil
}

func (s *Service) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteUser(ctx, id)
}

// Registration Request methods

func (s *Service) CreateRegistrationRequest(ctx context.Context, req *domain.RegistrationRequest) (*domain.RegistrationRequest, error) {
	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.PasswordHash), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	req.PasswordHash = string(hash)
	req.Status = domain.RegistrationStatusPending

	if err := s.repo.CreateRegistrationRequest(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}

func (s *Service) ListRegistrationRequests(ctx context.Context, status domain.RegistrationStatus, limit, offset int) ([]domain.RegistrationRequest, int, error) {
	if limit > 100 {
		limit = 100
	}
	if limit <= 0 {
		limit = 10
	}
	return s.repo.ListRegistrationRequests(ctx, status, limit, offset)
}

func (s *Service) ApproveRegistrationRequest(ctx context.Context, requestID uuid.UUID, approverID uuid.UUID) (*domain.User, error) {
	u, err := s.repo.ApproveRegistrationRequest(ctx, requestID, approverID)
	if err != nil {
		return nil, err
	}
	s.metrics.IncRegisteredUsers()
	return u, nil
}

func (s *Service) RejectRegistrationRequest(ctx context.Context, requestID uuid.UUID, approverID uuid.UUID, reason string) error {
	return s.repo.RejectRegistrationRequest(ctx, requestID, approverID, reason)
}
