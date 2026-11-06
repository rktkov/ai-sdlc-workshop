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

	"github.com/rktkov/ytfeed/internal/domain/discovery"
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

type stubComposer struct {
	batch feed.Batch
	err   error
	calls int
}

func (s *stubComposer) Compose(context.Context, string) (feed.Batch, error) {
	s.calls++
	return s.batch, s.err
}

func TestLiveFeedProviderCached(t *testing.T) {
	composer := &stubComposer{}
	provider := LiveFeedProvider{Store: stubFeedStore{batch: feedBatch()}, Composer: composer, UserID: "owner"}
	got, err := provider.Feed(context.Background(), "owner")
	if err != nil {
		t.Fatalf("Feed: %v", err)
	}
	if got.ID != "b1" || composer.calls != 0 {
		t.Fatalf("batch = %+v, compose calls = %d", got, composer.calls)
	}
}

func TestLiveFeedProviderCompose(t *testing.T) {
	composer := &stubComposer{batch: feedBatch()}
	provider := LiveFeedProvider{Store: stubFeedStore{err: feed.ErrNoCachedFeed}, Composer: composer, UserID: "owner"}
	got, err := provider.Feed(context.Background(), "owner")
	if err != nil {
		t.Fatalf("Feed: %v", err)
	}
	if got.ID != "b1" || composer.calls != 1 {
		t.Fatalf("batch = %+v, compose calls = %d", got, composer.calls)
	}
}

func TestLiveFeedProviderErrors(t *testing.T) {
	provider := LiveFeedProvider{Store: stubFeedStore{err: errors.New("db")}, Composer: &stubComposer{}}
	if _, err := provider.Feed(context.Background(), "owner"); err == nil {
		t.Fatal("Feed(store error) = nil, want an error")
	}
	provider = LiveFeedProvider{Store: stubFeedStore{err: feed.ErrNoCachedFeed}}
	if _, err := provider.Feed(context.Background(), "owner"); !errors.Is(err, feed.ErrNoCachedFeed) {
		t.Fatalf("Feed(nil composer) = %v, want ErrNoCachedFeed", err)
	}
	provider = LiveFeedProvider{Store: stubFeedStore{err: feed.ErrNoCachedFeed}, Composer: &stubComposer{err: feed.ErrNotAuthenticated}}
	if _, err := provider.Feed(context.Background(), "owner"); !errors.Is(err, feed.ErrNotAuthenticated) {
		t.Fatalf("Feed(compose error) = %v, want ErrNotAuthenticated", err)
	}
}

func TestFeedErrorMapping(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "auth", err: feed.ErrNotAuthenticated, want: http.StatusUnauthorized},
		{name: "interests", err: feed.ErrNoInterests, want: http.StatusConflict},
		{name: "profile", err: feed.ErrNoProfile, want: http.StatusConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			FeedHandler(stubFeedProvider{err: tt.err}).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/feed", nil))
			if recorder.Code != tt.want {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.want)
			}
		})
	}
}

func TestLiveFeedProviderRefresh(t *testing.T) {
	composer := &stubComposer{batch: feedBatch()}
	provider := LiveFeedProvider{Store: stubFeedStore{batch: feedBatch()}, Composer: composer, UserID: "owner"}
	ctx := WithRefresh(context.Background())
	if !RefreshRequested(ctx) {
		t.Fatal("RefreshRequested = false")
	}
	if _, err := provider.Feed(ctx, "owner"); err != nil {
		t.Fatalf("Feed: %v", err)
	}
	if composer.calls != 1 {
		t.Fatalf("compose calls = %d, want 1", composer.calls)
	}
}

type progressFeedProvider struct {
	batch feed.Batch
}

func (p progressFeedProvider) Feed(context.Context, string) (feed.Batch, error) {
	return p.batch, nil
}
func (p progressFeedProvider) FeedWithProgress(_ context.Context, _ string, progress feed.ProgressFunc) (feed.Batch, error) {
	progress("search", "searching: ai agents", map[string]any{"current": 1, "total": 5})
	return p.batch, nil
}

