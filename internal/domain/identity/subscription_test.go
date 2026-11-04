package identity

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"
)

type fakeSubscriptionSource struct {
	pages   map[string]SubscriptionPage
	metas   map[string]Subscription
	pageErr error
	metaErr error
	calls   int
}

func (f *fakeSubscriptionSource) SubscriptionPage(_ context.Context, _ string, pageToken string) (SubscriptionPage, error) {
	f.calls++
	if f.pageErr != nil {
		return SubscriptionPage{}, f.pageErr
	}
	page, ok := f.pages[pageToken]
	if !ok {
		return SubscriptionPage{}, fmt.Errorf("unexpected page token %q", pageToken)
	}
	return page, nil
}

func (f *fakeSubscriptionSource) Channels(_ context.Context, _ string, channelIDs []string) (map[string]Subscription, error) {
	if f.metaErr != nil {
		return nil, f.metaErr
	}
	out := make(map[string]Subscription, len(channelIDs))
	for _, id := range channelIDs {
		if meta, ok := f.metas[id]; ok {
			out[id] = meta
		}
	}
	return out, nil
}

type fakeSubscriptionStore struct {
	items map[string]Subscription
	pages int
}

func newFakeSubscriptionStore() *fakeSubscriptionStore {
	return &fakeSubscriptionStore{items: map[string]Subscription{}}
}

func (f *fakeSubscriptionStore) UpsertPage(_ context.Context, _ string, subs []Subscription) error {
	f.pages++
	for _, sub := range subs {
		f.items[sub.ChannelID] = sub
	}
	return nil
}

func (f *fakeSubscriptionStore) Count(context.Context, string) (int, error) {
	return len(f.items), nil
}

func (f *fakeSubscriptionStore) List(context.Context, string) ([]Subscription, error) {
	out := make([]Subscription, 0, len(f.items))
	for _, sub := range f.items {
		out = append(out, sub)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ChannelID < out[j].ChannelID })
	return out, nil
}

func TestImporterHappy(t *testing.T) {
	source := &fakeSubscriptionSource{
		pages: map[string]SubscriptionPage{
			"":   {Items: []Subscription{{ChannelID: "c1", Title: "C1"}, {ChannelID: "c2", Title: "C2"}}, NextPageToken: "p2"},
			"p2": {Items: []Subscription{{ChannelID: "c3", Title: "C3"}}},
		},
		metas: map[string]Subscription{
			"c1": {ChannelID: "c1", Title: "C1", SubscriberCount: 100, TopicTags: []string{"tech"}},
			"c2": {ChannelID: "c2", Title: "C2", SubscriberCount: 200},
			"c3": {ChannelID: "c3", Title: "C3", SubscriberCount: 300, TopicTags: []string{"science", "tech"}},
		},
	}
	store := newFakeSubscriptionStore()
	importer := &Importer{Source: source, Store: store, UserID: "owner"}

	count, err := importer.Import(context.Background(), "access")
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if count != 3 {
		t.Fatalf("count = %d, want 3", count)
	}
	if source.calls != 2 || store.pages != 2 {
		t.Fatalf("source calls = %d, store pages = %d", source.calls, store.pages)
	}
	if got := store.items["c1"]; got.SubscriberCount != 100 || len(got.TopicTags) != 1 {
		t.Fatalf("c1 = %+v", got)
	}
	if got := store.items["c3"]; len(got.TopicTags) != 2 {
		t.Fatalf("c3 = %+v", got)
	}
}

func TestImporterIdempotent(t *testing.T) {
	source := &fakeSubscriptionSource{
		pages: map[string]SubscriptionPage{"": {Items: []Subscription{{ChannelID: "c1", Title: "C1"}}}},
		metas: map[string]Subscription{"c1": {ChannelID: "c1", Title: "C1"}},
	}
	store := newFakeSubscriptionStore()
	importer := &Importer{Source: source, Store: store, UserID: "owner"}

	for i := 0; i < 2; i++ {
		if _, err := importer.Import(context.Background(), "access"); err != nil {
			t.Fatalf("Import %d: %v", i, err)
		}
	}
	if count, _ := store.Count(context.Background(), "owner"); count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
}

func TestImporterPartialPage(t *testing.T) {
	source := &fakeSubscriptionSource{
		pages:   map[string]SubscriptionPage{"": {Items: []Subscription{{ChannelID: "c1"}}, NextPageToken: "p2"}},
		pageErr: errors.New("boom"),
		metas:   map[string]Subscription{"c1": {ChannelID: "c1"}},
	}
	store := newFakeSubscriptionStore()
	importer := &Importer{Source: source, Store: store, UserID: "owner"}
	if _, err := importer.Import(context.Background(), "access"); err == nil {
		t.Fatal("Import = nil, want an error")
	}
	if count, _ := store.Count(context.Background(), "owner"); count != 0 {
		t.Fatalf("count = %d, want 0", count)
	}
}

func TestImporterMissingChannelID(t *testing.T) {
	source := &fakeSubscriptionSource{
		pages: map[string]SubscriptionPage{"": {Items: []Subscription{{Title: "no id"}}}},
		metas: map[string]Subscription{},
	}
	store := newFakeSubscriptionStore()
	importer := &Importer{Source: source, Store: store, UserID: "owner"}
	_, err := importer.Import(context.Background(), "access")
	if err == nil {
		t.Fatal("Import = nil, want an error")
	}
	if store.pages != 0 {
		t.Fatalf("store pages = %d, want 0", store.pages)
	}
}

func TestImporterNilDependencies(t *testing.T) {
	importer := &Importer{UserID: "owner"}
	if _, err := importer.Import(context.Background(), "access"); err == nil {
		t.Fatal("Import = nil, want an error")
	}
	importer = &Importer{Source: &fakeSubscriptionSource{}, UserID: "owner"}
	if _, err := importer.Import(context.Background(), "access"); err == nil {
		t.Fatal("Import = nil, want an error")
	}
}

func TestImporterChannelError(t *testing.T) {
	source := &fakeSubscriptionSource{
		pages:   map[string]SubscriptionPage{"": {Items: []Subscription{{ChannelID: "c1"}}}},
		metaErr: fmt.Errorf("channels down"),
	}
	store := newFakeSubscriptionStore()
	importer := &Importer{Source: source, Store: store, UserID: "owner"}
	if _, err := importer.Import(context.Background(), "access"); err == nil {
		t.Fatal("Import = nil, want an error")
	}
	if store.pages != 0 {
		t.Fatalf("store pages = %d, want 0", store.pages)
	}
}
