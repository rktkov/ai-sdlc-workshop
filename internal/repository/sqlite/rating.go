package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/feed"
)

// RatingRepository stores weekly ratings.
type RatingRepository struct {
	db *sql.DB
}

// NewRatingRepository returns a repository over the given database.
func NewRatingRepository(db *sql.DB) *RatingRepository {
	return &RatingRepository{db: db}
}

// FindByWeek returns one rating for a user and week.
func (r *RatingRepository) FindByWeek(ctx context.Context, userID string, weekStart time.Time) (feed.WeeklyRating, error) {
	row := r.db.QueryRowContext(ctx, `SELECT rating_id, user_id, week_start, value, created_at
		FROM weekly_ratings WHERE user_id = ? AND date(week_start) = date(?)`,
		userID, weekStart.UTC().Format("2006-01-02"))
	rating, err := scanRating(row)
	if errors.Is(err, sql.ErrNoRows) {
		return feed.WeeklyRating{}, feed.ErrRatingNotFound
	}
	if err != nil {
		return feed.WeeklyRating{}, err
	}
	return r.withBatches(ctx, rating)
}

// LastForUser returns the newest rating.
func (r *RatingRepository) LastForUser(ctx context.Context, userID string) (feed.WeeklyRating, error) {
	row := r.db.QueryRowContext(ctx, `SELECT rating_id, user_id, week_start, value, created_at
		FROM weekly_ratings WHERE user_id = ? ORDER BY week_start DESC LIMIT 1`, userID)
	rating, err := scanRating(row)
	if errors.Is(err, sql.ErrNoRows) {
		return feed.WeeklyRating{}, feed.ErrRatingNotFound
	}
	if err != nil {
		return feed.WeeklyRating{}, err
	}
	return r.withBatches(ctx, rating)
}

// Save inserts one rating and its batch links.
func (r *RatingRepository) Save(ctx context.Context, rating feed.WeeklyRating) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin rating: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO weekly_ratings
		(rating_id, user_id, week_start, value, created_at) VALUES (?, ?, ?, ?, ?)`,
		rating.ID, rating.UserID, rating.WeekStart.UTC().Format("2006-01-02"),
		string(rating.Value), rating.CreatedAt,
	); err != nil {
		_ = tx.Rollback()
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return feed.ErrRatingExists
		}
		return fmt.Errorf("insert rating: %w", err)
	}
	for _, batchID := range rating.BatchIDs {
		if _, err := tx.ExecContext(ctx, `INSERT INTO rating_batches
			(rating_id, batch_id) VALUES (?, ?)`, rating.ID, batchID,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert rating batch: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit rating: %w", err)
	}
	return nil
}

func (r *RatingRepository) withBatches(ctx context.Context, rating feed.WeeklyRating) (feed.WeeklyRating, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT batch_id FROM rating_batches
		WHERE rating_id = ? ORDER BY batch_id`, rating.ID)
	if err != nil {
		return feed.WeeklyRating{}, fmt.Errorf("read rating batches: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var batchID string
		if err := rows.Scan(&batchID); err != nil {
			return feed.WeeklyRating{}, fmt.Errorf("scan rating batch: %w", err)
		}
		rating.BatchIDs = append(rating.BatchIDs, batchID)
	}
	if err := rows.Err(); err != nil {
		return feed.WeeklyRating{}, fmt.Errorf("iterate rating batches: %w", err)
	}
	return rating, nil
}

func scanRating(row scanner) (feed.WeeklyRating, error) {
	var rating feed.WeeklyRating
	var weekStart, value string
	err := row.Scan(&rating.ID, &rating.UserID, &weekStart, &value, &rating.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return feed.WeeklyRating{}, err
		}
		return feed.WeeklyRating{}, fmt.Errorf("scan rating: %w", err)
	}
	parsed, err := time.Parse("2006-01-02", weekStart)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339, weekStart)
	}
	if err != nil {
		return feed.WeeklyRating{}, fmt.Errorf("parse week start: %w", err)
	}
	rating.WeekStart = parsed
	rating.Value = feed.RatingValue(value)
	return rating, nil
}

var _ feed.RatingRepository = (*RatingRepository)(nil)
