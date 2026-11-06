package discovery

import (
	"context"
	"fmt"
	"strings"
)

// RefreshSource selects the discovery sources for one refresh.
type RefreshSource string

const (
	RefreshMixed    RefreshSource = "mixed"
	RefreshTrending RefreshSource = "trending"
	RefreshGraph    RefreshSource = "graph"
	RefreshFresh    RefreshSource = "fresh"
	RefreshKeywords RefreshSource = "keywords"
)

// ParseRefreshSource validates one source mode.
func ParseRefreshSource(value string) (RefreshSource, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(RefreshMixed):
		return RefreshMixed, nil
	case string(RefreshTrending):
		return RefreshTrending, nil
	case string(RefreshGraph):
		return RefreshGraph, nil
	case string(RefreshFresh):
		return RefreshFresh, nil
	case string(RefreshKeywords):
		return RefreshKeywords, nil
	default:
		return "", fmt.Errorf("unknown refresh source %q", value)
	}
}

// UsesSearch reports whether the source runs keyword searches.
func (s RefreshSource) UsesSearch() bool {
	switch s {
	case RefreshMixed, RefreshTrending, RefreshGraph, RefreshKeywords:
		return true
	default:
		return false
	}
}

// UsesTrending reports whether the source reads mostPopular.
func (s RefreshSource) UsesTrending() bool {
	switch s {
	case RefreshMixed, RefreshTrending, RefreshFresh:
		return true
	default:
		return false
	}
}

// UsesGraph reports whether the source reads subscription uploads.
func (s RefreshSource) UsesGraph() bool {
	switch s {
	case RefreshMixed, RefreshGraph, RefreshFresh:
		return true
	default:
		return false
	}
}

type refreshSourceKey struct{}

// WithRefreshSource attaches a source mode to a context.
func WithRefreshSource(ctx context.Context, source RefreshSource) context.Context {
	return context.WithValue(ctx, refreshSourceKey{}, source)
}

// RefreshSourceFromContext returns the source mode, defaulting to mixed.
func RefreshSourceFromContext(ctx context.Context) RefreshSource {
	if source, ok := RefreshSourceFromContextOK(ctx); ok {
		return source
	}
	return RefreshMixed
}

// RefreshSourceFromContextOK reports the attached source mode, if any.
func RefreshSourceFromContextOK(ctx context.Context) (RefreshSource, bool) {
	source, ok := ctx.Value(refreshSourceKey{}).(RefreshSource)
	if !ok || source == "" {
		return "", false
	}
	return source, true
}
