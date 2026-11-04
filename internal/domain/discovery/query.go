// Package discovery plans and bounds the YouTube candidate pool.
package discovery

import (
	"errors"
	"fmt"
	"strings"
)

// CandidateSource tags where a candidate came from.
type CandidateSource string

const (
	SourceGraph    CandidateSource = "graph"
	SourceSearch   CandidateSource = "search"
	SourceTrending CandidateSource = "trending"
)

// Query is one planned search call.
type Query struct {
	Interest   string
	Text       string
	Source     CandidateSource
	Order      string
	MaxResults int
	Days       int
}

// SearchText returns the normalized search phrase.
func (q Query) SearchText() string {
	value := strings.ToLower(strings.TrimSpace(q.Interest))
	value = strings.ReplaceAll(value, "_", " ")
	value = strings.ReplaceAll(value, "-", " ")
	return strings.Join(strings.Fields(value), " ")
}

// QueryPlanner turns interests into deterministic search queries.
type QueryPlanner struct {
	MaxResults int
	Days       int
}

// Plan returns one query per interest in input order.
func (p QueryPlanner) Plan(interests []string) ([]Query, error) {
	normalized, err := normalizeInterests(interests)
	if err != nil {
		return nil, err
	}
	maxResults := p.MaxResults
	if maxResults == 0 {
		maxResults = 50
	}
	if maxResults < 1 || maxResults > 50 {
		return nil, fmt.Errorf("max results must be between 1 and 50")
	}
	days := p.Days
	if days == 0 {
		days = 30
	}
	if days < 1 {
		return nil, fmt.Errorf("days must be positive")
	}

	queries := make([]Query, 0, len(normalized))
	for _, interest := range normalized {
		query := Query{
			Interest:   interest,
			Source:     SourceSearch,
			Order:      "relevance",
			MaxResults: maxResults,
			Days:       days,
		}
		query.Text = query.SearchText()
		queries = append(queries, query)
	}
	return queries, nil
}

func normalizeInterests(values []string) ([]string, error) {
	if len(values) < 1 || len(values) > 10 {
		return nil, errors.New("interests must contain 1 to 10 values")
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		slug := normalizeSlug(value)
		if slug == "" {
			return nil, errors.New("interest must not be empty")
		}
		if seen[slug] {
			return nil, fmt.Errorf("interest %q is duplicated", value)
		}
		seen[slug] = true
		out = append(out, slug)
	}
	return out, nil
}

func normalizeSlug(value string) string {
	slug := strings.ToLower(strings.TrimSpace(value))
	slug = strings.ReplaceAll(slug, "_", "-")
	slug = strings.Join(strings.Fields(slug), "-")
	return slug
}
