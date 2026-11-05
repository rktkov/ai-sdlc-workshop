-- +goose Up
CREATE TABLE batches (
    batch_id          TEXT PRIMARY KEY,
    user_id           TEXT NOT NULL,
    state             TEXT NOT NULL,
    degradation_reason TEXT NOT NULL,
    payload           TEXT NOT NULL,
    created_at        DATETIME NOT NULL,
    completed_at      DATETIME
);

CREATE INDEX batches_user_completed ON batches(user_id, completed_at);

-- +goose Down
DROP INDEX batches_user_completed;
DROP TABLE batches;
