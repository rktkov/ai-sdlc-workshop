package contract

import (
	"testing"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/feed"
)

func TestStatsFromFeed(t *testing.T) {
	got := StatsFromFeed(feed.Stats{
		Swipes: map[string]int{"left": 2}, Reasons: map[string]int{"marketing": 1},
		ColdActive: 1, ColdExpired: 2,
		Quota:  []feed.QuotaPoint{{Day: "2026-11-05", Charged: 100}},
		Topics: map[string]int{"ai": 3},
	})
	if got.Swipes["left"] != 2 || got.ColdActive != 1 || len(got.Quota) != 1 || got.Topics["ai"] != 3 {
		t.Fatalf("stats = %+v", got)
	}
}

func TestRatingFromFeed(t *testing.T) {
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	got := RatingFromFeed(feed.WeeklyRating{
		ID: "r1", WeekStart: feed.WeekStart(now), Value: feed.RatingBetter,
		BatchIDs: []string{"b1"}, CreatedAt: now,
	})
	if got.ID != "r1" || got.WeekStart != "2026-11-02" || got.Value != "1" || len(got.BatchIDs) != 1 {
		t.Fatalf("rating = %+v", got)
	}
	if got.CreatedAt != "2026-11-06T12:00:00Z" {
		t.Fatalf("created = %q", got.CreatedAt)
	}
}
