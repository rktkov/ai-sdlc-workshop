package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/feed"
	"github.com/rktkov/ytfeed/internal/transport/contract"
)

type stubStatsRepo struct{}

func (stubStatsRepo) Swipes(context.Context, string) (map[string]int, error) {
	return map[string]int{"left": 2}, nil
}
func (stubStatsRepo) Reasons(context.Context, string) (map[string]int, error) {
	return map[string]int{"marketing": 1}, nil
}
func (stubStatsRepo) ColdPeriods(context.Context, string, time.Time) (int, int, error) {
	return 1, 2, nil
}
func (stubStatsRepo) QuotaTrend(context.Context, string, time.Time) ([]feed.QuotaPoint, error) {
	return []feed.QuotaPoint{{Day: "2026-11-05", Charged: 100}}, nil
}
func (stubStatsRepo) Topics(context.Context, string) (map[string]int, error) {
	return map[string]int{"ai": 3}, nil
}

func TestStatsHandler(t *testing.T) {
	service := &feed.StatsService{Repo: stubStatsRepo{}}
	recorder := httptest.NewRecorder()
	StatsHandler(service, "owner").ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/stats?days=7", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	var payload contract.Stats
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Swipes["left"] != 2 || payload.ColdActive != 1 || len(payload.Quota) != 1 {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestStatsUnhappy(t *testing.T) {
	service := &feed.StatsService{Repo: stubStatsRepo{}}
	for _, target := range []string{"/v1/stats?days=0", "/v1/stats?days=100", "/v1/stats?days=x"} {
		recorder := httptest.NewRecorder()
		StatsHandler(service, "owner").ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s status = %d, want 400", target, recorder.Code)
		}
	}
	recorder := httptest.NewRecorder()
	StatsHandler(service, "owner").ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/stats", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", recorder.Code)
	}
	recorder = httptest.NewRecorder()
	StatsHandler(nil, "owner").ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/stats", nil))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
}

type stubRatingRepo struct {
	items map[string]feed.WeeklyRating
}

func newStubRatingRepo() *stubRatingRepo {
	return &stubRatingRepo{items: map[string]feed.WeeklyRating{}}
}

func ratingKey(userID string, week time.Time) string {
	return userID + "|" + week.Format("2006-01-02")
}

func (s *stubRatingRepo) FindByWeek(_ context.Context, userID string, week time.Time) (feed.WeeklyRating, error) {
	value, ok := s.items[ratingKey(userID, week)]
	if !ok {
		return feed.WeeklyRating{}, feed.ErrRatingNotFound
	}
	return value, nil
}
func (s *stubRatingRepo) LastForUser(_ context.Context, userID string) (feed.WeeklyRating, error) {
	for _, value := range s.items {
		if value.UserID == userID {
			return value, nil
		}
	}
	return feed.WeeklyRating{}, feed.ErrRatingNotFound
}
func (s *stubRatingRepo) Save(_ context.Context, rating feed.WeeklyRating) error {
	key := ratingKey(rating.UserID, rating.WeekStart)
	if _, ok := s.items[key]; ok {
		return feed.ErrRatingExists
	}
	s.items[key] = rating
	return nil
}

func TestRatingHandler(t *testing.T) {
	repo := newStubRatingRepo()
	service := &feed.RatingService{Repo: repo, Now: func() time.Time {
		return time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	}, ID: func() string { return "r1" }}
	handler := RatingHandler(service, "owner")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/rating", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET status = %d", recorder.Code)
	}
	var status contract.RatingStatus
	if err := json.Unmarshal(recorder.Body.Bytes(), &status); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if !status.ShouldPrompt {
		t.Fatal("ShouldPrompt = false")
	}

	body := bytes.NewBufferString(`{"value":"1","batch_ids":["b1"]}`)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/rating", body))
	if recorder.Code != http.StatusOK {
		t.Fatalf("POST status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var rating contract.Rating
	if err := json.Unmarshal(recorder.Body.Bytes(), &rating); err != nil {
		t.Fatalf("decode rating: %v", err)
	}
	if rating.Value != "1" || len(rating.BatchIDs) != 1 {
		t.Fatalf("rating = %+v", rating)
	}

	body = bytes.NewBufferString(`{"value":"unrated","batch_ids":["b2"]}`)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/rating", body))
	if recorder.Code != http.StatusOK {
		t.Fatalf("dismiss status = %d", recorder.Code)
	}
}

func TestRatingUnhappy(t *testing.T) {
	service := &feed.RatingService{Repo: newStubRatingRepo()}
	handler := RatingHandler(service, "owner")
	tests := []struct {
		name string
		body string
	}{
		{name: "invalid json", body: `{`},
		{name: "invalid value", body: `{"value":"2","batch_ids":["b"]}`},
		{name: "no batches", body: `{"value":"1"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/rating", bytes.NewBufferString(tt.body)))
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", recorder.Code)
			}
		})
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, "/v1/rating", nil))
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", recorder.Code)
	}
	recorder = httptest.NewRecorder()
	RatingHandler(nil, "owner").ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/rating", nil))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
}
