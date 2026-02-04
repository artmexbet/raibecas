package domain

import (
	"github.com/google/uuid"
)

//go:generate easyjson -all users.go

// ListUsersQuery represents query parameters for listing users
type ListUsersQuery struct {
	Page     int    `json:"page" query:"page" validate:"min=1"`
	PageSize int    `json:"page_size" query:"page_size" validate:"min=1,max=100"`
	Search   string `json:"search" query:"search"`
	IsActive *bool  `json:"is_active" query:"is_active"`
}

// ListUsersResponse represents the response for listing users
type ListUsersResponse struct {
	Users      []User `json:"users"`
	TotalCount int    `json:"total_count"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
}

// UpdateUserRequest represents a request to update user information
type UpdateUserRequest struct {
	Email    *string `json:"email,omitempty" validate:"omitempty,email"`
	Username *string `json:"username,omitempty" validate:"omitempty,min=3,max=50"`
	FullName *string `json:"full_name,omitempty" validate:"omitempty,min=1,max=100"`
	IsActive *bool   `json:"is_active,omitempty"`
	Role     Role    `json:"role"`
}

// UpdateUserResponse represents the response after updating a user
type UpdateUserResponse struct {
	User User `json:"user"`
}

// DeleteUserResponse represents the response after deleting a user
type DeleteUserResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// GetUserRequest represents a request to get a single user
type GetUserRequest struct {
	ID uuid.UUID `json:"id" validate:"required,uuid"`
}

// GetUserResponse represents the response for getting a user
type GetUserResponse struct {
	User User `json:"user"`
}
