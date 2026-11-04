package cli

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/feed"
)

// RenderCard writes one card in the CLI-13 order using text only.
func RenderCard(w io.Writer, card feed.Card) error {
	if err := card.Validate(); err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString(card.Title)
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%s · %d subscribers · %d views · %s · %s\n",
		card.Channel,
		card.SubscriberCount,
		card.Views,
		card.Duration.Round(time.Minute),
		card.PublishedAt.UTC().Format("2006-01-02"),
	))
	if card.State == feed.StateAnalyzed {
		b.WriteString("Topic: " + card.Topic + "\n")
		b.WriteString("TL;DR: " + card.TLDR + "\n")
		if card.Similarity != "" {
			b.WriteString("Similarity: " + card.Similarity + "\n")
		}
		b.WriteString("Why this card: " + card.WhyThisCard + "\n")
	}
	b.WriteString("State: " + string(card.State) + "\n")
	if card.Reason != "" {
		b.WriteString("Reason: " + card.Reason + "\n")
	}

	_, err := io.WriteString(w, b.String())
	return err
}
