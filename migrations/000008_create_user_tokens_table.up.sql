CREATE TYPE token_type AS ENUM ('reset', 'verification');

CREATE TABLE user_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id),
    token VARCHAR(255) NOT NULL UNIQUE,
    type token_type NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, type)
);

CREATE INDEX idx_user_tokens_token ON user_tokens(token);
CREATE INDEX idx_user_tokens_user_type ON user_tokens(user_id, type); 