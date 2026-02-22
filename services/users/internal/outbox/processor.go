package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/artmexbet/raibecas/services/users/internal/domain"
)

const (
	defaultPollInterval = 5 * time.Second
	defaultBatchSize    = 10
	maxRetryCount       = 5
	lockTimeout         = 30 * time.Second // Cleanup locks older than 30 seconds
)

// Repository defines the interface for outbox data access
type Repository interface {
	GetUnprocessedEventsTx(ctx context.Context, limit int) (pgx.Tx, []domain.OutboxEvent, error)
	MarkEventAsProcessed(ctx context.Context, tx pgx.Tx, eventID uuid.UUID) error
	MarkEventAsFailed(ctx context.Context, tx pgx.Tx, eventID uuid.UUID, errorMsg string) error
	CleanupStaleLocks(ctx context.Context, timeout time.Duration) error
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

	// Cleanup ticker for stale locks (every 10 seconds)
	cleanupTicker := time.NewTicker(10 * time.Second)
	defer cleanupTicker.Stop()

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
		case <-cleanupTicker.C:
			if err := p.repo.CleanupStaleLocks(ctx, lockTimeout); err != nil {
				p.logger.Error("failed to cleanup stale locks", "error", err)
			}
		}
	}
}

// processEvents processes a batch of unprocessed events with transactional locking
func (p *Processor) processEvents(ctx context.Context) error {
	// Get events with row-level lock (SELECT FOR UPDATE)
	tx, events, err := p.repo.GetUnprocessedEventsTx(ctx, p.batchSize)
	if err != nil {
		return fmt.Errorf("failed to get unprocessed events: %w", err)
	}

	if len(events) == 0 {
		tx.Rollback(ctx) //nolint:errcheck
		return nil
	}

	p.logger.Info("processing outbox events", "count", len(events))

	// Process events within transaction
	for _, event := range events {
		if err := p.processEvent(ctx, tx, event); err != nil {
			p.logger.Error("failed to process event",
				"event_id", event.ID,
				"event_type", event.EventType,
				"error", err,
			)
			// Continue processing other events in batch
		}
	}

	// Commit transaction to release locks
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// processEvent processes a single outbox event within the transaction
func (p *Processor) processEvent(ctx context.Context, tx pgx.Tx, event domain.OutboxEvent) error {
	// Check retry limit
	if event.RetryCount >= maxRetryCount {
		p.logger.Warn("event exceeded max retry count",
			"event_id", event.ID,
			"retry_count", event.RetryCount,
		)
		// Mark as processed to stop retrying
		return p.repo.MarkEventAsProcessed(ctx, tx, event.ID)
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
		if markErr := p.repo.MarkEventAsFailed(ctx, tx, event.ID, errMsg); markErr != nil {
			p.logger.Error("failed to mark event as failed", "error", markErr)
		}
		return errors.New(errMsg)
	}

	// Publish to NATS
	if err := p.publisher.Publish(ctx, subject, data); err != nil {
		errMsg := fmt.Sprintf("failed to publish event: %v", err)
		if markErr := p.repo.MarkEventAsFailed(ctx, tx, event.ID, errMsg); markErr != nil {
			p.logger.Error("failed to mark event as failed", "error", markErr)
		}
		return errors.New(errMsg)
	}

	// Mark as processed
	if err := p.repo.MarkEventAsProcessed(ctx, tx, event.ID); err != nil {
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
	case domain.EventTypeUserUpdated:
		return "users.user.updated"
	case domain.EventTypeUserStatusChanged:
		return "users.user.status_changed"
	default:
		return ""
	}
}
