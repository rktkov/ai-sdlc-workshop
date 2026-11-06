package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"

	"github.com/rktkov/ytfeed/internal/domain/discovery"
	"github.com/rktkov/ytfeed/internal/domain/feed"
	"github.com/rktkov/ytfeed/internal/transport/contract"
)

type refreshKey struct{}

// WithRefresh marks a request that must bypass the cached batch.
func WithRefresh(ctx context.Context) context.Context {
	return context.WithValue(ctx, refreshKey{}, true)
}

// RefreshRequested reports whether the cached batch must be bypassed.
func RefreshRequested(ctx context.Context) bool {
	value, _ := ctx.Value(refreshKey{}).(bool)
	return value
}

// FeedProvider supplies the latest batch for one user.
type FeedProvider interface {
	Feed(ctx context.Context, userID string) (feed.Batch, error)
}

// FeedStore reads the latest completed batch.
type FeedStore interface {
	LatestCompleted(ctx context.Context, userID string) (feed.Batch, error)
}

// FeedComposer builds and saves one batch.
type FeedComposer interface {
	Compose(ctx context.Context, userID string) (feed.Batch, error)
}

// ProgressComposer builds a batch and reports each step.
type ProgressComposer interface {
	ComposeWithProgress(ctx context.Context, userID string, progress feed.ProgressFunc) (feed.Batch, error)
}

// FeedProgressProvider serves a batch and reports composition progress.
type FeedProgressProvider interface {
	FeedWithProgress(ctx context.Context, userID string, progress feed.ProgressFunc) (feed.Batch, error)
}

// FeedStreamProvider serves a batch and streams pending and analyzed cards.
type FeedStreamProvider interface {
	FeedWithStream(ctx context.Context, userID string, hooks feed.StreamHooks) (feed.Batch, error)
}

// LiveFeedProvider serves the cache or composes a new batch.
type LiveFeedProvider struct {
	Store    FeedStore
	Composer FeedComposer
	UserID   string
}

// Feed returns the cached batch or composes one.
func (p LiveFeedProvider) Feed(ctx context.Context, _ string) (feed.Batch, error) {
	return p.FeedWithProgress(ctx, "", nil)
}

// FeedWithProgress returns the cache or composes with progress events.
func (p LiveFeedProvider) FeedWithProgress(ctx context.Context, _ string, progress feed.ProgressFunc) (feed.Batch, error) {
	if p.Store != nil && !RefreshRequested(ctx) {
		batch, err := p.Store.LatestCompleted(ctx, p.UserID)
		if err == nil {
			if progress != nil {
				progress("cache", "using the cached batch", nil)
			}
			return batch, nil
		}
		if !errors.Is(err, feed.ErrNoCachedFeed) {
			return feed.Batch{}, err
		}
	}
	if p.Composer == nil {
		return feed.Batch{}, feed.ErrNoCachedFeed
	}
	if composer, ok := p.Composer.(ProgressComposer); ok {
		return composer.ComposeWithProgress(ctx, p.UserID, progress)
	}
	return p.Composer.Compose(ctx, p.UserID)
}

// FeedWithStream returns the cache or composes with card hooks.
func (p LiveFeedProvider) FeedWithStream(ctx context.Context, _ string, hooks feed.StreamHooks) (feed.Batch, error) {
	if p.Store != nil && !RefreshRequested(ctx) {
		batch, err := p.Store.LatestCompleted(ctx, p.UserID)
		if err == nil {
			return batch, nil
		}
		if !errors.Is(err, feed.ErrNoCachedFeed) {
			return feed.Batch{}, err
		}
	}
	if p.Composer == nil {
		return feed.Batch{}, feed.ErrNoCachedFeed
	}
	if composer, ok := p.Composer.(interface {
		ComposeWithHooks(context.Context, string, feed.StreamHooks) (feed.Batch, error)
	}); ok {
		return composer.ComposeWithHooks(ctx, p.UserID, hooks)
	}
	if hooks.Progress != nil {
		if composer, ok := p.Composer.(ProgressComposer); ok {
			return composer.ComposeWithProgress(ctx, p.UserID, hooks.Progress)
		}
	}
	return p.Composer.Compose(ctx, p.UserID)
}

// CachedFeedProvider serves the latest completed batch.
type CachedFeedProvider struct {
	Store  FeedStore
	UserID string
}

// Feed returns the cached batch for the configured user.
func (p CachedFeedProvider) Feed(ctx context.Context, _ string) (feed.Batch, error) {
	if p.Store == nil {
		return feed.Batch{}, feed.ErrNoCachedFeed
	}
	return p.Store.LatestCompleted(ctx, p.UserID)
}

