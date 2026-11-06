package cli

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/rktkov/ytfeed/internal/config"
	"github.com/rktkov/ytfeed/internal/domain/analysis"
	"github.com/rktkov/ytfeed/internal/domain/discovery"
	"github.com/rktkov/ytfeed/internal/domain/feed"
	"github.com/rktkov/ytfeed/internal/domain/identity"
	"github.com/rktkov/ytfeed/internal/domain/taste"
	"github.com/rktkov/ytfeed/internal/integration/llm"
	"github.com/rktkov/ytfeed/internal/integration/transcripts"
	"github.com/rktkov/ytfeed/internal/integration/youtube"
	"github.com/rktkov/ytfeed/internal/integration/ytdlp"
	"github.com/rktkov/ytfeed/internal/repository/sqlite"
	"github.com/rktkov/ytfeed/internal/repository/tokenfile"
)

type tokenReader interface {
	ValidToken(ctx context.Context) (identity.Token, error)
}

type fileTokenReader struct {
	store *tokenfile.FileStore
}

func (r fileTokenReader) ValidToken(ctx context.Context) (identity.Token, error) {
	return r.store.Read(ctx)
}

type profileReader interface {
	Interests(ctx context.Context, userID string) ([]string, error)
	Snapshot(ctx context.Context, userID string) (taste.ProfileSnapshot, error)
}

type channelLister interface {
	List(ctx context.Context, userID string) ([]identity.Subscription, error)
}

type exclusionReader interface {
	ForUser(ctx context.Context, userID string, since time.Time) (discovery.ExclusionSet, error)
}

type batchSaver interface {
	Save(ctx context.Context, batch feed.Batch) error
}

type pipelineRunner interface {
	Run(ctx context.Context) (feed.Result, error)
}

type pipelineFactory func(profile taste.ProfileSnapshot, interests []string, token identity.Token, exclusions discovery.ExclusionSet, source discovery.RefreshSource, subscriptions []identity.Subscription, progress feed.ProgressFunc) pipelineRunner

type liveComposer struct {
	UserID     string
	Size       int
	Source     discovery.RefreshSource
	Tokens     tokenReader
	Profiles   profileReader
	Channels   channelLister
	Exclusions exclusionReader
	Store      batchSaver
	Factory    pipelineFactory
	Now        func() time.Time
}

// Compose builds and saves one batch for the live feed.
func (c *liveComposer) Compose(ctx context.Context, userID string) (feed.Batch, error) {
	return c.ComposeWithProgress(ctx, userID, nil)
}

// ComposeWithProgress builds and saves one batch and reports each step.
func (c *liveComposer) ComposeWithProgress(ctx context.Context, userID string, progress feed.ProgressFunc) (feed.Batch, error) {
	if c.Tokens == nil || c.Profiles == nil || c.Store == nil || c.Factory == nil {
		return feed.Batch{}, errors.New("live composer is not configured")
	}
	now := c.now()
	source, ok := discovery.RefreshSourceFromContextOK(ctx)
	if !ok {
		source = c.Source
	}
	if source == "" {
		source = discovery.RefreshMixed
	}
	c.report(progress, "source", "refresh source: "+string(source), nil)
	c.report(progress, "auth", "checking the Google token", nil)
	token, err := c.Tokens.ValidToken(ctx)
	if err != nil {
		return feed.Batch{}, fmt.Errorf("%w: %v", feed.ErrNotAuthenticated, err)
	}
	c.report(progress, "auth_done", "token ready", nil)
	c.report(progress, "interests", "loading interests", nil)
	interests, err := c.Profiles.Interests(ctx, userID)
	if err != nil && !errors.Is(err, taste.ErrProfileNotFound) {
		return feed.Batch{}, err
	}
	if len(interests) == 0 {
		interests, err = c.interestsFromChannels(ctx, userID)
		if err != nil {
			return feed.Batch{}, err
		}
	}
	c.report(progress, "profile", "loading the taste profile", nil)
	profile, err := c.Profiles.Snapshot(ctx, userID)
	if err != nil {
		if errors.Is(err, taste.ErrProfileNotFound) {
			return feed.Batch{}, fmt.Errorf("%w: %v", feed.ErrNoProfile, err)
		}
		return feed.Batch{}, err
	}
	exclusions, err := c.loadExclusions(ctx, userID, now)
	if err != nil {
		return feed.Batch{}, err
	}
	subscriptions, err := c.loadSubscriptions(ctx, userID, source, progress)
	if err != nil {
		return feed.Batch{}, err
	}
	runner := c.Factory(profile, interests, token, exclusions, source, subscriptions, progress)
	if runner == nil {
		return feed.Batch{}, errors.New("pipeline factory returned nil")
	}
	result, err := runner.Run(ctx)
	if err != nil {
		return feed.Batch{}, err
	}
	batch := result.Batch
	if result.Shortfall == "quota" {
		c.report(progress, "degraded", "quota exhausted; serving a degraded feed", nil)
		batch = feed.Batch{
			ID: c.batchID(now), UserID: userID, Size: c.size(),
			State: feed.BatchDegraded, DegradationReason: feed.DegradeQuota,
			CreatedAt: now, CompletedAt: now,
		}
	}
	if batch.ID == "" {
		return feed.Batch{}, errors.New("pipeline returned no batch")
	}
	if err := c.Store.Save(ctx, batch); err != nil {
		return feed.Batch{}, err
	}
	c.report(progress, "done", "batch ready", nil)
	return batch, nil
}

