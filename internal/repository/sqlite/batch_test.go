package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/feed"
	"github.com/rktkov/ytfeed/internal/storage"
)

func batchFixture() feed.Batch {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	return feed.Batch{
		ID: "b1", UserID: "owner", Size: 1, State: feed.BatchCompleted,
		CreatedAt: now, CompletedAt: now,
		Cards: []feed.Card{
			{
				CardID: "c1", VideoID: "v1", ChannelID: "ch1", Title: "Title", Channel: "Channel",
				State: feed.StateAnalyzed, AnalysisID: "a1", Score: 0.8, Position: 0,
				Decision: feed.DecisionNone, Topic: "ai", TLDR: "tldr", WhyThisCard: "why",
			},
		},
	}
}

func TestBatchRoundTrip(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewBatchRepository(db)
	if err := repo.Save(context.Background(), batchFixture()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := repo.Get(context.Background(), "b1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != "b1" || len(got.Cards) != 1 || got.Cards[0].VideoID != "v1" {
		t.Fatalf("batch = %+v", got)
	}
	cached, err := repo.LatestCompleted(context.Background(), "owner")
	if err != nil {
		t.Fatalf("LatestCompleted: %v", err)
	}
	if cached.ID != "b1" {
		t.Fatalf("cached = %+v", cached)
	}
}

func TestBatchNoCached(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewBatchRepository(db)
	if _, err := repo.Get(context.Background(), "missing"); !errors.Is(err, feed.ErrBatchNotFound) {
		t.Fatalf("Get = %v, want ErrBatchNotFound", err)
	}
	if _, err := repo.LatestCompleted(context.Background(), "owner"); !errors.Is(err, feed.ErrNoCachedFeed) {
		t.Fatalf("LatestCompleted = %v, want ErrNoCachedFeed", err)
	}
}

func TestBatchInvalid(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	invalid := batchFixture()
	invalid.State = "bad"
	if err := NewBatchRepository(db).Save(context.Background(), invalid); err == nil {
		t.Fatal("Save(invalid) = nil, want an error")
	}
}

func TestBatchCorruptPayload(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(`INSERT INTO batches
		(batch_id, user_id, state, degradation_reason, payload, created_at, completed_at)
		VALUES ('b', 'owner', 'completed', '', '{', ?, ?)`, time.Now(), time.Now()); err != nil {
		t.Fatalf("insert corrupt: %v", err)
	}
	if _, err := NewBatchRepository(db).Get(context.Background(), "b"); err == nil {
		t.Fatal("Get(corrupt) = nil, want an error")
	}
}
