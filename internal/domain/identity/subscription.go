package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Subscription is one imported channel.
type Subscription struct {
	ChannelID       string
	Title           string
	SubscriberCount int64
	TopicTags       []string
}

// SubscriptionPage is one page from the subscription source.
type SubscriptionPage struct {
	Items         []Subscription
	NextPageToken string
}

// SubscriptionSource reads subscriptions and channel metadata.
type SubscriptionSource interface {
	SubscriptionPage(ctx context.Context, accessToken, pageToken string) (SubscriptionPage, error)
	Channels(ctx context.Context, accessToken string, channelIDs []string) (map[string]Subscription, error)
}

// SubscriptionStore upserts one page at a time.
type SubscriptionStore interface {
	UpsertPage(ctx context.Context, userID string, subscriptions []Subscription) error
	Count(ctx context.Context, userID string) (int, error)
	List(ctx context.Context, userID string) ([]Subscription, error)
}

// Importer pages through the account graph and upserts each page.
type Importer struct {
	Source SubscriptionSource
	Store  SubscriptionStore
	UserID string
}

// Import reads every page and returns the stored subscription count.
func (i *Importer) Import(ctx context.Context, accessToken string) (int, error) {
	if i.Source == nil || i.Store == nil {
		return 0, errors.New("subscription importer is not configured")
	}
	if strings.TrimSpace(i.UserID) == "" {
		return 0, errors.New("subscription importer has no user id")
	}

	pageToken := ""
	for {
		page, err := i.Source.SubscriptionPage(ctx, accessToken, pageToken)
		if err != nil {
			return 0, fmt.Errorf("read subscription page: %w", err)
		}
		if len(page.Items) > 0 {
			enriched, err := i.enrich(ctx, accessToken, page.Items)
			if err != nil {
				return 0, err
			}
			if err := i.Store.UpsertPage(ctx, i.UserID, enriched); err != nil {
				return 0, fmt.Errorf("store subscription page: %w", err)
			}
		}
		if page.NextPageToken == "" {
			break
		}
		pageToken = page.NextPageToken
	}
	return i.Store.Count(ctx, i.UserID)
}

func (i *Importer) enrich(ctx context.Context, accessToken string, items []Subscription) ([]Subscription, error) {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.ChannelID) == "" {
			return nil, errors.New("subscription has no channel id")
		}
		ids = append(ids, item.ChannelID)
	}
	metas, err := i.Source.Channels(ctx, accessToken, dedupeStrings(ids))
	if err != nil {
		return nil, fmt.Errorf("read channel metadata: %w", err)
	}

	out := make([]Subscription, 0, len(items))
	for _, item := range items {
		if meta, ok := metas[item.ChannelID]; ok {
			if meta.Title != "" {
				item.Title = meta.Title
			}
			item.SubscriberCount = meta.SubscriberCount
			item.TopicTags = append([]string(nil), meta.TopicTags...)
		}
		if item.Title == "" {
			item.Title = item.ChannelID
		}
		out = append(out, item)
	}
	return out, nil
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
