package service

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/artmexbet/raibecas/services/auth/internal/domain"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// RegistrationRepository defines the interface for registration request data access
type RegistrationRepository interface {
	CreateRegistrationRequest(ctx context.Context, req *domain.RegistrationRequest) (uuid.UUID, error)
	GetRegistrationRequestByID(ctx context.Context, id uuid.UUID) (*domain.RegistrationRequest, error)
	UpdateRegistrationRequestStatus(ctx context.Context, id uuid.UUID, status domain.RegistrationStatus, approvedBy *uuid.UUID) error
	ExistsPendingRegistrationByEmail(ctx context.Context, email string) (bool, error)
	ExistsPendingRegistrationByUsername(ctx context.Context, username string) (bool, error)
}

// RegistrationService handles registration business logic
type RegistrationService struct {
	regRepo    RegistrationRepository
	userRepo   UserRepository
	bcryptCost int
	logger     *slog.Logger
}

// NewRegistrationService creates a new registration service
func NewRegistrationService(
	regRepo RegistrationRepository,
	userRepo UserRepository,
	logger *slog.Logger,
) *RegistrationService {
	return &RegistrationService{
		regRepo:    regRepo,
		userRepo:   userRepo,
		bcryptCost: 12,
		logger:     logger,
	}
}

// CreateRegistrationRequest creates a new registration request
func (s *RegistrationService) CreateRegistrationRequest(ctx context.Context, req domain.RegisterRequest) (uuid.UUID, error) {
	// Validate email format
	if !emailRegex.MatchString(req.Email) {
		s.logger.WarnContext(ctx, "invalid email format in registration", "email", req.Email)
		return uuid.Nil, domain.ErrInvalidEmail
	}

	// Validate password strength (minimum 8 characters)
	if len(req.Password) < 8 {
		s.logger.WarnContext(ctx, "weak password in registration", "email", req.Email)
		return uuid.Nil, domain.ErrInvalidPassword
	}

	// Check if user already exists
	exists, err := s.userRepo.ExistsUserByEmail(ctx, req.Email)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to check user existence by email", "email", req.Email, "error", err)
		return uuid.Nil, fmt.Errorf("failed to check user existence: %w", err)
	}
	if exists {
		s.logger.WarnContext(ctx, "registration attempt with existing email", "email", req.Email)
		return uuid.Nil, domain.ErrEmailAlreadyExists
	}

	exists, err = s.userRepo.ExistsUserByUsername(ctx, req.Username)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to check user existence by username", "username", req.Username, "error", err)
		return uuid.Nil, fmt.Errorf("failed to check user existence: %w", err)
	}
	if exists {
		s.logger.WarnContext(ctx, "registration attempt with existing username", "username", req.Username)
		return uuid.Nil, domain.ErrUsernameAlreadyExists
	}

	// Check if registration request already exists
	exists, err = s.regRepo.ExistsPendingRegistrationByEmail(ctx, req.Email)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to check pending registration by email", "email", req.Email, "error", err)
		return uuid.Nil, fmt.Errorf("failed to check registration existence: %w", err)
	}
	if exists {
		s.logger.WarnContext(ctx, "duplicate registration request by email", "email", req.Email)
		return uuid.Nil, domain.ErrEmailAlreadyExists
	}

	exists, err = s.regRepo.ExistsPendingRegistrationByUsername(ctx, req.Username)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to check pending registration by username", "username", req.Username, "error", err)
		return uuid.Nil, fmt.Errorf("failed to check registration existence: %w", err)
	}
	if exists {
		s.logger.WarnContext(ctx, "duplicate registration request by username", "username", req.Username)
		return uuid.Nil, domain.ErrUsernameAlreadyExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.bcryptCost)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to hash password during registration", "email", req.Email, "error", err)
		return uuid.Nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// CreateUser registration request
	regReq := &domain.RegistrationRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: string(hashedPassword),
		Metadata: req.Metadata,
	}

	id, err := s.regRepo.CreateRegistrationRequest(ctx, regReq)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create registration request", "email", req.Email, "error", err)
		return uuid.Nil, fmt.Errorf("failed to create registration request: %w", err)
	}

	s.logger.InfoContext(ctx, "registration request created", "request_id", id, "email", req.Email, "username", req.Username)
	return id, nil
}

// ApproveRegistration approves a registration request and creates a user
func (s *RegistrationService) ApproveRegistration(ctx context.Context, requestID, approverID uuid.UUID, role string) (*domain.User, error) {
	// Get registration request
	regReq, err := s.regRepo.GetRegistrationRequestByID(ctx, requestID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get registration request", "request_id", requestID, "error", err)
		return nil, err
	}

	// Check if request is pending
	if regReq.Status != domain.StatusPending {
		s.logger.WarnContext(ctx, "attempt to approve non-pending registration", "request_id", requestID, "status", regReq.Status)
		return nil, domain.ErrRegistrationNotPending
	}

	// Default role if not specified
	if role == "" {
		role = "user"
	}

	// Create user with specified role
	user := &domain.User{
		Username:     regReq.Username,
		Email:        regReq.Email,
		PasswordHash: regReq.Password, // Already hashed
		Role:         domain.UserRole(role),
	}

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		s.logger.ErrorContext(ctx, "failed to create user from registration", "request_id", requestID, "email", regReq.Email, "error", err)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Update registration request status
	if err := s.regRepo.UpdateRegistrationRequestStatus(ctx, requestID, domain.StatusApproved, &approverID); err != nil {
		s.logger.ErrorContext(ctx, "failed to update registration status", "request_id", requestID, "error", err)
		return nil, fmt.Errorf("failed to update registration status: %w", err)
	}

	s.logger.InfoContext(ctx, "registration approved", "request_id", requestID, "user_id", user.ID, "approver_id", approverID)
	return user, nil
}

// RejectRegistration rejects a registration request
func (s *RegistrationService) RejectRegistration(ctx context.Context, requestID, approverID uuid.UUID) error {
	// Get registration request
	regReq, err := s.regRepo.GetRegistrationRequestByID(ctx, requestID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get registration request for rejection", "request_id", requestID, "error", err)
		return err
	}

	// Check if request is pending
	if regReq.Status != domain.StatusPending {
		s.logger.WarnContext(ctx, "attempt to reject non-pending registration", "request_id", requestID, "status", regReq.Status)
		return domain.ErrRegistrationNotPending
	}

	// Update registration request status
	if err := s.regRepo.UpdateRegistrationRequestStatus(ctx, requestID, domain.StatusRejected, &approverID); err != nil {
		s.logger.ErrorContext(ctx, "failed to update registration status to rejected", "request_id", requestID, "error", err)
		return fmt.Errorf("failed to update registration status: %w", err)
	}

	s.logger.InfoContext(ctx, "registration rejected", "request_id", requestID, "approver_id", approverID)
	return nil
}