func TestFeedProgressStream(t *testing.T) {
	handler := FeedHandler(progressFeedProvider{batch: feedBatch()})
	request := httptest.NewRequest(http.MethodGet, "/v1/feed", nil)
	request.Header.Set("Accept", "text/event-stream")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	body := recorder.Body.String()
	for _, want := range []string{"event: progress", "searching: ai agents", "event: batch.started", "event: card.upsert", "event: batch.done"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q:\n%s", want, body)
		}
	}
}

type sourceProvider struct {
	batch  feed.Batch
	source discovery.RefreshSource
}

func (p *sourceProvider) Feed(context.Context, string) (feed.Batch, error) {
	return p.batch, nil
}
func (p *sourceProvider) FeedWithProgress(ctx context.Context, _ string, _ feed.ProgressFunc) (feed.Batch, error) {
	p.source = discovery.RefreshSourceFromContext(ctx)
	return p.batch, nil
}

func TestFeedSourceHeader(t *testing.T) {
	provider := &sourceProvider{batch: feedBatch()}
	handler := FeedHandler(provider)
	request := httptest.NewRequest(http.MethodGet, "/v1/feed", nil)
	request.Header.Set("Accept", "text/event-stream")
	request.Header.Set("X-Ytfeed-Source", "trending")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	if provider.source != discovery.RefreshTrending {
		t.Fatalf("source = %q", provider.source)
	}
}

