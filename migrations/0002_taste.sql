-- +goose Up
CREATE TABLE signals (
    signal_id   TEXT PRIMARY KEY,
    gesture_id  TEXT NOT NULL UNIQUE,
    user_id     TEXT NOT NULL,
    card_id     TEXT NOT NULL,
    video_id    TEXT NOT NULL,
    channel_id  TEXT NOT NULL,
    topic       TEXT NOT NULL,
    gesture     TEXT NOT NULL,
    reason      TEXT NOT NULL,
    occurred_at DATETIME NOT NULL
);

CREATE TABLE cold_periods (
    cold_period_id    TEXT PRIMARY KEY,
    user_id           TEXT NOT NULL,
    scope             TEXT NOT NULL,
    target            TEXT NOT NULL,
    reason            TEXT NOT NULL,
    created_at        DATETIME NOT NULL,
    expires_at        DATETIME NOT NULL,
    last_refreshed_at DATETIME NOT NULL,
    UNIQUE (user_id, scope, target)
);

-- +goose Down
DROP TABLE cold_periods;
DROP TABLE signals;
