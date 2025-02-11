CREATE TABLE outbox_events (
    id UUID PRIMARY KEY,
    event_type VARCHAR(255) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    error TEXT,
    retry_count INT DEFAULT 0,
    retry_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL,
    processed_at TIMESTAMP,
    updated_at TIMESTAMP NOT NULL,
    region_code VARCHAR(10),
    CONSTRAINT chk_outbox_status CHECK (status IN ('pending', 'processing', 'processed', 'failed'))
);

CREATE INDEX idx_outbox_events_status ON outbox_events(status);
CREATE INDEX idx_outbox_events_created_at ON outbox_events(created_at);
CREATE INDEX idx_outbox_events_retry ON outbox_events(retry_at) WHERE status = 'failed'; 