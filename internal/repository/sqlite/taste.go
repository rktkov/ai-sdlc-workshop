// Package sqlite implements repositories on SQLite.
package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

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
func (r *TasteRepository) RecordGesture(ctx context.Context, signal taste.Signal, cold *taste.ColdPeriod) (bool, error) {
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
