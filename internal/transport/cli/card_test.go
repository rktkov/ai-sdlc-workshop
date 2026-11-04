package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/feed"
)

func TestRenderCardHappy(t *testing.T) {
	card := feed.Card{
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
		State:           feed.StateAnalyzed,
	}

	var buf bytes.Buffer
	if err := RenderCard(&buf, card); err != nil {
		t.Fatalf("RenderCard: %v", err)
	}

	out := buf.String()
	order := []string{"A useful video", "A channel", "ai-agents", "A short summary.", "similar to: prior video", "Matches depth over hype.", "analyzed"}
	last := -1
	for _, want := range order {
		index := strings.Index(out, want)
		if index < 0 {
			t.Fatalf("output is missing %q:\n%s", want, out)
		}
		if index <= last {
			t.Fatalf("%q is out of order:\n%s", want, out)
		}
		last = index
	}
}

func TestRenderCardUnhappy(t *testing.T) {
	card := feed.Card{State: feed.StateAnalyzed}

	var buf bytes.Buffer
	if err := RenderCard(&buf, card); err == nil {
		t.Fatal("RenderCard(invalid) = nil, want an error")
	}
	if buf.Len() != 0 {
		t.Fatalf("invalid card wrote output: %q", buf.String())
	}
}
