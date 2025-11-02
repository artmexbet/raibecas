package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"auth/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RegistrationRepository handles registration request data access
type RegistrationRepository struct {
	pool *pgxpool.Pool
}

// NewRegistrationRepository creates a new registration repository
func NewRegistrationRepository(pool *pgxpool.Pool) *RegistrationRepository {
	return &RegistrationRepository{pool: pool}
}

// Create creates a new registration request
func (r *RegistrationRepository) Create(ctx context.Context, req *domain.RegistrationRequest) error {
	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO registration_requests (id, username, email, password, status, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = r.pool.Exec(ctx, query,
		req.ID,
		req.Username,
		req.Email,
		req.Password,
		req.Status,
		metadata,
		req.CreatedAt,
		req.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create registration request: %w", err)
	}

	return nil
}

// GetByID retrieves a registration request by ID
func (r *RegistrationRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.RegistrationRequest, error) {
	var req domain.RegistrationRequest
	var metadata []byte

	query := `
		SELECT id, username, email, password, status, metadata, created_at, updated_at, approved_by, approved_at
		FROM registration_requests
		WHERE id = $1
	`

	err := r.pool.QueryRow(ctx, query, id).Scan(
		&req.ID,
		&req.Username,
		&req.Email,
		&req.Password,
		&req.Status,
		&metadata,
		&req.CreatedAt,
		&req.UpdatedAt,
		&req.ApprovedBy,
		&req.ApprovedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrRegistrationNotFound
		}
		return nil, fmt.Errorf("failed to get registration request: %w", err)
	}

	if len(metadata) > 0 {
		if err := json.Unmarshal(metadata, &req.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}

	return &req, nil
}

// UpdateStatus updates the status of a registration request
func (r *RegistrationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.RegistrationStatus, approvedBy *uuid.UUID) error {
	query := `
		UPDATE registration_requests
		SET status = $1, approved_by = $2, approved_at = CASE WHEN $1 = 'approved' THEN NOW() ELSE NULL END, updated_at = NOW()
		WHERE id = $3
	`

	result, err := r.pool.Exec(ctx, query, status, approvedBy, id)
	if err != nil {
		return fmt.Errorf("failed to update registration status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return domain.ErrRegistrationNotFound
	}

	return nil
}

// ExistsByEmail checks if a registration request with the given email exists
func (r *RegistrationRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM registration_requests WHERE email = $1 AND status = 'pending')`

	err := r.pool.QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check registration existence by email: %w", err)
	}

	return exists, nil
}

// ExistsByUsername checks if a registration request with the given username exists
func (r *RegistrationRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM registration_requests WHERE username = $1 AND status = 'pending')`

	err := r.pool.QueryRow(ctx, query, username).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check registration existence by username: %w", err)
	}

	return exists, nil
}
