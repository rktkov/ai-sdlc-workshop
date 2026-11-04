package feed

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/analysis"
	"github.com/rktkov/ytfeed/internal/domain/discovery"
	"github.com/rktkov/ytfeed/internal/domain/taste"
)

// BatchState is the lifecycle state of one batch.
type BatchState string

const (
	BatchComposing BatchState = "composing"
	BatchStreaming BatchState = "streaming"
	BatchCompleted BatchState = "completed"
	BatchDegraded  BatchState = "degraded"
)

// DegradationReason explains a degraded batch.
type DegradationReason string

const (
	DegradeQuota   DegradationReason = "quota_exhausted"
	DegradeNetwork DegradationReason = "network_unavailable"
	DegradeYTDLP   DegradationReason = "ytdlp_missing"
	DegradePartial DegradationReason = "partial"
)

// Batch is one ordered set of cards.
type Batch struct {
	ID                string
	UserID            string
	Size              int
	State             BatchState
	DegradationReason DegradationReason
	Cards             []Card
	CreatedAt         time.Time
	CompletedAt       time.Time
}

// ComposeBatch creates the first ordered batch from selected candidates.
func ComposeBatch(batchID, userID string, size int, candidates []discovery.SelectedCandidate, analyses map[string]analysis.Analysis) (Batch, error) {
	if batchID == "" || userID == "" {
		return Batch{}, errors.New("batch has no id or user")
	}
	if size < 1 || size > 20 {
		return Batch{}, errors.New("batch size must be between 1 and 20")
	}
	if len(candidates) > size {
		return Batch{}, errors.New("candidate count exceeds batch size")
	}
	seen := map[string]bool{}
	cards := make([]Card, 0, len(candidates))
	for _, candidate := range candidates {
		if seen[candidate.VideoID] {
			return Batch{}, fmt.Errorf("video %q is duplicated", candidate.VideoID)
		}
		seen[candidate.VideoID] = true
		card := Card{
			CardID:          "card-" + candidate.VideoID,
			VideoID:         candidate.VideoID,
			ChannelID:       candidate.ChannelID,
			Title:           candidate.Title,
			Channel:         candidate.ChannelID,
			SubscriberCount: int(candidate.SubscriberCount),
			State:           StatePending,
			DiscoveryRank:   candidate.DiscoveryRank,
		}
		if result, ok := analyses[candidate.VideoID]; ok {
			applyResult(&card, result)
		}
		cards = append(cards, card)
	}
	batch := Batch{
		ID: batchID, UserID: userID, Size: size, Cards: cards,
		State: BatchComposing, CreatedAt: time.Now().UTC(),
	}
	if len(cards) == 0 {
		batch.State = BatchDegraded
		batch.DegradationReason = DegradePartial
		return batch, nil
	}
	sortCards(batch.Cards)
	assignPositions(batch.Cards)
	batch.State = stateFor(batch.Cards)
	if batch.State == BatchCompleted {
		batch.CompletedAt = time.Now().UTC()
	}
	return batch, nil
}

// ApplyAnalysis updates one card and reorders only the unseen suffix.
func ApplyAnalysis(batch *Batch, cardID string, result analysis.Analysis, cursorPosition int) error {
	if batch == nil {
		return errors.New("batch is nil")
	}
	index := -1
	for i := range batch.Cards {
		if batch.Cards[i].CardID == cardID {
			index = i
			break
		}
	}
	if index == -1 {
		return fmt.Errorf("card %q is not in the batch", cardID)
	}
	card := &batch.Cards[index]
	late := card.Position <= cursorPosition && card.State == StatePending
	applyResult(card, result)
	if late {
		card.AnalyzedLate = true
	} else {
		reorderSuffix(batch.Cards, cursorPosition)
	}
	batch.State = stateFor(batch.Cards)
	if batch.State == BatchCompleted {
		batch.CompletedAt = time.Now().UTC()
	}
	return nil
}

