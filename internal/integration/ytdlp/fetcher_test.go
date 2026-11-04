package ytdlp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/rktkov/ytfeed/internal/domain/analysis"
	"github.com/rktkov/ytfeed/internal/testkit"
)

const videoID = "dQw4w9WgXcQ"

func TestFetch(t *testing.T) {
	dir := t.TempDir()
	script := testkit.WriteScript(t, dir, "yt-dlp", "#!/bin/sh\nprintf 'hello transcript'\n")
	text, err := New(script).Fetch(context.Background(), videoID)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if text != "hello transcript" {
		t.Fatalf("text = %q", text)
	}
}

func TestFetchArgs(t *testing.T) {
	dir := t.TempDir()
	argsFile := dir + "/args.txt"
	script := testkit.WriteScript(t, dir, "yt-dlp", fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$@\" > %q\nprintf 'text'\n", argsFile))
	if _, err := New(script).Fetch(context.Background(), videoID); err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read args: %v", err)
	}
	args := string(data)
	if !strings.Contains(args, "--skip-download") {
		t.Fatalf("args = %q, missing --skip-download", args)
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
	script := testkit.WriteScript(t, dir, "yt-dlp", "#!/bin/sh\nexit 0\n")
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
