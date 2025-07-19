-- +goose Up
CREATE TABLE users (
    id UUID PRIMARY KEY,
    created_at Timestamp NOT NULL,
    updated_at Timestamp NOT NULL,
    email Text NOT NULL UNIQUE
);

-- +goose Down
DROP TABLE users;