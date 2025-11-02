package domain

import (
	"time"

	"github.com/google/uuid"
)

// UserLoginEvent represents a user login event
type UserLoginEvent struct {
	UserID    uuid.UUID `json:"user_id"`
	DeviceID  string    `json:"device_id"`
	UserAgent string    `json:"user_agent"`
	IPAddress string    `json:"ip_address"`
	Timestamp time.Time `json:"timestamp"`
}

// UserLogoutEvent represents a user logout event
type UserLogoutEvent struct {
	UserID    uuid.UUID `json:"user_id"`
	Timestamp time.Time `json:"timestamp"`
}

// PasswordResetEvent represents a password reset event
type PasswordResetEvent struct {
	UserID    uuid.UUID `json:"user_id"`
	Method    string    `json:"method"`
	Timestamp time.Time `json:"timestamp"`
}

// UserRegisteredEvent represents a user registration event
type UserRegisteredEvent struct {
	UserID    uuid.UUID `json:"user_id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Timestamp time.Time `json:"timestamp"`
}

// RegistrationRequestedEvent represents a registration request event
type RegistrationRequestedEvent struct {
	RequestID uuid.UUID      `json:"request_id"`
	Username  string         `json:"username"`
	Email     string         `json:"email"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}
