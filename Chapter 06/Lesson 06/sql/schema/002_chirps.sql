-- +goose Up
CREATE TABLE chirps (
    id UUID PRIMARY KEY,
    created_at Timestamp NOT NULL,
    updated_at Timestamp NOT NULL,
    body Text NOT NULL,
    user_id UUID NOT NULL REFERENCES users (id)
        ON DELETE CASCADE
);

-- +goose Down
DROP TABLE chirps;