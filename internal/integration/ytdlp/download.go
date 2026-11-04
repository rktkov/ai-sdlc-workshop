package ytdlp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rktkov/ytfeed/internal/domain/analysis"
	"github.com/rktkov/ytfeed/internal/domain/library"
)

// Downloader downloads one video with yt-dlp.
type Downloader struct {
	Bin string
}

// NewDownloader returns a downloader for the given binary.
func NewDownloader(bin string) *Downloader {
	if strings.TrimSpace(bin) == "" {
		bin = "yt-dlp"
	}
	return &Downloader{Bin: bin}
}

// Download writes one media file and returns its extension and size.
func (d *Downloader) Download(ctx context.Context, videoID, directory string) (library.DownloadResult, error) {
	if !analysis.ValidVideoID(videoID) {
		return library.DownloadResult{}, analysis.ErrInvalidVideoID
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return library.DownloadResult{}, fmt.Errorf("create library directory: %w", err)
	}
	template := filepath.Join(directory, videoID+".%(ext)s")
	cmd := exec.CommandContext(ctx, d.Bin,
		"--no-playlist", "-f", "best", "-o", template,
		"https://www.youtube.com/watch?v="+videoID,
	)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return library.DownloadResult{}, fmt.Errorf("download video: %w", err)
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		return library.DownloadResult{}, fmt.Errorf("read library directory: %w", err)
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), videoID+".") || strings.HasSuffix(entry.Name(), ".part") {
			continue
		}
		names = append(names, entry.Name())
	}
	if len(names) == 0 {
		return library.DownloadResult{}, fmt.Errorf("download produced no file")
	}
	sort.Strings(names)
	name := names[0]
	info, err := os.Stat(filepath.Join(directory, name))
	if err != nil {
		return library.DownloadResult{}, fmt.Errorf("stat download: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 {
		return library.DownloadResult{}, fmt.Errorf("download produced no media")
	}
	extension := strings.TrimPrefix(name, videoID+".")
	return library.DownloadResult{Extension: extension, Bytes: info.Size()}, nil
}

var _ library.Downloader = (*Downloader)(nil)
