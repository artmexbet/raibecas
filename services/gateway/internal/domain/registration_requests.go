package domain

import (
	"time"

	"github.com/google/uuid"
)

//go:generate easyjson -all registration_requests.go

// RegistrationStatus represents the status of a registration request
type RegistrationStatus string

const (
	RegistrationStatusPending  RegistrationStatus = "pending"
	RegistrationStatusApproved RegistrationStatus = "approved"
	RegistrationStatusRejected RegistrationStatus = "rejected"
)

// RegistrationRequest represents a user registration request
type RegistrationRequest struct {
	ID         uuid.UUID          `json:"id"`
	Username   string             `json:"username"`
	Email      string             `json:"email"`
	Status     RegistrationStatus `json:"status"`
	Metadata   map[string]any     `json:"metadata,omitempty"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
	ApprovedBy *uuid.UUID         `json:"approved_by,omitempty"`
	ApprovedAt *time.Time         `json:"approved_at,omitempty"`
}

// CreateRegistrationRequestRequest represents a request to create a registration request
type CreateRegistrationRequestRequest struct {
	Username string         `json:"username" validate:"required,min=3,max=50"`
	Email    string         `json:"email" validate:"required,email"`
	Password string         `json:"password" validate:"required,min=8"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// CreateRegistrationRequestResponse represents the response after creating a registration request
type CreateRegistrationRequestResponse struct {
	RequestID uuid.UUID `json:"request_id"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
}

// ListRegistrationRequestsQuery represents query parameters for listing registration requests
type ListRegistrationRequestsQuery struct {
	Page     int                `json:"page" query:"page" validate:"min=1"`
	PageSize int                `json:"page_size" query:"page_size" validate:"min=1,max=100"`
	Status   RegistrationStatus `json:"status" query:"status"`
}

// ListRegistrationRequestsResponse represents the response for listing registration requests
type ListRegistrationRequestsResponse struct {
	Requests   []RegistrationRequest `json:"requests"`
	TotalCount int                   `json:"total_count"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"page_size"`
}

// ApproveRegistrationRequestRequest represents a request to approve a registration request
type ApproveRegistrationRequestRequest struct {
	RequestID uuid.UUID `json:"request_id" validate:"required,uuid"`
}

// ApproveRegistrationRequestResponse represents the response after approving a registration request
type ApproveRegistrationRequestResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	User    *User  `json:"user,omitempty"`
}

// RejectRegistrationRequestRequest represents a request to reject a registration request
type RejectRegistrationRequestRequest struct {
	RequestID uuid.UUID `json:"request_id" validate:"required,uuid"`
	Reason    string    `json:"reason,omitempty"`
}

// RejectRegistrationRequestResponse represents the response after rejecting a registration request
type RejectRegistrationRequestResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// GetRegistrationRequestResponse represents the response for getting a registration request
type GetRegistrationRequestResponse struct {
	Request RegistrationRequest `json:"request"`
}
