package mapper

import (
	"github.com/artmexbet/raibecas/libs/dto"

	"github.com/artmexbet/raibecas/services/users/internal/domain"
)

// UserToDTO converts domain.User to dto.User
func UserToDTO(u *domain.User) *dto.User {
	if u == nil {
		return nil
	}
	return &dto.User{
		ID:          u.ID,
		Email:       u.Email,
		Username:    u.Username,
		FullName:    u.FullName,
		Role:        u.Role,
		IsActive:    u.IsActive,
		CreatedAt:   u.CreatedAt,
		LastLoginAt: u.LastLoginAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

// UsersToDTO converts slice of domain.User to dto.User
func UsersToDTO(users []domain.User) []dto.User {
	result := make([]dto.User, 0, len(users))
	for i := range users {
		result = append(result, dto.User{
			ID:          users[i].ID,
			Email:       users[i].Email,
			Username:    users[i].Username,
			FullName:    users[i].FullName,
			Role:        users[i].Role,
			IsActive:    users[i].IsActive,
			CreatedAt:   users[i].CreatedAt,
			LastLoginAt: users[i].LastLoginAt,
			UpdatedAt:   users[i].UpdatedAt,
		})
	}
	return result
}

// RegistrationToDTO converts domain.RegistrationRequest to dto.RegistrationRequest
func RegistrationToDTO(r *domain.RegistrationRequest) *dto.RegistrationRequest {
	if r == nil {
		return nil
	}

	// Convert Metadata (domain.Metadata is map[string]interface{})
	metadata := make(map[string]any)
	for k, v := range r.Metadata {
		metadata[k] = v
	}

	return &dto.RegistrationRequest{
		ID:         r.ID,
		Username:   r.Username,
		Email:      r.Email,
		Status:     dto.RegistrationStatus(r.Status),
		Metadata:   metadata,
		CreatedAt:  r.CreatedAt,
		UpdatedAt:  r.UpdatedAt,
		ApprovedBy: r.ApprovedBy,
		ApprovedAt: r.ApprovedAt,
	}
}

// RegistrationsToDTO converts slice of domain.RegistrationRequest to dto.RegistrationRequest
func RegistrationsToDTO(reqs []domain.RegistrationRequest) []dto.RegistrationRequest {
	result := make([]dto.RegistrationRequest, 0, len(reqs))
	for i := range reqs {
		// Convert Metadata (domain.Metadata is map[string]interface{})
		metadata := make(map[string]any)
		for k, v := range reqs[i].Metadata {
			metadata[k] = v
		}

		result = append(result, dto.RegistrationRequest{
			ID:         reqs[i].ID,
			Username:   reqs[i].Username,
			Email:      reqs[i].Email,
			Status:     dto.RegistrationStatus(reqs[i].Status),
			Metadata:   metadata,
			CreatedAt:  reqs[i].CreatedAt,
			UpdatedAt:  reqs[i].UpdatedAt,
			ApprovedBy: reqs[i].ApprovedBy,
			ApprovedAt: reqs[i].ApprovedAt,
		})
	}
	return result
}
