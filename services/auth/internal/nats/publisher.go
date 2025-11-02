package nats

import (
	"encoding/json"
	"fmt"

	"auth/internal/domain"

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

// PublishUserRegistered publishes a user registered event
func (p *Publisher) PublishUserRegistered(event domain.UserRegisteredEvent) error {
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
func (p *Publisher) PublishUserLogin(event domain.UserLoginEvent) error {
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
func (p *Publisher) PublishUserLogout(event domain.UserLogoutEvent) error {
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
func (p *Publisher) PublishPasswordReset(event domain.PasswordResetEvent) error {
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
func (p *Publisher) PublishRegistrationRequested(event domain.RegistrationRequestedEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := p.conn.Publish("auth.registration.requested", data); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}
