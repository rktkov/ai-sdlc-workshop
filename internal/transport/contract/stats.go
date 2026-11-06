package contract

import "github.com/rktkov/ytfeed/internal/domain/feed"

// QuotaPoint is one day of quota usage.
type QuotaPoint struct {
	Day     string `json:"day"`
	Charged int    `json:"charged"`
}

// Stats is the read-only stats response.
type Stats struct {
	Swipes      map[string]int `json:"swipes"`
	Reasons     map[string]int `json:"reasons"`
	ColdActive  int            `json:"cold_active"`
	ColdExpired int            `json:"cold_expired"`
	Quota       []QuotaPoint   `json:"quota"`
	Topics      map[string]int `json:"topics"`
}

// StatsFromFeed maps the domain projection to the wire contract.
func StatsFromFeed(stats feed.Stats) Stats {
	out := Stats{
		Swipes: stats.Swipes, Reasons: stats.Reasons,
		ColdActive: stats.ColdActive, ColdExpired: stats.ColdExpired,
		Topics: stats.Topics,
	}
	for _, point := range stats.Quota {
		out.Quota = append(out.Quota, QuotaPoint{Day: point.Day, Charged: point.Charged})
	}
	return out
}

// Rating is one weekly rating.
type Rating struct {
	ID        string   `json:"rating_id"`
	WeekStart string   `json:"week_start"`
	Value     string   `json:"value"`
	BatchIDs  []string `json:"batch_ids"`
	CreatedAt string   `json:"created_at"`
}

// RatingStatus is the weekly prompt state.
type RatingStatus struct {
	ShouldPrompt bool    `json:"should_prompt"`
	WeekStart    string  `json:"week_start"`
	Last         *Rating `json:"last,omitempty"`
}

// RatingFromFeed maps a domain rating to the wire contract.
func RatingFromFeed(value feed.WeeklyRating) Rating {
	out := Rating{
		ID: value.ID, WeekStart: value.WeekStart.UTC().Format("2006-01-02"),
		Value: string(value.Value), BatchIDs: value.BatchIDs,
		CreatedAt: value.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	return out
}
