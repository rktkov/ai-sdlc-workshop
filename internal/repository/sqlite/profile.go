package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/taste"
)

// ProfileRepository stores the taste profile.
type ProfileRepository struct {
	db *sql.DB
}

// NewProfileRepository returns a repository over the given database.
func NewProfileRepository(db *sql.DB) *ProfileRepository {
	return &ProfileRepository{db: db}
}

// Save inserts one profile and its axes, examples and interests.
func (r *ProfileRepository) Save(ctx context.Context, profile taste.Profile) error {
	if err := profile.Validate(); err != nil {
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin profile: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO taste_profiles
		(user_id, note, rubric_version, created_at) VALUES (?, ?, ?, ?)`,
		profile.UserID, profile.Note, profile.RubricVersion, time.Now().UTC(),
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("insert profile: %w", err)
	}

	for pos, axis := range profile.Axes {
		if _, err := tx.ExecContext(ctx, `INSERT INTO taste_axes
			(user_id, name, weight, position) VALUES (?, ?, ?, ?)`,
			profile.UserID, axis.Name, axis.Weight, pos,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert axis %s: %w", axis.Name, err)
		}
		for i, text := range axis.PositiveExamples {
			if err := insertExample(ctx, tx, profile.UserID, axis.Name, "positive", text, i); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
		for i, text := range axis.NegativeExamples {
			if err := insertExample(ctx, tx, profile.UserID, axis.Name, "negative", text, i); err != nil {
				_ = tx.Rollback()
				return err
			}
		}
	}

	for pos, interest := range profile.Interests {
		if _, err := tx.ExecContext(ctx, `INSERT INTO taste_interests
			(user_id, slug, position) VALUES (?, ?, ?)`,
			profile.UserID, interest, pos,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert interest %s: %w", interest, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit profile: %w", err)
	}
	return nil
}

func insertExample(ctx context.Context, tx *sql.Tx, userID, axis, polarity, text string, position int) error {
	if _, err := tx.ExecContext(ctx, `INSERT INTO taste_examples
		(user_id, axis_name, polarity, text, position) VALUES (?, ?, ?, ?, ?)`,
		userID, axis, polarity, text, position,
	); err != nil {
		return fmt.Errorf("insert example for %s: %w", axis, err)
	}
	return nil
}

// Get returns one profile with its axes, examples and interests.
func (r *ProfileRepository) Get(ctx context.Context, userID string) (taste.Profile, error) {
	var profile taste.Profile
	err := r.db.QueryRowContext(ctx, `SELECT user_id, note, rubric_version
		FROM taste_profiles WHERE user_id = ?`, userID,
	).Scan(&profile.UserID, &profile.Note, &profile.RubricVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return taste.Profile{}, taste.ErrProfileNotFound
	}
	if err != nil {
		return taste.Profile{}, fmt.Errorf("read profile: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `SELECT name, weight FROM taste_axes
		WHERE user_id = ? ORDER BY position`, userID)
	if err != nil {
		return taste.Profile{}, fmt.Errorf("read axes: %w", err)
	}
	for rows.Next() {
		var axis taste.Axis
		if err := rows.Scan(&axis.Name, &axis.Weight); err != nil {
			_ = rows.Close()
			return taste.Profile{}, fmt.Errorf("scan axis: %w", err)
		}
		profile.Axes = append(profile.Axes, axis)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return taste.Profile{}, fmt.Errorf("iterate axes: %w", err)
	}
	if err := rows.Close(); err != nil {
		return taste.Profile{}, fmt.Errorf("close axes: %w", err)
	}
	for i := range profile.Axes {
		if err := r.fillExamples(ctx, userID, &profile.Axes[i]); err != nil {
			return taste.Profile{}, err
		}
	}

	interests, err := r.db.QueryContext(ctx, `SELECT slug FROM taste_interests
		WHERE user_id = ? ORDER BY position`, userID)
	if err != nil {
		return taste.Profile{}, fmt.Errorf("read interests: %w", err)
	}
	defer func() { _ = interests.Close() }()
	for interests.Next() {
		var slug string
		if err := interests.Scan(&slug); err != nil {
			return taste.Profile{}, fmt.Errorf("scan interest: %w", err)
		}
		profile.Interests = append(profile.Interests, slug)
	}
	if err := interests.Err(); err != nil {
		return taste.Profile{}, fmt.Errorf("iterate interests: %w", err)
	}
	return profile, nil
}

func (r *ProfileRepository) fillExamples(ctx context.Context, userID string, axis *taste.Axis) error {
	rows, err := r.db.QueryContext(ctx, `SELECT polarity, text FROM taste_examples
		WHERE user_id = ? AND axis_name = ? ORDER BY polarity, position`, userID, axis.Name)
	if err != nil {
		return fmt.Errorf("read examples for %s: %w", axis.Name, err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var polarity, text string
		if err := rows.Scan(&polarity, &text); err != nil {
			return fmt.Errorf("scan example: %w", err)
		}
		if polarity == "positive" {
			axis.PositiveExamples = append(axis.PositiveExamples, text)
		} else {
			axis.NegativeExamples = append(axis.NegativeExamples, text)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate examples: %w", err)
	}
	return nil
}

// SetInterests replaces the interest set for an existing profile.
func (r *ProfileRepository) SetInterests(ctx context.Context, userID string, interests []string) error {
	normalized, err := taste.NormalizeInterests(interests)
	if err != nil {
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin interests: %w", err)
	}

	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT 1 FROM taste_profiles WHERE user_id = ?`, userID).Scan(&exists); err != nil {
		_ = tx.Rollback()
		if errors.Is(err, sql.ErrNoRows) {
			return taste.ErrProfileNotFound
		}
		return fmt.Errorf("find profile: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM taste_interests WHERE user_id = ?`, userID); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("clear interests: %w", err)
	}
	for position, slug := range normalized {
		if _, err := tx.ExecContext(ctx, `INSERT INTO taste_interests
			(user_id, slug, position) VALUES (?, ?, ?)`, userID, slug, position,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert interest %s: %w", slug, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit interests: %w", err)
	}
	return nil
}
