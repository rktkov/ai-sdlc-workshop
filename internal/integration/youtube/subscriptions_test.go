package youtube

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/rktkov/ytfeed/internal/domain/identity"
)

func TestSubscriptionPages(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/subscriptions", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.URL.Query().Get("mine"); got != "true" {
			t.Errorf("mine = %q", got)
		}
		switch r.URL.Query().Get("pageToken") {
		case "":
			_, _ = fmt.Fprint(w, `{"items":[
				{"snippet":{"title":"One","resourceId":{"channelId":"c1"}}},
				{"snippet":{"title":"Two","resourceId":{"channelId":"c2"}}}
			],"nextPageToken":"p2"}`)
		case "p2":
			_, _ = fmt.Fprint(w, `{"items":[
				{"snippet":{"title":"Three","resourceId":{"channelId":"c3"}}}
			]}`)
		default:
			http.Error(w, "bad token", http.StatusBadRequest)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	source := NewSource(server.URL, server.Client())
	first, err := source.SubscriptionPage(context.Background(), "test-token", "")
	if err != nil {
		t.Fatalf("SubscriptionPage: %v", err)
	}
	if len(first.Items) != 2 || first.NextPageToken != "p2" {
		t.Fatalf("first = %+v", first)
	}
	if first.Items[0].ChannelID != "c1" || first.Items[0].Title != "One" {
		t.Fatalf("first item = %+v", first.Items[0])
	}

	second, err := source.SubscriptionPage(context.Background(), "test-token", "p2")
	if err != nil {
		t.Fatalf("SubscriptionPage page 2: %v", err)
	}
	if len(second.Items) != 1 || second.NextPageToken != "" {
		t.Fatalf("second = %+v", second)
	}
}

func TestChannels(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/channels", func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("id"); got != "c1,c2" {
			t.Errorf("id = %q", got)
		}
		_, _ = fmt.Fprint(w, `{"items":[
			{"id":"c1","snippet":{"title":"One"},"statistics":{"subscriberCount":"100"},"topicDetails":{"topicCategories":["https://en.wikipedia.org/wiki/Technology"]}},
			{"id":"c2","snippet":{"title":"Two"},"statistics":{"subscriberCount":"hidden"}}
		]}`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	source := NewSource(server.URL, server.Client())
	got, err := source.Channels(context.Background(), "test-token", []string{"c1", "c2"})
	if err != nil {
		t.Fatalf("Channels: %v", err)
	}
	if got["c1"].SubscriberCount != 100 || len(got["c1"].TopicTags) != 1 {
		t.Fatalf("c1 = %+v", got["c1"])
	}
	if got["c2"].SubscriberCount != 0 {
		t.Fatalf("c2 = %+v", got["c2"])
	}
}

func TestChannelsChunks(t *testing.T) {
	var calls int32
	mux := http.NewServeMux()
	mux.HandleFunc("/channels", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		_, _ = fmt.Fprint(w, `{"items":[]}`)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	ids := make([]string, 51)
	for i := range ids {
		ids[i] = fmt.Sprintf("c%d", i)
	}
	if _, err := NewSource(server.URL, server.Client()).Channels(context.Background(), "token", ids); err != nil {
		t.Fatalf("Channels: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("channel calls = %d, want 2", got)
	}
}

func TestSubscriptionUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()

	_, err := NewSource(server.URL, server.Client()).SubscriptionPage(context.Background(), "token", "")
	if !errors.Is(err, identity.ErrUnauthorized) {
		t.Fatalf("SubscriptionPage = %v, want ErrUnauthorized", err)
	}
}

func TestSubscriptionErrors(t *testing.T) {
	t.Run("malformed json", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprint(w, `{`)
		}))
		defer server.Close()
		if _, err := NewSource(server.URL, server.Client()).SubscriptionPage(context.Background(), "token", ""); err == nil {
			t.Fatal("SubscriptionPage = nil, want an error")
		}
	})

	t.Run("missing channel id", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprint(w, `{"items":[{"snippet":{"title":"No id"}}]}`)
		}))
		defer server.Close()
		if _, err := NewSource(server.URL, server.Client()).SubscriptionPage(context.Background(), "token", ""); err == nil {
			t.Fatal("SubscriptionPage = nil, want an error")
		}
	})

	t.Run("channels server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "boom", http.StatusInternalServerError)
		}))
		defer server.Close()
		if _, err := NewSource(server.URL, server.Client()).Channels(context.Background(), "token", []string{"c1"}); err == nil {
			t.Fatal("Channels = nil, want an error")
		}
	})
}

func TestDefaultBaseURL(t *testing.T) {
	source := NewSource("", nil)
	if source.BaseURL != DefaultBaseURL {
		t.Fatalf("BaseURL = %q", source.BaseURL)
	}
}

func TestRequestURL(t *testing.T) {
	source := NewSource("https://example.test/youtube/v3/", nil)
	got, err := source.requestURL("subscriptions", url.Values{"mine": {"true"}})
	if err != nil {
		t.Fatalf("requestURL: %v", err)
	}
	if !strings.HasPrefix(got, "https://example.test/youtube/v3/subscriptions?") {
		t.Fatalf("url = %q", got)
	}
}
