//go:build integration

package e2e

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/analysis"
	"github.com/rktkov/ytfeed/internal/domain/discovery"
	"github.com/rktkov/ytfeed/internal/domain/feed"
	"github.com/rktkov/ytfeed/internal/domain/library"
	"github.com/rktkov/ytfeed/internal/domain/taste"
	"github.com/rktkov/ytfeed/internal/integration/libraryfiles"
	"github.com/rktkov/ytfeed/internal/repository/sqlite"
	"github.com/rktkov/ytfeed/internal/storage"
	httpapi "github.com/rktkov/ytfeed/internal/transport/http"
)

func openDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func profileFixture() taste.Profile {
	return taste.Profile{
		UserID: "owner", RubricVersion: "v1",
		Axes: []taste.Axis{
			{Name: "depth", Weight: 0.5, PositiveExamples: []string{"p"}, NegativeExamples: []string{"n"}},
			{Name: "novelty", Weight: 0.5, PositiveExamples: []string{"p"}, NegativeExamples: []string{"n"}},
			{Name: "clarity", Weight: 0.5, PositiveExamples: []string{"p"}, NegativeExamples: []string{"n"}},
		},
		Interests: []string{"ai-agents"},
	}
}

func composedBatch(t *testing.T) feed.Batch {
	t.Helper()
	candidate := discovery.SelectedCandidate{
		Candidate: discovery.Candidate{
			VideoID: "dQw4w9WgXcQ", ChannelID: "c1", Title: "Title",
			Source: discovery.SourceSearch, DiscoveryRank: 0,
		},
	}
	result := analysis.Analysis{
		ID: "a1", State: analysis.StateAnalyzed, Topic: "ai-agents", TLDR: "tldr",
		Similarity: "new territory", WhyThisCard: "why",
		AxisScores:    []analysis.AxisScore{{Axis: "depth", Score: 0.8, Justification: "good"}},
		RubricVersion: "v1", ModelID: "model", CompletedAt: time.Now().UTC(),
	}
	batch, err := feed.ComposeBatch("b1", "owner", 1, []discovery.SelectedCandidate{candidate}, map[string]analysis.Analysis{"dQw4w9WgXcQ": result})
	if err != nil {
		t.Fatalf("ComposeBatch: %v", err)
	}
	return batch
}

