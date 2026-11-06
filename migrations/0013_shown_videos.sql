-- +goose Up
CREATE TABLE shown_videos (
    user_id  TEXT NOT NULL,
    video_id TEXT NOT NULL,
    batch_id TEXT NOT NULL,
    shown_at DATETIME NOT NULL,
    PRIMARY KEY (user_id, video_id)
);

CREATE INDEX shown_videos_user_time ON shown_videos(user_id, shown_at);

INSERT OR IGNORE INTO shown_videos (user_id, video_id, batch_id, shown_at)
SELECT b.user_id,
       json_extract(c.value, '$.VideoID'),
       b.batch_id,
       COALESCE(b.completed_at, b.created_at)
FROM batches b, json_each(b.payload, '$.Cards') c
WHERE json_extract(c.value, '$.VideoID') IS NOT NULL;

-- +goose Down
DROP INDEX shown_videos_user_time;
DROP TABLE shown_videos;
