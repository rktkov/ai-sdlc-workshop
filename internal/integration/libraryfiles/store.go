// Package libraryfiles owns media files under the data directory.
package libraryfiles

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/rktkov/ytfeed/internal/domain/library"
)

var (
	videoIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
	extPattern     = regexp.MustCompile(`^[a-z0-9]{1,8}$`)
)

// Store keeps media files at <dataDir>/library/<video_id>.<ext>.
type Store struct {
	dir string
}

// New returns a file store under dataDir.
func New(dataDir string) *Store {
	return &Store{dir: filepath.Join(dataDir, "library")}
}

// Dir returns the library directory.
func (s *Store) Dir() string {
	return s.dir
}

// Path returns the derived file path.
func (s *Store) Path(videoID, extension string) string {
	if extension == "" {
		return filepath.Join(s.dir, videoID)
	}
	return filepath.Join(s.dir, videoID+"."+extension)
}

// Stat returns the file size.
func (s *Store) Stat(videoID, extension string) (int64, error) {
	if !videoIDPattern.MatchString(videoID) || !extPattern.MatchString(extension) {
		return 0, errors.New("invalid library file name")
	}
	info, err := os.Stat(s.Path(videoID, extension))
	if errors.Is(err, os.ErrNotExist) {
		return 0, library.ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("stat library file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return 0, errors.New("library file is not regular")
	}
	return info.Size(), nil
}

// Delete removes the file. A missing file is not an error.
func (s *Store) Delete(videoID, extension string) error {
	if !videoIDPattern.MatchString(videoID) || !extPattern.MatchString(extension) {
		return errors.New("invalid library file name")
	}
	err := os.Remove(s.Path(videoID, extension))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("delete library file: %w", err)
	}
	return nil
}

// EnsureDir creates the library directory with 0700 permissions.
func (s *Store) EnsureDir() error {
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return fmt.Errorf("create library directory: %w", err)
	}
	return os.Chmod(s.dir, 0o700)
}

var _ library.FileStore = (*Store)(nil)
