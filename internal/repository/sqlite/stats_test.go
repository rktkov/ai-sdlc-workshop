package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/feed"
	"github.com/rktkov/ytfeed/internal/storage"
)

func TestStats(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	signals := []struct {
		id, gesture, reason, topic string
	}{
		{"s1", "left", "marketing", "ai-agents"},
		{"s2", "left", "ai-slop", "local-llms"},
		{"s3", "right", "", "ai-agents"},
		{"s4", "save", "", "ai-agents"},
	}
	for _, signal := range signals {
		if _, err := db.Exec(`INSERT INTO signals
			(signal_id, gesture_id, user_id, card_id, video_id, channel_id, topic, gesture, reason, occurred_at)
			VALUES (?, ?, 'owner', 'c', 'v', 'ch', ?, ?, ?, ?)`,
			signal.id, "g-"+signal.id, signal.topic, signal.gesture, signal.reason, now,
		); err != nil {
			t.Fatalf("insert signal: %v", err)
		}
	}
	if _, err := db.Exec(`INSERT INTO cold_periods
		(cold_period_id, user_id, scope, target, reason, created_at, expires_at, last_refreshed_at)
		VALUES ('c1', 'owner', 'channel', 'ch', 'marketing', ?, ?, ?)`,
		now.Add(-time.Hour), now.Add(time.Hour), now,
	); err != nil {
		t.Fatalf("insert active cold period: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO cold_periods
		(cold_period_id, user_id, scope, target, reason, created_at, expires_at, last_refreshed_at)
		VALUES ('c2', 'owner', 'topic', 'ai', 'other', ?, ?, ?)`,
		now.Add(-48*time.Hour), now.Add(-24*time.Hour), now.Add(-48*time.Hour),
	); err != nil {
		t.Fatalf("insert expired cold period: %v", err)
	}
	for _, point := range []struct {
		day     string
		charged int
	}{{"2026-11-04", 50}, {"2026-11-05", 100}} {
		if _, err := db.Exec(`INSERT INTO quota_ledgers
			(user_id, accounting_day, budget, reserved, charged, warned_at_80, exhausted, version, updated_at)
			VALUES ('owner', ?, 10000, 0, ?, 0, 0, 0, ?)`, point.day, point.charged, now); err != nil {
			t.Fatalf("insert quota: %v", err)
		}
	}

	service := &feed.StatsService{Repo: NewStatsRepository(db), Now: func() time.Time { return now }}
	stats, err := service.Get(context.Background(), "owner", 7)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if stats.Swipes["left"] != 2 || stats.Swipes["right"] != 1 || stats.Swipes["save"] != 1 {
		t.Fatalf("swipes = %v", stats.Swipes)
	}
	if stats.Reasons["marketing"] != 1 || stats.Reasons["ai-slop"] != 1 {
		t.Fatalf("reasons = %v", stats.Reasons)
	}
	if stats.ColdActive != 1 || stats.ColdExpired != 1 {
		t.Fatalf("cold = %d/%d", stats.ColdActive, stats.ColdExpired)
	}
	if len(stats.Quota) != 2 || stats.Quota[1].Charged != 100 {
		t.Fatalf("quota = %+v", stats.Quota)
	}
	if stats.Topics["ai-agents"] != 3 {
		t.Fatalf("topics = %v", stats.Topics)
	}
}

func TestStatsError(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec("DROP TABLE signals"); err != nil {
		t.Fatalf("drop signals: %v", err)
	}
	service := &feed.StatsService{Repo: NewStatsRepository(db)}
	if _, err := service.Get(context.Background(), "owner", 7); err == nil {
		t.Fatal("Get = nil, want an error")
	}
}
