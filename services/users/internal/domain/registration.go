package domain

import (
	"time"

	"github.com/google/uuid"
)

//go:generate easyjson -all registration.go

type RegistrationStatus string

const (
	RegistrationStatusPending  RegistrationStatus = "pending"
	RegistrationStatusApproved RegistrationStatus = "approved"
	RegistrationStatusRejected RegistrationStatus = "rejected"
)

//easyjson:json
type Metadata map[string]interface{}

//easyjson:json
type RegistrationRequest struct {
	ID           uuid.UUID          `json:"id"`
	Username     string             `json:"username"`
	Email        string             `json:"email"`
	PasswordHash string             `json:"-"`
	Status       RegistrationStatus `json:"status"`
	Metadata     Metadata           `json:"metadata"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
	ApprovedBy   *uuid.UUID         `json:"approved_by,omitempty"`
	ApprovedAt   *time.Time         `json:"approved_at,omitempty"`
}
