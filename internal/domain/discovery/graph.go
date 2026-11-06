package discovery

import (
	"sort"
	"strings"

	"github.com/rktkov/ytfeed/internal/domain/identity"
)

// GraphChannelIDs selects subscribed and related channel IDs for upload reads.
// Subscribed channels come first, ordered by subscriber count and channel ID.
// Related channels keep their input order and never repeat a subscribed channel.
func GraphChannelIDs(subscriptions []identity.Subscription, related []string, channelLimit, relatedLimit int) []string {
	if channelLimit <= 0 {
		channelLimit = 20
	}
	if relatedLimit <= 0 {
		relatedLimit = 10
	}
	ordered := append([]identity.Subscription(nil), subscriptions...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].SubscriberCount != ordered[j].SubscriberCount {
			return ordered[i].SubscriberCount > ordered[j].SubscriberCount
		}
		return ordered[i].ChannelID < ordered[j].ChannelID
	})

	seen := map[string]bool{}
	out := make([]string, 0, channelLimit+relatedLimit)
	for _, subscription := range ordered {
		channelID := strings.TrimSpace(subscription.ChannelID)
		if channelID == "" || seen[channelID] {
			continue
		}
		seen[channelID] = true
		out = append(out, channelID)
		if len(out) >= channelLimit {
			break
		}
	}
	added := 0
	for _, channelID := range related {
		channelID = strings.TrimSpace(channelID)
		if channelID == "" || seen[channelID] {
			continue
		}
		seen[channelID] = true
		out = append(out, channelID)
		added++
		if added >= relatedLimit {
			break
		}
	}
	return out
}
