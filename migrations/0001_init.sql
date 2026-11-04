-- +goose Up
CREATE TABLE users (
    user_id    TEXT PRIMARY KEY,
    created_at DATETIME NOT NULL
);

-- +goose Down
DROP TABLE users;
