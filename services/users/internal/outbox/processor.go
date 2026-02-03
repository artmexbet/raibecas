package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/artmexbet/raibecas/services/users/internal/domain"
)

const (
	defaultPollInterval = 5 * time.Second
	defaultBatchSize    = 10
	maxRetryCount       = 5
)

// Repository defines the interface for outbox data access
type Repository interface {
	GetUnprocessedEvents(ctx context.Context, limit int) ([]domain.OutboxEvent, error)
	MarkEventAsProcessed(ctx context.Context, eventID uuid.UUID) error
	MarkEventAsFailed(ctx context.Context, eventID uuid.UUID, errorMsg string) error
}

// Publisher defines the interface for publishing events
type Publisher interface {
	Publish(ctx context.Context, subject string, data []byte) error
}

// Processor handles the outbox pattern processing
type Processor struct {
	repo         Repository
	publisher    Publisher
	pollInterval time.Duration
	batchSize    int
	logger       *slog.Logger
}

// NewProcessor creates a new outbox processor
func NewProcessor(repo Repository, publisher Publisher, logger *slog.Logger) *Processor {
	if logger == nil {
		logger = slog.Default()
	}

	return &Processor{
		repo:         repo,
		publisher:    publisher,
		pollInterval: defaultPollInterval,
		batchSize:    defaultBatchSize,
		logger:       logger,
	}
}

// Start begins processing outbox events
func (p *Processor) Start(ctx context.Context) error {
	p.logger.Info("starting outbox processor",
		"poll_interval", p.pollInterval,
		"batch_size", p.batchSize,
	)

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	// Process immediately on start
	if err := p.processEvents(ctx); err != nil {
		p.logger.Error("failed to process events on start", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("stopping outbox processor")
			return ctx.Err()
		case <-ticker.C:
			if err := p.processEvents(ctx); err != nil {
				p.logger.Error("failed to process events", "error", err)
			}
		}
	}
}

// processEvents processes a batch of unprocessed events
func (p *Processor) processEvents(ctx context.Context) error {
	events, err := p.repo.GetUnprocessedEvents(ctx, p.batchSize)
	if err != nil {
		return fmt.Errorf("failed to get unprocessed events: %w", err)
	}

	if len(events) == 0 {
		return nil
	}

	p.logger.Info("processing outbox events", "count", len(events))

	for _, event := range events {
		if err := p.processEvent(ctx, event); err != nil {
			p.logger.Error("failed to process event",
				"event_id", event.ID,
				"event_type", event.EventType,
				"error", err,
			)
			continue
		}
	}

	return nil
}

// processEvent processes a single outbox event
func (p *Processor) processEvent(ctx context.Context, event domain.OutboxEvent) error {
	// Check retry limit
	if event.RetryCount >= maxRetryCount {
		p.logger.Warn("event exceeded max retry count",
			"event_id", event.ID,
			"retry_count", event.RetryCount,
		)
		// Mark as processed to stop retrying
		return p.repo.MarkEventAsProcessed(ctx, event.ID)
	}

	// Determine NATS subject based on event type
	subject := p.getSubjectForEvent(event.EventType)
	if subject == "" {
		return fmt.Errorf("unknown event type: %s", event.EventType)
	}

	// Serialize payload
	data, err := json.Marshal(event.Payload)
	if err != nil {
		errMsg := fmt.Sprintf("failed to marshal payload: %v", err)
		if markErr := p.repo.MarkEventAsFailed(ctx, event.ID, errMsg); markErr != nil {
			p.logger.Error("failed to mark event as failed", "error", markErr)
		}
		return fmt.Errorf(errMsg)
	}

	// Publish to NATS
	if err := p.publisher.Publish(ctx, subject, data); err != nil {
		errMsg := fmt.Sprintf("failed to publish event: %v", err)
		if markErr := p.repo.MarkEventAsFailed(ctx, event.ID, errMsg); markErr != nil {
			p.logger.Error("failed to mark event as failed", "error", markErr)
		}
		return fmt.Errorf(errMsg)
	}

	// Mark as processed
	if err := p.repo.MarkEventAsProcessed(ctx, event.ID); err != nil {
		return fmt.Errorf("failed to mark event as processed: %w", err)
	}

	p.logger.Info("event processed successfully",
		"event_id", event.ID,
		"event_type", event.EventType,
		"subject", subject,
	)

	return nil
}

// getSubjectForEvent maps event types to NATS subjects
func (p *Processor) getSubjectForEvent(eventType string) string {
	switch eventType {
	case domain.EventTypeUserRegistered:
		return "users.user.registered"
	default:
		return ""
	}
}
