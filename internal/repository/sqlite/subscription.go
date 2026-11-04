package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/identity"
)

// SubscriptionRepository stores imported channels and their topic tags.
type SubscriptionRepository struct {
	db *sql.DB
}

// NewSubscriptionRepository returns a repository over the given database.
func NewSubscriptionRepository(db *sql.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

// UpsertPage inserts or updates one page in a single transaction.
func (r *SubscriptionRepository) UpsertPage(ctx context.Context, userID string, subscriptions []identity.Subscription) error {
	if len(subscriptions) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin subscription page: %w", err)
	}

	now := time.Now().UTC()
	for _, sub := range subscriptions {
		if _, err := tx.ExecContext(ctx, `INSERT INTO channels
			(user_id, channel_id, title, subscriber_count, imported_at)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(user_id, channel_id) DO UPDATE SET
				title = excluded.title,
				subscriber_count = excluded.subscriber_count,
				imported_at = excluded.imported_at`,
			userID, sub.ChannelID, sub.Title, sub.SubscriberCount, now,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("upsert channel %s: %w", sub.ChannelID, err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM channel_topic_tags
			WHERE user_id = ? AND channel_id = ?`, userID, sub.ChannelID,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("clear tags for %s: %w", sub.ChannelID, err)
		}
		for position, tag := range sub.TopicTags {
			if _, err := tx.ExecContext(ctx, `INSERT INTO channel_topic_tags
				(user_id, channel_id, tag, position) VALUES (?, ?, ?, ?)`,
				userID, sub.ChannelID, tag, position,
			); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("insert tag for %s: %w", sub.ChannelID, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit subscription page: %w", err)
	}
	return nil
}

// Count returns the number of stored channels for one user.
func (r *SubscriptionRepository) Count(ctx context.Context, userID string) (int, error) {
	var count int
	if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM channels WHERE user_id = ?`, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count channels: %w", err)
	}
	return count, nil
}

// List returns the stored channels in channel-id order.
func (r *SubscriptionRepository) List(ctx context.Context, userID string) ([]identity.Subscription, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT channel_id, title, subscriber_count
		FROM channels WHERE user_id = ? ORDER BY channel_id`, userID)
	if err != nil {
		return nil, fmt.Errorf("list channels: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []identity.Subscription
	for rows.Next() {
		var sub identity.Subscription
		if err := rows.Scan(&sub.ChannelID, &sub.Title, &sub.SubscriberCount); err != nil {
			return nil, fmt.Errorf("scan channel: %w", err)
		}
		out = append(out, sub)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate channels: %w", err)
	}
	for i := range out {
		tags, err := r.topicTags(ctx, userID, out[i].ChannelID)
		if err != nil {
			return nil, err
		}
		out[i].TopicTags = tags
	}
	return out, nil
}

func (r *SubscriptionRepository) topicTags(ctx context.Context, userID, channelID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT tag FROM channel_topic_tags
		WHERE user_id = ? AND channel_id = ? ORDER BY position`, userID, channelID)
	if err != nil {
		return nil, fmt.Errorf("read tags for %s: %w", channelID, err)
	}
	defer func() { _ = rows.Close() }()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tags: %w", err)
	}
	return tags, nil
}
