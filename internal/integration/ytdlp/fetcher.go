// Package ytdlp fetches transcript text through yt-dlp metadata.
package ytdlp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/analysis"
)

// Fetcher reads caption metadata with yt-dlp and the caption text over HTTP.
type Fetcher struct {
	Bin        string
	HTTPClient *http.Client
}

// New returns a fetcher for the given binary.
func New(bin string) *Fetcher {
	if strings.TrimSpace(bin) == "" {
		bin = "yt-dlp"
	}
	return &Fetcher{Bin: bin, HTTPClient: &http.Client{Timeout: 30 * time.Second}}
}

type captionFormat struct {
	Ext      string `json:"ext"`
	URL      string `json:"url"`
	Protocol string `json:"protocol"`
}

type captions map[string][]captionFormat

// Fetch returns transcript text only. It never writes a subtitle file.
func (f *Fetcher) Fetch(ctx context.Context, videoID string) (string, error) {
	if !analysis.ValidVideoID(videoID) {
		return "", analysis.ErrInvalidVideoID
	}
	raw, err := f.captionJSON(ctx, videoID, "automatic_captions")
	if err != nil {
		return "", err
	}
	caps, err := parseCaptions(raw)
	if err != nil {
		return "", err
	}
	if len(caps) == 0 {
		raw, err = f.captionJSON(ctx, videoID, "subtitles")
		if err != nil {
			return "", err
		}
		caps, err = parseCaptions(raw)
		if err != nil {
			return "", err
		}
	}
	format, ok := pickEnglish(caps)
	if !ok {
		return "", analysis.ErrNoTranscript
	}
	text, err := f.download(ctx, format)
	if err != nil {
		return "", err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return "", analysis.ErrNoTranscript
	}
	return text, nil
}

func (f *Fetcher) captionJSON(ctx context.Context, videoID, field string) ([]byte, error) {
	args := []string{
		"--skip-download",
		"--no-warnings",
		"--no-playlist",
		"--print", "%(" + field + ")j",
		"https://www.youtube.com/watch?v=" + videoID,
	}
	var stdout bytes.Buffer
	cmd := exec.CommandContext(ctx, f.Bin, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w: %v", analysis.ErrNoTranscript, err)
	}
	return stdout.Bytes(), nil
}

func parseCaptions(raw []byte) (captions, error) {
	var caps captions
	if err := json.Unmarshal(raw, &caps); err != nil {
		return nil, fmt.Errorf("%w: decode captions", analysis.ErrNoTranscript)
	}
	return caps, nil
}

func pickEnglish(caps captions) (captionFormat, bool) {
	keys := []string{"en", "en-US", "en-GB"}
	for _, key := range keys {
		if formats, ok := caps[key]; ok {
			if format, ok := pickFormat(formats); ok {
				return format, true
			}
		}
	}
	for language, formats := range caps {
		if !strings.HasPrefix(language, "en") {
			continue
		}
		if format, ok := pickFormat(formats); ok {
			return format, true
		}
	}
	return captionFormat{}, false
}

func pickFormat(formats []captionFormat) (captionFormat, bool) {
	for _, format := range formats {
		if format.Ext == "json3" && format.URL != "" {
			return format, true
		}
	}
	for _, format := range formats {
		if format.Ext == "vtt" && format.URL != "" && !strings.Contains(format.URL, "hls_timedtext") {
			return format, true
		}
	}
	return captionFormat{}, false
}

func (f *Fetcher) download(ctx context.Context, format captionFormat) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, format.URL, nil)
	if err != nil {
		return "", fmt.Errorf("%w: create caption request", analysis.ErrNoTranscript)
	}
	client := f.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: download captions", analysis.ErrNoTranscript)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", analysis.ErrNoTranscript
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return "", analysis.ErrNoTranscript
	}
	if format.Ext == "json3" {
		return parseJSON3(body)
	}
	return parseVTT(body), nil
}

func parseJSON3(body []byte) (string, error) {
	var payload struct {
		Events []struct {
			Segs []struct {
				UTF8 string `json:"utf8"`
			} `json:"segs"`
		} `json:"events"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("%w: decode json3 captions", analysis.ErrNoTranscript)
	}
	var builder strings.Builder
	for _, event := range payload.Events {
		for _, segment := range event.Segs {
			builder.WriteString(segment.UTF8)
		}
		builder.WriteString("\n")
	}
	return strings.TrimSpace(builder.String()), nil
}

func parseVTT(body []byte) string {
	var lines []string
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "WEBVTT") || strings.HasPrefix(line, "NOTE") {
			continue
		}
		if strings.Contains(line, "-->") {
			continue
		}
		line = strings.ReplaceAll(line, "<", " ")
		line = strings.ReplaceAll(line, ">", " ")
		lines = append(lines, strings.Join(strings.Fields(line), " "))
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

var _ analysis.Fetcher = (*Fetcher)(nil)
