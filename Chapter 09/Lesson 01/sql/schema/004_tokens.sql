-- +goose Up
CREATE TABLE refresh_tokens (
    token text PRIMARY KEY,
    created_at Timestamp NOT NULL,
    updated_at Timestamp NOT NULL,
    user_id UUID NOT NULL REFERENCES users (id)
        ON DELETE CASCADE,
    expires_at Timestamp NOT NULL,
    revoked_at Timestamp
);

-- +goose Down
DROP TABLE refresh_tokens;