// FeedHandler streams one batch as SSE or returns one JSON object.
func FeedHandler(provider FeedProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if provider == nil {
			http.Error(w, "feed provider is nil", http.StatusInternalServerError)
			return
		}
		ctx := r.Context()
		if r.Header.Get("X-Ytfeed-Refresh") == "true" {
			ctx = WithRefresh(ctx)
		}
		if raw := r.Header.Get("X-Ytfeed-Source"); raw != "" {
			source, err := discovery.ParseRefreshSource(raw)
			if err != nil {
				writeFeedError(w, http.StatusBadRequest, "invalid_source")
				return
			}
			ctx = discovery.WithRefreshSource(ctx, source)
		}
		if !strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			batch, err := provider.Feed(ctx, "owner")
			switch {
			case errors.Is(err, feed.ErrNoCachedFeed):
				writeFeedError(w, http.StatusNotFound, "no_feed")
				return
			case errors.Is(err, feed.ErrNotAuthenticated):
				writeFeedError(w, http.StatusUnauthorized, "not_authenticated")
				return
			case errors.Is(err, feed.ErrNoInterests):
				writeFeedError(w, http.StatusConflict, "interests_not_set")
				return
			case errors.Is(err, feed.ErrNoProfile):
				writeFeedError(w, http.StatusConflict, "profile_not_set")
				return
			case err != nil:
				http.Error(w, "feed unavailable", http.StatusInternalServerError)
				return
			}
			payload := contract.FeedFromBatch(batch)
			if err := payload.Validate(); err != nil {
				http.Error(w, "invalid feed", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(payload); err != nil {
				http.Error(w, "encode feed", http.StatusInternalServerError)
			}
			return
		}
		streamFeed(w, provider, ctx)
	}
}

func writeFeedError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":"` + code + `"}`))
}

func streamFeed(w http.ResponseWriter, provider FeedProvider, ctx context.Context) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	var writeMu sync.Mutex
	write := func(name string, value any) {
		writeMu.Lock()
		defer writeMu.Unlock()
		writeEvent(w, flusher, name, value)
	}

	var batch feed.Batch
	var err error
	started := false
	if streamProvider, ok := provider.(FeedStreamProvider); ok {
		batch, err = streamProvider.FeedWithStream(ctx, "owner", feed.StreamHooks{
			Progress: func(stage, message string, attributes map[string]any) {
				write("progress", map[string]any{
					"stage": stage, "message": message, "attributes": attributes,
				})
			},
			Batch: func(value feed.Batch) error {
				started = true
				payload := contract.FeedFromBatch(value)
				writeBatchStarted(write, payload)
				for _, card := range payload.Cards {
					write("card.upsert", card)
				}
				return nil
			},
			Card: func(_ feed.Batch, card feed.Card) {
				write("card.upsert", contract.FeedCardFromCard(card))
			},
		})
	} else if progressProvider, ok := provider.(FeedProgressProvider); ok {
		batch, err = progressProvider.FeedWithProgress(ctx, "owner", func(stage, message string, attributes map[string]any) {
			write("progress", map[string]any{
				"stage": stage, "message": message, "attributes": attributes,
			})
		})
	} else {
		batch, err = provider.Feed(ctx, "owner")
	}
	if err != nil {
		write("error", map[string]any{"error": feedErrorCode(err)})
		return
	}
	if started {
		write("batch.done", map[string]any{
			"batch_id": batch.ID, "state": string(batch.State),
		})
		return
	}
	payload := contract.FeedFromBatch(batch)
	if err := payload.Validate(); err != nil {
		write("error", map[string]any{"error": "invalid_feed"})
		return
	}
	writeFeedEvents(w, flusher, payload)
}

func feedErrorCode(err error) string {
	switch {
	case errors.Is(err, feed.ErrNotAuthenticated):
		return "not_authenticated"
	case errors.Is(err, feed.ErrNoInterests):
		return "interests_not_set"
	case errors.Is(err, feed.ErrNoProfile):
		return "profile_not_set"
	case errors.Is(err, feed.ErrNoCachedFeed):
		return "no_feed"
	default:
		return "feed_unavailable"
	}
}

func writeBatchStarted(write func(string, any), payload contract.Feed) {
	write("batch.started", map[string]any{
		"batch_id": payload.BatchID, "state": payload.State, "size": len(payload.Cards),
		"banner": payload.Banner, "degradation_reason": payload.DegradationReason,
	})
}

func writeFeedEvents(w http.ResponseWriter, flusher http.Flusher, payload contract.Feed) {
	write := func(name string, value any) { writeEvent(w, flusher, name, value) }
	writeBatchStarted(write, payload)
	for _, card := range payload.Cards {
		write("card.upsert", card)
	}
	write("batch.done", map[string]any{
		"batch_id": payload.BatchID, "state": payload.State,
	})
}

func writeEvent(w http.ResponseWriter, flusher http.Flusher, name string, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	_, _ = w.Write([]byte("event: " + name + "\n"))
	_, _ = w.Write([]byte("data: " + string(data) + "\n\n"))
	flusher.Flush()
}
