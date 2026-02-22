package domain

import (
	"time"

	"github.com/google/uuid"
)

//go:generate easyjson -all user.go

//easyjson:json
type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	Username     string    `json:"username"`
	FullName     string    `json:"full_name"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	LastLoginAt  time.Time `json:"last_login_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Role constants
const (
	RoleUser       = "User"
	RoleAdmin      = "Admin"
	RoleSuperAdmin = "SuperAdmin"
)

// IsValidRole checks if role is valid
func IsValidRole(role string) bool {
	switch role {
	case RoleUser, RoleAdmin, RoleSuperAdmin:
		return true
	default:
		return false
	}
}

// UpdateUserParams represents parameters for updating a user
type UpdateUserParams struct {
	ID       uuid.UUID
	Email    *string
	Username *string
	FullName *string
	Role     *string
	IsActive *bool
}
