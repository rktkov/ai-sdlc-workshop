package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/rktkov/ytfeed/internal/domain/identity"
	"github.com/rktkov/ytfeed/internal/storage"
)

func TestSubscriptionUpsertHappy(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewSubscriptionRepository(db)
	page := []identity.Subscription{
		{ChannelID: "c1", Title: "One", SubscriberCount: 100, TopicTags: []string{"tech", "science"}},
		{ChannelID: "c2", Title: "Two", SubscriberCount: 200},
	}
	if err := repo.UpsertPage(context.Background(), "owner", page); err != nil {
		t.Fatalf("UpsertPage: %v", err)
	}

	count, err := repo.Count(context.Background(), "owner")
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}

	list, err := repo.List(context.Background(), "owner")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 || list[0].ChannelID != "c1" || list[1].ChannelID != "c2" {
		t.Fatalf("list = %+v", list)
	}
	if len(list[0].TopicTags) != 2 || list[0].TopicTags[0] != "tech" {
		t.Fatalf("tags = %v", list[0].TopicTags)
	}
}

func TestSubscriptionUpsertIdempotent(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewSubscriptionRepository(db)
	first := []identity.Subscription{{ChannelID: "c1", Title: "One", SubscriberCount: 100, TopicTags: []string{"old"}}}
	if err := repo.UpsertPage(context.Background(), "owner", first); err != nil {
		t.Fatalf("UpsertPage first: %v", err)
	}
	second := []identity.Subscription{{ChannelID: "c1", Title: "One updated", SubscriberCount: 150, TopicTags: []string{"new"}}}
	if err := repo.UpsertPage(context.Background(), "owner", second); err != nil {
		t.Fatalf("UpsertPage second: %v", err)
	}

	count, err := repo.Count(context.Background(), "owner")
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	list, err := repo.List(context.Background(), "owner")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if list[0].Title != "One updated" || list[0].SubscriberCount != 150 {
		t.Fatalf("list = %+v", list)
	}
	if len(list[0].TopicTags) != 1 || list[0].TopicTags[0] != "new" {
		t.Fatalf("tags = %v", list[0].TopicTags)
	}
}

func TestSubscriptionEmptyPage(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewSubscriptionRepository(db)
	if err := repo.UpsertPage(context.Background(), "owner", nil); err != nil {
		t.Fatalf("UpsertPage: %v", err)
	}
	if count, _ := repo.Count(context.Background(), "owner"); count != 0 {
		t.Fatalf("count = %d, want 0", count)
	}
}

func TestSubscriptionStoreError(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec("DROP TABLE channels"); err != nil {
		t.Fatalf("drop channels: %v", err)
	}

	repo := NewSubscriptionRepository(db)
	if err := repo.UpsertPage(context.Background(), "owner", []identity.Subscription{{ChannelID: "c1", Title: "One"}}); err == nil {
		t.Fatal("UpsertPage = nil, want an error")
	}
}
