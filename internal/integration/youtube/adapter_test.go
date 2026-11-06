package youtube

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/discovery"
	"github.com/rktkov/ytfeed/internal/domain/identity"
)

func TestSearch(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Errorf("Authorization = %q", got)
		}
		if got := query.Get("q"); got != "ai agents" {
			t.Errorf("q = %q", got)
		}
		if got := query.Get("type"); got != "video" {
			t.Errorf("type = %q", got)
		}
		if got := query.Get("maxResults"); got != "10" {
			t.Errorf("maxResults = %q", got)
		}
		if got := query.Get("order"); got != "relevance" {
			t.Errorf("order = %q", got)
		}
		if query.Get("publishedAfter") == "" {
			t.Error("publishedAfter is empty")
		}
		_, _ = fmt.Fprint(w, `{"items":[
			{"id":{"videoId":"v1"},"snippet":{"channelId":"c1","title":"One","description":"Desc","publishedAt":"2026-11-01T10:00:00Z"}}
		]}`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	source := NewSource(server.URL, server.Client())
	videos, err := source.Search(context.Background(), "token", discovery.Query{
		Interest: "ai-agents", Text: "ai agents", MaxResults: 10, Order: "relevance", Days: 30,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(videos) != 1 || videos[0].VideoID != "v1" || videos[0].ChannelID != "c1" {
		t.Fatalf("videos = %+v", videos)
	}
	if !videos[0].PublishedAt.Equal(time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("published = %v", videos[0].PublishedAt)
	}
}

func TestVideos(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/videos", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("id"); got != "v1,v2" {
			t.Errorf("id = %q", got)
		}
		_, _ = fmt.Fprint(w, `{"items":[
			{"id":"v1","snippet":{"channelId":"c1","title":"One","description":"Desc","publishedAt":"2026-11-01T10:00:00Z"},"statistics":{"viewCount":"1234"},"contentDetails":{"duration":"PT10M"},"topicDetails":{"topicCategories":["https://en.wikipedia.org/wiki/Technology"]}}
		]}`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	videos, err := NewSource(server.URL, server.Client()).Videos(context.Background(), "token", []string{"v1", "v2"})
	if err != nil {
		t.Fatalf("Videos: %v", err)
	}
	if len(videos) != 1 || videos[0].ViewCount != 1234 || videos[0].Duration != "PT10M" {
		t.Fatalf("videos = %+v", videos)
	}
	if len(videos[0].TopicTags) != 1 {
		t.Fatalf("tags = %v", videos[0].TopicTags)
	}
}

func TestMostPopular(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/videos", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("chart"); got != "mostPopular" {
			t.Errorf("chart = %q", got)
		}
		if got := r.URL.Query().Get("regionCode"); got != "GB" {
			t.Errorf("region = %q", got)
		}
		_, _ = fmt.Fprint(w, `{"items":[{"id":"v1","snippet":{"channelId":"c1","title":"One"}}]}`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	videos, err := NewSource(server.URL, server.Client()).MostPopular(context.Background(), "token", "GB")
	if err != nil {
		t.Fatalf("MostPopular: %v", err)
	}
	if len(videos) != 1 || videos[0].VideoID != "v1" {
		t.Fatalf("videos = %+v", videos)
	}
}

func TestAdapterUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()

	source := NewSource(server.URL, server.Client())
	if _, err := source.Search(context.Background(), "token", discovery.Query{Text: "x", MaxResults: 1, Days: 1}); !errors.Is(err, identity.ErrUnauthorized) {
		t.Fatalf("Search = %v, want ErrUnauthorized", err)
	}
	if _, err := source.Videos(context.Background(), "token", []string{"v1"}); !errors.Is(err, identity.ErrUnauthorized) {
		t.Fatalf("Videos = %v, want ErrUnauthorized", err)
	}
	if _, err := source.MostPopular(context.Background(), "token", "US"); !errors.Is(err, identity.ErrUnauthorized) {
		t.Fatalf("MostPopular = %v, want ErrUnauthorized", err)
	}
}

func TestAdapterErrors(t *testing.T) {
	t.Run("malformed search", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprint(w, `{`)
		}))
		defer server.Close()
		if _, err := NewSource(server.URL, server.Client()).Search(context.Background(), "token", discovery.Query{Text: "x", MaxResults: 1, Days: 1}); err == nil {
			t.Fatal("Search = nil, want an error")
		}
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "boom", http.StatusInternalServerError)
		}))
		defer server.Close()
		if _, err := NewSource(server.URL, server.Client()).Videos(context.Background(), "token", []string{"v1"}); err == nil {
			t.Fatal("Videos = nil, want an error")
		}
	})
}

func TestRequestURLPreservesQuery(t *testing.T) {
	source := NewSource("https://example.test/youtube/v3/", nil)
	got, err := source.requestURL("search", url.Values{"q": {"a b"}})
	if err != nil {
		t.Fatalf("requestURL: %v", err)
	}
	if got != "https://example.test/youtube/v3/search?q=a+b" {
		t.Fatalf("url = %q", got)
	}
}

func TestSearchPage(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("pageToken"); got != "p2" {
			t.Errorf("pageToken = %q", got)
		}
		_, _ = fmt.Fprint(w, `{"items":[
			{"id":{"videoId":"v2"},"snippet":{"channelId":"c2","title":"Two","publishedAt":"2026-11-02T10:00:00Z"}}
		],"nextPageToken":"p3"}`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	page, err := NewSource(server.URL, server.Client()).SearchPage(context.Background(), "token", discovery.Query{
		Text: "ai", MaxResults: 10, Order: "relevance", Days: 30,
	}, "p2")
	if err != nil {
		t.Fatalf("SearchPage: %v", err)
	}
	if len(page.Videos) != 1 || page.Videos[0].VideoID != "v2" || page.NextPageToken != "p3" {
		t.Fatalf("page = %+v", page)
	}
}

func TestPlaylistItems(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/playlistItems", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if got := query.Get("playlistId"); got != "uploads-c1" {
			t.Errorf("playlistId = %q", got)
		}
		if got := query.Get("maxResults"); got != "50" {
			t.Errorf("maxResults = %q", got)
		}
		if got := query.Get("pageToken"); got != "p2" {
			t.Errorf("pageToken = %q", got)
		}
		_, _ = fmt.Fprint(w, `{"items":[
			{"snippet":{"channelId":"c1","title":"One","description":"Desc","publishedAt":"2026-11-01T10:00:00Z","resourceId":{"videoId":"v1"}},"contentDetails":{"videoId":"v1"}},
			{"snippet":{"channelId":"c1","title":"No id"}}
		],"nextPageToken":"p3"}`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	page, err := NewSource(server.URL, server.Client()).PlaylistItems(context.Background(), "token", "uploads-c1", "p2")
	if err != nil {
		t.Fatalf("PlaylistItems: %v", err)
	}
	if len(page.Videos) != 1 || page.Videos[0].VideoID != "v1" || page.Videos[0].ChannelID != "c1" {
		t.Fatalf("videos = %+v", page.Videos)
	}
	if page.NextPageToken != "p3" {
		t.Fatalf("next = %q", page.NextPageToken)
	}
}

func TestPlaylistItemsEmptyID(t *testing.T) {
	if _, err := NewSource("https://example.test", nil).PlaylistItems(context.Background(), "token", " ", ""); err == nil {
		t.Fatal("PlaylistItems = nil, want an error")
	}
}
