// Package sqlite implements repositories on SQLite.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/taste"
)

// TasteRepository stores signals and cold periods.
type TasteRepository struct {
	db *sql.DB
}

// NewTasteRepository returns a repository over the given database.
func NewTasteRepository(db *sql.DB) *TasteRepository {
	return &TasteRepository{db: db}
}

// RecordGesture stores a signal and an optional cold period in one transaction.
// It returns false when the gesture already exists.
func (r *TasteRepository) RecordGesture(ctx context.Context, signal taste.Signal, cold *taste.ColdPeriod, updates []taste.WeightUpdate) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin gesture: %w", err)
	}

	var existing string
	err = tx.QueryRowContext(ctx, "SELECT signal_id FROM signals WHERE gesture_id = ?", signal.GestureID).Scan(&existing)
	if err == nil {
		if commitErr := tx.Commit(); commitErr != nil {
			return false, fmt.Errorf("commit duplicate gesture: %w", commitErr)
		}
		return false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		_ = tx.Rollback()
		return false, fmt.Errorf("find gesture: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO signals
		(signal_id, gesture_id, user_id, card_id, video_id, channel_id, topic, gesture, reason, occurred_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		signal.SignalID, signal.GestureID, signal.UserID, signal.CardID, signal.VideoID,
		signal.ChannelID, signal.Topic, string(signal.Gesture), string(signal.Reason), signal.OccurredAt,
	); err != nil {
		_ = tx.Rollback()
		return false, fmt.Errorf("insert signal: %w", err)
	}

	for _, score := range signal.ScoredAxes {
		if _, err := tx.ExecContext(ctx, `INSERT INTO signal_axis_scores
			(signal_id, axis_name, score) VALUES (?, ?, ?)`,
			signal.SignalID, score.Axis, score.Score,
		); err != nil {
			_ = tx.Rollback()
			return false, fmt.Errorf("insert score %s: %w", score.Axis, err)
		}
	}

	for _, update := range updates {
		result, err := tx.ExecContext(ctx, `UPDATE taste_axes SET weight = ?
			WHERE user_id = ? AND name = ?`, update.Weight, signal.UserID, update.Axis)
		if err != nil {
			_ = tx.Rollback()
			return false, fmt.Errorf("update axis %s: %w", update.Axis, err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			_ = tx.Rollback()
			return false, fmt.Errorf("check axis %s: %w", update.Axis, err)
		}
		if rows == 0 {
			_ = tx.Rollback()
			return false, fmt.Errorf("axis %s is not in the profile", update.Axis)
		}
	}

	if cold != nil {
		if _, err := tx.ExecContext(ctx, `INSERT INTO cold_periods
			(cold_period_id, user_id, scope, target, reason, created_at, expires_at, last_refreshed_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(user_id, scope, target) DO UPDATE SET
				expires_at = excluded.expires_at,
				last_refreshed_at = excluded.last_refreshed_at`,
			cold.ColdPeriodID, cold.UserID, string(cold.Scope), cold.Target, string(cold.Reason),
			cold.CreatedAt, cold.ExpiresAt, cold.LastRefreshedAt,
		); err != nil {
			_ = tx.Rollback()
			return false, fmt.Errorf("upsert cold period: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit gesture: %w", err)
	}
	return true, nil
}

// ListActiveColdPeriods returns the cold periods that have not expired.
func (r *TasteRepository) ListActiveColdPeriods(ctx context.Context, userID string) ([]taste.ColdPeriod, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT cold_period_id, user_id, scope, target, reason,
		created_at, expires_at, last_refreshed_at
		FROM cold_periods WHERE user_id = ? AND expires_at > ?
		ORDER BY expires_at`, userID, time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("list cold periods: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var periods []taste.ColdPeriod
	for rows.Next() {
		var cold taste.ColdPeriod
		var reason, scope string
		if err := rows.Scan(&cold.ColdPeriodID, &cold.UserID, &scope, &cold.Target, &reason,
			&cold.CreatedAt, &cold.ExpiresAt, &cold.LastRefreshedAt); err != nil {
			return nil, fmt.Errorf("scan cold period: %w", err)
		}
		cold.Scope = taste.Scope(scope)
		cold.Reason = taste.Reason(reason)
		periods = append(periods, cold)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cold periods: %w", err)
	}
	return periods, nil
}

// FindColdPeriod returns one cold period by target.
func (r *TasteRepository) FindColdPeriod(ctx context.Context, userID string, scope taste.Scope, target string) (taste.ColdPeriod, error) {
	var cold taste.ColdPeriod
	var reason, storedScope string
	err := r.db.QueryRowContext(ctx, `SELECT cold_period_id, user_id, scope, target, reason,
		created_at, expires_at, last_refreshed_at
		FROM cold_periods WHERE user_id = ? AND scope = ? AND target = ?`,
		userID, string(scope), target,
	).Scan(&cold.ColdPeriodID, &cold.UserID, &storedScope, &cold.Target, &reason,
		&cold.CreatedAt, &cold.ExpiresAt, &cold.LastRefreshedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return taste.ColdPeriod{}, taste.ErrColdPeriodNotFound
	}
	if err != nil {
		return taste.ColdPeriod{}, fmt.Errorf("find cold period: %w", err)
	}
	cold.Scope = taste.Scope(storedScope)
	cold.Reason = taste.Reason(reason)
	return cold, nil
}
