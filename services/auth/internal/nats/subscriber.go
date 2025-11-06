package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"auth/internal/domain"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

type IRegistrationService interface {
	ApproveRegistration(context.Context, uuid.UUID, uuid.UUID) (*domain.User, error)
	RejectRegistration(context.Context, uuid.UUID, uuid.UUID) error
}

// Subscriber handles subscribing to NATS events
type Subscriber struct {
	conn        *nats.Conn
	regService  IRegistrationService
	publisher   IEventPublisher
	subscribers []*nats.Subscription
}

// NewSubscriber creates a new NATS subscriber
func NewSubscriber(conn *nats.Conn, regService IRegistrationService, publisher IEventPublisher) *Subscriber {
	return &Subscriber{
		conn:       conn,
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
	sub1, err := s.conn.Subscribe("admin.registration.approved", s.handleRegistrationApproved)
	if err != nil {
		return fmt.Errorf("failed to subscribe to admin.registration.approved: %w", err)
	}
	s.subscribers = append(s.subscribers, sub1)

	// Subscribe to registration rejected events
	sub2, err := s.conn.Subscribe("admin.registration.rejected", s.handleRegistrationRejected)
	if err != nil {
		return fmt.Errorf("failed to subscribe to admin.registration.rejected: %w", err)
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
func (s *Subscriber) handleRegistrationApproved(msg *nats.Msg) {
	var event RegistrationApprovedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		slog.Error("Failed to unmarshal registration approved event", "error", err)
		return
	}

	ctx := context.Background()

	// Approve registration and create user
	user, err := s.regService.ApproveRegistration(ctx, event.RequestID, event.ApproverID)
	if err != nil {
		slog.Error("Failed to approve registration", "request_id", event.RequestID, "error", err)
		return
	}

	slog.Info("Registration approved, user created", "request_id", event.RequestID, "user_id", user.ID)

	// Publish user registered event with complete user data
	if err := s.publisher.PublishUserRegistered(domain.UserRegisteredEvent{
		UserID:    user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Timestamp: time.Now(),
	}); err != nil {
		slog.Error("Failed to publish user registered event", "error", err)
	}
}

// handleRegistrationRejected handles registration rejection events
func (s *Subscriber) handleRegistrationRejected(msg *nats.Msg) {
	var event RegistrationRejectedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		slog.Error("Failed to unmarshal registration rejected event", "error", err)
		return
	}

	ctx := context.Background()

	// Reject registration
	if err := s.regService.RejectRegistration(ctx, event.RequestID, event.ApproverID); err != nil {
		slog.Error("Failed to reject registration", "request_id", event.RequestID, "error", err)
		return
	}

	slog.Info("Registration rejected", "request_id", event.RequestID, "approver_id", event.ApproverID)
}
