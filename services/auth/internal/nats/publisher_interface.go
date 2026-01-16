package nats

import (
	"context"

	"github.com/artmexbet/raibecas/services/auth/internal/domain"
)

// EventPublisher defines the interface for publishing events
type EventPublisher interface {
	PublishUserLogin(ctx context.Context, event domain.UserLoginEvent) error
	PublishUserLogout(ctx context.Context, event domain.UserLogoutEvent) error
	PublishPasswordReset(ctx context.Context, event domain.PasswordResetEvent) error
	PublishRegistrationRequested(ctx context.Context, event domain.RegistrationRequestedEvent) error
	PublishUserRegistered(ctx context.Context, event domain.UserRegisteredEvent) error
}