// Validate checks the batch and card invariants.
func (b Batch) Validate() error {
	if b.ID == "" || b.UserID == "" {
		return errors.New("batch has no id or user")
	}
	if b.Size < 1 || b.Size > 20 {
		return errors.New("batch size must be between 1 and 20")
	}
	switch b.State {
	case BatchComposing, BatchStreaming, BatchCompleted:
		if b.DegradationReason != "" {
			return errors.New("non-degraded batch has a degradation reason")
		}
	case BatchDegraded:
		if b.DegradationReason == "" {
			return errors.New("degraded batch has no reason")
		}
	default:
		return fmt.Errorf("batch has an invalid state %q", b.State)
	}
	if len(b.Cards) > b.Size {
		return errors.New("batch has more cards than its size")
	}
	if len(b.Cards) == 0 && b.State != BatchDegraded {
		return errors.New("empty batch is not degraded")
	}
	seenVideo := map[string]bool{}
	seenCard := map[string]bool{}
	seenPosition := map[int]bool{}
	for _, card := range b.Cards {
		if err := card.Validate(); err != nil {
			return err
		}
		if card.CardID == "" {
			return errors.New("card has no id")
		}
		if seenCard[card.CardID] {
			return fmt.Errorf("card %q is duplicated", card.CardID)
		}
		seenCard[card.CardID] = true
		if seenVideo[card.VideoID] {
			return fmt.Errorf("video %q is duplicated", card.VideoID)
		}
		seenVideo[card.VideoID] = true
		if seenPosition[card.Position] {
			return fmt.Errorf("position %d is duplicated", card.Position)
		}
		seenPosition[card.Position] = true
		switch card.State {
		case StateAnalyzed:
			if card.AnalysisID == "" || card.Score < 0 || card.Score > 1 {
				return fmt.Errorf("card %q has an invalid analyzed state", card.CardID)
			}
		case StateFailed:
			if card.Reason == "" {
				return fmt.Errorf("card %q has no failure reason", card.CardID)
			}
		case StatePending:
		default:
			return fmt.Errorf("card %q has an invalid state", card.CardID)
		}
	}
	for position := 0; position < len(b.Cards); position++ {
		if !seenPosition[position] {
			return fmt.Errorf("position %d is missing", position)
		}
	}
	return nil
}

func applyResult(card *Card, result analysis.Analysis) {
	card.AnalysisID = result.ID
	switch result.State {
	case analysis.StateAnalyzed:
		card.State = StateAnalyzed
		card.Reason = ""
		card.Score = analysisScore(result)
		card.Topic = result.Topic
		card.TLDR = result.TLDR
		card.Similarity = result.Similarity
		card.WhyThisCard = result.WhyThisCard
		card.ScoredAxes = make([]taste.AxisScore, 0, len(result.AxisScores))
		for _, score := range result.AxisScores {
			card.ScoredAxes = append(card.ScoredAxes, taste.AxisScore{Axis: score.Axis, Score: score.Score})
		}
	case analysis.StateFailed:
		card.State = StateFailed
		card.Reason = string(result.Reason)
		card.Score = 0
		card.Topic = ""
		card.TLDR = ""
		card.Similarity = ""
		card.WhyThisCard = ""
	}
}

func analysisScore(result analysis.Analysis) float64 {
	if len(result.AxisScores) == 0 {
		return 0
	}
	total := 0.0
	for _, score := range result.AxisScores {
		total += score.Score
	}
	return total / float64(len(result.AxisScores))
}

func sortCards(cards []Card) {
	sort.SliceStable(cards, func(i, j int) bool {
		iRank := stateRank(cards[i].State)
		jRank := stateRank(cards[j].State)
		if iRank != jRank {
			return iRank < jRank
		}
		if cards[i].State == StateAnalyzed && cards[i].Score != cards[j].Score {
			return cards[i].Score > cards[j].Score
		}
		if cards[i].DiscoveryRank != cards[j].DiscoveryRank {
			return cards[i].DiscoveryRank < cards[j].DiscoveryRank
		}
		return cards[i].CardID < cards[j].CardID
	})
}

func reorderSuffix(cards []Card, cursorPosition int) {
	prefix := make([]Card, 0, len(cards))
	suffix := make([]Card, 0, len(cards))
	for _, card := range cards {
		if card.Position <= cursorPosition {
			prefix = append(prefix, card)
		} else {
			suffix = append(suffix, card)
		}
	}
	sort.SliceStable(prefix, func(i, j int) bool { return prefix[i].Position < prefix[j].Position })
	sortCards(suffix)
	combined := append(prefix, suffix...)
	copy(cards, combined)
	assignPositions(cards)
}

func assignPositions(cards []Card) {
	for i := range cards {
		cards[i].Position = i
	}
}

func stateFor(cards []Card) BatchState {
	for _, card := range cards {
		if card.State == StatePending {
			return BatchStreaming
		}
	}
	return BatchCompleted
}

func stateRank(state State) int {
	switch state {
	case StateAnalyzed:
		return 0
	case StateFailed:
		return 1
	default:
		return 2
	}
}
