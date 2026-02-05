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

// GetUnprocessedEventsTx retrieves unprocessed outbox events with row-level locking (SELECT FOR UPDATE)
// Returns a transaction that must be committed or rolled back by caller
func (p *Postgres) GetUnprocessedEventsTx(ctx context.Context, limit int) (pgx.Tx, []domain.OutboxEvent, error) {
	// Start transaction with serializable isolation for consistency
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	query := `
		SELECT id, aggregate_id, aggregate_type, event_type, payload, created_at, processed_at, retry_count, last_error
		FROM outbox
		WHERE processed_at IS NULL AND processing_started_at IS NULL
		ORDER BY created_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`

	rows, err := tx.Query(ctx, query, limit)
	if err != nil {
		tx.Rollback(ctx) //nolint:errcheck
		return nil, nil, fmt.Errorf("failed to query unprocessed events: %w", err)
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
			tx.Rollback(ctx) //nolint:errcheck
			return nil, nil, fmt.Errorf("failed to scan event: %w", err)
		}

		if err := json.Unmarshal(payloadJSON, &event.Payload); err != nil {
			tx.Rollback(ctx) //nolint:errcheck
			return nil, nil, fmt.Errorf("failed to unmarshal payload: %w", err)
		}

		events = append(events, event)
	}

	if err := rows.Err(); err != nil {
		tx.Rollback(ctx) //nolint:errcheck
		return nil, nil, fmt.Errorf("error iterating events: %w", err)
	}

	// Mark events as processing started to prevent other instances from picking them up
	if len(events) > 0 {
		eventIDs := make([]uuid.UUID, len(events))
		for i, e := range events {
			eventIDs[i] = e.ID
		}

		query := `
			UPDATE outbox 
			SET processing_started_at = $1 
			WHERE id = ANY($2)
		`

		_, err := tx.Exec(ctx, query, time.Now(), eventIDs)
		if err != nil {
			tx.Rollback(ctx) //nolint:errcheck
			return nil, nil, fmt.Errorf("failed to mark events as processing: %w", err)
		}
	}

	return tx, events, nil
}

// MarkEventAsProcessed marks an outbox event as successfully processed and commits the transaction
func (p *Postgres) MarkEventAsProcessed(ctx context.Context, tx pgx.Tx, eventID uuid.UUID) error {
	query := `
		UPDATE outbox
		SET processed_at = $1, processing_started_at = NULL
		WHERE id = $2
	`

	result, err := tx.Exec(ctx, query, time.Now(), eventID)
	if err != nil {
		return fmt.Errorf("failed to mark event as processed: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("event not found")
	}

	return nil
}

// MarkEventAsFailed updates retry count and error message for failed event within transaction
func (p *Postgres) MarkEventAsFailed(ctx context.Context, tx pgx.Tx, eventID uuid.UUID, errorMsg string) error {
	query := `
		UPDATE outbox
		SET retry_count = retry_count + 1, last_error = $1, processing_started_at = NULL
		WHERE id = $2
	`

	_, err := tx.Exec(ctx, query, errorMsg, eventID)
	if err != nil {
		return fmt.Errorf("failed to mark event as failed: %w", err)
	}

	return nil
}

// CleanupStaleLocks resets processing_started_at for events stuck longer than timeout
func (p *Postgres) CleanupStaleLocks(ctx context.Context, timeout time.Duration) error {
	query := `
		UPDATE outbox
		SET processing_started_at = NULL
		WHERE processing_started_at IS NOT NULL 
		AND processed_at IS NULL 
		AND NOW() - processing_started_at > $1
	`

	result, err := p.pool.Exec(ctx, query, timeout)
	if err != nil {
		return fmt.Errorf("failed to cleanup stale locks: %w", err)
	}

	if result.RowsAffected() > 0 {
		// Log cleanup happened
		_ = result.RowsAffected()
	}

	return nil
}
