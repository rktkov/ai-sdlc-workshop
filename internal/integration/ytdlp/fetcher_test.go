package ytdlp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/rktkov/ytfeed/internal/domain/analysis"
	"github.com/rktkov/ytfeed/internal/testkit"
)

const videoID = "dQw4w9WgXcQ"

func TestFetchJSON3(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"events":[{"segs":[{"utf8":"hello "},{"utf8":"world"}]}]}`)
	}))
	defer server.Close()

	script := captionScript(t, t.TempDir(), "json3", server.URL)
	text, err := New(script).Fetch(context.Background(), videoID)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if text != "hello world" {
		t.Fatalf("text = %q", text)
	}
}

func TestFetchVTT(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, "WEBVTT\n\n00:00:00.000 --> 00:00:02.000\nhello world\n")
	}))
	defer server.Close()

	script := captionScript(t, t.TempDir(), "vtt", server.URL)
	text, err := New(script).Fetch(context.Background(), videoID)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if text != "hello world" {
		t.Fatalf("text = %q", text)
	}
}

func TestFetchArgs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"events":[]}`)
	}))
	defer server.Close()

	dir := t.TempDir()
	argsFile := dir + "/args.txt"
	body := captionBody("json3", server.URL)
	script := testkit.WriteScript(t, dir, "yt-dlp",
		fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$@\" > %q\nprintf '%%s' '%s'\n", argsFile, body))
	if _, err := New(script).Fetch(context.Background(), videoID); err != nil && !errors.Is(err, analysis.ErrNoTranscript) {
		t.Fatalf("Fetch: %v", err)
	}
	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read args: %v", err)
	}
	args := string(data)
	if !strings.Contains(args, "--skip-download") || !strings.Contains(args, "--print") {
		t.Fatalf("args = %q", args)
	}
	if !strings.Contains(args, "https://www.youtube.com/watch?v="+videoID) {
		t.Fatalf("args = %q, missing URL", args)
	}
}

func TestFetchNoTranscript(t *testing.T) {
	dir := t.TempDir()
	script := testkit.WriteScript(t, dir, "yt-dlp", "#!/bin/sh\nexit 1\n")
	if _, err := New(script).Fetch(context.Background(), videoID); !errors.Is(err, analysis.ErrNoTranscript) {
		t.Fatalf("Fetch = %v, want ErrNoTranscript", err)
	}
}

func TestFetchEmpty(t *testing.T) {
	dir := t.TempDir()
	script := testkit.WriteScript(t, dir, "yt-dlp", "#!/bin/sh\nprintf '{}'\n")
	if _, err := New(script).Fetch(context.Background(), videoID); !errors.Is(err, analysis.ErrNoTranscript) {
		t.Fatalf("Fetch = %v, want ErrNoTranscript", err)
	}
}

func TestFetchHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	script := captionScript(t, t.TempDir(), "json3", server.URL)
	if _, err := New(script).Fetch(context.Background(), videoID); !errors.Is(err, analysis.ErrNoTranscript) {
		t.Fatalf("Fetch = %v, want ErrNoTranscript", err)
	}
}

func TestFetchInvalidID(t *testing.T) {
	if _, err := New("/bin/true").Fetch(context.Background(), "short"); !errors.Is(err, analysis.ErrInvalidVideoID) {
		t.Fatalf("Fetch = %v, want ErrInvalidVideoID", err)
	}
}

func TestNewDefault(t *testing.T) {
	if New("").Bin != "yt-dlp" {
		t.Fatalf("Bin = %q", New("").Bin)
	}
}

func captionScript(t *testing.T, dir, ext, baseURL string) string {
	t.Helper()
	return testkit.WriteScript(t, dir, "yt-dlp",
		fmt.Sprintf("#!/bin/sh\nprintf '%%s' '%s'\n", captionBody(ext, baseURL)))
}

func captionBody(ext, baseURL string) string {
	return fmt.Sprintf(`{"en":[{"ext":%q,"url":%q}]}`, ext, baseURL+"/caps")
}
