package feed

import (
	"testing"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/analysis"
	"github.com/rktkov/ytfeed/internal/domain/discovery"
)

func selected(videoID string, rank int) discovery.SelectedCandidate {
	return discovery.SelectedCandidate{
		Candidate: discovery.Candidate{
			VideoID: videoID, ChannelID: "ch-" + videoID, Title: "title " + videoID,
			Source: discovery.SourceSearch, DiscoveryRank: rank,
		},
		Position: rank,
	}
}

func analyzed(videoID string, score float64) analysis.Analysis {
	return analysis.Analysis{
		ID: "analysis-" + videoID, VideoID: videoID, State: analysis.StateAnalyzed,
		Topic: "ai-agents", TLDR: "tldr", Similarity: "new territory", WhyThisCard: "why",
		AxisScores:    []analysis.AxisScore{{Axis: "depth", Score: score, Justification: "good"}},
		CompletedAt:   time.Now().UTC(),
		RubricVersion: "v1", ModelID: "model",
	}
}

func failed(videoID string) analysis.Analysis {
	return analysis.Analysis{
		ID: "analysis-" + videoID, VideoID: videoID, State: analysis.StateFailed,
		Reason: analysis.ReasonNoTranscript, RubricVersion: "v1", ModelID: "model",
		CompletedAt: time.Now().UTC(),
	}
}

func TestComposeBatchOrder(t *testing.T) {
	candidates := []discovery.SelectedCandidate{
		selected("v1", 0), selected("v2", 1), selected("v3", 2), selected("v4", 3), selected("v5", 4),
	}
	analyses := map[string]analysis.Analysis{
		"v1": analyzed("v1", 0.9),
		"v2": analyzed("v2", 0.5),
		"v3": analyzed("v3", 0.7),
	}
	batch, err := ComposeBatch("batch-1", "owner", 5, candidates, analyses)
	if err != nil {
		t.Fatalf("ComposeBatch: %v", err)
	}
	want := []string{"v1", "v3", "v2", "v4", "v5"}
	for i, card := range batch.Cards {
		if card.VideoID != want[i] {
			t.Fatalf("position %d = %s, want %s", i, card.VideoID, want[i])
		}
		if card.Position != i {
			t.Fatalf("card %s position = %d", card.VideoID, card.Position)
		}
	}
	if batch.State != BatchStreaming {
		t.Fatalf("state = %q", batch.State)
	}
}

func TestComposeBatchCompleted(t *testing.T) {
	candidates := []discovery.SelectedCandidate{selected("v1", 0)}
	batch, err := ComposeBatch("batch-1", "owner", 1, candidates, map[string]analysis.Analysis{"v1": analyzed("v1", 0.8)})
	if err != nil {
		t.Fatalf("ComposeBatch: %v", err)
	}
	if batch.State != BatchCompleted {
		t.Fatalf("state = %q", batch.State)
	}
	if batch.Cards[0].Score != 0.8 {
		t.Fatalf("score = %v", batch.Cards[0].Score)
	}
}

func TestComposeBatchUnhappy(t *testing.T) {
	if _, err := ComposeBatch("b", "owner", 0, nil, nil); err == nil {
		t.Fatal("ComposeBatch(size 0) = nil, want an error")
	}
	if _, err := ComposeBatch("b", "owner", 21, nil, nil); err == nil {
		t.Fatal("ComposeBatch(size 21) = nil, want an error")
	}
	duplicate := []discovery.SelectedCandidate{selected("v1", 0), selected("v1", 1)}
	if _, err := ComposeBatch("b", "owner", 2, duplicate, nil); err == nil {
		t.Fatal("ComposeBatch(duplicate) = nil, want an error")
	}
	tooMany := []discovery.SelectedCandidate{
		selected("v1", 0), selected("v2", 1), selected("v3", 2),
	}
	if _, err := ComposeBatch("b", "owner", 2, tooMany, nil); err == nil {
		t.Fatal("ComposeBatch(over size) = nil, want an error")
	}
}

func TestApplyAnalysisAhead(t *testing.T) {
	candidates := []discovery.SelectedCandidate{selected("v1", 0), selected("v2", 1), selected("v3", 2)}
	batch, err := ComposeBatch("b", "owner", 3, candidates, map[string]analysis.Analysis{"v1": analyzed("v1", 0.9)})
	if err != nil {
		t.Fatalf("ComposeBatch: %v", err)
	}
	if err := ApplyAnalysis(&batch, "card-v3", analyzed("v3", 0.8), 0); err != nil {
		t.Fatalf("ApplyAnalysis: %v", err)
	}
	want := []string{"v1", "v3", "v2"}
	for i, card := range batch.Cards {
		if card.VideoID != want[i] || card.Position != i {
			t.Fatalf("cards = %+v", batch.Cards)
		}
	}
	if batch.Cards[1].AnalyzedLate {
		t.Fatal("v3 marked analyzed late ahead of the cursor")
	}
}

