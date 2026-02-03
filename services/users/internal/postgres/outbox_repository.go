package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/artmexbet/raibecas/services/users/internal/domain"
)

// CreateOutboxEvent creates a new outbox event in the database
func (p *Postgres) CreateOutboxEvent(ctx context.Context, tx pgx.Tx, event *domain.OutboxEvent) error {
	payloadJSON, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	query := `
		INSERT INTO outbox (id, aggregate_id, aggregate_type, event_type, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = tx.Exec(ctx, query,
		event.ID,
		event.AggregateID,
		event.AggregateType,
		event.EventType,
		payloadJSON,
		event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create outbox event: %w", err)
	}

	return nil
}

// GetUnprocessedEvents retrieves unprocessed outbox events
func (p *Postgres) GetUnprocessedEvents(ctx context.Context, limit int) ([]domain.OutboxEvent, error) {
	query := `
		SELECT id, aggregate_id, aggregate_type, event_type, payload, created_at, processed_at, retry_count, last_error
		FROM outbox
		WHERE processed_at IS NULL
		ORDER BY created_at ASC
		LIMIT $1
	`

	rows, err := p.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query unprocessed events: %w", err)
	}
	defer rows.Close()

	var events []domain.OutboxEvent
	for rows.Next() {
		var event domain.OutboxEvent
		var payloadJSON []byte

		err := rows.Scan(
			&event.ID,
			&event.AggregateID,
			&event.AggregateType,
			&event.EventType,
			&payloadJSON,
			&event.CreatedAt,
			&event.ProcessedAt,
			&event.RetryCount,
			&event.LastError,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		if err := json.Unmarshal(payloadJSON, &event.Payload); err != nil {
			return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events: %w", err)
	}

	return events, nil
}

// MarkEventAsProcessed marks an outbox event as successfully processed
func (p *Postgres) MarkEventAsProcessed(ctx context.Context, eventID uuid.UUID) error {
	query := `
		UPDATE outbox
		SET processed_at = $1
		WHERE id = $2 AND processed_at IS NULL
	`

	result, err := p.pool.Exec(ctx, query, time.Now(), eventID)
	if err != nil {
		return fmt.Errorf("failed to mark event as processed: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("event not found or already processed")
	}

	return nil
}

// MarkEventAsFailed updates retry count and error message for failed event
func (p *Postgres) MarkEventAsFailed(ctx context.Context, eventID uuid.UUID, errorMsg string) error {
	query := `
		UPDATE outbox
		SET retry_count = retry_count + 1, last_error = $1
		WHERE id = $2 AND processed_at IS NULL
	`

	_, err := p.pool.Exec(ctx, query, errorMsg, eventID)
	if err != nil {
		return fmt.Errorf("failed to mark event as failed: %w", err)
	}

	return nil
}