func (c *liveComposer) loadExclusions(ctx context.Context, userID string, now time.Time) (discovery.ExclusionSet, error) {
	if c.Exclusions == nil {
		return discovery.ExclusionSet{}, nil
	}
	return c.Exclusions.ForUser(ctx, userID, now.AddDate(0, 0, -90))
}

func (c *liveComposer) loadSubscriptions(ctx context.Context, userID string, source discovery.RefreshSource, progress feed.ProgressFunc) ([]identity.Subscription, error) {
	if !source.UsesGraph() || c.Channels == nil {
		return nil, nil
	}
	subscriptions, err := c.Channels.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	c.report(progress, "subscriptions", fmt.Sprintf("loaded %d subscriptions for the graph", len(subscriptions)), nil)
	return subscriptions, nil
}

func (c *liveComposer) report(progress feed.ProgressFunc, stage, message string, attributes map[string]any) {
	if progress != nil {
		progress(stage, message, attributes)
	}
}

func (c *liveComposer) interestsFromChannels(ctx context.Context, userID string) ([]string, error) {
	if c.Channels == nil {
		return nil, feed.ErrNoInterests
	}
	channels, err := c.Channels.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []string
	for _, channel := range channels {
		slug := slugify(channel.Title)
		if slug == "" || seen[slug] {
			continue
		}
		seen[slug] = true
		out = append(out, slug)
		if len(out) == 5 {
			break
		}
	}
	if len(out) == 0 {
		return nil, feed.ErrNoInterests
	}
	return out, nil
}

func (c *liveComposer) now() time.Time {
	if c.Now == nil {
		return time.Now().UTC()
	}
	return c.Now().UTC()
}

func (c *liveComposer) size() int {
	if c.Size < 1 || c.Size > 20 {
		return 20
	}
	return c.Size
}

func (c *liveComposer) batchID(now time.Time) string {
	return fmt.Sprintf("batch-%d", now.UnixNano())
}

func newLiveComposer(cfg config.Config, db *sql.DB) *liveComposer {
	profiles := sqlite.NewProfileRepository(db)
	tasteService := &taste.Service{Profiles: profiles}
	dataDir := cfg.Storage.DataDir
	factory := func(profile taste.ProfileSnapshot, interests []string, token identity.Token, exclusions discovery.ExclusionSet, source discovery.RefreshSource, subscriptions []identity.Subscription, progress feed.ProgressFunc) pipelineRunner {
		youtubeSource := youtube.NewSource("", nil)
		youtubeSource.Region = cfg.YouTube.Region
		quota := &discovery.QuotaService{Store: sqlite.NewQuotaRepository(db)}
		fetcher := ytdlp.New(cfg.Analysis.YTDLP)
		staging := transcripts.NewWithLimit(dataDir, cfg.Analysis.MaxBytes)
		llmClient := llm.New(cfg.LLM.BaseURL, cfg.LLM.Model, cfg.LLM.APIKey, cfg.LLM.JSONMode, cfg.LLM.Timeout)
		analyzer := &analysis.Analyzer{LLM: llmClient, Profile: profile, ModelID: cfg.LLM.Model}
		return &feed.Pipeline{
			UserID: "owner", AccessToken: token.AccessToken, Interests: interests,
			Size: cfg.Feed.Size, MicroTarget: 0.2, RubricVersion: profile.RubricVersion,
			ModelID: cfg.LLM.Model, Staging: cfg.Analysis.Staging, Source: source, Exclusions: exclusions,
			Subscriptions: subscriptions, GraphChannelLimit: cfg.Feed.GraphChannelLimit, GraphRelatedLimit: cfg.Feed.GraphRelatedLimit, ChannelCap: cfg.Feed.ChannelCap,
			Planner: discovery.QueryPlanner{}, YouTube: youtubeSource, Quota: quota,
			Fetcher: fetcher, Transcripts: staging, Analyzer: analyzer,
			Analyses: sqlite.NewAnalysisRepository(db),
			Progress: progress,
		}
	}
	var tokens tokenReader = fileTokenReader{store: tokenfile.New(dataDir)}
	if cfg.YouTube.OAuthClientFile != "" {
		if client, err := identity.LoadClient(cfg.YouTube.OAuthClientFile); err == nil {
			tokens = &identity.Manager{Store: tokenfile.New(dataDir), Client: client}
		}
	}
	source, err := discovery.ParseRefreshSource(cfg.Feed.RefreshSource)
	if err != nil {
		source = discovery.RefreshMixed
	}
	return &liveComposer{
		UserID: "owner", Size: cfg.Feed.Size, Source: source,
		Tokens: tokens, Profiles: tasteService,
		Channels:   sqlite.NewSubscriptionRepository(db),
		Exclusions: sqlite.NewExclusionRepository(db),
		Store:      sqlite.NewBatchRepository(db), Factory: factory,
	}
}
