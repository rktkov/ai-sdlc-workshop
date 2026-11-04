package identity

import (
	"strings"
	"testing"
)

func TestSeedSourcesListed(t *testing.T) {
	sources := SeedSources(false)
	if len(sources) != 3 {
		t.Fatalf("sources = %d, want 3", len(sources))
	}
	bySource := map[SeedSource]SeedSourceInfo{}
	for _, info := range sources {
		bySource[info.Source] = info
	}
	if bySource[SeedExternalTrends].Available {
		t.Fatal("external-trends is available without an adapter")
	}
	if bySource[SeedSubscriptionsOnly].Available {
		t.Fatal("subscriptions-only is available without subscriptions")
	}
	if bySource[SeedYouTubeTrending].Available {
		t.Fatal("youtube-trending is available before the discovery adapter")
	}

	withSubs := SeedSources(true)
	for _, info := range withSubs {
		if info.Source == SeedSubscriptionsOnly && !info.Available {
			t.Fatal("subscriptions-only is unavailable with subscriptions")
		}
	}
}

func TestParseSeedSource(t *testing.T) {
	for _, value := range []string{"youtube-trending", "external-trends", "subscriptions-only", " SUBScriptions-only "} {
		if _, err := ParseSeedSource(value); err != nil {
			t.Fatalf("ParseSeedSource(%q): %v", value, err)
		}
	}
	if _, err := ParseSeedSource("unknown"); err == nil {
		t.Fatal("ParseSeedSource(unknown) = nil, want an error")
	}
}

func TestSelectSeedSource(t *testing.T) {
	source, _, err := SelectSeedSource("subscriptions-only", true)
	if err != nil {
		t.Fatalf("SelectSeedSource: %v", err)
	}
	if source != SeedSubscriptionsOnly {
		t.Fatalf("source = %q", source)
	}

	_, sources, err := SelectSeedSource("external-trends", true)
	if err == nil {
		t.Fatal("SelectSeedSource(external-trends) = nil, want an error")
	}
	if len(sources) != 3 {
		t.Fatalf("sources = %d, want 3", len(sources))
	}
	if !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("error = %q", err)
	}

	if _, _, err := SelectSeedSource("subscriptions-only", false); err == nil {
		t.Fatal("SelectSeedSource(no subscriptions) = nil, want an error")
	}
	if _, _, err := SelectSeedSource("bogus", true); err == nil {
		t.Fatal("SelectSeedSource(bogus) = nil, want an error")
	}
}
