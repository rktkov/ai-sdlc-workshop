package contract

import (
	"fmt"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/feed"
)

// FeedCard is one card in the feed contract.
type FeedCard struct {
	CardID          string  `json:"card_id"`
	VideoID         string  `json:"video_id"`
	ChannelID       string  `json:"channel_id"`
	Title           string  `json:"title"`
	Channel         string  `json:"channel"`
	SubscriberCount int     `json:"subscriber_count"`
	Views           int     `json:"views"`
	DurationSeconds int64   `json:"duration_seconds"`
	PublishedAt     string  `json:"published_at"`
	Topic           string  `json:"topic,omitempty"`
	TLDR            string  `json:"tldr,omitempty"`
	Similarity      string  `json:"similarity,omitempty"`
	WhyThisCard     string  `json:"why_this_card,omitempty"`
	State           string  `json:"state"`
	Reason          string  `json:"reason,omitempty"`
	AnalysisID      string  `json:"analysis_id,omitempty"`
	Score           float64 `json:"score,omitempty"`
	Position        int     `json:"position"`
	Decision        string  `json:"decision"`
	AnalyzedLate    bool    `json:"analyzed_late"`
}

// Feed is one batch response.
type Feed struct {
	BatchID           string     `json:"batch_id"`
	State             string     `json:"state"`
	DegradationReason string     `json:"degradation_reason,omitempty"`
	Banner            string     `json:"banner,omitempty"`
	Cards             []FeedCard `json:"cards"`
}

// FeedFromBatch maps a feed batch to the wire contract.
func FeedFromBatch(batch feed.Batch) Feed {
	out := Feed{
		BatchID: batch.ID, State: string(batch.State),
		DegradationReason: string(batch.DegradationReason),
		Banner:            bannerFor(batch),
		Cards:             make([]FeedCard, 0, len(batch.Cards)),
	}
	for _, card := range batch.Cards {
		out.Cards = append(out.Cards, FeedCard{
			CardID: card.CardID, VideoID: card.VideoID, ChannelID: card.ChannelID,
			Title: card.Title, Channel: card.Channel, SubscriberCount: card.SubscriberCount,
			Views: card.Views, DurationSeconds: int64(card.Duration.Seconds()),
			PublishedAt: card.PublishedAt.UTC().Format(time.RFC3339),
			Topic:       card.Topic, TLDR: card.TLDR, Similarity: card.Similarity,
			WhyThisCard: card.WhyThisCard, State: string(card.State), Reason: card.Reason,
			AnalysisID: card.AnalysisID, Score: card.Score, Position: card.Position,
			Decision: string(card.Decision), AnalyzedLate: card.AnalyzedLate,
		})
	}
	return out
}

func bannerFor(batch feed.Batch) string {
	if batch.State != feed.BatchDegraded {
		return ""
	}
	switch batch.DegradationReason {
	case feed.DegradeQuota:
		return "quota exhausted — cached feed"
	case feed.DegradeNetwork:
		return "network unavailable — cached feed"
	case feed.DegradeYTDLP:
		return "yt-dlp missing — saved videos only"
	case feed.DegradePartial:
		return "partial feed"
	default:
		return "degraded feed"
	}
}

// Validate checks the feed contract.
func (f Feed) Validate() error {
	if f.BatchID == "" {
		return fmt.Errorf("feed has no batch id")
	}
	switch f.State {
	case string(feed.BatchComposing), string(feed.BatchStreaming), string(feed.BatchCompleted), string(feed.BatchDegraded):
	default:
		return fmt.Errorf("feed has an invalid state %q", f.State)
	}
	for i, card := range f.Cards {
		if card.CardID == "" || card.VideoID == "" {
			return fmt.Errorf("card %d has no id or video", i)
		}
		if card.Position != i {
			return fmt.Errorf("card %d has position %d", i, card.Position)
		}
	}
	return nil
}
