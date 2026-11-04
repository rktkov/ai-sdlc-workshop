-- +goose Up
CREATE TABLE library_entries (
    user_id          TEXT NOT NULL,
    video_id         TEXT NOT NULL,
    state            TEXT NOT NULL,
    title            TEXT NOT NULL,
    channel_id       TEXT NOT NULL,
    duration_seconds INTEGER NOT NULL,
    file_extension   TEXT NOT NULL,
    file_bytes       INTEGER NOT NULL,
    saved_at         DATETIME NOT NULL,
    downloaded_at    DATETIME,
    updated_at       DATETIME NOT NULL,
    PRIMARY KEY (user_id, video_id),
    CHECK (state IN ('saved', 'downloaded')),
    CHECK (duration_seconds >= 0),
    CHECK (state = 'saved' OR file_bytes > 0)
);

CREATE INDEX library_entries_saved ON library_entries(user_id, saved_at);

-- +goose Down
DROP INDEX library_entries_saved;
DROP TABLE library_entries;
