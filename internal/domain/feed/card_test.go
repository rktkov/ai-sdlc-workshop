package feed

import (
	"testing"
	"time"
)

func validCard() Card {
	return Card{
		VideoID:         "video-1",
		Title:           "A useful video",
		Channel:         "A channel",
		SubscriberCount: 120000,
		Views:           45000,
		Duration:        12 * time.Minute,
		PublishedAt:     time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		Topic:           "ai-agents",
		TLDR:            "A short summary.",
		Similarity:      "similar to: prior video",
		WhyThisCard:     "Matches depth over hype.",
		State:           StateAnalyzed,
	}
}

func TestCardValidateHappy(t *testing.T) {
	if err := validCard().Validate(); err != nil {
		t.Fatalf("Validate() = %v", err)
	}

	pending := validCard()
	pending.State = StatePending
	pending.Topic = ""
	pending.TLDR = ""
	pending.WhyThisCard = ""
	if err := pending.Validate(); err != nil {
		t.Fatalf("pending Validate() = %v", err)
	}

	failed := validCard()
	failed.State = StateFailed
	failed.Reason = "no_transcript"
	if err := failed.Validate(); err != nil {
		t.Fatalf("failed Validate() = %v", err)
	}
}

func TestCardValidateUnhappy(t *testing.T) {
	tests := []struct {
		name string
		card Card
	}{
		{name: "no video", card: func() Card { c := validCard(); c.VideoID = ""; return c }()},
		{name: "no title", card: func() Card { c := validCard(); c.Title = ""; return c }()},
		{name: "no channel", card: func() Card { c := validCard(); c.Channel = ""; return c }()},
		{name: "analyzed fields", card: func() Card { c := validCard(); c.Topic = ""; return c }()},
		{name: "failed reason", card: func() Card { c := validCard(); c.State = StateFailed; return c }()},
		{name: "bad state", card: func() Card { c := validCard(); c.State = State("unknown"); return c }()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.card.Validate(); err == nil {
				t.Fatal("Validate() = nil, want an error")
			}
		})
	}
}
