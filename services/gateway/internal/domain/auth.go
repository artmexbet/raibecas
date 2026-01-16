package domain

import "github.com/google/uuid"

//go:generate easyjson -all auth.go

// Auth DTOs - Request/Response models for authentication endpoints

// LoginRequest represents a login request
type LoginRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
	DeviceID  string `json:"deviceId,omitempty"`
	UserAgent string `json:"userAgent,omitempty"`
	IPAddress string `json:"ipAddress,omitempty"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"`
}

// RefreshTokenRequest represents a token refresh request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
	DeviceID     string `json:"deviceId,omitempty"`
	UserAgent    string `json:"userAgent,omitempty"`
	IPAddress    string `json:"ipAddress,omitempty"`
}

// RefreshTokenResponse represents a token refresh response
type RefreshTokenResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"`
}

// LogoutRequest represents a logout request
type LogoutRequest struct {
	Token string `json:"token" validate:"required"`
}

// LogoutAllRequest represents a logout all devices request
type LogoutAllRequest struct {
	Token string `json:"token" validate:"required"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	Token       string `json:"token" validate:"required"`
	OldPassword string `json:"oldPassword" validate:"required,min=8"`
	NewPassword string `json:"newPassword" validate:"required,min=8"`
}

// ValidateTokenRequest represents a token validation request
type ValidateTokenRequest struct {
	Token string `json:"token" validate:"required"`
}

// ValidateTokenResponse represents a token validation response
type ValidateTokenResponse struct {
	Valid  bool      `json:"valid"`
	UserID uuid.UUID `json:"userId,omitempty"`
	Role   string    `json:"role,omitempty"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Message string `json:"message"`
}
