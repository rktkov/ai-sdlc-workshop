package identity

import (
	"fmt"
	"strings"
)

// SeedSource is the onboarding source for interest proposals.
type SeedSource string

const (
	// SeedYouTubeTrending uses the mostPopular chart and the subscription graph.
	SeedYouTubeTrending SeedSource = "youtube-trending"
	// SeedExternalTrends uses an external trend adapter.
	SeedExternalTrends SeedSource = "external-trends"
	// SeedSubscriptionsOnly uses only the imported subscription graph.
	SeedSubscriptionsOnly SeedSource = "subscriptions-only"
)

// SeedSourceInfo describes one source and its availability.
type SeedSourceInfo struct {
	Source    SeedSource
	Available bool
	Reason    string
}

// SeedSources returns all sources. Unavailable sources are listed, never replaced.
func SeedSources(hasSubscriptions bool) []SeedSourceInfo {
	subscriptionReason := "uses the imported subscription graph"
	if !hasSubscriptions {
		subscriptionReason = "no subscriptions imported"
	}
	return []SeedSourceInfo{
		{Source: SeedYouTubeTrending, Available: false, Reason: "the discovery adapter is not implemented yet"},
		{Source: SeedExternalTrends, Available: false, Reason: "no adapter is configured"},
		{Source: SeedSubscriptionsOnly, Available: hasSubscriptions, Reason: subscriptionReason},
	}
}

// ParseSeedSource validates one source name.
func ParseSeedSource(value string) (SeedSource, error) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case string(SeedYouTubeTrending):
		return SeedYouTubeTrending, nil
	case string(SeedExternalTrends):
		return SeedExternalTrends, nil
	case string(SeedSubscriptionsOnly):
		return SeedSubscriptionsOnly, nil
	default:
		return "", fmt.Errorf("unknown seed source %q", value)
	}
}

// SelectSeedSource validates a choice and rejects an unavailable source.
func SelectSeedSource(value string, hasSubscriptions bool) (SeedSource, []SeedSourceInfo, error) {
	sources := SeedSources(hasSubscriptions)
	source, err := ParseSeedSource(value)
	if err != nil {
		return "", sources, err
	}
	for _, info := range sources {
		if info.Source != source {
			continue
		}
		if !info.Available {
			return "", sources, fmt.Errorf("seed source %s is unavailable: %s", source, info.Reason)
		}
		return source, sources, nil
	}
	return "", sources, fmt.Errorf("unknown seed source %q", value)
}
