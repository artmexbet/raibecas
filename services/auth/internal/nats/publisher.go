package nats

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// Publisher handles publishing events to NATS
type Publisher struct {
	conn *nats.Conn
}

// NewPublisher creates a new NATS publisher
func NewPublisher(conn *nats.Conn) *Publisher {
	return &Publisher{conn: conn}
}

// UserRegisteredEvent represents a user registration event
type UserRegisteredEvent struct {
	UserID    uuid.UUID `json:"user_id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Timestamp time.Time `json:"timestamp"`
}

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

// RegistrationRequestedEvent represents a registration request event
type RegistrationRequestedEvent struct {
	RequestID uuid.UUID      `json:"request_id"`
	Username  string         `json:"username"`
	Email     string         `json:"email"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}

// PublishUserRegistered publishes a user registered event
func (p *Publisher) PublishUserRegistered(event UserRegisteredEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := p.conn.Publish("auth.user.registered", data); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// PublishUserLogin publishes a user login event
func (p *Publisher) PublishUserLogin(event UserLoginEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := p.conn.Publish("auth.user.login", data); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// PublishUserLogout publishes a user logout event
func (p *Publisher) PublishUserLogout(event UserLogoutEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := p.conn.Publish("auth.user.logout", data); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// PublishPasswordReset publishes a password reset event
func (p *Publisher) PublishPasswordReset(event PasswordResetEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := p.conn.Publish("auth.password.reset", data); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// PublishRegistrationRequested publishes a registration requested event
func (p *Publisher) PublishRegistrationRequested(event RegistrationRequestedEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := p.conn.Publish("auth.registration.requested", data); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}
