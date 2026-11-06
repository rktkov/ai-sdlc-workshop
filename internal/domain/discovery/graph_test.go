package discovery

import (
	"testing"

	"github.com/rktkov/ytfeed/internal/domain/identity"
)

func TestGraphChannelIDs(t *testing.T) {
	subscriptions := []identity.Subscription{
		{ChannelID: "c2", SubscriberCount: 100},
		{ChannelID: "c1", SubscriberCount: 300},
		{ChannelID: "c3", SubscriberCount: 100},
		{ChannelID: "", SubscriberCount: 500},
	}
	got := GraphChannelIDs(subscriptions, []string{"c2", "c9", "", "c8", "c7"}, 2, 2)
	want := []string{"c1", "c2", "c9", "c8"}
	if len(got) != len(want) {
		t.Fatalf("ids = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ids = %v, want %v", got, want)
		}
	}
}

func TestGraphChannelIDsDefaults(t *testing.T) {
	subscriptions := make([]identity.Subscription, 25)
	for i := range subscriptions {
		subscriptions[i] = identity.Subscription{ChannelID: "c" + string(rune('a'+i))}
	}
	related := make([]string, 15)
	for i := range related {
		related[i] = "r" + string(rune('a'+i))
	}
	got := GraphChannelIDs(subscriptions, related, 0, 0)
	if len(got) != 30 {
		t.Fatalf("ids = %d, want 30", len(got))
	}
}

func TestGraphChannelIDsEmpty(t *testing.T) {
	if got := GraphChannelIDs(nil, nil, 5, 5); len(got) != 0 {
		t.Fatalf("ids = %v, want none", got)
	}
}
