package service

import (
	"context"
	"fmt"
	"regexp"

	"auth/internal/domain"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// RegistrationRepository defines the interface for registration request data access
type RegistrationRepository interface {
	Create(ctx context.Context, req *domain.RegistrationRequest) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.RegistrationRequest, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.RegistrationStatus, approvedBy *uuid.UUID) error
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ExistsByUsername(ctx context.Context, username string) (bool, error)
}

// RegistrationService handles registration business logic
type RegistrationService struct {
	regRepo    RegistrationRepository
	userRepo   IUserRepository
	bcryptCost int
}

// NewRegistrationService creates a new registration service
func NewRegistrationService(
	regRepo RegistrationRepository,
	userRepo IUserRepository,
) *RegistrationService {
	return &RegistrationService{
		regRepo:    regRepo,
		userRepo:   userRepo,
		bcryptCost: 12,
	}
}

// CreateRegistrationRequest creates a new registration request
func (s *RegistrationService) CreateRegistrationRequest(ctx context.Context, req domain.RegisterRequest) (uuid.UUID, error) {
	// Validate email format
	if !emailRegex.MatchString(req.Email) {
		return uuid.Nil, domain.ErrInvalidEmail
	}

	// Validate password strength (minimum 8 characters)
	if len(req.Password) < 8 {
		return uuid.Nil, domain.ErrInvalidPassword
	}

	// Check if user already exists
	exists, err := s.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to check user existence: %w", err)
	}
	if exists {
		return uuid.Nil, domain.ErrEmailAlreadyExists
	}

	exists, err = s.userRepo.ExistsByUsername(ctx, req.Username)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to check user existence: %w", err)
	}
	if exists {
		return uuid.Nil, domain.ErrUsernameAlreadyExists
	}

	// Check if registration request already exists
	exists, err = s.regRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to check registration existence: %w", err)
	}
	if exists {
		return uuid.Nil, domain.ErrEmailAlreadyExists
	}

	exists, err = s.regRepo.ExistsByUsername(ctx, req.Username)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to check registration existence: %w", err)
	}
	if exists {
		return uuid.Nil, domain.ErrUsernameAlreadyExists
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.bcryptCost)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// CreateUser registration request
	regReq := &domain.RegistrationRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: string(hashedPassword),
		Metadata: req.Metadata,
	}

	id, err := s.regRepo.Create(ctx, regReq)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create registration request: %w", err)
	}

	return id, nil
}

// ApproveRegistration approves a registration request and creates a user
func (s *RegistrationService) ApproveRegistration(ctx context.Context, requestID, approverID uuid.UUID) (*domain.User, error) {
	// Get registration request
	regReq, err := s.regRepo.GetByID(ctx, requestID)
	if err != nil {
		return nil, err
	}

	// Check if request is pending
	if regReq.Status != domain.StatusPending {
		return nil, domain.ErrRegistrationNotPending
	}

	// CreateUser user
	user := &domain.User{
		Username:     regReq.Username,
		Email:        regReq.Email,
		PasswordHash: regReq.Password, // Already hashed
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Update registration request status
	if err := s.regRepo.UpdateStatus(ctx, requestID, domain.StatusApproved, &approverID); err != nil {
		return nil, fmt.Errorf("failed to update registration status: %w", err)
	}

	return user, nil
}

// RejectRegistration rejects a registration request
func (s *RegistrationService) RejectRegistration(ctx context.Context, requestID, approverID uuid.UUID) error {
	// Get registration request
	regReq, err := s.regRepo.GetByID(ctx, requestID)
	if err != nil {
		return err
	}

	// Check if request is pending
	if regReq.Status != domain.StatusPending {
		return domain.ErrRegistrationNotPending
	}

	// Update registration request status
	if err := s.regRepo.UpdateStatus(ctx, requestID, domain.StatusRejected, &approverID); err != nil {
		return fmt.Errorf("failed to update registration status: %w", err)
	}

	return nil
}
