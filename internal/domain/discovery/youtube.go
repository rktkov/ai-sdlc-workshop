package discovery

import (
	"context"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/identity"
)

// Video is the normalized metadata used by discovery and analysis.
type Video struct {
	VideoID         string
	ChannelID       string
	Title           string
	Description     string
	PublishedAt     time.Time
	ViewCount       int64
	SubscriberCount int64
	Duration        string
	TopicTags       []string
	Source          CandidateSource
}

// VideoPage is one page of search or upload results.
type VideoPage struct {
	Videos        []Video
	NextPageToken string
}

// YouTube is the one adapter port for every YouTube Data API call.
type YouTube interface {
	Search(ctx context.Context, accessToken string, query Query) ([]Video, error)
	SearchPage(ctx context.Context, accessToken string, query Query, pageToken string) (VideoPage, error)
	Channels(ctx context.Context, accessToken string, channelIDs []string) (map[string]identity.Subscription, error)
	Videos(ctx context.Context, accessToken string, videoIDs []string) ([]Video, error)
	PlaylistItems(ctx context.Context, accessToken, playlistID, pageToken string) (VideoPage, error)
	SubscriptionPage(ctx context.Context, accessToken, pageToken string) (identity.SubscriptionPage, error)
	MostPopular(ctx context.Context, accessToken, region string) ([]Video, error)
}
