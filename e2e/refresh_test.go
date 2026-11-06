//go:build integration

package e2e

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/discovery"
	"github.com/rktkov/ytfeed/internal/domain/feed"
	"github.com/rktkov/ytfeed/internal/domain/identity"
	"github.com/rktkov/ytfeed/internal/repository/sqlite"
	"github.com/rktkov/ytfeed/internal/storage"
)

type refreshYouTube struct {
	videos []discovery.Video
}

func (y refreshYouTube) Search(context.Context, string, discovery.Query) ([]discovery.Video, error) {
	return y.videos, nil
}

func (y refreshYouTube) SearchPage(context.Context, string, discovery.Query, string) (discovery.VideoPage, error) {
	return discovery.VideoPage{Videos: y.videos}, nil
}

func (y refreshYouTube) PlaylistItems(context.Context, string, string, string) (discovery.VideoPage, error) {
	return discovery.VideoPage{}, nil
}

func (y refreshYouTube) Channels(context.Context, string, []string) (map[string]identity.Subscription, error) {
	out := make(map[string]identity.Subscription, len(y.videos))
	for _, video := range y.videos {
		out[video.ChannelID] = identity.Subscription{
			ChannelID: video.ChannelID, Title: video.Title, SubscriberCount: 200000,
		}
	}
	return out, nil
}

func (y refreshYouTube) Videos(context.Context, string, []string) ([]discovery.Video, error) {
	return y.videos, nil
}

func (y refreshYouTube) SubscriptionPage(context.Context, string, string) (identity.SubscriptionPage, error) {
	return identity.SubscriptionPage{}, nil
}

func (y refreshYouTube) MostPopular(context.Context, string, string) ([]discovery.Video, error) {
	return nil, nil
}

func refreshVideos(count int) []discovery.Video {
	out := make([]discovery.Video, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, discovery.Video{
			VideoID:   fmt.Sprintf("video-%02d", i),
			ChannelID: fmt.Sprintf("channel-%02d", i),
			Title:     "Video",
		})
	}
	return out
}

func TestRefreshReturnsTwentyNewVideos(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	source := refreshYouTube{videos: refreshVideos(50)}
	batches := sqlite.NewBatchRepository(db)
	exclusions := sqlite.NewExclusionRepository(db)
	now := time.Now().UTC()
	var first []string
	for refresh := 0; refresh < 2; refresh++ {
		excluded, err := exclusions.ForUser(context.Background(), "owner", now.AddDate(0, 0, -90))
		if err != nil {
			t.Fatalf("ForUser: %v", err)
		}
		pipeline := &feed.Pipeline{
			UserID: "owner", AccessToken: "token", Interests: []string{"ai", "go", "llm", "video", "data"},
			Size: 20, MicroTarget: 0.2, RubricVersion: "v1", ModelID: "model",
			BatchID: fmt.Sprintf("refresh-%d", refresh), Source: discovery.RefreshKeywords,
			Exclusions: excluded, ChannelCap: 2, AnalysisConcurrency: 2,
			Planner: discovery.QueryPlanner{}, YouTube: source, Quota: resumeQuota{},
			Fetcher: &resumeFetcher{}, Transcripts: resumeStore{}, Analyzer: &resumeAnalyzer{},
			Analyses: sqlite.NewAnalysisRepository(db),
		}
		result, err := pipeline.Run(context.Background())
		if err != nil {
			t.Fatalf("Run %d: %v", refresh, err)
		}
		if len(result.Batch.Cards) != 20 {
			t.Fatalf("refresh %d cards = %d, want 20", refresh, len(result.Batch.Cards))
		}
		seen := map[string]bool{}
		ids := make([]string, 0, len(result.Batch.Cards))
		for _, card := range result.Batch.Cards {
			if seen[card.VideoID] {
				t.Fatalf("refresh %d repeated %s", refresh, card.VideoID)
			}
			seen[card.VideoID] = true
			ids = append(ids, card.VideoID)
		}
		if refresh == 0 {
			first = ids
		} else {
			before := map[string]bool{}
			for _, id := range first {
				before[id] = true
			}
			for _, id := range ids {
				if before[id] {
					t.Fatalf("second refresh repeated %s", id)
				}
			}
		}
		if err := batches.Save(context.Background(), result.Batch); err != nil {
			t.Fatalf("Save %d: %v", refresh, err)
		}
	}
}

func TestRefreshShortfallDoesNotRepeat(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	source := refreshYouTube{videos: refreshVideos(25)}
	batches := sqlite.NewBatchRepository(db)
	exclusions := sqlite.NewExclusionRepository(db)
	now := time.Now().UTC()
	first := map[string]bool{}
	for refresh := 0; refresh < 2; refresh++ {
		excluded, err := exclusions.ForUser(context.Background(), "owner", now.AddDate(0, 0, -90))
		if err != nil {
			t.Fatalf("ForUser: %v", err)
		}
		pipeline := &feed.Pipeline{
			UserID: "owner", AccessToken: "token", Interests: []string{"ai", "go", "llm", "video", "data"},
			Size: 20, MicroTarget: 0.2, RubricVersion: "v1", ModelID: "model",
			BatchID: fmt.Sprintf("short-%d", refresh), Source: discovery.RefreshKeywords,
			Exclusions: excluded, ChannelCap: 2, AnalysisConcurrency: 2,
			Planner: discovery.QueryPlanner{}, YouTube: source, Quota: resumeQuota{},
			Fetcher: &resumeFetcher{}, Transcripts: resumeStore{}, Analyzer: &resumeAnalyzer{},
			Analyses: sqlite.NewAnalysisRepository(db),
		}
		result, err := pipeline.Run(context.Background())
		if err != nil {
			t.Fatalf("Run %d: %v", refresh, err)
		}
		for _, card := range result.Batch.Cards {
			if refresh == 1 && first[card.VideoID] {
				t.Fatalf("second refresh repeated %s", card.VideoID)
			}
			first[card.VideoID] = true
		}
		if refresh == 1 {
			if len(result.Batch.Cards) != 5 {
				t.Fatalf("second cards = %d, want 5", len(result.Batch.Cards))
			}
			if result.Batch.State != feed.BatchDegraded || result.Batch.DegradationReason != feed.DegradeShortfall {
				t.Fatalf("batch = %+v", result.Batch)
			}
		}
		if err := batches.Save(context.Background(), result.Batch); err != nil {
			t.Fatalf("Save %d: %v", refresh, err)
		}
	}
}
