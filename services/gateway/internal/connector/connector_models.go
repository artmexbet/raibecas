package connector

import "github.com/google/uuid"

//go:generate easyjson -all connector_models.go

// Auth connector models

// LogoutRequest represents a logout request to auth service
type LogoutRequest struct {
	TokenID        string    `json:"token_id"`
	AccessTokenJTI string    `json:"access_token_jti"`
	UserID         uuid.UUID `json:"user_id"`
	Token          string    `json:"token"`
}

// LogoutAllRequest represents a logout all request to auth service
type LogoutAllRequest struct {
	UserID uuid.UUID `json:"user_id"`
	Token  string    `json:"token"`
}

// ChangePasswordRequest represents a change password request to auth service
type ChangePasswordRequest struct {
	UserID      uuid.UUID `json:"user_id"`
	Token       string    `json:"token"`
	OldPassword string    `json:"old_password"`
	NewPassword string    `json:"new_password"`
}

// Users connector models

// UpdateUserRequestWrapper wraps user update request with ID
type UpdateUserRequestWrapper struct {
	ID      uuid.UUID         `json:"id"`
	Updates UpdateUserUpdates `json:"updates"`
}

// UpdateUserUpdates represents the updates for a user
type UpdateUserUpdates struct {
	Email    *string `json:"email,omitempty"`
	Username *string `json:"username,omitempty"`
	FullName *string `json:"full_name,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
}

// DeleteUserRequest represents a delete user request
type DeleteUserRequest struct {
	ID uuid.UUID `json:"id"`
}

// ApproveRegistrationRequest represents an approve registration request
type ApproveRegistrationRequest struct {
	RequestID  uuid.UUID `json:"request_id"`
	ApproverID uuid.UUID `json:"approver_id"`
}

// RejectRegistrationRequest represents a reject registration request
type RejectRegistrationRequest struct {
	RequestID  uuid.UUID `json:"request_id"`
	ApproverID uuid.UUID `json:"approver_id"`
	Reason     string    `json:"reason,omitempty"`
}