func TestGestureColdPeriodAndResume(t *testing.T) {
	db := openDB(t)
	profiles := sqlite.NewProfileRepository(db)
	if err := profiles.Save(context.Background(), profileFixture()); err != nil {
		t.Fatalf("Save profile: %v", err)
	}
	tasteService := &taste.Service{Store: sqlite.NewTasteRepository(db), Profiles: profiles}
	sessionRepo := sqlite.NewSessionRepository(db)
	sessionService := &feed.SessionService{Store: sessionRepo, IdleTimeout: time.Hour}
	batch := composedBatch(t)

	session, resumed, err := sessionService.Open(context.Background(), "owner", batch.ID, batch.Cards[0].CardID, "lease-1")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if resumed {
		t.Fatal("new session reported resumed")
	}
	gestures := &feed.GestureService{Sessions: sessionRepo, Taste: tasteService}
	result, err := gestures.Apply(context.Background(), session, &batch, feed.GestureCommand{
		GestureID: "g1", CardID: batch.Cards[0].CardID, Gesture: feed.GestureLeft, Reason: taste.ReasonMarketing,
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if batch.Cards[0].Decision != feed.DecisionRejected {
		t.Fatalf("decision = %q", batch.Cards[0].Decision)
	}
	cold, err := sqlite.NewTasteRepository(db).ListActiveColdPeriods(context.Background(), "owner")
	if err != nil {
		t.Fatalf("ListActiveColdPeriods: %v", err)
	}
	if len(cold) != 1 {
		t.Fatalf("cold periods = %d, want 1", len(cold))
	}
	if err := sessionService.Close(context.Background(), result.Session.ID); err != nil {
		t.Fatalf("Close: %v", err)
	}
	reopened, resumed, err := sessionService.Open(context.Background(), "owner", batch.ID, batch.Cards[0].CardID, "lease-2")
	if err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	if !resumed || reopened.CurrentCardID != batch.Cards[0].CardID || reopened.ResumeCount != 1 {
		t.Fatalf("reopened = %+v, resumed = %t", reopened, resumed)
	}
}

func TestCachedFeedServed(t *testing.T) {
	db := openDB(t)
	if err := sqlite.NewBatchRepository(db).Save(context.Background(), composedBatch(t)); err != nil {
		t.Fatalf("Save batch: %v", err)
	}
	provider := httpapi.CachedFeedProvider{Store: sqlite.NewBatchRepository(db), UserID: "owner"}
	handler := httpapi.FeedHandler(provider)
	request := httptest.NewRequest(http.MethodGet, "/v1/feed", nil)
	request.Header.Set("Accept", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	var payload struct {
		BatchID string `json:"batch_id"`
		Cards   []any  `json:"cards"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.BatchID != "b1" || len(payload.Cards) != 1 {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestWeeklyRating(t *testing.T) {
	db := openDB(t)
	repo := sqlite.NewRatingRepository(db)
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	service := &feed.RatingService{Repo: repo, Now: func() time.Time { return now }}
	prompt, err := service.ShouldPrompt(context.Background(), "owner")
	if err != nil || !prompt {
		t.Fatalf("ShouldPrompt = %t, %v", prompt, err)
	}
	rating, err := service.Record(context.Background(), "owner", feed.RatingBetter, []string{"b1"})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if rating.Value != feed.RatingBetter {
		t.Fatalf("rating = %+v", rating)
	}
	prompt, err = service.ShouldPrompt(context.Background(), "owner")
	if err != nil || prompt {
		t.Fatalf("ShouldPrompt after record = %t, %v", prompt, err)
	}
	dismissed, err := service.Dismiss(context.Background(), "other", []string{"b2"})
	if err != nil {
		t.Fatalf("Dismiss: %v", err)
	}
	if dismissed.Value != feed.RatingUnrated {
		t.Fatalf("dismissed = %+v", dismissed)
	}
}

type writingDownloader struct{ dir string }

func (d *writingDownloader) Download(_ context.Context, videoID, dir string) (library.DownloadResult, error) {
	d.dir = dir
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return library.DownloadResult{}, err
	}
	path := filepath.Join(dir, videoID+".mp4")
	if err := os.WriteFile(path, []byte("media"), 0o600); err != nil {
		return library.DownloadResult{}, err
	}
	return library.DownloadResult{Extension: "mp4", Bytes: 5}, nil
}

type recordingPlayer struct{ path string }

func (p *recordingPlayer) Open(_ context.Context, path string) error {
	p.path = path
	return nil
}

func TestLibraryDownloadAndPlay(t *testing.T) {
	dir := t.TempDir()
	db, err := storage.Open(context.Background(), filepath.Join(dir, "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	files := libraryfiles.New(dir)
	downloader := &writingDownloader{}
	player := &recordingPlayer{}
	service := &library.Service{
		Repo: sqlite.NewLibraryRepository(db), Files: files,
		Downloader: downloader, Player: player,
	}
	if _, err := service.Save(context.Background(), library.Entry{
		UserID: "owner", VideoID: "dQw4w9WgXcQ", Title: "Title",
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	entry, err := service.Download(context.Background(), "owner", "dQw4w9WgXcQ")
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if entry.State != library.StateDownloaded || entry.FileExtension != "mp4" {
		t.Fatalf("entry = %+v", entry)
	}
	if err := service.Open(context.Background(), "owner", "dQw4w9WgXcQ"); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if player.path != filepath.Join(files.Dir(), "dQw4w9WgXcQ.mp4") {
		t.Fatalf("player path = %q", player.path)
	}
}
