package discovery

import (
	"math"
	"testing"
)

func candidate(id string, source CandidateSource, rank int, subscribers int64) Candidate {
	return Candidate{
		VideoID: id, ChannelID: "ch-" + id, Title: "title " + id,
		Source: source, DiscoveryRank: rank, SubscriberCount: subscribers,
	}
}

func TestSelectHappy(t *testing.T) {
	var candidates []Candidate
	for i := 0; i < 20; i++ {
		source := SourceSearch
		if i%3 == 0 {
			source = SourceGraph
		}
		subscribers := int64(200000)
		if i%5 == 0 {
			subscribers = 1000
		}
		candidates = append(candidates, candidate(string(rune('a'+i)), source, i, subscribers))
	}
	selection, err := Select(candidates, ExclusionSet{}, 20, 0.2)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if len(selection.Selected) != 20 {
		t.Fatalf("selected = %d", len(selection.Selected))
	}
	for i, selected := range selection.Selected {
		if selected.Position != i {
			t.Fatalf("position %d = %d", i, selected.Position)
		}
	}
	if selection.Shortfall != "" {
		t.Fatalf("shortfall = %q", selection.Shortfall)
	}
}

func TestSelectDedupe(t *testing.T) {
	candidates := []Candidate{
		candidate("v1", SourceSearch, 1, 1000),
		candidate("v1", SourceGraph, 0, 1000),
	}
	selection, err := Select(candidates, ExclusionSet{}, 20, 0.2)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if len(selection.Selected) != 1 {
		t.Fatalf("selected = %d", len(selection.Selected))
	}
	if len(selection.Excluded) != 1 || selection.Excluded[0].Reason != ExclusionDuplicate {
		t.Fatalf("excluded = %+v", selection.Excluded)
	}
}

func TestSelectExclusions(t *testing.T) {
	candidates := []Candidate{
		candidate("watched", SourceSearch, 0, 1000),
		candidate("saved", SourceSearch, 1, 1000),
		candidate("shown", SourceSearch, 2, 1000),
		candidate("keep", SourceSearch, 3, 1000),
	}
	exclusions := ExclusionSet{
		Watched: map[string]bool{"watched": true},
		Saved:   map[string]bool{"saved": true},
		Shown7d: map[string]bool{"shown": true},
	}
	selection, err := Select(candidates, exclusions, 20, 0.2)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if len(selection.Selected) != 1 || selection.Selected[0].VideoID != "keep" {
		t.Fatalf("selected = %+v", selection.Selected)
	}
	reasons := map[ExclusionReason]bool{}
	for _, excluded := range selection.Excluded {
		reasons[excluded.Reason] = true
	}
	for _, want := range []ExclusionReason{ExclusionWatched, ExclusionSaved, ExclusionShown7d} {
		if !reasons[want] {
			t.Fatalf("missing reason %q in %+v", want, selection.Excluded)
		}
	}
}

func TestSelectMicroBloggerTarget(t *testing.T) {
	candidates := []Candidate{
		candidate("micro-1", SourceSearch, 0, 1000),
		candidate("micro-2", SourceSearch, 1, 2000),
	}
	for i := 0; i < 8; i++ {
		candidates = append(candidates, candidate(string(rune('a'+i)), SourceSearch, 2+i, 500000))
	}
	selection, err := Select(candidates, ExclusionSet{}, 10, 0.2)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if len(selection.Selected) != 10 {
		t.Fatalf("selected = %d", len(selection.Selected))
	}
	micro := 0
	for _, selected := range selection.Selected {
		if selected.MicroBlogger {
			micro++
		}
	}
	if micro != 2 {
		t.Fatalf("micro = %d, want 2", micro)
	}
	if math.Abs(selection.MicroBloggerShare-0.2) > 1e-9 {
		t.Fatalf("share = %v", selection.MicroBloggerShare)
	}
	if selection.Shortfall != "" {
		t.Fatalf("shortfall = %q", selection.Shortfall)
	}
}

func TestSelectMicroBloggerShortfall(t *testing.T) {
	candidates := []Candidate{candidate("micro", SourceSearch, 0, 1000)}
	for i := 0; i < 9; i++ {
		candidates = append(candidates, candidate(string(rune('a'+i)), SourceSearch, 1+i, 500000))
	}
	selection, err := Select(candidates, ExclusionSet{}, 10, 0.2)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if selection.Shortfall != "micro_blogger_shortfall" {
		t.Fatalf("shortfall = %q", selection.Shortfall)
	}
	if len(selection.Selected) != 10 {
		t.Fatalf("selected = %d", len(selection.Selected))
	}
}

func TestSelectFewerThanSize(t *testing.T) {
	candidates := []Candidate{
		candidate("v1", SourceSearch, 0, 1000),
		candidate("v2", SourceSearch, 1, 1000),
	}
	selection, err := Select(candidates, ExclusionSet{}, 20, 0.2)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if len(selection.Selected) != 2 {
		t.Fatalf("selected = %d", len(selection.Selected))
	}
	if selection.Shortfall != "no_candidates" {
		t.Fatalf("shortfall = %q", selection.Shortfall)
	}
}

func TestSelectEmpty(t *testing.T) {
	selection, err := Select(nil, ExclusionSet{}, 20, 0.2)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	if len(selection.Selected) != 0 || selection.Shortfall != "no_candidates" {
		t.Fatalf("selection = %+v", selection)
	}
}

func TestSelectInvalidSize(t *testing.T) {
	if _, err := Select(nil, ExclusionSet{}, 0, 0.2); err == nil {
		t.Fatal("Select(size 0) = nil, want an error")
	}
	if _, err := Select(nil, ExclusionSet{}, 21, 0.2); err == nil {
		t.Fatal("Select(size 21) = nil, want an error")
	}
}

func TestSelectSourcePriority(t *testing.T) {
	candidates := []Candidate{
		candidate("trending", SourceTrending, 0, 1000),
		candidate("search", SourceSearch, 0, 1000),
		candidate("graph", SourceGraph, 0, 1000),
	}
	selection, err := Select(candidates, ExclusionSet{}, 3, 0.2)
	if err != nil {
		t.Fatalf("Select: %v", err)
	}
	want := []string{"graph", "trending", "search"}
	for i, selected := range selection.Selected {
		if selected.VideoID != want[i] {
			t.Fatalf("selected %d = %q, want %q", i, selected.VideoID, want[i])
		}
	}
}
