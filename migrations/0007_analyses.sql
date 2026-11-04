-- +goose Up
CREATE TABLE analyses (
    analysis_id       TEXT PRIMARY KEY,
    user_id           TEXT NOT NULL,
    video_id          TEXT NOT NULL,
    profile_version   INTEGER NOT NULL,
    rubric_version    TEXT NOT NULL,
    model_id          TEXT NOT NULL,
    state             TEXT NOT NULL,
    reason            TEXT NOT NULL,
    topic             TEXT NOT NULL,
    tldr              TEXT NOT NULL,
    marketing_score   REAL NOT NULL,
    ai_slop_score     REAL NOT NULL,
    similarity        TEXT NOT NULL,
    new_territory     INTEGER NOT NULL,
    why_this_card     TEXT NOT NULL,
    transcript_sha256 TEXT NOT NULL,
    transcript_bytes  INTEGER NOT NULL,
    created_at        DATETIME NOT NULL,
    completed_at      DATETIME NOT NULL,
    UNIQUE (user_id, video_id, rubric_version),
    CHECK (state IN ('analyzed', 'failed')),
    CHECK (
        (state = 'analyzed' AND reason = '') OR
        (state = 'failed' AND reason <> '')
    ),
    CHECK (new_territory IN (0, 1))
);

CREATE TABLE analysis_axis_scores (
    analysis_id   TEXT NOT NULL,
    axis_name     TEXT NOT NULL,
    score         REAL NOT NULL,
    justification TEXT NOT NULL,
    position      INTEGER NOT NULL,
    PRIMARY KEY (analysis_id, axis_name),
    FOREIGN KEY (analysis_id) REFERENCES analyses(analysis_id) ON DELETE CASCADE
);

CREATE INDEX analyses_video_completed ON analyses(video_id, completed_at);

-- +goose Down
DROP INDEX analyses_video_completed;
DROP TABLE analysis_axis_scores;
DROP TABLE analyses;
