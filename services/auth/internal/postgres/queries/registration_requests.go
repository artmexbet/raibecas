package queries

import (
	"auth/internal/domain"
	"encoding/json"
	"fmt"
	"utills/pg"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (r *RegistrationRequest) ToDomain() (domain.RegistrationRequest, error) {
	id, err := pg.GoogleUUIDFromPG(r.ID)
	if err != nil {
		return domain.RegistrationRequest{}, fmt.Errorf("cannot convert ID to domain UUID: %w", err)
	}
	var metadata map[string]any
	if err = json.Unmarshal(r.Metadata, &metadata); err != nil {
		return domain.RegistrationRequest{}, fmt.Errorf("cannot unmarshal metadata: %w", err)
	}

	var approvedByID *uuid.UUID
	if r.ApprovedBy.Valid {
		_id, err := pg.GoogleUUIDFromPG(r.ApprovedBy)
		if err != nil {
			return domain.RegistrationRequest{}, fmt.Errorf("cannot convert ApprovedBy to domain UUID: %w", err)
		}
		approvedByID = &_id
	}

	return domain.RegistrationRequest{
		ID:         id,
		Username:   r.Username,
		Email:      r.Email,
		Password:   r.Password,
		Status:     domain.RegistrationStatus(r.Status),
		Metadata:   metadata,
		CreatedAt:  r.CreatedAt.Time,
		UpdatedAt:  r.UpdatedAt.Time,
		ApprovedBy: approvedByID,
		ApprovedAt: &r.ApprovedAt.Time,
	}, nil
}

func FromDomainRegistrationRequest(req domain.RegistrationRequest) (RegistrationRequest, error) {
	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		return RegistrationRequest{}, fmt.Errorf("cannot marshal metadata: %w", err)
	}

	var approvedByUUID pgtype.UUID
	if req.ApprovedBy != nil {
		approvedByUUID = pg.UUIDFromGoogle(*req.ApprovedBy)
	}

	var approvedAtTS pgtype.Timestamp
	if req.ApprovedAt != nil {
		approvedAtTS = pg.ConvertToPGTime(*req.ApprovedAt)
	}

	return RegistrationRequest{
		ID:         pg.UUIDFromGoogle(req.ID),
		Username:   req.Username,
		Email:      req.Email,
		Password:   req.Password,
		Status:     string(req.Status),
		Metadata:   metadata,
		CreatedAt:  pg.ConvertToPGTime(req.CreatedAt),
		UpdatedAt:  pg.ConvertToPGTime(req.UpdatedAt),
		ApprovedBy: approvedByUUID,
		ApprovedAt: approvedAtTS,
	}, nil
}
