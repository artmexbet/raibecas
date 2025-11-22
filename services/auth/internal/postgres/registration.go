package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/artmexbet/raibecas/services/auth/internal/domain"
	"github.com/artmexbet/raibecas/services/auth/internal/postgres/queries"

	"github.com/google/uuid"
)

func (p *Postgres) CreateRegistrationRequest(ctx context.Context, req *domain.RegistrationRequest) (uuid.UUID, error) {
	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		return uuid.Nil, fmt.Errorf("could not marshal metadata: %v", err)
	}
	resp, err := p.q.CreateRegistrationRequest(ctx,
		queries.CreateRegistrationRequestParams{
			Username: req.Username,
			Email:    req.Email,
			Password: req.Password,
			Metadata: metadata,
		},
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("could not create registration request: %v", err)
	}
	return resp.ID, nil
}

func (p *Postgres) GetRegistrationRequestByID(ctx context.Context, id uuid.UUID) (*domain.RegistrationRequest, error) {
	r, err := p.q.GetRegistrationRequestByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("could not get registration request by id: %v", err)
	}
	req, err := r.ToDomain()
	if err != nil {
		return nil, fmt.Errorf("could not convert registration request to domain: %v", err)
	}
	return &req, nil
}

func (p *Postgres) UpdateRegistrationRequestStatus(ctx context.Context, id uuid.UUID, status domain.RegistrationStatus, approvedBy *uuid.UUID) error {
	err := p.q.UpdateRegistrationStatus(ctx, queries.UpdateRegistrationStatusParams{
		ID:         id,
		Status:     status.String(),
		ApprovedBy: approvedBy,
	})
	if err != nil {
		return fmt.Errorf("could not update registration request status: %v", err)
	}
	return nil
}

func (p *Postgres) ExistsPendingRegistrationByEmail(ctx context.Context, email string) (bool, error) {
	exists, err := p.q.RegistrationExistsByEmail(ctx, email)
	if err != nil {
		return false, fmt.Errorf("could not check existence of pending registration by email: %v", err)
	}

	return exists, nil
}

func (p *Postgres) ExistsPendingRegistrationByUsername(ctx context.Context, username string) (bool, error) {
	exists, err := p.q.RegistrationExistsByUsername(ctx, username)
	if err != nil {
		return false, fmt.Errorf("could not check existence of pending registration by username: %v", err)
	}

	return exists, nil
}
