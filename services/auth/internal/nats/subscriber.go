package nats

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	"github.com/artmexbet/raibecas/libs/natsw"
	"github.com/artmexbet/raibecas/services/auth/internal/domain"
)

type IRegistrationService interface {
	ApproveRegistration(context.Context, uuid.UUID, uuid.UUID) (*domain.User, error)
	RejectRegistration(context.Context, uuid.UUID, uuid.UUID) error
}

// Subscriber handles subscribing to NATS events
type Subscriber struct {
	client      *natsw.Client
	regService  IRegistrationService
	publisher   EventPublisher
	subscribers []*nats.Subscription
}

// NewSubscriber creates a new NATS subscriber
func NewSubscriber(conn *nats.Conn, regService IRegistrationService, publisher EventPublisher) *Subscriber {
	// Создаём клиент с middleware
	client := natsw.NewClient(conn,
		natsw.WithLogger(slog.Default()),
		natsw.WithRecover(),
	)

	return &Subscriber{
		client:     client,
		regService: regService,
		publisher:  publisher,
	}
}

// RegistrationApprovedEvent represents a registration approval event from admin service
type RegistrationApprovedEvent struct {
	RequestID  uuid.UUID `json:"request_id"`
	ApproverID uuid.UUID `json:"approver_id"`
}

// RegistrationRejectedEvent represents a registration rejection event from admin service
type RegistrationRejectedEvent struct {
	RequestID  uuid.UUID `json:"request_id"`
	ApproverID uuid.UUID `json:"approver_id"`
	Reason     string    `json:"reason,omitempty"`
}

// Start starts all NATS subscriptions
func (s *Subscriber) Start(_ context.Context) error {
	// Subscribe to registration approved events
	sub1, err := s.client.Subscribe(SubjectAdminRegistrationApproved, s.handleRegistrationApproved)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", SubjectAdminRegistrationApproved, err)
	}
	s.subscribers = append(s.subscribers, sub1)

	// Subscribe to registration rejected events
	sub2, err := s.client.Subscribe(SubjectAdminRegistrationRejected, s.handleRegistrationRejected)
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", SubjectAdminRegistrationRejected, err)
	}
	s.subscribers = append(s.subscribers, sub2)

	slog.Info("NATS subscribers started successfully")
	return nil
}

// Stop stops all NATS subscriptions
func (s *Subscriber) Stop() error {
	for _, sub := range s.subscribers {
		if err := sub.Unsubscribe(); err != nil {
			return fmt.Errorf("failed to unsubscribe: %w", err)
		}
	}
	slog.Info("NATS subscribers stopped successfully")
	return nil
}

// handleRegistrationApproved handles registration approval events
func (s *Subscriber) handleRegistrationApproved(msg *natsw.Message) error {
	var event RegistrationApprovedEvent
	if err := msg.UnmarshalData(&event); err != nil {
		return fmt.Errorf("failed to unmarshal registration approved event: %w", err)
	}

	// Используем контекст из сообщения (содержит trace context)
	ctx := msg.Ctx

	// Approve registration and create user
	user, err := s.regService.ApproveRegistration(ctx, event.RequestID, event.ApproverID)
	if err != nil {
		return fmt.Errorf("failed to approve registration: %w", err)
	}

	slog.InfoContext(ctx, "Registration approved, user created",
		"request_id", event.RequestID,
		"user_id", user.ID)

	// Publish user registered event with complete user data
	if err := s.publisher.PublishUserRegistered(ctx, domain.UserRegisteredEvent{
		UserID:    user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Timestamp: time.Now(),
	}); err != nil {
		return fmt.Errorf("failed to publish user registered event: %w", err)
	}

	return nil
}

// handleRegistrationRejected handles registration rejection events
func (s *Subscriber) handleRegistrationRejected(msg *natsw.Message) error {
	var event RegistrationRejectedEvent
	if err := msg.UnmarshalData(&event); err != nil {
		return fmt.Errorf("failed to unmarshal registration rejected event: %w", err)
	}

	// Используем контекст из сообщения (содержит trace context)
	ctx := msg.Ctx

	// Reject registration
	if err := s.regService.RejectRegistration(ctx, event.RequestID, event.ApproverID); err != nil {
		return fmt.Errorf("failed to reject registration: %w", err)
	}

	slog.InfoContext(ctx, "Registration rejected",
		"request_id", event.RequestID,
		"approver_id", event.ApproverID)

	return nil
}
