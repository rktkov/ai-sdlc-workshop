package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/rktkov/ytfeed/internal/domain/feed"
	"github.com/rktkov/ytfeed/internal/transport/contract"
)

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

// LiveFeedProvider serves the cache or composes a new batch.
type LiveFeedProvider struct {
	Store    FeedStore
	Composer FeedComposer
	UserID   string
}

// Feed returns the cached batch or composes one.
func (p LiveFeedProvider) Feed(ctx context.Context, _ string) (feed.Batch, error) {
	if p.Store != nil {
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
		batch, err := provider.Feed(r.Context(), "owner")
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

		if strings.Contains(r.Header.Get("Accept"), "application/json") {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(payload); err != nil {
				http.Error(w, "encode feed", http.StatusInternalServerError)
			}
			return
		}
		streamFeed(w, payload)
	}
}

func writeFeedError(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":"` + code + `"}`))
}

func streamFeed(w http.ResponseWriter, payload contract.Feed) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	writeEvent(w, flusher, "batch.started", map[string]any{
		"batch_id": payload.BatchID, "state": payload.State, "size": len(payload.Cards),
		"banner": payload.Banner, "degradation_reason": payload.DegradationReason,
	})
	for _, card := range payload.Cards {
		writeEvent(w, flusher, "card.upsert", card)
	}
	writeEvent(w, flusher, "batch.done", map[string]any{
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
