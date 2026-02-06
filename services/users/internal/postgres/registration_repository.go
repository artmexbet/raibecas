package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/mailru/easyjson"

	"github.com/artmexbet/raibecas/services/users/internal/domain"
	"github.com/artmexbet/raibecas/services/users/internal/postgres/queries"
)

func (p *Postgres) CreateRegistrationRequest(ctx context.Context, req *domain.RegistrationRequest) error {
	metadataBytes, err := easyjson.Marshal(req.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	r, err := p.q.CreateRegistrationRequest(ctx, queries.CreateRegistrationRequestParams{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: req.PasswordHash,
		Status:       string(req.Status),
		Metadata:     metadataBytes,
	})
	if err != nil {
		return fmt.Errorf("failed to create registration request: %w", err)
	}
	req.ID = r.ID
	req.CreatedAt = r.CreatedAt
	req.UpdatedAt = r.UpdatedAt
	return nil
}

func (p *Postgres) ListRegistrationRequests(ctx context.Context, status domain.RegistrationStatus, limit, offset int) ([]domain.RegistrationRequest, int, error) {
	// Count
	total, err := p.q.CountRegistrationRequests(ctx, string(status))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count registration requests: %w", err)
	}

	// List
	dbReqs, err := p.q.ListRegistrationRequests(ctx, queries.ListRegistrationRequestsParams{
		StatusFilter: string(status),
		Limit:        int32(limit),
		Offset:       int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list registration requests: %w", err)
	}

	reqs := make([]domain.RegistrationRequest, len(dbReqs))
	for i, r := range dbReqs {
		var meta domain.Metadata
		if len(r.Metadata) > 0 {
			if err := easyjson.Unmarshal(r.Metadata, &meta); err != nil {
				return nil, 0, fmt.Errorf("failed to map registration request: %w", err)
			}
		}
		reqs[i] = domain.RegistrationRequest{
			ID:       r.ID,
			Username: r.Username,
			Email:    r.Email,
			// PasswordHash omitted in list
			Status:     domain.RegistrationStatus(r.Status),
			Metadata:   meta,
			CreatedAt:  r.CreatedAt,
			UpdatedAt:  r.UpdatedAt,
			ApprovedBy: r.ApprovedBy,
			ApprovedAt: r.ApprovedAt,
		}
	}
	return reqs, int(total), nil
}

func (p *Postgres) GetRegistrationRequestByID(ctx context.Context, id uuid.UUID) (*domain.RegistrationRequest, error) {
	r, err := p.q.GetRegistrationRequestByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get registration request: %w", err)
	}
	req, err := toDomainRegistrationRequest(r)
	if err != nil {
		return nil, fmt.Errorf("failed to map registration request: %w", err)
	}
	return &req, nil
}

func (p *Postgres) RejectRegistrationRequest(ctx context.Context, id uuid.UUID, approverID uuid.UUID, reason string) error {
	tag, err := p.q.RejectRegistrationRequest(ctx, queries.RejectRegistrationRequestParams{
		ID:         id,
		ApprovedBy: &approverID,
		Reason:     reason,
	})
	if err != nil {
		return fmt.Errorf("failed to reject registration request: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("request not found or not pending")
	}
	return nil
}

func (p *Postgres) ApproveRegistrationRequest(ctx context.Context, requestID uuid.UUID, approverID uuid.UUID, role string) (*domain.User, error) {
	// Validate role
	if role == "" {
		role = domain.RoleUser // Default role
	}
	if !domain.IsValidRole(role) {
		return nil, fmt.Errorf("invalid role: %s", role)
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	qtx := p.q.WithTx(tx)

	// Get request to get details for user creation
	req, err := qtx.GetRegistrationRequestByID(ctx, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to get request: %w", err)
	}
	if req.Status != "pending" {
		return nil, fmt.Errorf("request not found or not pending")
	}

	fullName := "" // Default empty  //todo: extract from metadata if available
	u, err := qtx.CreateUser(ctx, queries.CreateUserParams{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: req.PasswordHash,
		Role:         queries.RoleEnum(role),
		IsActive:     true,
		FullName:     &fullName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	tag, err := qtx.UpdateRegistrationRequestStatus(ctx, queries.UpdateRegistrationRequestStatusParams{
		ID:         requestID,
		Status:     "approved",
		ApprovedBy: &approverID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update registration request: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, fmt.Errorf("failed to update registration request status")
	}

	// Create outbox event for user registration
	outboxEvent := &domain.OutboxEvent{
		ID:            uuid.New(),
		AggregateID:   u.ID,
		AggregateType: domain.AggregateTypeUser,
		EventType:     domain.EventTypeUserRegistered,
		Payload: map[string]interface{}{
			"user_id":       u.ID.String(),
			"username":      req.Username,
			"email":         req.Email,
			"password_hash": req.PasswordHash,
			"role":          role,
			"is_active":     true,
		},
		CreatedAt:  u.CreatedAt,
		RetryCount: 0,
	}

	if err := p.CreateOutboxEvent(ctx, tx, outboxEvent); err != nil {
		return nil, fmt.Errorf("failed to create outbox event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &domain.User{
		ID:        u.ID,
		Username:  req.Username,
		Email:     req.Email,
		FullName:  fullName,
		Role:      role,
		IsActive:  true,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}, nil
}

func toDomainRegistrationRequest(r queries.RegistrationRequest) (domain.RegistrationRequest, error) {
	var meta domain.Metadata
	if len(r.Metadata) > 0 {
		if err := easyjson.Unmarshal(r.Metadata, &meta); err != nil {
			return domain.RegistrationRequest{}, err
		}
	}
	return domain.RegistrationRequest{
		ID:           r.ID,
		Username:     r.Username,
		Email:        r.Email,
		PasswordHash: r.PasswordHash,
		Status:       domain.RegistrationStatus(r.Status),
		Metadata:     meta,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
		ApprovedBy:   r.ApprovedBy,
		ApprovedAt:   r.ApprovedAt,
	}, nil
}
