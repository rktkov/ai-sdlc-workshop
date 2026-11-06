package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/discovery"
)

// ExclusionRepository builds the discovery exclusion set from local history.
type ExclusionRepository struct {
	db *sql.DB
}

// NewExclusionRepository returns an exclusion reader.
func NewExclusionRepository(db *sql.DB) *ExclusionRepository {
	return &ExclusionRepository{db: db}
}

// ForUser returns watched, saved and recently shown video IDs.
func (r *ExclusionRepository) ForUser(ctx context.Context, userID string, since time.Time) (discovery.ExclusionSet, error) {
	watched, err := r.ids(ctx, `SELECT DISTINCT video_id FROM signals
		WHERE user_id = ? AND gesture IN ('right', 'save')`, userID)
	if err != nil {
		return discovery.ExclusionSet{}, err
	}
	saved, err := r.ids(ctx, `SELECT video_id FROM library_entries WHERE user_id = ?`, userID)
	if err != nil {
		return discovery.ExclusionSet{}, err
	}
	shown, err := r.shown(ctx, userID, since)
	if err != nil {
		return discovery.ExclusionSet{}, err
	}
	return discovery.ExclusionSet{Watched: watched, Saved: saved, Shown7d: shown}, nil
}

func (r *ExclusionRepository) ids(ctx context.Context, query, userID string) (map[string]bool, error) {
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("read exclusion ids: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan exclusion id: %w", err)
		}
		out[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate exclusion ids: %w", err)
	}
	return out, nil
}

func (r *ExclusionRepository) shown(ctx context.Context, userID string, since time.Time) (map[string]bool, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT video_id FROM shown_videos
		WHERE user_id = ? AND shown_at >= ?`, userID, since)
	if err != nil {
		return nil, fmt.Errorf("read shown videos: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]bool{}
	for rows.Next() {
		var videoID string
		if err := rows.Scan(&videoID); err != nil {
			return nil, fmt.Errorf("scan shown video: %w", err)
		}
		out[videoID] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate shown videos: %w", err)
	}
	return out, nil
}

var _ interface {
	ForUser(ctx context.Context, userID string, since time.Time) (discovery.ExclusionSet, error)
} = (*ExclusionRepository)(nil)
