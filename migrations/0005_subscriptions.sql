-- +goose Up
CREATE TABLE channels (
    user_id          TEXT NOT NULL,
    channel_id       TEXT NOT NULL,
    title            TEXT NOT NULL,
    subscriber_count INTEGER NOT NULL,
    imported_at      DATETIME NOT NULL,
    PRIMARY KEY (user_id, channel_id)
);

CREATE TABLE channel_topic_tags (
    user_id    TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    tag        TEXT NOT NULL,
    position   INTEGER NOT NULL,
    PRIMARY KEY (user_id, channel_id, tag),
    FOREIGN KEY (user_id, channel_id) REFERENCES channels(user_id, channel_id) ON DELETE CASCADE
);

CREATE INDEX channels_user_id ON channels(user_id);

-- +goose Down
DROP INDEX channels_user_id;
DROP TABLE channel_topic_tags;
DROP TABLE channels;
