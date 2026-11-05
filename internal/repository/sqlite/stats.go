package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/feed"
)

// StatsRepository reads feed facts for the stats projection.
type StatsRepository struct {
	db *sql.DB
}

// NewStatsRepository returns a repository over the given database.
func NewStatsRepository(db *sql.DB) *StatsRepository {
	return &StatsRepository{db: db}
}

// Swipes returns gesture counts.
func (r *StatsRepository) Swipes(ctx context.Context, userID string) (map[string]int, error) {
	return r.countBy(ctx, `SELECT gesture, count(*) FROM signals WHERE user_id = ? GROUP BY gesture`, userID)
}

// Reasons returns rejection reason counts.
func (r *StatsRepository) Reasons(ctx context.Context, userID string) (map[string]int, error) {
	return r.countBy(ctx, `SELECT reason, count(*) FROM signals WHERE user_id = ? AND reason <> '' GROUP BY reason`, userID)
}

// ColdPeriods returns active and expired counts at one instant.
func (r *StatsRepository) ColdPeriods(ctx context.Context, userID string, now time.Time) (int, int, error) {
	var active, expired int
	if err := r.db.QueryRowContext(ctx, `SELECT
		COALESCE(SUM(CASE WHEN expires_at > ? THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN expires_at <= ? THEN 1 ELSE 0 END), 0)
		FROM cold_periods WHERE user_id = ?`, now, now, userID,
	).Scan(&active, &expired); err != nil {
		return 0, 0, fmt.Errorf("count cold periods: %w", err)
	}
	return active, expired, nil
}

// QuotaTrend returns charged units per accounting day.
func (r *StatsRepository) QuotaTrend(ctx context.Context, userID string, since time.Time) ([]feed.QuotaPoint, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT accounting_day, charged
		FROM quota_ledgers WHERE user_id = ? AND accounting_day >= ?
		ORDER BY accounting_day`, userID, since.UTC().Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("read quota trend: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []feed.QuotaPoint
	for rows.Next() {
		var point feed.QuotaPoint
		if err := rows.Scan(&point.Day, &point.Charged); err != nil {
			return nil, fmt.Errorf("scan quota trend: %w", err)
		}
		out = append(out, point)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate quota trend: %w", err)
	}
	return out, nil
}

// Topics returns the most frequent topics.
func (r *StatsRepository) Topics(ctx context.Context, userID string) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT topic, count(*) AS n FROM signals
		WHERE user_id = ? AND topic <> '' GROUP BY topic ORDER BY n DESC, topic LIMIT 10`, userID)
	if err != nil {
		return nil, fmt.Errorf("read topics: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := map[string]int{}
	for rows.Next() {
		var topic string
		var count int
		if err := rows.Scan(&topic, &count); err != nil {
			return nil, fmt.Errorf("scan topic: %w", err)
		}
		out[topic] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate topics: %w", err)
	}
	return out, nil
}

func (r *StatsRepository) countBy(ctx context.Context, query, userID string) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("read counts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := map[string]int{}
	for rows.Next() {
		var key string
		var count int
		if err := rows.Scan(&key, &count); err != nil {
			return nil, fmt.Errorf("scan count: %w", err)
		}
		out[key] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate counts: %w", err)
	}
	return out, nil
}

var _ feed.StatsRepository = (*StatsRepository)(nil)
