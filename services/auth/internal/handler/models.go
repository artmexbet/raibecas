package handler

import (
	"github.com/artmexbet/raibecas/services/auth/internal/domain"

	"github.com/google/uuid"
)

// Response is a unified response wrapper for all NATS responses
type Response struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	DeviceID  string `json:"device_id,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
	IPAddress string `json:"ip_address,omitempty"`
}

func (r *LoginRequest) ToDomain() domain.LoginRequest {
	return domain.LoginRequest{
		Email:     r.Email,
		Password:  r.Password,
		DeviceID:  r.DeviceID,
		UserAgent: r.UserAgent,
		IPAddress: r.IPAddress,
	}
}

// LoginResponse represents a login response
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// ValidateRequest represents a token validation request
type ValidateRequest struct {
	Token string `json:"token"`
}

// ValidateResponse represents a token validation response
type ValidateResponse struct {
	Valid  bool      `json:"valid"`
	UserID uuid.UUID `json:"user_id,omitempty"`
	Role   string    `json:"role,omitempty"`
}

// RefreshRequest represents a token refresh request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
	DeviceID     string `json:"device_id,omitempty"`
	UserAgent    string `json:"user_agent,omitempty"`
	IPAddress    string `json:"ip_address,omitempty"`
}

func (r *RefreshRequest) ToDomain() domain.RefreshRequest {
	return domain.RefreshRequest{
		RefreshToken: r.RefreshToken,
		DeviceID:     r.DeviceID,
		UserAgent:    r.UserAgent,
		IPAddress:    r.IPAddress,
	}
}

// LogoutRequest represents a logout request
type LogoutRequest struct {
	UserID uuid.UUID `json:"user_id"`
	Token  string    `json:"token"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Message string `json:"message"`
}

// LogoutAllRequest represents a logout all request
type LogoutAllRequest struct {
	UserID uuid.UUID `json:"user_id"`
	Token  string    `json:"token"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	UserID      uuid.UUID `json:"user_id"`
	Token       string    `json:"token"`
	OldPassword string    `json:"old_password"`
	NewPassword string    `json:"new_password"`
}

func (r *ChangePasswordRequest) ToDomain() domain.ChangePasswordRequest {
	return domain.ChangePasswordRequest{
		UserID:      r.UserID,
		OldPassword: r.OldPassword,
		NewPassword: r.NewPassword,
	}
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Username string                 `json:"username"`
	Email    string                 `json:"email"`
	Password string                 `json:"password"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (r *RegisterRequest) ToDomain() domain.RegisterRequest {
	return domain.RegisterRequest{
		Username: r.Username,
		Email:    r.Email,
		Password: r.Password,
		Metadata: r.Metadata,
	}
}

// RegisterResponse represents a registration response
type RegisterResponse struct {
	RequestID uuid.UUID `json:"request_id"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
}
