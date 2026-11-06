package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/feed"
)

// BatchRepository stores composed batches as one JSON snapshot.
type BatchRepository struct {
	db *sql.DB
}

// NewBatchRepository returns a repository over the given database.
func NewBatchRepository(db *sql.DB) *BatchRepository {
	return &BatchRepository{db: db}
}

// Save inserts or replaces one batch.
func (r *BatchRepository) Save(ctx context.Context, batch feed.Batch) error {
	if err := batch.Validate(); err != nil {
		return err
	}
	payload, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("encode batch: %w", err)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin batch save: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO batches
		(batch_id, user_id, state, degradation_reason, payload, created_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(batch_id) DO UPDATE SET
			state = excluded.state,
			degradation_reason = excluded.degradation_reason,
			payload = excluded.payload,
			completed_at = excluded.completed_at`,
		batch.ID, batch.UserID, string(batch.State), string(batch.DegradationReason),
		string(payload), batch.CreatedAt, nullableTime(batch.CompletedAt),
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("save batch: %w", err)
	}
	now := time.Now().UTC()
	for _, card := range batch.Cards {
		if card.VideoID == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO shown_videos
			(user_id, video_id, batch_id, shown_at) VALUES (?, ?, ?, ?)`,
			batch.UserID, card.VideoID, batch.ID, now,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("mark shown video: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM shown_videos WHERE shown_at < ?`,
		now.Add(-90*24*time.Hour),
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prune shown videos: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit batch save: %w", err)
	}
	return nil
}

// Get returns one batch by ID.
func (r *BatchRepository) Get(ctx context.Context, batchID string) (feed.Batch, error) {
	var payload string
	err := r.db.QueryRowContext(ctx, `SELECT payload FROM batches WHERE batch_id = ?`, batchID).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return feed.Batch{}, feed.ErrBatchNotFound
	}
	if err != nil {
		return feed.Batch{}, fmt.Errorf("read batch: %w", err)
	}
	return decodeBatch(payload)
}

// LatestCompleted returns the newest completed batch.
func (r *BatchRepository) LatestCompleted(ctx context.Context, userID string) (feed.Batch, error) {
	var payload string
	err := r.db.QueryRowContext(ctx, `SELECT payload FROM batches
		WHERE user_id = ? AND state = 'completed'
		ORDER BY completed_at DESC LIMIT 1`, userID).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return feed.Batch{}, feed.ErrNoCachedFeed
	}
	if err != nil {
		return feed.Batch{}, fmt.Errorf("read cached feed: %w", err)
	}
	return decodeBatch(payload)
}

func decodeBatch(payload string) (feed.Batch, error) {
	var batch feed.Batch
	if err := json.Unmarshal([]byte(payload), &batch); err != nil {
		return feed.Batch{}, fmt.Errorf("decode batch: %w", err)
	}
	return batch, nil
}

var _ interface {
	Save(ctx context.Context, batch feed.Batch) error
	Get(ctx context.Context, batchID string) (feed.Batch, error)
	LatestCompleted(ctx context.Context, userID string) (feed.Batch, error)
} = (*BatchRepository)(nil)