func TestApplyAnalysisBehindCursor(t *testing.T) {
	candidates := []discovery.SelectedCandidate{selected("v1", 0), selected("v2", 1), selected("v3", 2)}
	batch, err := ComposeBatch("b", "owner", 3, candidates, nil)
	if err != nil {
		t.Fatalf("ComposeBatch: %v", err)
	}
	if err := ApplyAnalysis(&batch, "card-v2", analyzed("v2", 0.8), 2); err != nil {
		t.Fatalf("ApplyAnalysis: %v", err)
	}
	if batch.Cards[1].VideoID != "v2" || batch.Cards[1].Position != 1 {
		t.Fatalf("card moved behind the cursor: %+v", batch.Cards)
	}
	if !batch.Cards[1].AnalyzedLate {
		t.Fatal("v2 was not marked analyzed late")
	}
}

func TestApplyAnalysisFailed(t *testing.T) {
	candidates := []discovery.SelectedCandidate{selected("v1", 0)}
	batch, err := ComposeBatch("b", "owner", 1, candidates, nil)
	if err != nil {
		t.Fatalf("ComposeBatch: %v", err)
	}
	if err := ApplyAnalysis(&batch, "card-v1", failed("v1"), 0); err != nil {
		t.Fatalf("ApplyAnalysis: %v", err)
	}
	if batch.Cards[0].State != StateFailed || batch.Cards[0].Reason != string(analysis.ReasonNoTranscript) {
		t.Fatalf("card = %+v", batch.Cards[0])
	}
	if batch.State != BatchCompleted {
		t.Fatalf("state = %q", batch.State)
	}
}

func TestApplyAnalysisUnknown(t *testing.T) {
	batch := Batch{ID: "b", UserID: "owner", Size: 1, State: BatchStreaming, Cards: []Card{
		{CardID: "c1", VideoID: "v1", Position: 0, DiscoveryRank: 0, State: StatePending},
	}}
	if err := ApplyAnalysis(&batch, "missing", analyzed("missing", 0.5), 0); err == nil {
		t.Fatal("ApplyAnalysis(missing) = nil, want an error")
	}
}

func TestApplyAnalysisDensePositions(t *testing.T) {
	candidates := []discovery.SelectedCandidate{
		selected("v1", 0), selected("v2", 1), selected("v3", 2), selected("v4", 3),
	}
	batch, err := ComposeBatch("b", "owner", 4, candidates, nil)
	if err != nil {
		t.Fatalf("ComposeBatch: %v", err)
	}
	if err := ApplyAnalysis(&batch, "card-v4", analyzed("v4", 0.9), 1); err != nil {
		t.Fatalf("ApplyAnalysis: %v", err)
	}
	if err := ApplyAnalysis(&batch, "card-v3", analyzed("v3", 0.7), 1); err != nil {
		t.Fatalf("ApplyAnalysis: %v", err)
	}
	seen := map[int]bool{}
	for _, card := range batch.Cards {
		if seen[card.Position] {
			t.Fatalf("duplicate position %d", card.Position)
		}
		seen[card.Position] = true
	}
	if len(seen) != len(batch.Cards) {
		t.Fatalf("positions = %v", seen)
	}
}

func TestBatchValidate(t *testing.T) {
	valid := Batch{
		ID: "b", UserID: "owner", Size: 2, State: BatchStreaming,
		Cards: []Card{
			{CardID: "c1", VideoID: "v1", Position: 0, DiscoveryRank: 0, State: StatePending, Title: "t", Channel: "c"},
			{CardID: "c2", VideoID: "v2", Position: 1, DiscoveryRank: 1, State: StatePending, Title: "t", Channel: "c"},
		},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	tests := []struct {
		name string
		edit func(*Batch)
	}{
		{name: "no id", edit: func(b *Batch) { b.ID = "" }},
		{name: "no user", edit: func(b *Batch) { b.UserID = "" }},
		{name: "bad size", edit: func(b *Batch) { b.Size = 0 }},
		{name: "gap", edit: func(b *Batch) { b.Cards[1].Position = 2 }},
		{name: "duplicate position", edit: func(b *Batch) { b.Cards[1].Position = 0 }},
		{name: "duplicate video", edit: func(b *Batch) { b.Cards[1].VideoID = "v1" }},
		{name: "bad state", edit: func(b *Batch) { b.State = "bad" }},
		{name: "degraded reason", edit: func(b *Batch) { b.State = BatchDegraded }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batch := valid
			batch.Cards = append([]Card(nil), valid.Cards...)
			tt.edit(&batch)
			if err := batch.Validate(); err == nil {
				t.Fatal("Validate = nil, want an error")
			}
		})
	}
}
