package ytdlp

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rktkov/ytfeed/internal/domain/analysis"
	"github.com/rktkov/ytfeed/internal/testkit"
)

func TestDownload(t *testing.T) {
	dir := t.TempDir()
	script := testkit.WriteScript(t, dir, "yt-dlp", `#!/bin/bash
out=""
while [ $# -gt 0 ]; do
  if [ "$1" = "-o" ]; then out="$2"; fi
  shift
done
out="${out//\%(ext)s/mp4}"
printf 'media' > "$out"
`)
	result, err := NewDownloader(script).Download(context.Background(), "dQw4w9WgXcQ", filepath.Join(dir, "library"))
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if result.Extension != "mp4" || result.Bytes != 5 {
		t.Fatalf("result = %+v", result)
	}
	if _, err := os.Stat(filepath.Join(dir, "library", "dQw4w9WgXcQ.mp4")); err != nil {
		t.Fatalf("file: %v", err)
	}
}

func TestDownloadFailure(t *testing.T) {
	dir := t.TempDir()
	script := testkit.WriteScript(t, dir, "yt-dlp", "#!/bin/sh\nexit 1\n")
	if _, err := NewDownloader(script).Download(context.Background(), "dQw4w9WgXcQ", filepath.Join(dir, "library")); err == nil {
		t.Fatal("Download = nil, want an error")
	}
}

func TestDownloadNoFile(t *testing.T) {
	dir := t.TempDir()
	script := testkit.WriteScript(t, dir, "yt-dlp", "#!/bin/sh\nexit 0\n")
	if _, err := NewDownloader(script).Download(context.Background(), "dQw4w9WgXcQ", filepath.Join(dir, "library")); err == nil {
		t.Fatal("Download(no file) = nil, want an error")
	}
}

func TestDownloadInvalidID(t *testing.T) {
	if _, err := NewDownloader("/bin/true").Download(context.Background(), "short", t.TempDir()); !errors.Is(err, analysis.ErrInvalidVideoID) {
		t.Fatalf("Download = %v, want ErrInvalidVideoID", err)
	}
}
