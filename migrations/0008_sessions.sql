-- +goose Up
CREATE TABLE sessions (
    session_id       TEXT PRIMARY KEY,
    user_id          TEXT NOT NULL,
    batch_id         TEXT NOT NULL,
    state            TEXT NOT NULL,
    current_card_id  TEXT NOT NULL,
    furthest_card_id TEXT NOT NULL,
    last_gesture_id  TEXT NOT NULL,
    lease_owner      TEXT NOT NULL,
    last_activity_at DATETIME NOT NULL,
    expires_at       DATETIME NOT NULL,
    opened_at        DATETIME NOT NULL,
    closed_at        DATETIME,
    resume_count     INTEGER NOT NULL,
    version          INTEGER NOT NULL,
    CHECK (state IN ('open', 'closed')),
    CHECK (resume_count >= 0)
);

CREATE UNIQUE INDEX sessions_one_open
    ON sessions(user_id) WHERE state = 'open';

-- +goose Down
DROP INDEX sessions_one_open;
DROP TABLE sessions;
