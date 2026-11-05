package httpapi

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/feed"
	"github.com/rktkov/ytfeed/internal/repository/sqlite"
	"github.com/rktkov/ytfeed/internal/storage"
	"github.com/rktkov/ytfeed/internal/transport/contract"
)

type stubFeedProvider struct {
	batch feed.Batch
	err   error
}

func (s stubFeedProvider) Feed(context.Context, string) (feed.Batch, error) {
	return s.batch, s.err
}

func feedBatch() feed.Batch {
	return feed.Batch{
		ID: "b1", UserID: "owner", Size: 1, State: feed.BatchCompleted,
		Cards: []feed.Card{
			{
				CardID: "c1", VideoID: "v1", ChannelID: "ch1", Title: "Title",
				Channel: "Channel", State: feed.StateAnalyzed, AnalysisID: "a1",
				Score: 0.8, Position: 0, Decision: feed.DecisionNone,
				Topic: "ai-agents", TLDR: "tldr", Similarity: "new territory", WhyThisCard: "why",
				PublishedAt: time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC),
			},
		},
	}
}

func TestFeedJSON(t *testing.T) {
	handler := FeedHandler(stubFeedProvider{batch: feedBatch()})
	request := httptest.NewRequest(http.MethodGet, "/v1/feed", nil)
	request.Header.Set("Accept", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("content type = %q", got)
	}
	var payload contract.Feed
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if err := payload.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if payload.BatchID != "b1" || len(payload.Cards) != 1 {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestFeedSSE(t *testing.T) {
	handler := FeedHandler(stubFeedProvider{batch: feedBatch()})
	request := httptest.NewRequest(http.MethodGet, "/v1/feed", nil)
	request.Header.Set("Accept", "text/event-stream")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); !strings.Contains(got, "text/event-stream") {
		t.Fatalf("content type = %q", got)
	}
	body := recorder.Body.String()
	for _, want := range []string{"event: batch.started", "event: card.upsert", "event: batch.done"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q:\n%s", want, body)
		}
	}
	scanner := bufio.NewScanner(strings.NewReader(body))
	events := 0
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "event: ") {
			events++
		}
	}
	if events != 3 {
		t.Fatalf("events = %d, want 3", events)
	}
}

func TestFeedNoFeed(t *testing.T) {
	handler := FeedHandler(stubFeedProvider{err: feed.ErrNoCachedFeed})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/feed", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func TestFeedError(t *testing.T) {
	handler := FeedHandler(stubFeedProvider{err: errors.New("db down")})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/feed", nil))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
}

func TestFeedUnhappy(t *testing.T) {
	t.Run("nil provider", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		FeedHandler(nil).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/feed", nil))
		if recorder.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", recorder.Code)
		}
	})
	t.Run("method", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		FeedHandler(stubFeedProvider{batch: feedBatch()}).ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/feed", nil))
		if recorder.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want 405", recorder.Code)
		}
	})
}

type stubFeedStore struct {
	batch feed.Batch
	err   error
}

func (s stubFeedStore) LatestCompleted(context.Context, string) (feed.Batch, error) {
	return s.batch, s.err
}

func TestCachedFeedProvider(t *testing.T) {
	provider := CachedFeedProvider{Store: stubFeedStore{batch: feedBatch()}, UserID: "owner"}
	got, err := provider.Feed(context.Background(), "ignored")
	if err != nil {
		t.Fatalf("Feed: %v", err)
	}
	if got.ID != "b1" {
		t.Fatalf("batch = %+v", got)
	}
	if _, err := (CachedFeedProvider{}).Feed(context.Background(), "owner"); !errors.Is(err, feed.ErrNoCachedFeed) {
		t.Fatalf("Feed(nil) = %v, want ErrNoCachedFeed", err)
	}
}

func TestFeedRoute(t *testing.T) {
	handler := HandlerWithFeatures("token", "dev", 11, nil, FeedHandler(stubFeedProvider{batch: feedBatch()}), false)
	request := httptest.NewRequest(http.MethodGet, "/v1/feed", nil)
	request.RemoteAddr = "127.0.0.1:1234"
	request.Header.Set("Authorization", "Bearer token")
	request.Header.Set("Accept", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func TestCachedFeedServed(t *testing.T) {
	dir := t.TempDir()
	db, err := storage.Open(context.Background(), filepath.Join(dir, "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()
	if err := sqlite.NewBatchRepository(db).Save(context.Background(), feedBatch()); err != nil {
		t.Fatalf("Save batch: %v", err)
	}
	provider := CachedFeedProvider{Store: sqlite.NewBatchRepository(db), UserID: "owner"}
	payload, err := provider.Feed(context.Background(), "owner")
	if err != nil {
		t.Fatalf("Feed: %v", err)
	}
	if payload.ID != "b1" {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestCachedFeedMissing(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()
	provider := CachedFeedProvider{Store: sqlite.NewBatchRepository(db), UserID: "owner"}
	if _, err := provider.Feed(context.Background(), "owner"); !errors.Is(err, feed.ErrNoCachedFeed) {
		t.Fatalf("Feed = %v, want ErrNoCachedFeed", err)
	}
}
