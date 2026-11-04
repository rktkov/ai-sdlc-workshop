// Package youtube implements the YouTube Data API adapters.
package youtube

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/rktkov/ytfeed/internal/domain/identity"
)

// DefaultBaseURL is the YouTube Data API v3 endpoint.
const DefaultBaseURL = "https://www.googleapis.com/youtube/v3"

// Source reads subscriptions and channel metadata.
type Source struct {
	BaseURL string
	Client  *http.Client
	Region  string
}

// NewSource returns a YouTube source.
func NewSource(baseURL string, client *http.Client) *Source {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DefaultBaseURL
	}
	if client == nil {
		client = &http.Client{}
	}
	return &Source{BaseURL: strings.TrimRight(baseURL, "/"), Client: client}
}

type subscriptionPage struct {
	Items []struct {
		Snippet struct {
			Title      string `json:"title"`
			ResourceID struct {
				ChannelID string `json:"channelId"`
			} `json:"resourceId"`
		} `json:"snippet"`
	} `json:"items"`
	NextPageToken string `json:"nextPageToken"`
}

// SubscriptionPage reads one page of subscriptions.
func (s *Source) SubscriptionPage(ctx context.Context, accessToken, pageToken string) (identity.SubscriptionPage, error) {
	params := url.Values{}
	params.Set("part", "snippet,contentDetails")
	params.Set("mine", "true")
	params.Set("maxResults", "50")
	if pageToken != "" {
		params.Set("pageToken", pageToken)
	}
	body, err := s.get(ctx, "subscriptions", params, accessToken)
	if err != nil {
		return identity.SubscriptionPage{}, err
	}

	var page subscriptionPage
	if err := json.Unmarshal(body, &page); err != nil {
		return identity.SubscriptionPage{}, fmt.Errorf("decode subscriptions: %w", err)
	}
	out := identity.SubscriptionPage{NextPageToken: page.NextPageToken}
	for _, item := range page.Items {
		if strings.TrimSpace(item.Snippet.ResourceID.ChannelID) == "" {
			return identity.SubscriptionPage{}, fmt.Errorf("subscription has no channel id")
		}
		out.Items = append(out.Items, identity.Subscription{
			ChannelID: item.Snippet.ResourceID.ChannelID,
			Title:     item.Snippet.Title,
		})
	}
	return out, nil
}

type channelPage struct {
	Items []struct {
		ID      string `json:"id"`
		Snippet struct {
			Title string `json:"title"`
		} `json:"snippet"`
		Statistics struct {
			SubscriberCount string `json:"subscriberCount"`
		} `json:"statistics"`
		TopicDetails struct {
			TopicCategories []string `json:"topicCategories"`
		} `json:"topicDetails"`
	} `json:"items"`
}

// Channels reads metadata for up to any number of channel IDs.
func (s *Source) Channels(ctx context.Context, accessToken string, channelIDs []string) (map[string]identity.Subscription, error) {
	out := make(map[string]identity.Subscription, len(channelIDs))
	for _, chunk := range chunkIDs(channelIDs, 50) {
		params := url.Values{}
		params.Set("part", "snippet,statistics,topicDetails")
		params.Set("id", strings.Join(chunk, ","))
		body, err := s.get(ctx, "channels", params, accessToken)
		if err != nil {
			return nil, err
		}
		var page channelPage
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("decode channels: %w", err)
		}
		for _, item := range page.Items {
			subscriberCount, _ := strconv.ParseInt(item.Statistics.SubscriberCount, 10, 64)
			out[item.ID] = identity.Subscription{
				ChannelID:       item.ID,
				Title:           item.Snippet.Title,
				SubscriberCount: subscriberCount,
				TopicTags:       append([]string(nil), item.TopicDetails.TopicCategories...),
			}
		}
	}
	return out, nil
}

func (s *Source) get(ctx context.Context, endpoint string, params url.Values, accessToken string) ([]byte, error) {
	requestURL, err := s.requestURL(endpoint, params)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create youtube request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call youtube %s: %w", endpoint, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("read youtube %s: %w", endpoint, err)
	}
	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return nil, identity.ErrUnauthorized
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("youtube %s returned %d", endpoint, resp.StatusCode)
	default:
		return body, nil
	}
}

func (s *Source) requestURL(endpoint string, params url.Values) (string, error) {
	parsed, err := url.Parse(s.BaseURL + "/" + endpoint)
	if err != nil {
		return "", fmt.Errorf("parse youtube base url: %w", err)
	}
	parsed.RawQuery = params.Encode()
	return parsed.String(), nil
}

func chunkIDs(values []string, size int) [][]string {
	if size <= 0 {
		size = 50
	}
	var out [][]string
	for len(values) > 0 {
		n := size
		if len(values) < n {
			n = len(values)
		}
		out = append(out, values[:n])
		values = values[n:]
	}
	return out
}
