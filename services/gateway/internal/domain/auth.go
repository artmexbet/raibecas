package domain

import (
	"time"

	"github.com/google/uuid"
)

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

// AuthServiceLoginResponse represents the full response from Auth service (internal)
type AuthServiceLoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenID      string    `json:"token_id"`
	Fingerprint  string    `json:"fingerprint"`
	ExpiresIn    int       `json:"expires_in"`
	User         *UserInfo `json:"user,omitempty"`
}

// LoginResponse represents a login response sent to client (public)
type LoginResponse struct {
	AccessToken string    `json:"access_token"`
	ExpiresIn   int       `json:"expires_in"`
	TokenType   string    `json:"token_type"`
	User        *UserInfo `json:"user,omitempty"`
}

// UserInfo represents user information sent to client
type UserInfo struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// RefreshTokenRequest represents a token refresh request (cookie-based)
type RefreshTokenRequest struct {
	DeviceID  string `json:"deviceId,omitempty"`
	UserAgent string `json:"userAgent,omitempty"`
	IPAddress string `json:"ipAddress,omitempty"`
}

// AuthServiceRefreshRequest represents the request to Auth service (internal)
type AuthServiceRefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
	TokenID      string `json:"token_id"`
	Fingerprint  string `json:"fingerprint"`
	DeviceID     string `json:"device_id,omitempty"`
	UserAgent    string `json:"user_agent,omitempty"`
	IPAddress    string `json:"ip_address,omitempty"`
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

// AuthServiceValidateRequest represents validation request to Auth service (internal)
type AuthServiceValidateRequest struct {
	Token           string `json:"token"`
	Fingerprint     string `json:"fingerprint"`
	SkipFingerprint bool   `json:"skip_fingerprint,omitempty"` // Для WS: браузер не может передать fingerprint
}

// ValidateTokenResponse represents a token validation response
type ValidateTokenResponse struct {
	Valid  bool      `json:"valid"`
	UserID uuid.UUID `json:"user_id,omitempty"`
	Role   string    `json:"role,omitempty"`
	JTI    string    `json:"jti,omitempty"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Message string `json:"message"`
}
