package queries

import (
	"encoding/json"
	"fmt"

	"github.com/artmexbet/raibecas/services/auth/internal/domain"
)

func (r *RegistrationRequest) ToDomain() (domain.RegistrationRequest, error) {
	var metadata map[string]any
	if err := json.Unmarshal(r.Metadata, &metadata); err != nil {
		return domain.RegistrationRequest{}, fmt.Errorf("cannot unmarshal metadata: %w", err)
	}

	return domain.RegistrationRequest{
		ID:         r.ID,
		Username:   r.Username,
		Email:      r.Email,
		Password:   r.Password,
		Status:     domain.RegistrationStatus(r.Status),
		Metadata:   metadata,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
		ApprovedBy: r.ApprovedBy,
		ApprovedAt: r.ApprovedAt,
	}, nil
}

func FromDomainRegistrationRequest(req domain.RegistrationRequest) (RegistrationRequest, error) {
	metadata, err := json.Marshal(req.Metadata)
	if err != nil {
		return RegistrationRequest{}, fmt.Errorf("cannot marshal metadata: %w", err)
	}

	return RegistrationRequest{
		ID:         req.ID,
		Username:   req.Username,
		Email:      req.Email,
		Password:   req.Password,
		Status:     string(req.Status),
		Metadata:   metadata,
		CreatedAt:  req.CreatedAt,
		UpdatedAt:  req.UpdatedAt,
		ApprovedBy: req.ApprovedBy,
		ApprovedAt: req.ApprovedAt,
	}, nil
}
