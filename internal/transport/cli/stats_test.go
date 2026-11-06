package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rktkov/ytfeed/internal/transport/contract"
)

func TestStatsText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"swipes":{"left":2},"reasons":{"marketing":1},"cold_active":1,"cold_expired":2,"quota":[{"day":"2026-11-05","charged":100}],"topics":{"ai":3}}`)
	}))
	defer server.Close()

	var out, errOut bytes.Buffer
	if code := Run(context.Background(), []string{"stats"}, feedTestEnv(t), &out, &errOut, server.URL); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut.String())
	}
	for _, want := range []string{"Swipes: left=2", "Cold periods: active=1 expired=2", "2026-11-05 100", "ai 3"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, out.String())
		}
	}
}

func TestStatsJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"swipes":{"left":2},"reasons":{},"cold_active":0,"cold_expired":0,"quota":[],"topics":{}}`)
	}))
	defer server.Close()

	var out, errOut bytes.Buffer
	if code := Run(context.Background(), []string{"stats", "--json"}, feedTestEnv(t), &out, &errOut, server.URL); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut.String())
	}
	var payload contract.Stats
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("stdout is not one JSON object: %v", err)
	}
	if payload.Swipes["left"] != 2 {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestStatsUnhappy(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := Run(context.Background(), []string{"stats", "--days", "0"}, feedTestEnv(t), &out, &errOut, ""); code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()
	if code := Run(context.Background(), []string{"stats"}, feedTestEnv(t), &out, &errOut, server.URL); code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
}

func TestRateStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/rating" {
			t.Errorf("%s %s", r.Method, r.URL.Path)
		}
		_, _ = fmt.Fprint(w, `{"should_prompt":true,"week_start":"2026-11-02"}`)
	}))
	defer server.Close()

	var out, errOut bytes.Buffer
	if code := Run(context.Background(), []string{"rate"}, feedTestEnv(t), &out, &errOut, server.URL); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "weekly rating: due") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestRateRecord(t *testing.T) {
	var posted map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			_ = json.NewDecoder(r.Body).Decode(&posted)
			_, _ = fmt.Fprint(w, `{"rating_id":"r1","week_start":"2026-11-02","value":"1","batch_ids":["b1"]}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	var out, errOut bytes.Buffer
	if code := Run(context.Background(), []string{"rate", "--value", "1", "--batch-id", "b1"}, feedTestEnv(t), &out, &errOut, server.URL); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut.String())
	}
	if posted["value"] != "1" {
		t.Fatalf("posted = %+v", posted)
	}
	if !strings.Contains(out.String(), "rating recorded") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestRateDismissAndErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["value"] != "unrated" {
				t.Errorf("value = %v", body["value"])
			}
			_, _ = fmt.Fprint(w, `{"rating_id":"r1","week_start":"2026-11-02","value":"unrated","batch_ids":["b1"]}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	var out, errOut bytes.Buffer
	if code := Run(context.Background(), []string{"rate", "--dismiss", "--batch-id", "b1"}, feedTestEnv(t), &out, &errOut, server.URL); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut.String())
	}
	if code := Run(context.Background(), []string{"rate", "--value", "2"}, feedTestEnv(t), &out, &errOut, server.URL); code != 2 {
		t.Fatalf("invalid value exit = %d, want 2", code)
	}
	if code := Run(context.Background(), []string{"rate", "--value", "1", "--dismiss"}, feedTestEnv(t), &out, &errOut, server.URL); code != 2 {
		t.Fatalf("conflicting flags exit = %d, want 2", code)
	}
}

func TestRateUsesCurrentBatch(t *testing.T) {
	var posted map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/feed":
			_, _ = fmt.Fprint(w, `{"batch_id":"b-live","state":"completed","cards":[]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/rating":
			_ = json.NewDecoder(r.Body).Decode(&posted)
			_, _ = fmt.Fprint(w, `{"rating_id":"r1","week_start":"2026-11-02","value":"0","batch_ids":["b-live"]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var out, errOut bytes.Buffer
	if code := Run(context.Background(), []string{"rate", "--value", "0"}, feedTestEnv(t), &out, &errOut, server.URL); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut.String())
	}
	ids, ok := posted["batch_ids"].([]any)
	if !ok || len(ids) != 1 || ids[0] != "b-live" {
		t.Fatalf("posted = %+v", posted)
	}
}

func TestFeedWeeklyPromptHint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/rating" {
			_, _ = io.WriteString(w, `{"should_prompt":true,"week_start":"2026-11-02"}`)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "event: batch.done\ndata: {}\n\n")
	}))
	defer server.Close()

	var out, errOut bytes.Buffer
	if code := Run(context.Background(), []string{"feed"}, feedTestEnv(t), &out, &errOut, server.URL); code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "weekly rating due") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}
