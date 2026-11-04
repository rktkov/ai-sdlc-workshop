//go:build integration

package e2e

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"

	"github.com/rktkov/ytfeed/internal/testkit"
)

func TestFakeYouTube(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// The test client checks the body. A write error is not actionable here.
		_, _ = w.Write([]byte(`{"items":[{"id":"video-1"}]}`))
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("get fake YouTube: %v", err)
	}
	defer func() { _ = resp.Body.Close() }() // test response, no recovery path

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read fake YouTube: %v", err)
	}

	var payload struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode fake YouTube: %v", err)
	}
	if len(payload.Items) != 1 || payload.Items[0].ID != "video-1" {
		t.Fatalf("unexpected payload: %s", raw)
	}
}

func TestFakeLLM(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// The test client checks the body. A write error is not actionable here.
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"topic\":\"ai\"}"}}]}`))
	}))
	defer server.Close()

	resp, err := http.Post(server.URL, "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("post fake LLM: %v", err)
	}
	defer func() { _ = resp.Body.Close() }() // test response, no recovery path

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read fake LLM: %v", err)
	}
	if !strings.Contains(string(raw), "choices") {
		t.Fatalf("unexpected payload: %s", raw)
	}
}

func TestFakeScripts(t *testing.T) {
	dir := t.TempDir()
	path := testkit.WriteScript(t, dir, "yt-dlp", "#!/bin/sh\nprintf 'transcript'\n")

	out, err := exec.Command(path).CombinedOutput()
	if err != nil {
		t.Fatalf("run fake yt-dlp: %v: %s", err, out)
	}
	if string(out) != "transcript" {
		t.Fatalf("fake yt-dlp output = %q", out)
	}
}
