-- Create outbox table for transactional outbox pattern
CREATE TABLE IF NOT EXISTS outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_id UUID NOT NULL,
    aggregate_type VARCHAR(100) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMP,
    processing_started_at TIMESTAMP,
    retry_count INT NOT NULL DEFAULT 0,
    last_error TEXT,

    -- Index for polling unprocessed events
    CONSTRAINT check_retry_count CHECK (retry_count >= 0)
);

-- Index for efficient polling of unprocessed events
CREATE INDEX idx_outbox_unprocessed ON outbox(created_at)
WHERE processed_at IS NULL AND processing_started_at IS NULL;

-- Index for cleaning up stale locks (timeout mechanism)
CREATE INDEX idx_outbox_stale_locks ON outbox(processing_started_at)
WHERE processed_at IS NULL AND processing_started_at IS NOT NULL;

-- Index for monitoring and debugging
CREATE INDEX idx_outbox_aggregate ON outbox(aggregate_type, aggregate_id);
