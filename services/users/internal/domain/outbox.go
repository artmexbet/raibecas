package domain

import (
	"time"

	"github.com/google/uuid"
)

// OutboxEvent represents an event to be published via outbox pattern
type OutboxEvent struct {
	ID            uuid.UUID              `json:"id"`
	AggregateID   uuid.UUID              `json:"aggregate_id"`
	AggregateType string                 `json:"aggregate_type"`
	EventType     string                 `json:"event_type"`
	Payload       map[string]interface{} `json:"payload"`
	CreatedAt     time.Time              `json:"created_at"`
	ProcessedAt   *time.Time             `json:"processed_at,omitempty"`
	RetryCount    int                    `json:"retry_count"`
	LastError     *string                `json:"last_error,omitempty"`
}

// Event types
const (
	EventTypeUserRegistered    = "user.registered"
	EventTypeUserUpdated       = "user.updated"
	EventTypeUserStatusChanged = "user.status_changed"
)

// Aggregate types
const (
	AggregateTypeUser = "user"
)

// UserRegisteredPayload represents the payload for user.registered event
type UserRegisteredPayload struct {
	UserID       uuid.UUID `json:"user_id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
	Role         string    `json:"role"`
	IsActive     bool      `json:"is_active"`
}

// UserUpdatedPayload represents the payload for user.updated event
type UserUpdatedPayload struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	IsActive bool      `json:"is_active"`
}

// UserStatusChangedPayload represents the payload for user.status_changed event
type UserStatusChangedPayload struct {
	UserID   uuid.UUID `json:"user_id"`
	Role     *string   `json:"role,omitempty"`
	IsActive *bool     `json:"is_active,omitempty"`
}
