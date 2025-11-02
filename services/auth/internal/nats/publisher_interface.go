package nats

import "auth/internal/domain"

// IEventPublisher defines the interface for publishing events
type IEventPublisher interface {
	PublishUserLogin(domain.UserLoginEvent) error
	PublishUserLogout(domain.UserLogoutEvent) error
	PublishPasswordReset(domain.PasswordResetEvent) error
	PublishRegistrationRequested(domain.RegistrationRequestedEvent) error
	PublishUserRegistered(domain.UserRegisteredEvent) error
}
