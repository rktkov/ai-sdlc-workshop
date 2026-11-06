package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/discovery"
)

// Region is the default YouTube region for search and trending.
const defaultRegion = "US"

type snippet struct {
	ChannelID   string `json:"channelId"`
	Title       string `json:"title"`
	Description string `json:"description"`
	PublishedAt string `json:"publishedAt"`
}

type searchItem struct {
	ID struct {
		VideoID string `json:"videoId"`
	} `json:"id"`
	Snippet snippet `json:"snippet"`
}

type videoItem struct {
	ID         string  `json:"id"`
	Snippet    snippet `json:"snippet"`
	Statistics struct {
		ViewCount string `json:"viewCount"`
	} `json:"statistics"`
	ContentDetails struct {
		Duration string `json:"duration"`
	} `json:"contentDetails"`
	TopicDetails struct {
		TopicCategories []string `json:"topicCategories"`
	} `json:"topicDetails"`
}

// Search runs one search.list call.
func (s *Source) Search(ctx context.Context, accessToken string, query discovery.Query) ([]discovery.Video, error) {
	page, err := s.SearchPage(ctx, accessToken, query, "")
	if err != nil {
		return nil, err
	}
	return page.Videos, nil
}

// SearchPage runs one search.list call with an optional page token.
func (s *Source) SearchPage(ctx context.Context, accessToken string, query discovery.Query, pageToken string) (discovery.VideoPage, error) {
	params := url.Values{}
	params.Set("part", "snippet")
	params.Set("type", "video")
	params.Set("q", queryText(query))
	params.Set("maxResults", strconv.Itoa(maxResults(query.MaxResults)))
	params.Set("order", order(query.Order))
	if pageToken != "" {
		params.Set("pageToken", pageToken)
	}
	days := query.Days
	if days <= 0 {
		days = 30
	}
	params.Set("publishedAfter", time.Now().UTC().AddDate(0, 0, -days).Format(time.RFC3339))
	if region := s.region(); region != "" {
		params.Set("regionCode", region)
	}

	body, err := s.get(ctx, "search", params, accessToken)
	if err != nil {
		return discovery.VideoPage{}, err
	}
	var page struct {
		Items         []searchItem `json:"items"`
		NextPageToken string       `json:"nextPageToken"`
	}
	if err := json.Unmarshal(body, &page); err != nil {
		return discovery.VideoPage{}, fmt.Errorf("decode search: %w", err)
	}
	out := make([]discovery.Video, 0, len(page.Items))
	for _, item := range page.Items {
		if item.ID.VideoID == "" {
			continue
		}
		out = append(out, discovery.Video{
			VideoID:     item.ID.VideoID,
			ChannelID:   item.Snippet.ChannelID,
			Title:       item.Snippet.Title,
			Description: item.Snippet.Description,
			PublishedAt: parseTime(item.Snippet.PublishedAt),
		})
	}
	return discovery.VideoPage{Videos: out, NextPageToken: page.NextPageToken}, nil
}

