// Package ytdlp fetches transcript text through yt-dlp.
package ytdlp

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/rktkov/ytfeed/internal/domain/analysis"
)

// Fetcher runs yt-dlp and returns transcript text from stdout.
type Fetcher struct {
	Bin string
}

// New returns a fetcher for the given binary.
func New(bin string) *Fetcher {
	if strings.TrimSpace(bin) == "" {
		bin = "yt-dlp"
	}
	return &Fetcher{Bin: bin}
}

// Fetch returns transcript text only.
func (f *Fetcher) Fetch(ctx context.Context, videoID string) (string, error) {
	if !analysis.ValidVideoID(videoID) {
		return "", analysis.ErrInvalidVideoID
	}
	args := []string{
		"--skip-download",
		"--write-auto-subs",
		"--write-subs",
		"--sub-langs", "en.*,en",
		"--sub-format", "vtt",
		"-o", "-",
		"https://www.youtube.com/watch?v=" + videoID,
	}
	var stdout bytes.Buffer
	cmd := exec.CommandContext(ctx, f.Bin, args...)
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %v", analysis.ErrNoTranscript, err)
	}
	text := strings.TrimSpace(stdout.String())
	if text == "" {
		return "", analysis.ErrNoTranscript
	}
	return text, nil
}

var _ analysis.Fetcher = (*Fetcher)(nil)
