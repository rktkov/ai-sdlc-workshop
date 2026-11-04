-- +goose Up
CREATE TABLE signal_axis_scores (
    signal_id TEXT NOT NULL,
    axis_name TEXT NOT NULL,
    score     REAL NOT NULL,
    PRIMARY KEY (signal_id, axis_name),
    FOREIGN KEY (signal_id) REFERENCES signals(signal_id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE signal_axis_scores;
