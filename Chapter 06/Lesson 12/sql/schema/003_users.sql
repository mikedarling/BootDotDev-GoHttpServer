-- +goose Up
ALTER TABLE users
ADD COLUMN hashed_password Text NOT NULL DEFAULT 'unset';

-- +goose Down
ALTER TABLE users
DROP COLUMN hashed_password;