func TestFeedInvalidSource(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/feed", nil)
	request.Header.Set("X-Ytfeed-Source", "random")
	FeedHandler(stubFeedProvider{batch: feedBatch()}).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

type streamFeedProvider struct {
	batch feed.Batch
}

func (p streamFeedProvider) Feed(context.Context, string) (feed.Batch, error) {
	return p.batch, nil
}

func (p streamFeedProvider) FeedWithStream(_ context.Context, _ string, hooks feed.StreamHooks) (feed.Batch, error) {
	pending := p.batch
	pending.State = feed.BatchStreaming
	for i := range pending.Cards {
		pending.Cards[i].State = feed.StatePending
	}
	if hooks.Batch != nil {
		if err := hooks.Batch(pending); err != nil {
			return feed.Batch{}, err
		}
	}
	if hooks.Card != nil {
		hooks.Card(p.batch, p.batch.Cards[0])
	}
	return p.batch, nil
}

func TestFeedStreamsProgressiveCards(t *testing.T) {
	handler := FeedHandler(streamFeedProvider{batch: feedBatch()})
	request := httptest.NewRequest(http.MethodGet, "/v1/feed", nil)
	request.Header.Set("Accept", "text/event-stream")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	body := recorder.Body.String()
	if !strings.Contains(body, "event: batch.started") {
		t.Fatalf("body = %s", body)
	}
	if strings.Count(body, "event: card.upsert") != 2 {
		t.Fatalf("card events = %d", strings.Count(body, "event: card.upsert"))
	}
	if !strings.Contains(body, `"state":"streaming"`) {
		t.Fatalf("body = %s", body)
	}
	if !strings.Contains(body, `"state":"completed"`) {
		t.Fatalf("body = %s", body)
	}
	if strings.Count(body, "event: batch.done") != 1 {
		t.Fatalf("done events = %d", strings.Count(body, "event: batch.done"))
	}
}

type hookComposer struct {
	batch  feed.Batch
	called bool
}

func (h *hookComposer) Compose(context.Context, string) (feed.Batch, error) {
	return h.batch, nil
}
func (h *hookComposer) ComposeWithHooks(_ context.Context, _ string, hooks feed.StreamHooks) (feed.Batch, error) {
	h.called = true
	if hooks.Batch != nil {
		_ = hooks.Batch(h.batch)
	}
	return h.batch, nil
}

type progressComposer struct {
	batch  feed.Batch
	called bool
}

func (p *progressComposer) Compose(context.Context, string) (feed.Batch, error) {
	return p.batch, nil
}
func (p *progressComposer) ComposeWithProgress(_ context.Context, _ string, progress feed.ProgressFunc) (feed.Batch, error) {
	p.called = true
	if progress != nil {
		progress("search", "searching", nil)
	}
	return p.batch, nil
}

func TestLiveFeedProviderStreamCached(t *testing.T) {
	composer := &hookComposer{batch: feedBatch()}
	provider := LiveFeedProvider{Store: stubFeedStore{batch: feedBatch()}, Composer: composer, UserID: "owner"}
	called := false
	got, err := provider.FeedWithStream(context.Background(), "owner", feed.StreamHooks{
		Batch: func(feed.Batch) error { called = true; return nil },
	})
	if err != nil {
		t.Fatalf("FeedWithStream: %v", err)
	}
	if got.ID != "b1" || called || composer.called {
		t.Fatalf("batch = %+v, called = %t, composer = %t", got, called, composer.called)
	}
}

func TestLiveFeedProviderStreamHooks(t *testing.T) {
	composer := &hookComposer{batch: feedBatch()}
	provider := LiveFeedProvider{Store: stubFeedStore{err: feed.ErrNoCachedFeed}, Composer: composer, UserID: "owner"}
	if _, err := provider.FeedWithStream(context.Background(), "owner", feed.StreamHooks{}); err != nil {
		t.Fatalf("FeedWithStream: %v", err)
	}
	if !composer.called {
		t.Fatal("ComposeWithHooks was not called")
	}
}

func TestLiveFeedProviderStreamProgressFallback(t *testing.T) {
	composer := &progressComposer{batch: feedBatch()}
	provider := LiveFeedProvider{Store: stubFeedStore{err: feed.ErrNoCachedFeed}, Composer: composer, UserID: "owner"}
	var stages []string
	if _, err := provider.FeedWithStream(context.Background(), "owner", feed.StreamHooks{
		Progress: func(stage, _ string, _ map[string]any) { stages = append(stages, stage) },
	}); err != nil {
		t.Fatalf("FeedWithStream: %v", err)
	}
	if !composer.called || len(stages) != 1 || stages[0] != "search" {
		t.Fatalf("called = %t, stages = %v", composer.called, stages)
	}
}

func TestLiveFeedProviderStreamErrors(t *testing.T) {
	provider := LiveFeedProvider{Store: stubFeedStore{err: errors.New("db")}, Composer: &stubComposer{}}
	if _, err := provider.FeedWithStream(context.Background(), "owner", feed.StreamHooks{}); err == nil {
		t.Fatal("FeedWithStream(store error) = nil, want an error")
	}
	provider = LiveFeedProvider{Store: stubFeedStore{err: feed.ErrNoCachedFeed}}
	if _, err := provider.FeedWithStream(context.Background(), "owner", feed.StreamHooks{}); !errors.Is(err, feed.ErrNoCachedFeed) {
		t.Fatalf("FeedWithStream(nil composer) = %v, want ErrNoCachedFeed", err)
	}
}

type failingStreamProvider struct{}

func (failingStreamProvider) Feed(context.Context, string) (feed.Batch, error) {
	return feed.Batch{}, feed.ErrNoCachedFeed
}
func (failingStreamProvider) FeedWithStream(context.Context, string, feed.StreamHooks) (feed.Batch, error) {
	return feed.Batch{}, feed.ErrNotAuthenticated
}

func TestFeedStreamErrorEvent(t *testing.T) {
	handler := FeedHandler(failingStreamProvider{})
	request := httptest.NewRequest(http.MethodGet, "/v1/feed", nil)
	request.Header.Set("Accept", "text/event-stream")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if !strings.Contains(recorder.Body.String(), "not_authenticated") {
		t.Fatalf("body = %s", recorder.Body.String())
	}
}
