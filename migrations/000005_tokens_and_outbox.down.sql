-- Drop indexes
DROP INDEX IF EXISTS idx_outbox_events_created_at;
DROP INDEX IF EXISTS idx_outbox_events_status;
DROP INDEX IF EXISTS idx_user_tokens_user_type;
DROP INDEX IF EXISTS idx_user_tokens_token;

-- Drop tables
DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS user_tokens;

-- Drop custom types
DROP TYPE IF EXISTS token_type; 