package cli

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/discovery"
	"github.com/rktkov/ytfeed/internal/domain/feed"
	"github.com/rktkov/ytfeed/internal/domain/identity"
	"github.com/rktkov/ytfeed/internal/domain/taste"
)

type fakeTokenReader struct {
	token identity.Token
	err   error
}

func (f fakeTokenReader) ValidToken(context.Context) (identity.Token, error) { return f.token, f.err }

type fakeProfileReader struct {
	interests    []string
	profile      taste.ProfileSnapshot
	interestsErr error
	snapshotErr  error
}

func (f fakeProfileReader) Interests(context.Context, string) ([]string, error) {
	return f.interests, f.interestsErr
}
func (f fakeProfileReader) Snapshot(context.Context, string) (taste.ProfileSnapshot, error) {
	return f.profile, f.snapshotErr
}

type fakeChannelLister struct {
	channels []identity.Subscription
	err      error
}

func (f fakeChannelLister) List(context.Context, string) ([]identity.Subscription, error) {
	return f.channels, f.err
}

type fakeExclusionReader struct {
	set discovery.ExclusionSet
	err error
}

func (f fakeExclusionReader) ForUser(context.Context, string, time.Time) (discovery.ExclusionSet, error) {
	return f.set, f.err
}

type fakeBatchSaver struct {
	batch feed.Batch
	err   error
}

func (f *fakeBatchSaver) Save(_ context.Context, batch feed.Batch) error {
	f.batch = batch
	return f.err
}

type fakeRunner struct {
	result feed.Result
	err    error
}

func (f fakeRunner) Run(context.Context) (feed.Result, error) { return f.result, f.err }

func liveBatch() feed.Batch {
	return feed.Batch{
		ID: "b1", UserID: "owner", Size: 1, State: feed.BatchCompleted,
		Cards: []feed.Card{{
			CardID: "c", VideoID: "v", Title: "t", Channel: "ch", State: feed.StateAnalyzed,
			AnalysisID: "a", Score: 0.5, Position: 0,
		}},
	}
}

func liveProfile() taste.ProfileSnapshot {
	return taste.ProfileSnapshot{
		UserID: "owner", RubricVersion: "v1",
		Axes: []taste.Axis{{Name: "depth", Weight: 0.5, PositiveExamples: []string{"p"}, NegativeExamples: []string{"n"}}},
	}
}

func liveComposerFixture() (*liveComposer, *fakeBatchSaver, *[]string) {
	store := &fakeBatchSaver{}
	var seen []string
	composer := &liveComposer{
		UserID: "owner", Size: 1,
		Tokens:     fakeTokenReader{token: identity.Token{AccessToken: "access"}},
		Profiles:   fakeProfileReader{interests: []string{"ai-agents"}, profile: liveProfile()},
		Channels:   fakeChannelLister{},
		Exclusions: fakeExclusionReader{set: discovery.ExclusionSet{Watched: map[string]bool{"old": true}}},
		Store:      store,
		Factory: func(profile taste.ProfileSnapshot, interests []string, token identity.Token, exclusions discovery.ExclusionSet) pipelineRunner {
			seen = append(seen, interests...)
			if len(exclusions.Watched) != 1 {
				seen = append(seen, "missing-exclusions")
			}
			return fakeRunner{result: feed.Result{Batch: liveBatch()}}
		},
	}
	return composer, store, &seen
}

func TestLiveComposerHappy(t *testing.T) {
	composer, store, seen := liveComposerFixture()
	batch, err := composer.Compose(context.Background(), "owner")
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	if batch.ID != "b1" || store.batch.ID != "b1" {
		t.Fatalf("batch = %+v", batch)
	}
	if len(*seen) != 1 || (*seen)[0] != "ai-agents" {
		t.Fatalf("factory saw %v", *seen)
	}
}

func TestLiveComposerTokenMissing(t *testing.T) {
	composer, _, _ := liveComposerFixture()
	composer.Tokens = fakeTokenReader{err: identity.ErrTokenNotFound}
	if _, err := composer.Compose(context.Background(), "owner"); !errors.Is(err, feed.ErrNotAuthenticated) {
		t.Fatalf("Compose = %v, want ErrNotAuthenticated", err)
	}
}

func TestLiveComposerNoInterests(t *testing.T) {
	composer, _, _ := liveComposerFixture()
	composer.Profiles = fakeProfileReader{interestsErr: taste.ErrProfileNotFound, profile: liveProfile()}
	composer.Channels = fakeChannelLister{}
	if _, err := composer.Compose(context.Background(), "owner"); !errors.Is(err, feed.ErrNoInterests) {
		t.Fatalf("Compose = %v, want ErrNoInterests", err)
	}
}

func TestLiveComposerInterestsFromChannels(t *testing.T) {
	composer, _, seen := liveComposerFixture()
	composer.Profiles = fakeProfileReader{profile: liveProfile()}
	composer.Channels = fakeChannelLister{channels: []identity.Subscription{
		{Title: "AI Agents"}, {Title: "Local LLMs"}, {Title: "AI Agents"},
	}}
	if _, err := composer.Compose(context.Background(), "owner"); err != nil {
		t.Fatalf("Compose: %v", err)
	}
	if len(*seen) != 2 || (*seen)[0] != "ai-agents" || (*seen)[1] != "local-llms" {
		t.Fatalf("interests = %v", *seen)
	}
}

func TestLiveComposerNoProfile(t *testing.T) {
	composer, _, _ := liveComposerFixture()
	composer.Profiles = fakeProfileReader{interests: []string{"ai"}, snapshotErr: taste.ErrProfileNotFound}
	if _, err := composer.Compose(context.Background(), "owner"); !errors.Is(err, feed.ErrNoProfile) {
		t.Fatalf("Compose = %v, want ErrNoProfile", err)
	}
}

func TestLiveComposerQuotaShortfall(t *testing.T) {
	composer, store, _ := liveComposerFixture()
	composer.Factory = func(taste.ProfileSnapshot, []string, identity.Token, discovery.ExclusionSet) pipelineRunner {
		return fakeRunner{result: feed.Result{Shortfall: "quota"}}
	}
	batch, err := composer.Compose(context.Background(), "owner")
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	if batch.State != feed.BatchDegraded || batch.DegradationReason != feed.DegradeQuota {
		t.Fatalf("batch = %+v", batch)
	}
	if store.batch.State != feed.BatchDegraded {
		t.Fatalf("stored = %+v", store.batch)
	}
}

func TestLiveComposerErrors(t *testing.T) {
	composer, _, _ := liveComposerFixture()
	composer.Exclusions = fakeExclusionReader{err: errors.New("db")}
	if _, err := composer.Compose(context.Background(), "owner"); err == nil {
		t.Fatal("Compose(exclusion error) = nil, want an error")
	}
	composer, _, _ = liveComposerFixture()
	composer.Factory = func(taste.ProfileSnapshot, []string, identity.Token, discovery.ExclusionSet) pipelineRunner {
		return fakeRunner{err: errors.New("pipeline")}
	}
	if _, err := composer.Compose(context.Background(), "owner"); err == nil {
		t.Fatal("Compose(pipeline error) = nil, want an error")
	}
	composer, _, _ = liveComposerFixture()
	composer.Store = &fakeBatchSaver{err: errors.New("save")}
	if _, err := composer.Compose(context.Background(), "owner"); err == nil {
		t.Fatal("Compose(store error) = nil, want an error")
	}
	if _, err := (&liveComposer{}).Compose(context.Background(), "owner"); err == nil {
		t.Fatal("Compose(empty) = nil, want an error")
	}
}
