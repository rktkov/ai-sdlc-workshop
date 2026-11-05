//go:build integration

package e2e

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/analysis"
	"github.com/rktkov/ytfeed/internal/domain/discovery"
	"github.com/rktkov/ytfeed/internal/domain/feed"
	"github.com/rktkov/ytfeed/internal/domain/identity"
	"github.com/rktkov/ytfeed/internal/repository/sqlite"
	"github.com/rktkov/ytfeed/internal/storage"
)

type resumePlanner struct{}

func (resumePlanner) Plan([]string) ([]discovery.Query, error) {
	return []discovery.Query{{Text: "ai", MaxResults: 1, Days: 30}}, nil
}

type resumeYouTube struct{}

func (resumeYouTube) Search(context.Context, string, discovery.Query) ([]discovery.Video, error) {
	return []discovery.Video{{VideoID: "dQw4w9WgXcQ", ChannelID: "c1", Title: "Video"}}, nil
}
func (resumeYouTube) Channels(context.Context, string, []string) (map[string]identity.Subscription, error) {
	return map[string]identity.Subscription{"c1": {ChannelID: "c1", Title: "Channel", SubscriberCount: 1000}}, nil
}
func (resumeYouTube) Videos(context.Context, string, []string) ([]discovery.Video, error) {
	return []discovery.Video{{VideoID: "dQw4w9WgXcQ", ChannelID: "c1", Title: "Video"}}, nil
}
func (resumeYouTube) SubscriptionPage(context.Context, string, string) (identity.SubscriptionPage, error) {
	return identity.SubscriptionPage{}, nil
}
func (resumeYouTube) MostPopular(context.Context, string, string) ([]discovery.Video, error) {
	return nil, nil
}

type resumeQuota struct{}

func (resumeQuota) Reserve(context.Context, string, discovery.QuotaCall) (discovery.Reservation, error) {
	return discovery.Reservation{ID: "r"}, nil
}
func (resumeQuota) Charge(context.Context, string) (discovery.Ledger, bool, error) {
	return discovery.Ledger{}, false, nil
}
func (resumeQuota) Release(context.Context, string) error { return nil }

type resumeFetcher struct{ calls int }

func (f *resumeFetcher) Fetch(context.Context, string) (string, error) {
	f.calls++
	return "transcript", nil
}

type resumeStore struct{}

func (resumeStore) Put(context.Context, string, string) (analysis.Staged, error) {
	return analysis.Staged{Hash: "hash", Bytes: 10}, nil
}
func (resumeStore) Get(context.Context, string, string) (analysis.Staged, error) {
	return analysis.Staged{}, analysis.ErrStageMiss
}
func (resumeStore) Delete(context.Context, string) error { return nil }
func (resumeStore) Sweep(context.Context, time.Time) (int, error) {
	return 0, nil
}

type resumeAnalyzer struct{ calls int }

func (a *resumeAnalyzer) Analyze(context.Context, string) (analysis.Analysis, error) {
	a.calls++
	return analysis.Analysis{
		State: analysis.StateAnalyzed, Topic: "ai", TLDR: "tldr",
		Similarity: "new territory", WhyThisCard: "why",
		AxisScores:    []analysis.AxisScore{{Axis: "depth", Score: 0.8, Justification: "good"}},
		RubricVersion: "v1", ModelID: "model", CompletedAt: time.Now().UTC(),
	}, nil
}

func TestResumeWithoutRefetch(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	for attempt := 0; attempt < 2; attempt++ {
		fetcher := &resumeFetcher{}
		analyzer := &resumeAnalyzer{}
		pipeline := &feed.Pipeline{
			UserID: "owner", AccessToken: "token", Interests: []string{"ai"},
			Size: 1, RubricVersion: "v1", ModelID: "model", BatchID: "b1", Staging: false,
			Planner: resumePlanner{}, YouTube: resumeYouTube{}, Quota: resumeQuota{},
			Fetcher: fetcher, Transcripts: resumeStore{}, Analyzer: analyzer,
			Analyses: sqlite.NewAnalysisRepository(db),
		}
		result, err := pipeline.Run(context.Background())
		if err != nil {
			t.Fatalf("Run %d: %v", attempt, err)
		}
		if len(result.Analyses) != 1 {
			t.Fatalf("analyses = %d", len(result.Analyses))
		}
		if attempt == 0 && fetcher.calls != 1 {
			t.Fatalf("first fetch calls = %d, want 1", fetcher.calls)
		}
		if attempt == 1 && fetcher.calls != 0 {
			t.Fatalf("resume fetch calls = %d, want 0", fetcher.calls)
		}
	}
}
