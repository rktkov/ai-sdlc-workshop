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

func ratingFixture() feed.WeeklyRating {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	return feed.WeeklyRating{
		ID: "r1", UserID: "owner", WeekStart: feed.WeekStart(now),
		Value: feed.RatingBetter, BatchIDs: []string{"b1", "b2"}, CreatedAt: now,
	}
}

func TestRatingRoundTrip(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewRatingRepository(db)
	want := ratingFixture()
	if err := repo.Save(context.Background(), want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := repo.FindByWeek(context.Background(), "owner", want.WeekStart)
	if err != nil {
		t.Fatalf("FindByWeek: %v", err)
	}
	if got.Value != want.Value || len(got.BatchIDs) != 2 {
		t.Fatalf("rating = %+v", got)
	}
	last, err := repo.LastForUser(context.Background(), "owner")
	if err != nil {
		t.Fatalf("LastForUser: %v", err)
	}
	if last.ID != "r1" {
		t.Fatalf("last = %+v", last)
	}
}

func TestRatingDuplicate(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewRatingRepository(db)
	if err := repo.Save(context.Background(), ratingFixture()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	duplicate := ratingFixture()
	duplicate.ID = "r2"
	if err := repo.Save(context.Background(), duplicate); !errors.Is(err, feed.ErrRatingExists) {
		t.Fatalf("Save = %v, want ErrRatingExists", err)
	}
}

func TestRatingNotFound(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewRatingRepository(db)
	if _, err := repo.FindByWeek(context.Background(), "owner", time.Now()); !errors.Is(err, feed.ErrRatingNotFound) {
		t.Fatalf("FindByWeek = %v, want ErrRatingNotFound", err)
	}
	if _, err := repo.LastForUser(context.Background(), "owner"); !errors.Is(err, feed.ErrRatingNotFound) {
		t.Fatalf("LastForUser = %v, want ErrRatingNotFound", err)
	}
}

func TestRatingStoreError(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec("DROP TABLE rating_batches"); err != nil {
		t.Fatalf("drop rating_batches: %v", err)
	}
	if err := NewRatingRepository(db).Save(context.Background(), ratingFixture()); err == nil {
		t.Fatal("Save = nil, want an error")
	}
}
