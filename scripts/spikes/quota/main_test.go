package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMaxUnits(t *testing.T) {
	batch := plannedBatch{Interests: []string{"a", "b", "c", "d", "e"}, MaxResults: 50}
	if got, want := batch.maxUnits(), 510; got != want {
		t.Fatalf("maxUnits() = %d, want %d", got, want)
	}
}

func TestChunkStrings(t *testing.T) {
	values := make([]string, 120)
	for i := range values {
		values[i] = fmt.Sprintf("id-%d", i)
	}
	chunks := chunkStrings(values, 50)
	if len(chunks) != 3 || len(chunks[0]) != 50 || len(chunks[2]) != 20 {
		t.Fatalf("chunks = %d, sizes %d/%d", len(chunks), len(chunks[0]), len(chunks[2]))
	}
}

func TestDedupe(t *testing.T) {
	got := dedupe([]string{"a", "b", "a", "", "c", "b"})
	if strings.Join(got, ",") != "a,b,c" {
		t.Fatalf("dedupe = %v", got)
	}
}

func TestRunHappyPath(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") != "test-key" {
			t.Errorf("search key = %q", r.URL.Query().Get("key"))
		}
		_, _ = w.Write([]byte(`{"items":[
			{"id":{"videoId":"v1"},"snippet":{"channelId":"c1"}},
			{"id":{"videoId":"v2"},"snippet":{"channelId":"c2"}},
			{"id":{"videoId":"v1"},"snippet":{"channelId":"c1"}}
		]}`))
	})
	mux.HandleFunc("/videos", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"items":[
			{"id":"v1","snippet":{"channelId":"c1"}},
			{"id":"v2","snippet":{"channelId":"c3"}}
		]}`))
	})
	mux.HandleFunc("/channels", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"items":[{"id":"c1"},{"id":"c2"},{"id":"c3"}]}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	r := &runner{apiKey: "test-key", baseURL: server.URL, client: server.Client()}
	rep := run(context.Background(), r, plannedBatch{Interests: []string{"one"}, MaxResults: 50, Days: 30})

	if rep.TotalUnits != 102 {
		t.Fatalf("units = %d, want 102", rep.TotalUnits)
	}
	if rep.UniqueVideos != 2 || rep.UniqueChannels != 3 {
		t.Fatalf("unique videos/channels = %d/%d", rep.UniqueVideos, rep.UniqueChannels)
	}
	if rep.MeetsStrictTarget {
		t.Fatal("MeetsStrictTarget = true for 102 units")
	}
	if rep.BatchesPerDayAt80 != 78 {
		t.Fatalf("batches = %d, want 78", rep.BatchesPerDayAt80)
	}
	if !strings.Contains(rep.Verdict, "GO within the daily budget") {
		t.Fatalf("verdict = %q", rep.Verdict)
	}
}

func TestRunQuotaExhausted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"code":403,"message":"quota","errors":[{"reason":"quotaExceeded"}]}}`))
	}))
	defer server.Close()

	r := &runner{apiKey: "test-key", baseURL: server.URL, client: server.Client()}
	rep := run(context.Background(), r, plannedBatch{Interests: []string{"one", "two"}, MaxResults: 50, Days: 30})

	if !rep.QuotaExhausted {
		t.Fatal("QuotaExhausted = false")
	}
	if rep.TotalUnits != 0 {
		t.Fatalf("units = %d, want 0", rep.TotalUnits)
	}
	if len(rep.Calls) != 1 {
		t.Fatalf("calls = %d, want 1", len(rep.Calls))
	}
	if !strings.Contains(rep.Verdict, "INCOMPLETE") {
		t.Fatalf("verdict = %q", rep.Verdict)
	}
}

func TestRunRedactsKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"code":500,"message":"bad test-key"}}`))
	}))
	defer server.Close()

	r := &runner{apiKey: "test-key", baseURL: server.URL, client: server.Client()}
	rep := run(context.Background(), r, plannedBatch{Interests: []string{"one"}, MaxResults: 50, Days: 30})

	if len(rep.Calls) != 1 {
		t.Fatalf("calls = %d", len(rep.Calls))
	}
	if strings.Contains(rep.Calls[0].Error, "test-key") {
		t.Fatalf("error leaks the key: %q", rep.Calls[0].Error)
	}
	if !strings.Contains(rep.Calls[0].Error, "REDACTED") {
		t.Fatalf("error = %q, want REDACTED", rep.Calls[0].Error)
	}
}

func TestVerdicts(t *testing.T) {
	tests := []struct {
		name  string
		units int
		go_   bool
	}{
		{name: "100 units", units: 100, go_: true},
		{name: "505 units", units: 505, go_: true},
		{name: "5000 units", units: 5000, go_: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batches := usableBudget / tt.units
			if got := batches >= 3; got != tt.go_ {
				t.Fatalf("batches = %d, go = %t, want %t", batches, got, tt.go_)
			}
			if got := tt.units <= strictBatchTarget; got != (tt.units <= 100) {
				t.Fatalf("strict = %t", got)
			}
		})
	}
}
