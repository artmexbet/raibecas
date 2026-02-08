package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/artmexbet/raibecas/services/users/internal/domain"
	"github.com/artmexbet/raibecas/services/users/internal/postgres"
)

// Errors are defined in errors.go

type Service struct {
	userRepo   UserRepository
	regRepo    RegistrationRepository
	outboxRepo OutboxRepository
	metrics    Metrics
}

func New(
	userRepo UserRepository,
	regRepo RegistrationRepository,
	outboxRepo OutboxRepository,
	metrics Metrics,
) *Service {
	return &Service{
		userRepo:   userRepo,
		regRepo:    regRepo,
		outboxRepo: outboxRepo,
		metrics:    metrics,
	}
}

// User methods

func (s *Service) ListUsers(ctx context.Context, params postgres.ListUsersParams) ([]domain.User, int, error) {
	// Validate and normalize parameters
	if params.Limit > 100 {
		params.Limit = 100
	}
	if params.Limit <= 0 {
		params.Limit = 10
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	users, total, err := s.userRepo.ListUsers(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}

	return users, total, nil
}

func (s *Service) GetUserByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	if id == uuid.Nil {
		return nil, ErrInvalidUserID
	}

	u, err := s.userRepo.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if u == nil {
		return nil, ErrNotFound
	}
	return u, nil
}

func (s *Service) UpdateUser(ctx context.Context, params domain.UpdateUserParams) (*domain.User, error) {
	if params.ID == uuid.Nil {
		return nil, ErrInvalidUserID
	}

	// Validate role if provided
	if params.Role != nil && *params.Role != "" {
		if !domain.IsValidRole(*params.Role) {
			return nil, fmt.Errorf("invalid role: %s", *params.Role)
		}
	}

	u, err := s.userRepo.UpdateUser(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	if u == nil {
		return nil, ErrNotFound
	}
	return u, nil
}

func (s *Service) DeleteUser(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return ErrInvalidUserID
	}

	if err := s.userRepo.DeleteUser(ctx, id); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// Registration Request methods

func (s *Service) CreateRegistrationRequest(ctx context.Context, req *domain.RegistrationRequest) (*domain.RegistrationRequest, error) {
	if req == nil {
		return nil, ErrRegistrationRequestNil
	}
	if req.Email == "" || req.Username == "" || req.PasswordHash == "" {
		return nil, ErrMissingRequiredFields
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.PasswordHash), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	req.PasswordHash = string(hash)
	req.Status = domain.RegistrationStatusPending

	if err := s.regRepo.CreateRegistrationRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("failed to create registration request: %w", err)
	}
	return req, nil
}

func (s *Service) ListRegistrationRequests(ctx context.Context, status domain.RegistrationStatus, limit, offset int) ([]domain.RegistrationRequest, int, error) {
	// Validate and normalize parameters
	if limit > 100 {
		limit = 100
	}
	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	reqs, total, err := s.regRepo.ListRegistrationRequests(ctx, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list registration requests: %w", err)
	}

	return reqs, total, nil
}

func (s *Service) ApproveRegistrationRequest(ctx context.Context, requestID uuid.UUID, approverID uuid.UUID, role string) (*domain.User, error) {
	if requestID == uuid.Nil || approverID == uuid.Nil {
		return nil, ErrInvalidRequestOrApproverID
	}

	u, err := s.regRepo.ApproveRegistrationRequest(ctx, requestID, approverID, role)
	if err != nil {
		return nil, fmt.Errorf("failed to approve registration request: %w", err)
	}
	if u == nil {
		return nil, ErrNotFound
	}

	s.metrics.IncRegisteredUsers()
	return u, nil
}

func (s *Service) RejectRegistrationRequest(ctx context.Context, requestID uuid.UUID, approverID uuid.UUID, reason string) error {
	if requestID == uuid.Nil || approverID == uuid.Nil {
		return ErrInvalidRequestOrApproverID
	}

	if err := s.regRepo.RejectRegistrationRequest(ctx, requestID, approverID, reason); err != nil {
		return fmt.Errorf("failed to reject registration request: %w", err)
	}
	return nil
}
