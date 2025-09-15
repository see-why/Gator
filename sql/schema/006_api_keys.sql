-- +goose Up
ALTER TABLE users ADD COLUMN api_key TEXT UNIQUE;

-- +goose Down
ALTER TABLE users DROP COLUMN api_key;
