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