// PlaylistItems reads one page of an uploads playlist.
func (s *Source) PlaylistItems(ctx context.Context, accessToken, playlistID, pageToken string) (discovery.VideoPage, error) {
	if strings.TrimSpace(playlistID) == "" {
		return discovery.VideoPage{}, fmt.Errorf("playlist id is empty")
	}
	params := url.Values{}
	params.Set("part", "snippet,contentDetails")
	params.Set("playlistId", playlistID)
	params.Set("maxResults", "50")
	if pageToken != "" {
		params.Set("pageToken", pageToken)
	}
	body, err := s.get(ctx, "playlistItems", params, accessToken)
	if err != nil {
		return discovery.VideoPage{}, err
	}
	var page struct {
		Items []struct {
			Snippet struct {
				ChannelID   string `json:"channelId"`
				Title       string `json:"title"`
				Description string `json:"description"`
				PublishedAt string `json:"publishedAt"`
				ResourceID  struct {
					VideoID string `json:"videoId"`
				} `json:"resourceId"`
			} `json:"snippet"`
			ContentDetails struct {
				VideoID string `json:"videoId"`
			} `json:"contentDetails"`
		} `json:"items"`
		NextPageToken string `json:"nextPageToken"`
	}
	if err := json.Unmarshal(body, &page); err != nil {
		return discovery.VideoPage{}, fmt.Errorf("decode playlist items: %w", err)
	}
	out := make([]discovery.Video, 0, len(page.Items))
	for _, item := range page.Items {
		videoID := item.ContentDetails.VideoID
		if videoID == "" {
			videoID = item.Snippet.ResourceID.VideoID
		}
		if videoID == "" {
			continue
		}
		out = append(out, discovery.Video{
			VideoID:     videoID,
			ChannelID:   item.Snippet.ChannelID,
			Title:       item.Snippet.Title,
			Description: item.Snippet.Description,
			PublishedAt: parseTime(item.Snippet.PublishedAt),
		})
	}
	return discovery.VideoPage{Videos: out, NextPageToken: page.NextPageToken}, nil
}

// Videos returns metadata for video IDs in chunks of 50.
func (s *Source) Videos(ctx context.Context, accessToken string, videoIDs []string) ([]discovery.Video, error) {
	var out []discovery.Video
	for _, chunk := range chunkIDs(videoIDs, 50) {
		params := url.Values{}
		params.Set("part", "snippet,statistics,contentDetails,topicDetails")
		params.Set("id", strings.Join(chunk, ","))
		body, err := s.get(ctx, "videos", params, accessToken)
		if err != nil {
			return nil, err
		}
		var page struct {
			Items []videoItem `json:"items"`
		}
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("decode videos: %w", err)
		}
		out = append(out, parseVideos(page.Items)...)
	}
	return out, nil
}

// MostPopular runs videos.list?chart=mostPopular.
func (s *Source) MostPopular(ctx context.Context, accessToken, region string) ([]discovery.Video, error) {
	params := url.Values{}
	params.Set("part", "snippet,statistics,contentDetails,topicDetails")
	params.Set("chart", "mostPopular")
	params.Set("maxResults", "50")
	if region == "" {
		region = s.region()
	}
	if region != "" {
		params.Set("regionCode", region)
	}
	body, err := s.get(ctx, "videos", params, accessToken)
	if err != nil {
		return nil, err
	}
	var page struct {
		Items []videoItem `json:"items"`
	}
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("decode most popular: %w", err)
	}
	return parseVideos(page.Items), nil
}

func parseVideos(items []videoItem) []discovery.Video {
	out := make([]discovery.Video, 0, len(items))
	for _, item := range items {
		if item.ID == "" {
			continue
		}
		viewCount, _ := strconv.ParseInt(item.Statistics.ViewCount, 10, 64)
		out = append(out, discovery.Video{
			VideoID:     item.ID,
			ChannelID:   item.Snippet.ChannelID,
			Title:       item.Snippet.Title,
			Description: item.Snippet.Description,
			PublishedAt: parseTime(item.Snippet.PublishedAt),
			ViewCount:   viewCount,
			Duration:    item.ContentDetails.Duration,
			TopicTags:   append([]string(nil), item.TopicDetails.TopicCategories...),
		})
	}
	return out
}

func parseTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func queryText(query discovery.Query) string {
	if strings.TrimSpace(query.Text) != "" {
		return query.Text
	}
	return query.SearchText()
}

func maxResults(value int) int {
	if value < 1 || value > 50 {
		return 50
	}
	return value
}

func order(value string) string {
	if strings.TrimSpace(value) == "" {
		return "relevance"
	}
	return value
}

func (s *Source) region() string {
	if strings.TrimSpace(s.Region) == "" {
		return defaultRegion
	}
	return s.Region
}

var _ discovery.YouTube = (*Source)(nil)
