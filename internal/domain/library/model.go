// Package library owns saved and downloaded videos.
package library

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// State is saved or downloaded.
type State string

const (
	StateSaved      State = "saved"
	StateDownloaded State = "downloaded"
)

var (
	// ErrNotFound means the entry does not exist.
	ErrNotFound = errors.New("library entry not found")
	// ErrExists means the entry already exists.
	ErrExists = errors.New("library entry already exists")
)

var (
	videoIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
	extPattern     = regexp.MustCompile(`^[a-z0-9]{1,8}$`)
)

// Entry is one saved or downloaded video.
type Entry struct {
	UserID          string
	VideoID         string
	State           State
	Title           string
	ChannelID       string
	DurationSeconds int64
	FileExtension   string
	FileBytes       int64
	SavedAt         time.Time
	DownloadedAt    time.Time
	UpdatedAt       time.Time
}

// Validate checks the entry invariants.
func (e Entry) Validate() error {
	if strings.TrimSpace(e.UserID) == "" || !videoIDPattern.MatchString(e.VideoID) {
		return errors.New("library entry has no user or valid video id")
	}
	if length := len(strings.TrimSpace(e.Title)); length < 1 || length > 500 {
		return errors.New("library title must be 1 to 500 characters")
	}
	if e.DurationSeconds < 0 {
		return errors.New("library duration is negative")
	}
	switch e.State {
	case StateSaved:
		if e.FileExtension != "" || e.FileBytes != 0 || !e.DownloadedAt.IsZero() {
			return errors.New("saved entry has file fields")
		}
	case StateDownloaded:
		if !extPattern.MatchString(e.FileExtension) || e.FileBytes <= 0 || e.DownloadedAt.IsZero() {
			return errors.New("downloaded entry has invalid file fields")
		}
	default:
		return fmt.Errorf("library entry has an invalid state %q", e.State)
	}
	return nil
}

// DownloadResult is one completed media file.
type DownloadResult struct {
	Extension string
	Bytes     int64
}

// Repository stores library entries.
type Repository interface {
	Get(ctx context.Context, userID, videoID string) (Entry, error)
	List(ctx context.Context, userID string) ([]Entry, error)
	Save(ctx context.Context, entry Entry) error
	MarkDownloaded(ctx context.Context, userID, videoID, extension string, bytes int64, now time.Time) (Entry, error)
	Remove(ctx context.Context, userID, videoID string) error
}

// FileStore owns media files under the library directory.
type FileStore interface {
	Dir() string
	Path(videoID, extension string) string
	Stat(videoID, extension string) (int64, error)
	Delete(videoID, extension string) error
}

// Downloader downloads one video explicitly.
type Downloader interface {
	Download(ctx context.Context, videoID, directory string) (DownloadResult, error)
}

// Service applies library use cases.
type Service struct {
	Repo       Repository
	Files      FileStore
	Downloader Downloader
	Now        func() time.Time
}

// Save inserts one saved entry. A duplicate returns the stored entry.
func (s *Service) Save(ctx context.Context, entry Entry) (Entry, error) {
	if s.Repo == nil {
		return Entry{}, errors.New("library repository is nil")
	}
	now := s.now()
	entry.State = StateSaved
	entry.FileExtension = ""
	entry.FileBytes = 0
	entry.DownloadedAt = time.Time{}
	entry.SavedAt = now
	entry.UpdatedAt = now
	if err := entry.Validate(); err != nil {
		return Entry{}, err
	}
	if err := s.Repo.Save(ctx, entry); err != nil {
		if errors.Is(err, ErrExists) {
			return s.Repo.Get(ctx, entry.UserID, entry.VideoID)
		}
		return Entry{}, err
	}
	return entry, nil
}

// Download saves the media first, then marks the entry downloaded.
func (s *Service) Download(ctx context.Context, userID, videoID string) (Entry, error) {
	if s.Repo == nil || s.Files == nil || s.Downloader == nil {
		return Entry{}, errors.New("library service is not configured")
	}
	entry, err := s.Repo.Get(ctx, userID, videoID)
	if err != nil {
		return Entry{}, err
	}
	if entry.State == StateDownloaded {
		return entry, nil
	}
	result, err := s.Downloader.Download(ctx, videoID, s.Files.Dir())
	if err != nil {
		return Entry{}, err
	}
	if !extPattern.MatchString(result.Extension) {
		return Entry{}, errors.New("downloader returned an invalid extension")
	}
	bytes, err := s.Files.Stat(videoID, result.Extension)
	if err != nil {
		return Entry{}, err
	}
	updated, err := s.Repo.MarkDownloaded(ctx, userID, videoID, result.Extension, bytes, s.now())
	if err != nil {
		return Entry{}, err
	}
	return updated, nil
}

// Remove deletes the file and then the entry.
func (s *Service) Remove(ctx context.Context, userID, videoID string) error {
	if s.Repo == nil || s.Files == nil {
		return errors.New("library service is not configured")
	}
	entry, err := s.Repo.Get(ctx, userID, videoID)
	if err != nil {
		return err
	}
	if entry.State == StateDownloaded {
		if err := s.Files.Delete(videoID, entry.FileExtension); err != nil {
			return err
		}
	}
	return s.Repo.Remove(ctx, userID, videoID)
}

// List returns the library ordered by save time.
func (s *Service) List(ctx context.Context, userID string) ([]Entry, error) {
	if s.Repo == nil {
		return nil, errors.New("library repository is nil")
	}
	return s.Repo.List(ctx, userID)
}

func (s *Service) now() time.Time {
	if s.Now == nil {
		return time.Now().UTC()
	}
	return s.Now().UTC()
}
