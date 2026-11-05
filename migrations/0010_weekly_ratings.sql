-- +goose Up
CREATE TABLE weekly_ratings (
    rating_id  TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL,
    week_start DATE NOT NULL,
    value      TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    UNIQUE (user_id, week_start),
    CHECK (value IN ('-1', '0', '1', 'unrated'))
);

CREATE TABLE rating_batches (
    rating_id TEXT NOT NULL,
    batch_id  TEXT NOT NULL,
    PRIMARY KEY (rating_id, batch_id),
    FOREIGN KEY (rating_id) REFERENCES weekly_ratings(rating_id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE rating_batches;
DROP TABLE weekly_ratings;
