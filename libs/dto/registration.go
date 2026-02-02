package dto

import (
	"time"

	"github.com/google/uuid"
)

//go:generate easyjson -all registration.go

// RegistrationStatus represents the status of a registration request
type RegistrationStatus string

const (
	RegistrationStatusPending  RegistrationStatus = "pending"
	RegistrationStatusApproved RegistrationStatus = "approved"
	RegistrationStatusRejected RegistrationStatus = "rejected"
)

// RegistrationRequest represents a registration request
//
//easyjson:json
type RegistrationRequest struct {
	ID         uuid.UUID          `json:"id"`
	Username   string             `json:"username"`
	Email      string             `json:"email"`
	Status     RegistrationStatus `json:"status"`
	Metadata   map[string]any     `json:"metadata,omitempty"`
	CreatedAt  time.Time          `json:"createdAt"`
	UpdatedAt  time.Time          `json:"updatedAt"`
	ApprovedBy *uuid.UUID         `json:"approvedBy,omitempty"`
	ApprovedAt *time.Time         `json:"approvedAt,omitempty"`
}

// CreateRegistrationRequest represents a request to create a registration
//
//easyjson:json
type CreateRegistrationRequest struct {
	Username string         `json:"username"`
	Email    string         `json:"email"`
	Password string         `json:"password"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// CreateRegistrationResponse represents a response after creating a registration
//
//easyjson:json
type CreateRegistrationResponse struct {
	RequestID uuid.UUID `json:"requestId"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
}

// ListRegistrationsRequest represents a request to list registrations
//
//easyjson:json
type ListRegistrationsRequest struct {
	Page     int                `json:"page"`
	PageSize int                `json:"pageSize"`
	Status   RegistrationStatus `json:"status,omitempty"`
}

// ListRegistrationsResponse represents a response with list of registrations
//
//easyjson:json
type ListRegistrationsResponse struct {
	Requests   []RegistrationRequest `json:"requests"`
	TotalCount int                   `json:"totalCount"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"pageSize"`
}

// ApproveRegistrationRequest represents a request to approve a registration
//
//easyjson:json
type ApproveRegistrationRequest struct {
	RequestID  uuid.UUID `json:"requestId"`
	ApproverID uuid.UUID `json:"approverId"`
}

// ApproveRegistrationResponse represents a response after approving
//
//easyjson:json
type ApproveRegistrationResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	User    *User  `json:"user,omitempty"`
}

// RejectRegistrationRequest represents a request to reject a registration
//
//easyjson:json
type RejectRegistrationRequest struct {
	RequestID  uuid.UUID `json:"requestId"`
	ApproverID uuid.UUID `json:"approverId"`
	Reason     string    `json:"reason,omitempty"`
}

// RejectRegistrationResponse represents a response after rejecting
//
//easyjson:json
type RejectRegistrationResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
