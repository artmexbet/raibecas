package postgres

import (
	"auth/internal/domain"
	"auth/internal/postgres/queries"
	"context"
	"encoding/json"
	"fmt"
	"utills/pg"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (p *Postgres) CreateRegistrationRequest(ctx context.Context, req *domain.RegistrationRequest) error {
	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		return fmt.Errorf("could not marshal metadata: %v", err)
	}
	_, err = p.q.CreateRegistrationRequest(ctx,
		queries.CreateRegistrationRequestParams{
			Username: req.Username,
			Email:    req.Email,
			Password: req.Password,
			Metadata: metadata,
		},
	)
	if err != nil {
		return fmt.Errorf("could not create registration request: %v", err)
	}
	return nil
}

func (p *Postgres) GetRegistrationRequestByID(ctx context.Context, id uuid.UUID) (*domain.RegistrationRequest, error) {
	r, err := p.q.GetRegistrationRequestByID(ctx, pg.UUIDFromGoogle(id))
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
	var pgApprovedBy pgtype.UUID
	if approvedBy != nil {
		pgApprovedBy = pg.UUIDFromGoogle(*approvedBy)
	}
	err := p.q.UpdateRegistrationStatus(ctx, queries.UpdateRegistrationStatusParams{
		ID:         pg.UUIDFromGoogle(id),
		Status:     status.String(),
		ApprovedBy: pgApprovedBy,
	})
	if err != nil {
		return fmt.Errorf("could not update registration request status: %v", err)
	}
	return nil
}

// TODO: Add more methods
// TODO: Create repository level wrapping this postgres methods
