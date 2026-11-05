package contract

import (
	"testing"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/feed"
)

func TestFeedFromBatch(t *testing.T) {
	batch := feed.Batch{
		ID: "b1", State: feed.BatchCompleted,
		Cards: []feed.Card{
			{
				CardID: "c1", VideoID: "v1", ChannelID: "ch1", Title: "Title", Channel: "Channel",
				SubscriberCount: 10, Views: 20, Duration: 90 * time.Second,
				PublishedAt: time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC),
				Topic:       "ai-agents", TLDR: "tldr", Similarity: "new territory", WhyThisCard: "why",
				State: feed.StateAnalyzed, AnalysisID: "a1", Score: 0.8, Position: 0,
				Decision: feed.DecisionNone,
			},
		},
	}
	got := FeedFromBatch(batch)
	if err := got.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if got.BatchID != "b1" || len(got.Cards) != 1 {
		t.Fatalf("feed = %+v", got)
	}
	card := got.Cards[0]
	if card.CardID != "c1" || card.DurationSeconds != 90 || card.State != "analyzed" {
		t.Fatalf("card = %+v", card)
	}
	if card.PublishedAt != "2026-11-04T12:00:00Z" {
		t.Fatalf("published = %q", card.PublishedAt)
	}
}

func TestFeedValidateUnhappy(t *testing.T) {
	valid := Feed{BatchID: "b", State: "completed", Cards: []FeedCard{{CardID: "c", VideoID: "v", Position: 0}}}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	tests := []struct {
		name string
		edit func(*Feed)
	}{
		{name: "no batch", edit: func(f *Feed) { f.BatchID = "" }},
		{name: "bad state", edit: func(f *Feed) { f.State = "bad" }},
		{name: "no card id", edit: func(f *Feed) { f.Cards[0].CardID = "" }},
		{name: "bad position", edit: func(f *Feed) { f.Cards[0].Position = 1 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := valid
			value.Cards = append([]FeedCard(nil), valid.Cards...)
			tt.edit(&value)
			if err := value.Validate(); err == nil {
				t.Fatal("Validate = nil, want an error")
			}
		})
	}
}

func TestFeedBanner(t *testing.T) {
	batch := feed.Batch{
		ID: "b", State: feed.BatchDegraded, DegradationReason: feed.DegradeQuota,
		Cards: []feed.Card{{CardID: "c", VideoID: "v", Title: "t", Channel: "ch", State: feed.StatePending, Position: 0}},
	}
	got := FeedFromBatch(batch)
	if got.DegradationReason != "quota_exhausted" || got.Banner == "" {
		t.Fatalf("feed = %+v", got)
	}
}
