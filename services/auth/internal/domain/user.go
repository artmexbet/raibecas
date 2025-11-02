package domain

import (
	"time"

	"github.com/google/uuid"
)

// UserRole represents user roles in the system
type UserRole string

func (u UserRole) String() string {
	return string(u)
}

const (
	RoleUser  UserRole = "user"
	RoleAdmin UserRole = "admin"
)

// User represents a registered user in the system
type User struct {
	ID           uuid.UUID `db:"id" json:"id"`
	Username     string    `db:"username" json:"username"`
	Email        string    `db:"email" json:"email"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Role         UserRole  `db:"role" json:"role"`
	IsActive     bool      `db:"is_active" json:"is_active"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

// RegistrationStatus represents the status of a registration request
type RegistrationStatus string

func (r RegistrationStatus) String() string {
	return string(r)
}

const (
	StatusPending  RegistrationStatus = "pending"
	StatusApproved RegistrationStatus = "approved"
	StatusRejected RegistrationStatus = "rejected"
)

// RegistrationRequest represents a user registration request
type RegistrationRequest struct {
	ID         uuid.UUID          `db:"id" json:"id"`
	Username   string             `db:"username" json:"username"`
	Email      string             `db:"email" json:"email"`
	Password   string             `db:"password" json:"-"`
	Status     RegistrationStatus `db:"status" json:"status"`
	Metadata   map[string]any     `db:"metadata" json:"metadata,omitempty"`
	CreatedAt  time.Time          `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time          `db:"updated_at" json:"updated_at"`
	ApprovedBy *uuid.UUID         `db:"approved_by" json:"approved_by,omitempty"`
	ApprovedAt *time.Time         `db:"approved_at" json:"approved_at,omitempty"`
}

// RefreshToken represents a refresh token stored in Redis
type RefreshToken struct {
	Token     string    `json:"token"`
	UserID    uuid.UUID `json:"user_id"`
	DeviceID  string    `json:"device_id"`
	UserAgent string    `json:"user_agent"`
	IPAddress string    `json:"ip_address"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email     string
	Password  string
	DeviceID  string
	UserAgent string
	IPAddress string
}

// TokenPair represents access and refresh tokens
type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

// RefreshRequest represents a token refresh request
type RefreshRequest struct {
	RefreshToken string
	DeviceID     string
	UserAgent    string
	IPAddress    string
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	UserID      uuid.UUID
	OldPassword string
	NewPassword string
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Username string
	Email    string
	Password string
	Metadata map[string]any
}
