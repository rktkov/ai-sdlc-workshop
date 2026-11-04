package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/library"
	"github.com/rktkov/ytfeed/internal/storage"
)

func libraryFixture() library.Entry {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	return library.Entry{
		UserID: "owner", VideoID: "dQw4w9WgXcQ", State: library.StateSaved,
		Title: "Title", ChannelID: "ch1", DurationSeconds: 90,
		SavedAt: now, UpdatedAt: now,
	}
}

func TestLibraryRoundTrip(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewLibraryRepository(db)
	entry := libraryFixture()
	if err := repo.Save(context.Background(), entry); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := repo.Get(context.Background(), "owner", "dQw4w9WgXcQ")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "Title" || got.State != library.StateSaved {
		t.Fatalf("entry = %+v", got)
	}
	list, err := repo.List(context.Background(), "owner")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list = %+v", list)
	}
}

func TestLibraryDuplicate(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewLibraryRepository(db)
	if err := repo.Save(context.Background(), libraryFixture()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := repo.Save(context.Background(), libraryFixture()); !errors.Is(err, library.ErrExists) {
		t.Fatalf("Save duplicate = %v, want ErrExists", err)
	}
}

func TestLibraryMarkDownloaded(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewLibraryRepository(db)
	if err := repo.Save(context.Background(), libraryFixture()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	now := time.Now().UTC()
	entry, err := repo.MarkDownloaded(context.Background(), "owner", "dQw4w9WgXcQ", "mp4", 42, now)
	if err != nil {
		t.Fatalf("MarkDownloaded: %v", err)
	}
	if entry.State != library.StateDownloaded || entry.FileExtension != "mp4" || entry.FileBytes != 42 {
		t.Fatalf("entry = %+v", entry)
	}
}

func TestLibraryRemove(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewLibraryRepository(db)
	if err := repo.Save(context.Background(), libraryFixture()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := repo.Remove(context.Background(), "owner", "dQw4w9WgXcQ"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := repo.Get(context.Background(), "owner", "dQw4w9WgXcQ"); !errors.Is(err, library.ErrNotFound) {
		t.Fatalf("Get = %v, want ErrNotFound", err)
	}
	if err := repo.Remove(context.Background(), "owner", "dQw4w9WgXcQ"); !errors.Is(err, library.ErrNotFound) {
		t.Fatalf("Remove missing = %v, want ErrNotFound", err)
	}
	if _, err := repo.MarkDownloaded(context.Background(), "owner", "missing", "mp4", 1, time.Now()); !errors.Is(err, library.ErrNotFound) {
		t.Fatalf("MarkDownloaded missing = %v, want ErrNotFound", err)
	}
}

func TestLibraryStoreError(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec("DROP TABLE library_entries"); err != nil {
		t.Fatalf("drop library: %v", err)
	}
	repo := NewLibraryRepository(db)
	if err := repo.Save(context.Background(), libraryFixture()); err == nil {
		t.Fatal("Save = nil, want an error")
	}
	if _, err := repo.Get(context.Background(), "owner", "v"); err == nil {
		t.Fatal("Get = nil, want an error")
	}
	if _, err := repo.List(context.Background(), "owner"); err == nil {
		t.Fatal("List = nil, want an error")
	}
}
