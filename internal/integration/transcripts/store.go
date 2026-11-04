// Package transcripts stages transcript text in a restricted temporary directory.
package transcripts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/analysis"
)

const defaultMaxBytes int64 = 8 << 20

// Store is the filesystem TranscriptStore adapter.
type Store struct {
	dir      string
	maxBytes int64
}

// New returns a store under <dataDir>/tmp/transcripts.
func New(dataDir string) *Store {
	return NewWithLimit(dataDir, defaultMaxBytes)
}

// NewWithLimit returns a store with an explicit size cap.
func NewWithLimit(dataDir string, maxBytes int64) *Store {
	if maxBytes <= 0 {
		maxBytes = defaultMaxBytes
	}
	return &Store{
		dir:      filepath.Join(dataDir, "tmp", "transcripts"),
		maxBytes: maxBytes,
	}
}

// Dir returns the staging directory.
func (s *Store) Dir() string {
	return s.dir
}

// Put writes one transcript atomically with 0600 permissions.
func (s *Store) Put(_ context.Context, videoID, text string) (analysis.Staged, error) {
	if !analysis.ValidVideoID(videoID) {
		return analysis.Staged{}, analysis.ErrInvalidVideoID
	}
	if int64(len(text)) > s.maxBytes {
		return analysis.Staged{}, analysis.ErrTooLarge
	}
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return analysis.Staged{}, fmt.Errorf("create transcript directory: %w", err)
	}
	if err := os.Chmod(s.dir, 0o700); err != nil {
		return analysis.Staged{}, fmt.Errorf("set transcript directory permissions: %w", err)
	}

	temp, err := os.CreateTemp(s.dir, videoID+".*.part")
	if err != nil {
		return analysis.Staged{}, fmt.Errorf("create transcript file: %w", err)
	}
	tempPath := temp.Name()
	defer func() { _ = os.Remove(tempPath) }()

	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return analysis.Staged{}, fmt.Errorf("set transcript permissions: %w", err)
	}
	if _, err := temp.WriteString(text); err != nil {
		_ = temp.Close()
		return analysis.Staged{}, fmt.Errorf("write transcript: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return analysis.Staged{}, fmt.Errorf("sync transcript: %w", err)
	}
	if err := temp.Close(); err != nil {
		return analysis.Staged{}, fmt.Errorf("close transcript: %w", err)
	}
	path := filepath.Join(s.dir, videoID+".txt")
	if err := os.Rename(tempPath, path); err != nil {
		return analysis.Staged{}, fmt.Errorf("replace transcript: %w", err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return analysis.Staged{}, fmt.Errorf("set transcript permissions: %w", err)
	}
	if dir, err := os.Open(s.dir); err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}

	sum := sha256.Sum256([]byte(text))
	return analysis.Staged{Hash: hex.EncodeToString(sum[:]), Bytes: int64(len(text)), Text: text}, nil
}

// Get reads one staged transcript and verifies its hash when wantHash is set.
func (s *Store) Get(_ context.Context, videoID, wantHash string) (analysis.Staged, error) {
	if !analysis.ValidVideoID(videoID) {
		return analysis.Staged{}, analysis.ErrInvalidVideoID
	}
	path := filepath.Join(s.dir, videoID+".txt")
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return analysis.Staged{}, analysis.ErrStageMiss
	}
	if err != nil {
		return analysis.Staged{}, fmt.Errorf("stat transcript: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return analysis.Staged{}, analysis.ErrStageMiss
	}
	if info.Size() > s.maxBytes {
		return analysis.Staged{}, analysis.ErrStageMiss
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return analysis.Staged{}, analysis.ErrStageMiss
	}
	if int64(len(data)) > s.maxBytes {
		return analysis.Staged{}, analysis.ErrStageMiss
	}
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	if wantHash != "" && !strings.EqualFold(hash, wantHash) {
		return analysis.Staged{}, analysis.ErrStageMiss
	}
	return analysis.Staged{Hash: hash, Bytes: int64(len(data)), Text: string(data)}, nil
}

// Delete removes a staged transcript. A missing file is not an error.
func (s *Store) Delete(_ context.Context, videoID string) error {
	if !analysis.ValidVideoID(videoID) {
		return analysis.ErrInvalidVideoID
	}
	err := os.Remove(filepath.Join(s.dir, videoID+".txt"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("delete transcript: %w", err)
	}
	return nil
}

// Sweep removes staged and partial files older than the cutoff.
func (s *Store) Sweep(_ context.Context, olderThan time.Time) (int, error) {
	entries, err := os.ReadDir(s.dir)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read transcript directory: %w", err)
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".txt") && !strings.HasSuffix(name, ".part") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(olderThan) {
			continue
		}
		if err := os.Remove(filepath.Join(s.dir, name)); err == nil {
			count++
		}
	}
	return count, nil
}

var _ analysis.TranscriptStore = (*Store)(nil)
