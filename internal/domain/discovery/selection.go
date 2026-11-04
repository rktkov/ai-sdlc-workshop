package discovery

import (
	"errors"
	"math"
	"sort"
)

// ExclusionReason is why a candidate is not selected.
type ExclusionReason string

const (
	ExclusionWatched   ExclusionReason = "watched"
	ExclusionSaved     ExclusionReason = "saved"
	ExclusionShown7d   ExclusionReason = "shown_7d"
	ExclusionDuplicate ExclusionReason = "duplicate"
)

const microBloggerLimit = 100000

// Candidate is one raw discovery result.
type Candidate struct {
	VideoID         string
	ChannelID       string
	Title           string
	SubscriberCount int64
	TopicTags       []string
	Source          CandidateSource
	DiscoveryRank   int
}

// ExclusionSet is supplied by feed.
type ExclusionSet struct {
	Watched map[string]bool
	Saved   map[string]bool
	Shown7d map[string]bool
}

// SelectedCandidate is one candidate in display order.
type SelectedCandidate struct {
	Candidate
	Position     int
	MicroBlogger bool
}

// ExcludedCandidate records one removed candidate.
type ExcludedCandidate struct {
	Candidate
	Reason ExclusionReason
}

// Selection is the deterministic result of one pool selection.
type Selection struct {
	Selected          []SelectedCandidate
	Excluded          []ExcludedCandidate
	MicroBloggerShare float64
	Shortfall         string
}

// Select applies exclusions, dedupe and the micro-blogger target.
func Select(candidates []Candidate, exclusions ExclusionSet, size int, microTarget float64) (Selection, error) {
	if size < 1 || size > 20 {
		return Selection{}, errors.New("selection size must be between 1 and 20")
	}
	if microTarget <= 0 || microTarget > 1 {
		microTarget = 0.2
	}

	selection := Selection{}
	seen := map[string]bool{}
	var kept []Candidate
	for _, candidate := range candidates {
		if candidate.VideoID == "" {
			continue
		}
		if seen[candidate.VideoID] {
			selection.Excluded = append(selection.Excluded, ExcludedCandidate{Candidate: candidate, Reason: ExclusionDuplicate})
			continue
		}
		seen[candidate.VideoID] = true
		if reason := exclusionReason(candidate.VideoID, exclusions); reason != "" {
			selection.Excluded = append(selection.Excluded, ExcludedCandidate{Candidate: candidate, Reason: reason})
			continue
		}
		kept = append(kept, candidate)
	}

	sort.SliceStable(kept, func(i, j int) bool {
		iPriority := sourcePriority(kept[i].Source)
		jPriority := sourcePriority(kept[j].Source)
		if iPriority != jPriority {
			return iPriority < jPriority
		}
		if kept[i].DiscoveryRank != kept[j].DiscoveryRank {
			return kept[i].DiscoveryRank < kept[j].DiscoveryRank
		}
		return kept[i].VideoID < kept[j].VideoID
	})

	targetMicro := int(math.Ceil(float64(size) * microTarget))
	chosen := map[string]bool{}
	for _, candidate := range kept {
		if len(chosen) >= targetMicro {
			break
		}
		if candidate.SubscriberCount < microBloggerLimit {
			chosen[candidate.VideoID] = true
		}
	}
	for _, candidate := range kept {
		if len(chosen) >= size {
			break
		}
		chosen[candidate.VideoID] = true
	}

	micro := 0
	for _, candidate := range kept {
		if !chosen[candidate.VideoID] {
			continue
		}
		isMicro := candidate.SubscriberCount < microBloggerLimit
		if isMicro {
			micro++
		}
		selection.Selected = append(selection.Selected, SelectedCandidate{
			Candidate:    candidate,
			Position:     len(selection.Selected),
			MicroBlogger: isMicro,
		})
	}
	if len(selection.Selected) > 0 {
		selection.MicroBloggerShare = float64(micro) / float64(len(selection.Selected))
	}
	selection.Shortfall = shortfall(selection, size, microTarget, len(selection.Excluded))
	return selection, nil
}

func shortfall(selection Selection, size int, microTarget float64, excluded int) string {
	if len(selection.Selected) == 0 {
		if excluded > 0 {
			return "dedupe"
		}
		return "no_candidates"
	}
	if len(selection.Selected) < size {
		if excluded > 0 {
			return "dedupe"
		}
		return "no_candidates"
	}
	if selection.MicroBloggerShare < microTarget {
		return "micro_blogger_shortfall"
	}
	return ""
}

func exclusionReason(videoID string, exclusions ExclusionSet) ExclusionReason {
	if exclusions.Watched[videoID] {
		return ExclusionWatched
	}
	if exclusions.Saved[videoID] {
		return ExclusionSaved
	}
	if exclusions.Shown7d[videoID] {
		return ExclusionShown7d
	}
	return ""
}

func sourcePriority(source CandidateSource) int {
	switch source {
	case SourceGraph:
		return 0
	case SourceSearch:
		return 1
	case SourceTrending:
		return 2
	default:
		return 3
	}
}
