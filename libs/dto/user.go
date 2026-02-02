package dto

import (
	"time"

	"github.com/google/uuid"
)

//go:generate easyjson -all user.go

// User represents a user in the system (shared between services)
//
//easyjson:json
type User struct {
	ID          uuid.UUID `json:"id"`
	Email       string    `json:"email"`
	Username    string    `json:"username"`
	FullName    string    `json:"fullName"`
	Role        string    `json:"role"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
	LastLoginAt time.Time `json:"lastLoginAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// ListUsersRequest represents a request to list users
//
//easyjson:json
type ListUsersRequest struct {
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
	Search   string `json:"search,omitempty"`
	IsActive *bool  `json:"isActive,omitempty"`
}

// ListUsersResponse represents a response with list of users
//
//easyjson:json
type ListUsersResponse struct {
	Users      []User `json:"users"`
	TotalCount int    `json:"totalCount"`
	Page       int    `json:"page"`
	PageSize   int    `json:"pageSize"`
}

// GetUserRequest represents a request to get a single user
//
//easyjson:json
type GetUserRequest struct {
	ID uuid.UUID `json:"id"`
}

// GetUserResponse represents a response with a single user
//
//easyjson:json
type GetUserResponse struct {
	User User `json:"user"`
}

// UpdateUserRequest represents a request to update user fields
//
//easyjson:json
type UpdateUserRequest struct {
	ID      uuid.UUID         `json:"id"`
	Updates UpdateUserPayload `json:"updates"`
}

// UpdateUserPayload contains fields to update
//
//easyjson:json
type UpdateUserPayload struct {
	Email    *string `json:"email,omitempty"`
	Username *string `json:"username,omitempty"`
	FullName *string `json:"fullName,omitempty"`
	IsActive *bool   `json:"isActive,omitempty"`
}

// UpdateUserResponse represents a response after updating a user
//
//easyjson:json
type UpdateUserResponse struct {
	User User `json:"user"`
}

// DeleteUserRequest represents a request to delete a user
//
//easyjson:json
type DeleteUserRequest struct {
	ID uuid.UUID `json:"id"`
}

// DeleteUserResponse represents a response after deleting a user
//
//easyjson:json
type DeleteUserResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}
