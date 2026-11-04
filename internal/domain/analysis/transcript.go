package analysis

import (
	"context"
	"errors"
	"regexp"
	"time"
)

var (
	// ErrStageMiss means a staged transcript is missing, unreadable or corrupt.
	ErrStageMiss = errors.New("transcript stage miss")
	// ErrTooLarge means the transcript exceeds the size cap.
	ErrTooLarge = errors.New("transcript too large")
	// ErrInvalidVideoID means the ID cannot be used as a staging key.
	ErrInvalidVideoID = errors.New("invalid video id")
	// ErrNoTranscript means the fetcher produced no transcript.
	ErrNoTranscript = errors.New("no transcript")
)

var videoIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)

// ValidVideoID reports whether the ID is safe to use as a staging key.
func ValidVideoID(videoID string) bool {
	return videoIDPattern.MatchString(videoID)
}

// Staged is one staged transcript.
type Staged struct {
	Hash  string
	Bytes int64
	Text  string
}

// TranscriptStore stages transcript text between fetch and analysis commit.
type TranscriptStore interface {
	Put(ctx context.Context, videoID, text string) (Staged, error)
	Get(ctx context.Context, videoID, wantHash string) (Staged, error)
	Delete(ctx context.Context, videoID string) error
	Sweep(ctx context.Context, olderThan time.Time) (int, error)
}

// Fetcher obtains transcript text only.
type Fetcher interface {
	Fetch(ctx context.Context, videoID string) (string, error)
}
