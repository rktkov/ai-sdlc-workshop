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

func newSessionService(t *testing.T) (*feed.SessionService, *SessionRepository, *time.Time) {
	t.Helper()
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	repo := NewSessionRepository(db)
	service := &feed.SessionService{
		Store:       repo,
		Now:         func() time.Time { return now },
		IdleTimeout: time.Hour,
		ID:          func() string { return "session-1" },
	}
	return service, repo, &now
}

func TestSessionOpenNew(t *testing.T) {
	service, _, now := newSessionService(t)
	session, resumed, err := service.Open(context.Background(), "owner", "batch-1", "card-1", "lease-1")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if resumed {
		t.Fatal("new session reported resumed")
	}
	if session.ID != "session-1" || session.State != feed.SessionOpen {
		t.Fatalf("session = %+v", session)
	}
	if session.CurrentCardID != "card-1" || session.FurthestCardID != "card-1" {
		t.Fatalf("cursor = %+v", session)
	}
	if !session.ExpiresAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("expiry = %v", session.ExpiresAt)
	}
}

func TestSessionLiveLease(t *testing.T) {
	service, _, _ := newSessionService(t)
	if _, _, err := service.Open(context.Background(), "owner", "batch-1", "card-1", "lease-1"); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, _, err := service.Open(context.Background(), "owner", "batch-1", "card-1", "lease-2"); !errors.Is(err, feed.ErrLiveLease) {
		t.Fatalf("Open = %v, want ErrLiveLease", err)
	}
}

func TestSessionSameLease(t *testing.T) {
	service, _, _ := newSessionService(t)
	first, _, err := service.Open(context.Background(), "owner", "batch-1", "card-1", "lease-1")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	second, resumed, err := service.Open(context.Background(), "owner", "batch-1", "card-1", "lease-1")
	if err != nil {
		t.Fatalf("Open same: %v", err)
	}
	if resumed || second.ID != first.ID {
		t.Fatalf("session = %+v, resumed = %t", second, resumed)
	}
}

func TestSessionStaleLease(t *testing.T) {
	service, repo, now := newSessionService(t)
	first, _, err := service.Open(context.Background(), "owner", "batch-1", "card-1", "lease-1")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	*now = now.Add(2 * time.Hour)
	reclaimed, resumed, err := service.Open(context.Background(), "owner", "batch-1", "card-1", "lease-2")
	if err != nil {
		t.Fatalf("Open stale: %v", err)
	}
	if !resumed || reclaimed.ID != first.ID || reclaimed.LeaseOwner != "lease-2" {
		t.Fatalf("session = %+v, resumed = %t", reclaimed, resumed)
	}
	open, err := repo.GetOpen(context.Background(), "owner")
	if err != nil {
		t.Fatalf("GetOpen: %v", err)
	}
	if open.LeaseOwner != "lease-2" {
		t.Fatalf("lease = %q", open.LeaseOwner)
	}
}

func TestSessionClosedResume(t *testing.T) {
	service, repo, _ := newSessionService(t)
	first, _, err := service.Open(context.Background(), "owner", "batch-1", "card-1", "lease-1")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := service.Close(context.Background(), first.ID); err != nil {
		t.Fatalf("Close: %v", err)
	}
	reopened, resumed, err := service.Open(context.Background(), "owner", "batch-1", "card-1", "lease-2")
	if err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	if !resumed || reopened.ID != first.ID || reopened.State != feed.SessionOpen {
		t.Fatalf("session = %+v, resumed = %t", reopened, resumed)
	}
	if reopened.ResumeCount != 1 || reopened.LeaseOwner != "lease-2" {
		t.Fatalf("session = %+v", reopened)
	}
	if _, err := repo.GetLatest(context.Background(), "owner", "batch-2"); !errors.Is(err, feed.ErrNoSession) {
		t.Fatalf("GetLatest = %v, want ErrNoSession", err)
	}
}

func TestSessionOneOpen(t *testing.T) {
	_, repo, _ := newSessionService(t)
	now := time.Now().UTC()
	first := feed.Session{
		ID: "s1", UserID: "owner", BatchID: "batch-1", State: feed.SessionOpen,
		CurrentCardID: "card-1", FurthestCardID: "card-1", LeaseOwner: "lease-1",
		LastActivityAt: now, ExpiresAt: now.Add(time.Hour), OpenedAt: now,
	}
	if err := repo.Create(context.Background(), first); err != nil {
		t.Fatalf("Create: %v", err)
	}
	second := first
	second.ID = "s2"
	if err := repo.Create(context.Background(), second); !errors.Is(err, feed.ErrLiveLease) {
		t.Fatalf("Create = %v, want ErrLiveLease", err)
	}
}

func TestSessionSaveStale(t *testing.T) {
	service, repo, _ := newSessionService(t)
	session, _, err := service.Open(context.Background(), "owner", "batch-1", "card-1", "lease-1")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := repo.Save(context.Background(), session, 99); !errors.Is(err, feed.ErrStaleSession) {
		t.Fatalf("Save = %v, want ErrStaleSession", err)
	}
}

func TestSessionTouch(t *testing.T) {
	service, _, now := newSessionService(t)
	session, _, err := service.Open(context.Background(), "owner", "batch-1", "card-1", "lease-1")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	*now = now.Add(30 * time.Minute)
	touched, err := service.Touch(context.Background(), session)
	if err != nil {
		t.Fatalf("Touch: %v", err)
	}
	if !touched.ExpiresAt.After(session.ExpiresAt) || touched.Version <= session.Version {
		t.Fatalf("touched = %+v", touched)
	}
	if err := service.Close(context.Background(), session.ID); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := repoGetOpenForTest(t, service); !errors.Is(err, feed.ErrNoOpenSession) {
		t.Fatalf("GetOpen = %v, want ErrNoOpenSession", err)
	}
}

func repoGetOpenForTest(t *testing.T, service *feed.SessionService) (feed.Session, error) {
	t.Helper()
	return service.Store.GetOpen(context.Background(), "owner")
}

func TestSessionErrors(t *testing.T) {
	service, repo, _ := newSessionService(t)
	if _, _, err := service.Open(context.Background(), "", "batch-1", "card-1", "lease-1"); err == nil {
		t.Fatal("Open(no user) = nil, want an error")
	}
	if _, _, err := service.Open(context.Background(), "owner", "", "card-1", "lease-1"); err == nil {
		t.Fatal("Open(no batch) = nil, want an error")
	}
	if _, _, err := service.Open(context.Background(), "owner", "batch-1", "", "lease-1"); err == nil {
		t.Fatal("Open(no card) = nil, want an error")
	}
	if _, _, err := service.Open(context.Background(), "owner", "batch-1", "card-1", ""); err == nil {
		t.Fatal("Open(no lease) = nil, want an error")
	}
	if _, err := repo.GetOpen(context.Background(), "owner"); !errors.Is(err, feed.ErrNoOpenSession) {
		t.Fatalf("GetOpen = %v, want ErrNoOpenSession", err)
	}
	if _, err := repo.GetLatest(context.Background(), "owner", "batch-1"); !errors.Is(err, feed.ErrNoSession) {
		t.Fatalf("GetLatest = %v, want ErrNoSession", err)
	}
	if err := service.Close(context.Background(), "missing"); !errors.Is(err, feed.ErrNoOpenSession) {
		t.Fatalf("Close = %v, want ErrNoOpenSession", err)
	}
}
