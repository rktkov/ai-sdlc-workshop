package discovery

import (
	"errors"
	"math"
	"sort"
)

// ExclusionReason is why a candidate is not selected.
type ExclusionReason string

const (
	ExclusionWatched    ExclusionReason = "watched"
	ExclusionSaved      ExclusionReason = "saved"
	ExclusionShown7d    ExclusionReason = "shown_7d"
	ExclusionDuplicate  ExclusionReason = "duplicate"
	ExclusionChannelCap ExclusionReason = "channel_cap"
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

// Select applies exclusions, dedupe, the channel cap and the micro-blogger target.
func Select(candidates []Candidate, exclusions ExclusionSet, size int, microTarget float64, channelCap int) (Selection, error) {
	if size < 1 || size > 20 {
		return Selection{}, errors.New("selection size must be between 1 and 20")
	}
	if microTarget <= 0 || microTarget > 1 {
		microTarget = 0.2
	}
	if channelCap <= 0 {
		channelCap = 2
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
		if kept[i].DiscoveryRank != kept[j].DiscoveryRank {
			return kept[i].DiscoveryRank < kept[j].DiscoveryRank
		}
		return kept[i].VideoID < kept[j].VideoID
	})

	chosen, capExcluded := chooseCandidates(kept, size, int(math.Ceil(float64(size)*microTarget)), channelCap)
	selection.Excluded = append(selection.Excluded, capExcluded...)
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
	selection.Shortfall = shortfall(selection, size, microTarget, len(selection.Excluded)-len(capExcluded), len(capExcluded))
	return selection, nil
}

func shortfall(selection Selection, size int, microTarget float64, excluded, capExcluded int) string {
	if len(selection.Selected) == 0 {
		if capExcluded > 0 {
			return "channel_cap"
		}
		if excluded > 0 {
			return "dedupe"
		}
		return "no_candidates"
	}
	if len(selection.Selected) < size {
		if capExcluded > 0 {
			return "channel_cap"
		}
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

func chooseCandidates(kept []Candidate, size, targetMicro, channelCap int) (map[string]bool, []ExcludedCandidate) {
	chosen := map[string]bool{}
	channelCount := map[string]int{}
	canTake := func(candidate Candidate) bool {
		if candidate.ChannelID == "" {
			return true
		}
		return channelCount[candidate.ChannelID] < channelCap
	}
	take := func(candidate Candidate) {
		chosen[candidate.VideoID] = true
		if candidate.ChannelID != "" {
			channelCount[candidate.ChannelID]++
		}
	}
	for _, candidate := range kept {
		if len(chosen) >= targetMicro {
			break
		}
		if candidate.SubscriberCount < microBloggerLimit && canTake(candidate) {
			take(candidate)
		}
	}
	for _, candidate := range kept {
		if len(chosen) >= size {
			break
		}
		if chosen[candidate.VideoID] || !canTake(candidate) {
			continue
		}
		take(candidate)
	}
	var capExcluded []ExcludedCandidate
	for _, candidate := range kept {
		if chosen[candidate.VideoID] {
			continue
		}
		if candidate.ChannelID != "" && channelCount[candidate.ChannelID] >= channelCap {
			capExcluded = append(capExcluded, ExcludedCandidate{Candidate: candidate, Reason: ExclusionChannelCap})
		}
	}
	return chosen, capExcluded
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
	case SourceTrending:
		return 1
	case SourceSearch:
		return 2
	default:
		return 3
	}
}

// SourceRank orders sources by selection priority. Lower is better.
func SourceRank(source CandidateSource) int {
	return sourcePriority(source)
}
