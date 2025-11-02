package queries

import (
	"auth/internal/domain"
	"fmt"
	"utills/pg"
)

func (u *User) ToDomain() (domain.User, error) {
	id, err := pg.GoogleUUIDFromPG(u.ID)
	if err != nil {
		return domain.User{}, fmt.Errorf("failed to convert UUID: %w", err)
	}
	return domain.User{
		ID:           id,
		Username:     u.Username,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Role:         domain.UserRole(u.Role),
		IsActive:     u.IsActive,
		CreatedAt:    u.CreatedAt.Time,
		UpdatedAt:    u.UpdatedAt.Time,
	}, nil
}

func FromDomainUser(user domain.User) User {
	return User{
		ID:           pg.UUIDFromGoogle(user.ID),
		Username:     user.Username,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Role:         user.Role.String(),
		IsActive:     user.IsActive,
		CreatedAt:    pg.ConvertToPGTime(user.CreatedAt),
		UpdatedAt:    pg.ConvertToPGTime(user.UpdatedAt),
	}
}
