-- +goose Up
CREATE TABLE taste_profiles (
    user_id        TEXT PRIMARY KEY,
    note           TEXT NOT NULL,
    rubric_version TEXT NOT NULL,
    created_at     DATETIME NOT NULL
);

CREATE TABLE taste_axes (
    user_id  TEXT NOT NULL,
    name     TEXT NOT NULL,
    weight   REAL NOT NULL,
    position INTEGER NOT NULL,
    PRIMARY KEY (user_id, name),
    FOREIGN KEY (user_id) REFERENCES taste_profiles(user_id) ON DELETE CASCADE
);

CREATE TABLE taste_examples (
    example_id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    TEXT NOT NULL,
    axis_name  TEXT NOT NULL,
    polarity   TEXT NOT NULL,
    text       TEXT NOT NULL,
    position   INTEGER NOT NULL,
    FOREIGN KEY (user_id, axis_name) REFERENCES taste_axes(user_id, name) ON DELETE CASCADE
);

CREATE TABLE taste_interests (
    user_id  TEXT NOT NULL,
    slug     TEXT NOT NULL,
    position INTEGER NOT NULL,
    PRIMARY KEY (user_id, slug),
    FOREIGN KEY (user_id) REFERENCES taste_profiles(user_id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE taste_interests;
DROP TABLE taste_examples;
DROP TABLE taste_axes;
DROP TABLE taste_profiles;
