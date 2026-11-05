package feed

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// RatingValue is the weekly quality answer.
type RatingValue string

const (
	RatingWorse   RatingValue = "-1"
	RatingSame    RatingValue = "0"
	RatingBetter  RatingValue = "1"
	RatingUnrated RatingValue = "unrated"
)

var (
	// ErrRatingNotFound means no rating exists for the week.
	ErrRatingNotFound = errors.New("weekly rating not found")
	// ErrRatingExists means the week already has a rating.
	ErrRatingExists = errors.New("weekly rating already exists")
)

// WeeklyRating is one weekly quality answer.
type WeeklyRating struct {
	ID        string
	UserID    string
	WeekStart time.Time
	Value     RatingValue
	BatchIDs  []string
	CreatedAt time.Time
}

// RatingRepository stores weekly ratings.
type RatingRepository interface {
	FindByWeek(ctx context.Context, userID string, weekStart time.Time) (WeeklyRating, error)
	LastForUser(ctx context.Context, userID string) (WeeklyRating, error)
	Save(ctx context.Context, rating WeeklyRating) error
}

// RatingService decides and records the weekly prompt.
type RatingService struct {
	Repo RatingRepository
	Now  func() time.Time
	ID   func() string
}

// ShouldPrompt reports whether the current week has no rating.
func (s *RatingService) ShouldPrompt(ctx context.Context, userID string) (bool, error) {
	if s.Repo == nil {
		return false, errors.New("rating repository is nil")
	}
	last, err := s.Repo.LastForUser(ctx, userID)
	if errors.Is(err, ErrRatingNotFound) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return last.WeekStart.Before(s.weekStart()), nil
}

// Record stores one rating for the current week.
func (s *RatingService) Record(ctx context.Context, userID string, value RatingValue, batchIDs []string) (WeeklyRating, error) {
	if s.Repo == nil {
		return WeeklyRating{}, errors.New("rating repository is nil")
	}
	now := s.now()
	rating := WeeklyRating{
		ID: s.newID(), UserID: userID, WeekStart: WeekStart(now),
		Value: value, BatchIDs: append([]string(nil), batchIDs...), CreatedAt: now,
	}
	if err := rating.Validate(); err != nil {
		return WeeklyRating{}, err
	}
	if err := s.Repo.Save(ctx, rating); err != nil {
		if errors.Is(err, ErrRatingExists) {
			return s.Repo.FindByWeek(ctx, userID, rating.WeekStart)
		}
		return WeeklyRating{}, err
	}
	return rating, nil
}

// Dismiss records the current week as unrated.
func (s *RatingService) Dismiss(ctx context.Context, userID string, batchIDs []string) (WeeklyRating, error) {
	return s.Record(ctx, userID, RatingUnrated, batchIDs)
}

// WeekStart returns the Monday of the ISO week in UTC.
func WeekStart(instant time.Time) time.Time {
	utc := instant.UTC()
	day := time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
	weekday := int(day.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return day.AddDate(0, 0, -(weekday - 1))
}

// Validate checks the rating invariants.
func (r WeeklyRating) Validate() error {
	if r.ID == "" || r.UserID == "" {
		return errors.New("rating has no id or user")
	}
	switch r.Value {
	case RatingWorse, RatingSame, RatingBetter, RatingUnrated:
	default:
		return fmt.Errorf("rating has an invalid value %q", r.Value)
	}
	if !r.WeekStart.Equal(WeekStart(r.WeekStart)) {
		return errors.New("rating week does not start on Monday")
	}
	if len(r.BatchIDs) == 0 {
		return errors.New("rating has no batch ids")
	}
	seen := map[string]bool{}
	for _, id := range r.BatchIDs {
		if id == "" {
			return errors.New("rating has an empty batch id")
		}
		if seen[id] {
			return fmt.Errorf("batch %q is duplicated", id)
		}
		seen[id] = true
	}
	return nil
}

func (s *RatingService) now() time.Time {
	if s.Now == nil {
		return time.Now().UTC()
	}
	return s.Now().UTC()
}

func (s *RatingService) weekStart() time.Time {
	return WeekStart(s.now())
}

func (s *RatingService) newID() string {
	if s.ID != nil {
		return s.ID()
	}
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("rating-%d", time.Now().UnixNano())
	}
	return "rating-" + hex.EncodeToString(buf)
}
