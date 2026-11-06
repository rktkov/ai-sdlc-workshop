package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/feed"
	"github.com/rktkov/ytfeed/internal/storage"
)

func TestExclusionsForUser(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	for _, signal := range []struct {
		id, video, gesture string
	}{
		{"s1", "v1", "right"},
		{"s2", "v2", "save"},
		{"s3", "v3", "left"},
	} {
		if _, err := db.Exec(`INSERT INTO signals
			(signal_id, gesture_id, user_id, card_id, video_id, channel_id, topic, gesture, reason, occurred_at)
			VALUES (?, ?, 'owner', 'c', ?, 'ch', 'ai', ?, '', ?)`,
			signal.id, "g-"+signal.id, signal.video, signal.gesture, now,
		); err != nil {
			t.Fatalf("insert signal: %v", err)
		}
	}
	if _, err := db.Exec(`INSERT INTO library_entries
		(user_id, video_id, state, title, channel_id, duration_seconds,
		 file_extension, file_bytes, saved_at, downloaded_at, updated_at)
		VALUES ('owner', 'v2', 'saved', 'Title', 'ch', 0, '', 0, ?, NULL, ?)`,
		now, now,
	); err != nil {
		t.Fatalf("insert library: %v", err)
	}
	batch := feed.Batch{
		ID: "b1", UserID: "owner", Size: 1, State: feed.BatchCompleted,
		Cards:     []feed.Card{{CardID: "c4", VideoID: "v4", Title: "t", Channel: "ch", State: feed.StatePending, Position: 0}},
		CreatedAt: now, CompletedAt: now,
	}
	if err := NewBatchRepository(db).Save(context.Background(), batch); err != nil {
		t.Fatalf("save batch: %v", err)
	}

	set, err := NewExclusionRepository(db).ForUser(context.Background(), "owner", now.AddDate(0, 0, -7))
	if err != nil {
		t.Fatalf("ForUser: %v", err)
	}
	if !set.Watched["v1"] || !set.Watched["v2"] || set.Watched["v3"] {
		t.Fatalf("watched = %v", set.Watched)
	}
	if !set.Saved["v2"] {
		t.Fatalf("saved = %v", set.Saved)
	}
	if !set.Shown7d["v4"] {
		t.Fatalf("shown = %v", set.Shown7d)
	}
}

func TestExclusionsError(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec("DROP TABLE signals"); err != nil {
		t.Fatalf("drop signals: %v", err)
	}
	if _, err := NewExclusionRepository(db).ForUser(context.Background(), "owner", time.Now()); err == nil {
		t.Fatal("ForUser = nil, want an error")
	}
}
