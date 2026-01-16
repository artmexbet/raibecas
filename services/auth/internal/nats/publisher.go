package nats

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"

	"github.com/artmexbet/raibecas/libs/natsw"

	"github.com/artmexbet/raibecas/services/auth/internal/domain"
)

// Publisher handles publishing events to NATS
type Publisher struct {
	client *natsw.Client
}

// NewPublisher creates a new NATS publisher
func NewPublisher(conn *nats.Conn) *Publisher {
	// Создаём клиент для публикации с автоматической пропагацией trace context
	client := natsw.NewClient(conn)

	return &Publisher{client: client}
}

// PublishUserRegistered publishes a user registered event
func (p *Publisher) PublishUserRegistered(ctx context.Context, event domain.UserRegisteredEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Используем переданный контекст для пропагации trace
	if err := p.client.Publish(ctx, SubjectAuthUserRegistered, data); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// PublishUserLogin publishes a user login event
func (p *Publisher) PublishUserLogin(ctx context.Context, event domain.UserLoginEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := p.client.Publish(ctx, SubjectAuthUserLogin, data); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// PublishUserLogout publishes a user logout event
func (p *Publisher) PublishUserLogout(ctx context.Context, event domain.UserLogoutEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := p.client.Publish(ctx, SubjectAuthUserLogout, data); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// PublishPasswordReset publishes a password reset event
func (p *Publisher) PublishPasswordReset(ctx context.Context, event domain.PasswordResetEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := p.client.Publish(ctx, SubjectAuthPasswordReset, data); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

// PublishRegistrationRequested publishes a registration requested event
func (p *Publisher) PublishRegistrationRequested(ctx context.Context, event domain.RegistrationRequestedEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := p.client.Publish(ctx, SubjectAuthRegistrationRequested, data); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}
