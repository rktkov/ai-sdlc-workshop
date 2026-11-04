package feed

import (
	"fmt"
	"time"
)

// State is the analysis state of a card.
type State string

const (
	StatePending  State = "pending"
	StateAnalyzed State = "analyzed"
	StateFailed   State = "failed"
)

// Card is one video presented for judgment.
type Card struct {
	VideoID         string
	Title           string
	Channel         string
	SubscriberCount int
	Views           int
	Duration        time.Duration
	PublishedAt     time.Time
	Topic           string
	TLDR            string
	Similarity      string
	WhyThisCard     string
	State           State
	Reason          string
}

// Validate checks the card fields needed for rendering.
func (c Card) Validate() error {
	if c.VideoID == "" {
		return fmt.Errorf("card has no video id")
	}
	if c.Title == "" {
		return fmt.Errorf("card has no title")
	}
	if c.Channel == "" {
		return fmt.Errorf("card has no channel")
	}
	switch c.State {
	case StatePending:
		return nil
	case StateAnalyzed:
		if c.Topic == "" || c.TLDR == "" || c.WhyThisCard == "" {
			return fmt.Errorf("analyzed card is missing presentation fields")
		}
		return nil
	case StateFailed:
		if c.Reason == "" {
			return fmt.Errorf("failed card has no reason")
		}
		return nil
	default:
		return fmt.Errorf("card has an invalid state %q", c.State)
	}
}
