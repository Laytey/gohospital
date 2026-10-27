CREATE TABLE IF NOT EXISTS outbox_events (
    id UUID PRIMARY KEY,
    topic TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMP
);

-- Индекс для быстрого поиска неотправленных событий
CREATE INDEX IF NOT EXISTS idx_outbox_events_unsent ON outbox_events(created_at) WHERE sent_at IS NULL;