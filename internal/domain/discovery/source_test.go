package discovery

import (
	"context"
	"testing"
)

func TestParseRefreshSource(t *testing.T) {
	for _, value := range []string{"mixed", "trending", "graph", "fresh", "keywords", " MIXED "} {
		if _, err := ParseRefreshSource(value); err != nil {
			t.Fatalf("ParseRefreshSource(%q): %v", value, err)
		}
	}
	if _, err := ParseRefreshSource("random"); err == nil {
		t.Fatal("ParseRefreshSource(random) = nil, want an error")
	}
}

func TestRefreshSourceFlags(t *testing.T) {
	tests := []struct {
		source   RefreshSource
		search   bool
		trending bool
		graph    bool
	}{
		{RefreshMixed, true, true, true},
		{RefreshTrending, true, true, false},
		{RefreshGraph, true, false, true},
		{RefreshFresh, false, true, true},
		{RefreshKeywords, true, false, false},
	}
	for _, tt := range tests {
		if tt.source.UsesSearch() != tt.search {
			t.Fatalf("%s UsesSearch = %t", tt.source, tt.source.UsesSearch())
		}
		if tt.source.UsesTrending() != tt.trending {
			t.Fatalf("%s UsesTrending = %t", tt.source, tt.source.UsesTrending())
		}
		if tt.source.UsesGraph() != tt.graph {
			t.Fatalf("%s UsesGraph = %t", tt.source, tt.source.UsesGraph())
		}
	}
}

func TestRefreshSourceContext(t *testing.T) {
	if got := RefreshSourceFromContext(context.Background()); got != RefreshMixed {
		t.Fatalf("default = %q", got)
	}
	ctx := WithRefreshSource(context.Background(), RefreshGraph)
	if got := RefreshSourceFromContext(ctx); got != RefreshGraph {
		t.Fatalf("source = %q", got)
	}
}

func TestRefreshSourceFromContextOK(t *testing.T) {
	if _, ok := RefreshSourceFromContextOK(context.Background()); ok {
		t.Fatal("empty context reports a source")
	}
	ctx := WithRefreshSource(context.Background(), RefreshFresh)
	if source, ok := RefreshSourceFromContextOK(ctx); !ok || source != RefreshFresh {
		t.Fatalf("source = %q, %t", source, ok)
	}
}
