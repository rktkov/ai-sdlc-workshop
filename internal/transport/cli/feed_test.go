package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rktkov/ytfeed/internal/transport/contract"
)

func feedTestEnv(t *testing.T) func(string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "api-token"), []byte("secret"), 0o600); err != nil {
		t.Fatalf("write token: %v", err)
	}
	return func(key string) string {
		if key == "YTFEED_DATA_DIR" {
			return dir
		}
		return ""
	}
}

func wireCard() contract.FeedCard {
	return contract.FeedCard{
		CardID: "c1", VideoID: "v1", ChannelID: "ch1", Title: "A card",
		Channel: "A channel", State: "analyzed", Topic: "ai-agents", TLDR: "tldr",
		Similarity: "new territory", WhyThisCard: "why", Position: 0,
		PublishedAt: "2026-11-04T12:00:00Z",
	}
}

func TestFeedSSE(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/rating" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("Accept"); got != "text/event-stream" {
			t.Errorf("Accept = %q", got)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "event: batch.started\ndata: {}\n\n")
		data, _ := json.Marshal(wireCard())
		_, _ = fmt.Fprintf(w, "event: card.upsert\ndata: %s\n\n", data)
		_, _ = fmt.Fprint(w, "event: batch.done\ndata: {}\n\n")
	}))
	defer server.Close()

	var out, errOut bytes.Buffer
	code := Run(context.Background(), []string{"feed"}, feedTestEnv(t), &out, &errOut, server.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "A card") || !strings.Contains(out.String(), "State: analyzed") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestFeedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/rating" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(contract.Feed{BatchID: "b1", State: "completed", Cards: []contract.FeedCard{wireCard()}})
	}))
	defer server.Close()

	var out, errOut bytes.Buffer
	code := Run(context.Background(), []string{"feed", "--json"}, feedTestEnv(t), &out, &errOut, server.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut.String())
	}
	var payload contract.Feed
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("stdout is not one JSON object: %v\n%s", err, out.String())
	}
	if payload.BatchID != "b1" || len(payload.Cards) != 1 {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestFeedUnhappy(t *testing.T) {
	t.Run("no feed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "no feed", http.StatusNotFound)
		}))
		defer server.Close()
		var out, errOut bytes.Buffer
		if code := Run(context.Background(), []string{"feed"}, feedTestEnv(t), &out, &errOut, server.URL); code != 6 {
			t.Fatalf("exit = %d, want 6", code)
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		}))
		defer server.Close()
		var out, errOut bytes.Buffer
		if code := Run(context.Background(), []string{"feed"}, feedTestEnv(t), &out, &errOut, server.URL); code != 2 {
			t.Fatalf("exit = %d, want 2", code)
		}
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "boom", http.StatusInternalServerError)
		}))
		defer server.Close()
		var out, errOut bytes.Buffer
		if code := Run(context.Background(), []string{"feed"}, feedTestEnv(t), &out, &errOut, server.URL); code != 1 {
			t.Fatalf("exit = %d, want 1", code)
		}
	})

	t.Run("bad flag", func(t *testing.T) {
		var out, errOut bytes.Buffer
		if code := Run(context.Background(), []string{"feed", "--bad"}, feedTestEnv(t), &out, &errOut, ""); code != 2 {
			t.Fatalf("exit = %d, want 2", code)
		}
	})
}

func TestFeedSSEBanner(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, `event: batch.started`+"\n"+`data: {"banner":"quota exhausted — cached feed"}`+"\n\n")
		_, _ = fmt.Fprint(w, "event: batch.done\ndata: {}\n\n")
	}))
	defer server.Close()

	var out, errOut bytes.Buffer
	code := Run(context.Background(), []string{"feed"}, feedTestEnv(t), &out, &errOut, server.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "banner: quota exhausted") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestFeedOnboardingErrors(t *testing.T) {
	tests := []struct {
		status int
		code   string
		text   string
	}{
		{status: 401, code: "not_authenticated", text: "ytfeed onboard"},
		{status: 409, code: "interests_not_set", text: "interests not set"},
		{status: 409, code: "profile_not_set", text: "taste profile not set"},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = fmt.Fprintf(w, `{"error":%q}`, tt.code)
			}))
			defer server.Close()
			var out, errOut bytes.Buffer
			if code := Run(context.Background(), []string{"feed"}, feedTestEnv(t), &out, &errOut, server.URL); code != 2 {
				t.Fatalf("exit = %d, want 2", code)
			}
			if !strings.Contains(errOut.String(), tt.text) {
				t.Fatalf("stderr = %q, want %q", errOut.String(), tt.text)
			}
		})
	}
}

func TestFeedRefreshHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/rating" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("X-Ytfeed-Refresh"); got != "true" {
			t.Errorf("refresh header = %q", got)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "event: batch.done\ndata: {}\n\n")
	}))
	defer server.Close()

	var out, errOut bytes.Buffer
	if code := Run(context.Background(), []string{"feed", "--refresh"}, feedTestEnv(t), &out, &errOut, server.URL); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut.String())
	}
}

func TestFeedSSEProgress(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/rating" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "event: progress\ndata: {\"stage\":\"search\",\"message\":\"searching: ai agents\"}\n\n")
		_, _ = fmt.Fprint(w, "event: batch.started\ndata: {\"batch_id\":\"b1\",\"state\":\"completed\",\"size\":1}\n\n")
		data, _ := json.Marshal(wireCard())
		_, _ = fmt.Fprintf(w, "event: card.upsert\ndata: %s\n\n", data)
		_, _ = fmt.Fprint(w, "event: batch.done\ndata: {\"batch_id\":\"b1\",\"state\":\"completed\"}\n\n")
	}))
	defer server.Close()

	var out, errOut bytes.Buffer
	if code := Run(context.Background(), []string{"feed"}, feedTestEnv(t), &out, &errOut, server.URL); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "searching: ai agents") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestFeedErrorEvent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/rating" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "event: error\ndata: {\"error\":\"not_authenticated\"}\n\n")
	}))
	defer server.Close()

	var out, errOut bytes.Buffer
	if code := Run(context.Background(), []string{"feed"}, feedTestEnv(t), &out, &errOut, server.URL); code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	if !strings.Contains(errOut.String(), "not authenticated") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestFeedErrorText(t *testing.T) {
	for code, want := range map[string]string{
		"not_authenticated": "not authenticated",
		"interests_not_set": "interests not set",
		"profile_not_set":   "taste profile not set",
		"no_feed":           "no feed available",
		"other":             "feed unavailable",
	} {
		if got := feedErrorText(code); !strings.Contains(got, want) {
			t.Fatalf("feedErrorText(%q) = %q", code, got)
		}
	}
}

func TestFeedSourceFlag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/rating" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("X-Ytfeed-Source"); got != "trending" {
			t.Errorf("source = %q", got)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "event: batch.done\ndata: {}\n\n")
	}))
	defer server.Close()
	var out, errOut bytes.Buffer
	if code := Run(context.Background(), []string{"feed", "--refresh", "--source", "trending"}, feedTestEnv(t), &out, &errOut, server.URL); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut.String())
	}
}

func TestFeedInvalidSource(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := Run(context.Background(), []string{"feed", "--source", "random"}, feedTestEnv(t), &out, &errOut, ""); code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
}
