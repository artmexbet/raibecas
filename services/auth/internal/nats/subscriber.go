package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"auth/internal/service"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// Subscriber handles subscribing to NATS events
type Subscriber struct {
	conn        *nats.Conn
	regService  *service.RegistrationService
	publisher   *Publisher
	subscribers []*nats.Subscription
}

// NewSubscriber creates a new NATS subscriber
func NewSubscriber(conn *nats.Conn, regService *service.RegistrationService, publisher *Publisher) *Subscriber {
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
func (s *Subscriber) Start(ctx context.Context) error {
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

	log.Println("NATS subscribers started successfully")
	return nil
}

// Stop stops all NATS subscriptions
func (s *Subscriber) Stop() error {
	for _, sub := range s.subscribers {
		if err := sub.Unsubscribe(); err != nil {
			return fmt.Errorf("failed to unsubscribe: %w", err)
		}
	}
	log.Println("NATS subscribers stopped successfully")
	return nil
}

// handleRegistrationApproved handles registration approval events
func (s *Subscriber) handleRegistrationApproved(msg *nats.Msg) {
	var event RegistrationApprovedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("Failed to unmarshal registration approved event: %v", err)
		return
	}

	ctx := context.Background()

	// Approve registration and create user
	userID, err := s.regService.ApproveRegistration(ctx, event.RequestID, event.ApproverID)
	if err != nil {
		log.Printf("Failed to approve registration %s: %v", event.RequestID, err)
		return
	}

	log.Printf("Registration %s approved, user %s created", event.RequestID, userID)

	// Publish user registered event
	if err := s.publisher.PublishUserRegistered(UserRegisteredEvent{
		UserID:    userID,
		Username:  "", // Would need to fetch from DB or pass in event
		Email:     "", // Would need to fetch from DB or pass in event
		Timestamp: time.Now(),
	}); err != nil {
		log.Printf("Failed to publish user registered event: %v", err)
	}
}

// handleRegistrationRejected handles registration rejection events
func (s *Subscriber) handleRegistrationRejected(msg *nats.Msg) {
	var event RegistrationRejectedEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("Failed to unmarshal registration rejected event: %v", err)
		return
	}

	ctx := context.Background()

	// Reject registration
	if err := s.regService.RejectRegistration(ctx, event.RequestID, event.ApproverID); err != nil {
		log.Printf("Failed to reject registration %s: %v", event.RequestID, err)
		return
	}

	log.Printf("Registration %s rejected by %s", event.RequestID, event.ApproverID)
}
