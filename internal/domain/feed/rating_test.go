package feed

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeRatingRepo struct {
	items map[string]WeeklyRating
	err   error
}

func newFakeRatingRepo() *fakeRatingRepo {
	return &fakeRatingRepo{items: map[string]WeeklyRating{}}
}

func ratingKey(userID string, week time.Time) string {
	return userID + "|" + week.Format("2006-01-02")
}

func (f *fakeRatingRepo) FindByWeek(_ context.Context, userID string, week time.Time) (WeeklyRating, error) {
	if f.err != nil {
		return WeeklyRating{}, f.err
	}
	value, ok := f.items[ratingKey(userID, week)]
	if !ok {
		return WeeklyRating{}, ErrRatingNotFound
	}
	return value, nil
}

func (f *fakeRatingRepo) LastForUser(_ context.Context, userID string) (WeeklyRating, error) {
	if f.err != nil {
		return WeeklyRating{}, f.err
	}
	var latest WeeklyRating
	found := false
	for _, value := range f.items {
		if value.UserID != userID {
			continue
		}
		if !found || value.WeekStart.After(latest.WeekStart) {
			latest = value
			found = true
		}
	}
	if !found {
		return WeeklyRating{}, ErrRatingNotFound
	}
	return latest, nil
}

func (f *fakeRatingRepo) Save(_ context.Context, rating WeeklyRating) error {
	if f.err != nil {
		return f.err
	}
	key := ratingKey(rating.UserID, rating.WeekStart)
	if _, ok := f.items[key]; ok {
		return ErrRatingExists
	}
	f.items[key] = rating
	return nil
}

func TestWeekStart(t *testing.T) {
	for input, want := range map[string]string{
		"2026-11-05T12:00:00Z": "2026-11-02",
		"2026-11-02T00:00:00Z": "2026-11-02",
		"2026-11-08T23:59:00Z": "2026-11-02",
	} {
		parsed, _ := time.Parse(time.RFC3339, input)
		if got := WeekStart(parsed).Format("2006-01-02"); got != want {
			t.Fatalf("WeekStart(%s) = %s, want %s", input, got, want)
		}
	}
}

func TestRatingValidate(t *testing.T) {
	monday := WeekStart(time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC))
	valid := WeeklyRating{ID: "r", UserID: "owner", WeekStart: monday, Value: RatingBetter, BatchIDs: []string{"b1"}}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	tests := []struct {
		name string
		edit func(*WeeklyRating)
	}{
		{name: "no id", edit: func(r *WeeklyRating) { r.ID = "" }},
		{name: "no user", edit: func(r *WeeklyRating) { r.UserID = "" }},
		{name: "bad value", edit: func(r *WeeklyRating) { r.Value = "2" }},
		{name: "not monday", edit: func(r *WeeklyRating) { r.WeekStart = r.WeekStart.AddDate(0, 0, 1) }},
		{name: "no batches", edit: func(r *WeeklyRating) { r.BatchIDs = nil }},
		{name: "empty batch", edit: func(r *WeeklyRating) { r.BatchIDs = []string{""} }},
		{name: "duplicate batch", edit: func(r *WeeklyRating) { r.BatchIDs = []string{"b1", "b1"} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := valid
			tt.edit(&value)
			if err := value.Validate(); err == nil {
				t.Fatal("Validate = nil, want an error")
			}
		})
	}
}

func TestRatingPrompt(t *testing.T) {
	repo := newFakeRatingRepo()
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	service := &RatingService{Repo: repo, Now: func() time.Time { return now }, ID: func() string { return "r1" }}
	prompt, err := service.ShouldPrompt(context.Background(), "owner")
	if err != nil || !prompt {
		t.Fatalf("ShouldPrompt = %t, %v", prompt, err)
	}
	if _, err := service.Record(context.Background(), "owner", RatingBetter, []string{"b1"}); err != nil {
		t.Fatalf("Record: %v", err)
	}
	prompt, err = service.ShouldPrompt(context.Background(), "owner")
	if err != nil || prompt {
		t.Fatalf("ShouldPrompt after rating = %t, %v", prompt, err)
	}
	now = now.AddDate(0, 0, 7)
	prompt, err = service.ShouldPrompt(context.Background(), "owner")
	if err != nil || !prompt {
		t.Fatalf("ShouldPrompt next week = %t, %v", prompt, err)
	}
}

func TestRatingRecord(t *testing.T) {
	repo := newFakeRatingRepo()
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	service := &RatingService{Repo: repo, Now: func() time.Time { return now }, ID: func() string { return "r1" }}
	rating, err := service.Record(context.Background(), "owner", RatingWorse, []string{"b1", "b2"})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if rating.Value != RatingWorse || len(rating.BatchIDs) != 2 {
		t.Fatalf("rating = %+v", rating)
	}
	duplicate, err := service.Record(context.Background(), "owner", RatingSame, []string{"b3"})
	if err != nil {
		t.Fatalf("Record duplicate: %v", err)
	}
	if duplicate.ID != "r1" || duplicate.Value != RatingWorse {
		t.Fatalf("duplicate = %+v", duplicate)
	}
	dismissed, err := service.Dismiss(context.Background(), "other", []string{"b1"})
	if err != nil {
		t.Fatalf("Dismiss: %v", err)
	}
	if dismissed.Value != RatingUnrated {
		t.Fatalf("dismissed = %+v", dismissed)
	}
}

func TestRatingUnhappy(t *testing.T) {
	repo := newFakeRatingRepo()
	service := &RatingService{Repo: repo}
	if _, err := service.Record(context.Background(), "owner", "bad", []string{"b1"}); err == nil {
		t.Fatal("Record(bad value) = nil, want an error")
	}
	if _, err := service.Record(context.Background(), "owner", RatingSame, nil); err == nil {
		t.Fatal("Record(no batches) = nil, want an error")
	}
	if _, err := (&RatingService{}).ShouldPrompt(context.Background(), "owner"); err == nil {
		t.Fatal("ShouldPrompt(nil repo) = nil, want an error")
	}
	if _, err := (&RatingService{}).Record(context.Background(), "owner", RatingSame, []string{"b"}); err == nil {
		t.Fatal("Record(nil repo) = nil, want an error")
	}
	repo.err = errors.New("db")
	if _, err := service.ShouldPrompt(context.Background(), "owner"); err == nil {
		t.Fatal("ShouldPrompt(error) = nil, want an error")
	}
}
