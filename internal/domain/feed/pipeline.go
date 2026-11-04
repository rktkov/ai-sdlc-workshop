package feed

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/analysis"
	"github.com/rktkov/ytfeed/internal/domain/discovery"
	"github.com/rktkov/ytfeed/internal/domain/identity"
)

// Planner turns interests into search queries.
type Planner interface {
	Plan(interests []string) ([]discovery.Query, error)
}

// Quota reserves and charges YouTube calls.
type Quota interface {
	Reserve(ctx context.Context, userID string, call discovery.QuotaCall) (discovery.Reservation, error)
	Charge(ctx context.Context, reservationID string) (discovery.Ledger, bool, error)
	Release(ctx context.Context, reservationID string) error
}

// Analyses stores terminal analyses.
type Analyses interface {
	FindByKey(ctx context.Context, userID, videoID, rubricVersion string) (analysis.Analysis, error)
	Save(ctx context.Context, value analysis.Analysis) error
}

// Analyzer turns transcript text into a terminal analysis.
type Analyzer interface {
	Analyze(ctx context.Context, transcript string) (analysis.Analysis, error)
}

// Pipeline runs discovery, selection and analysis in order.
type Pipeline struct {
	UserID        string
	AccessToken   string
	Interests     []string
	Size          int
	MicroTarget   float64
	RubricVersion string
	ModelID       string
	Staging       bool
	Exclusions    discovery.ExclusionSet

	Planner     Planner
	YouTube     discovery.YouTube
	Quota       Quota
	Fetcher     analysis.Fetcher
	Transcripts analysis.TranscriptStore
	Analyzer    Analyzer
	Analyses    Analyses

	Now   func() time.Time
	NewID func(videoID, rubricVersion string) string
}

// Result is one completed batch pipeline run.
type Result struct {
	Selection discovery.Selection
	Analyses  []analysis.Analysis
	Shortfall string
}

// Run executes the stages in order.
func (p *Pipeline) Run(ctx context.Context) (Result, error) {
	if err := p.validate(); err != nil {
		return Result{}, err
	}
	queries, err := p.Planner.Plan(p.Interests)
	if err != nil {
		return Result{}, fmt.Errorf("plan queries: %w", err)
	}
	videos, shortfall, err := p.discover(ctx, queries)
	if err != nil {
		return Result{}, err
	}
	if shortfall == "quota" {
		return Result{Shortfall: shortfall}, nil
	}

	candidates := make([]discovery.Candidate, 0, len(videos))
	for position, video := range videos {
		candidates = append(candidates, discovery.Candidate{
			VideoID:         video.VideoID,
			ChannelID:       video.ChannelID,
			Title:           video.Title,
			SubscriberCount: video.SubscriberCount,
			TopicTags:       video.TopicTags,
			Source:          discovery.SourceSearch,
			DiscoveryRank:   position,
		})
	}
	selection, err := discovery.Select(candidates, p.Exclusions, p.Size, p.MicroTarget)
	if err != nil {
		return Result{}, err
	}
	analyses, err := p.analyze(ctx, selection.Selected)
	if err != nil {
		return Result{}, err
	}
	return Result{Selection: selection, Analyses: analyses, Shortfall: selection.Shortfall}, nil
}

func (p *Pipeline) validate() error {
	if p.Planner == nil || p.YouTube == nil || p.Quota == nil || p.Fetcher == nil || p.Analyzer == nil || p.Analyses == nil {
		return errors.New("pipeline is not configured")
	}
	if p.Staging && p.Transcripts == nil {
		return errors.New("pipeline has no transcript store")
	}
	if p.UserID == "" || p.AccessToken == "" {
		return errors.New("pipeline has no user or access token")
	}
	if len(p.Interests) < 1 || len(p.Interests) > 10 {
		return errors.New("pipeline needs 1 to 10 interests")
	}
	if p.Size < 1 || p.Size > 20 {
		return errors.New("pipeline size must be between 1 and 20")
	}
	if p.RubricVersion == "" {
		return errors.New("pipeline has no rubric version")
	}
	return nil
}

func (p *Pipeline) discover(ctx context.Context, queries []discovery.Query) ([]discovery.Video, string, error) {
	var videos []discovery.Video
	for _, query := range queries {
		reservation, err := p.Quota.Reserve(ctx, p.UserID, discovery.CallSearch)
		if errors.Is(err, discovery.ErrQuotaExhausted) {
			return videos, "quota", nil
		}
		if err != nil {
			return nil, "", fmt.Errorf("reserve search quota: %w", err)
		}
		found, err := p.YouTube.Search(ctx, p.AccessToken, query)
		if err != nil {
			_ = p.charge(ctx, reservation.ID)
			if errors.Is(err, identity.ErrUnauthorized) {
				return nil, "", err
			}
			return videos, "quota", nil
		}
		if err := p.charge(ctx, reservation.ID); err != nil {
			return nil, "", err
		}
		videos = append(videos, found...)
	}
	if len(videos) == 0 {
		return nil, "", nil
	}
	videos = dedupeVideos(videos)
	enriched, shortfall, err := p.enrich(ctx, videos)
	if err != nil {
		return nil, "", err
	}
	return enriched, shortfall, nil
}

func (p *Pipeline) enrich(ctx context.Context, videos []discovery.Video) ([]discovery.Video, string, error) {
	ids := make([]string, 0, len(videos))
	for _, video := range videos {
		ids = append(ids, video.VideoID)
	}
	details, shortfall, err := p.videos(ctx, ids)
	if err != nil {
		return nil, "", err
	}
	if shortfall == "quota" {
		return videos, shortfall, nil
	}
	channelIDs := make([]string, 0, len(videos))
	for _, video := range videos {
		channelIDs = append(channelIDs, video.ChannelID)
	}
	channels, shortfall, err := p.channels(ctx, channelIDs)
	if err != nil {
		return nil, "", err
	}
	if shortfall == "quota" {
		return videos, shortfall, nil
	}

	detailByID := make(map[string]discovery.Video, len(details))
	for _, detail := range details {
		detailByID[detail.VideoID] = detail
	}
	for i := range videos {
		if detail, ok := detailByID[videos[i].VideoID]; ok {
			videos[i] = mergeVideo(videos[i], detail)
		}
		if channel, ok := channels[videos[i].ChannelID]; ok {
			videos[i].SubscriberCount = channel.SubscriberCount
			if videos[i].Title == "" {
				videos[i].Title = channel.Title
			}
		}
	}
	return videos, "", nil
}

func (p *Pipeline) videos(ctx context.Context, ids []string) ([]discovery.Video, string, error) {
	reservation, err := p.Quota.Reserve(ctx, p.UserID, discovery.CallVideos)
	if errors.Is(err, discovery.ErrQuotaExhausted) {
		return nil, "quota", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("reserve videos quota: %w", err)
	}
	details, err := p.YouTube.Videos(ctx, p.AccessToken, ids)
	if err != nil {
		_ = p.charge(ctx, reservation.ID)
		return nil, "", nil
	}
	if err := p.charge(ctx, reservation.ID); err != nil {
		return nil, "", err
	}
	return details, "", nil
}

func (p *Pipeline) channels(ctx context.Context, ids []string) (map[string]identity.Subscription, string, error) {
	reservation, err := p.Quota.Reserve(ctx, p.UserID, discovery.CallChannels)
	if errors.Is(err, discovery.ErrQuotaExhausted) {
		return nil, "quota", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("reserve channels quota: %w", err)
	}
	channels, err := p.YouTube.Channels(ctx, p.AccessToken, ids)
	if err != nil {
		_ = p.charge(ctx, reservation.ID)
		return nil, "", nil
	}
	if err := p.charge(ctx, reservation.ID); err != nil {
		return nil, "", err
	}
	return channels, "", nil
}

func (p *Pipeline) charge(ctx context.Context, reservationID string) error {
	if _, _, err := p.Quota.Charge(ctx, reservationID); err != nil {
		return fmt.Errorf("charge quota: %w", err)
	}
	return nil
}

func (p *Pipeline) analyze(ctx context.Context, selected []discovery.SelectedCandidate) ([]analysis.Analysis, error) {
	out := make([]analysis.Analysis, 0, len(selected))
	for _, candidate := range selected {
		existing, err := p.Analyses.FindByKey(ctx, p.UserID, candidate.VideoID, p.RubricVersion)
		if err == nil {
			out = append(out, existing)
			continue
		}
		if !errors.Is(err, analysis.ErrAnalysisNotFound) {
			return nil, fmt.Errorf("find analysis: %w", err)
		}
		value, err := p.analyzeOne(ctx, candidate)
		if err != nil {
			return nil, err
		}
		out = append(out, value)
	}
	return out, nil
}

func (p *Pipeline) analyzeOne(ctx context.Context, candidate discovery.SelectedCandidate) (analysis.Analysis, error) {
	transcript, err := p.Fetcher.Fetch(ctx, candidate.VideoID)
	if err != nil {
		return p.saveFailed(ctx, candidate, analysis.ReasonNoTranscript)
	}

	var staged analysis.Staged
	hasStage := false
	if p.Staging {
		staged, err = p.Transcripts.Put(ctx, candidate.VideoID, transcript)
		if err == nil {
			hasStage = true
		}
	}

	value, err := p.Analyzer.Analyze(ctx, transcript)
	if err != nil {
		value = analysis.Analysis{State: analysis.StateFailed, Reason: analysis.ReasonInvalidModelOutput}
	}
	value.UserID = p.UserID
	value.VideoID = candidate.VideoID
	value.RubricVersion = p.RubricVersion
	if value.ModelID == "" {
		value.ModelID = p.modelID()
	}
	if value.ID == "" {
		value.ID = p.newID(candidate.VideoID, p.RubricVersion)
	}
	if hasStage {
		value.TranscriptHash = staged.Hash
		value.TranscriptBytes = staged.Bytes
	}
	if value.CompletedAt.IsZero() {
		value.CompletedAt = p.now()
	}
	if value.CreatedAt.IsZero() {
		value.CreatedAt = value.CompletedAt
	}
	if err := p.Analyses.Save(ctx, value); err != nil {
		if errors.Is(err, analysis.ErrAnalysisExists) {
			existing, findErr := p.Analyses.FindByKey(ctx, p.UserID, candidate.VideoID, p.RubricVersion)
			if findErr == nil {
				value = existing
			} else {
				return analysis.Analysis{}, findErr
			}
		} else {
			return analysis.Analysis{}, fmt.Errorf("save analysis: %w", err)
		}
	}
	if hasStage {
		_ = p.Transcripts.Delete(ctx, candidate.VideoID)
	}
	return value, nil
}

func (p *Pipeline) saveFailed(ctx context.Context, candidate discovery.SelectedCandidate, reason analysis.Reason) (analysis.Analysis, error) {
	value := analysis.Analysis{
		ID:             p.newID(candidate.VideoID, p.RubricVersion),
		UserID:         p.UserID,
		VideoID:        candidate.VideoID,
		ProfileVersion: 1,
		RubricVersion:  p.RubricVersion,
		ModelID:        p.modelID(),
		State:          analysis.StateFailed,
		Reason:         reason,
		CreatedAt:      p.now(),
		CompletedAt:    p.now(),
	}
	if err := p.Analyses.Save(ctx, value); err != nil {
		return analysis.Analysis{}, fmt.Errorf("save failed analysis: %w", err)
	}
	return value, nil
}

func (p *Pipeline) now() time.Time {
	if p.Now == nil {
		return time.Now().UTC()
	}
	return p.Now().UTC()
}

func (p *Pipeline) modelID() string {
	if p.ModelID == "" {
		return "unknown"
	}
	return p.ModelID
}

func (p *Pipeline) newID(videoID, rubricVersion string) string {
	if p.NewID != nil {
		return p.NewID(videoID, rubricVersion)
	}
	return fmt.Sprintf("analysis-%s-%s", videoID, rubricVersion)
}

func dedupeVideos(videos []discovery.Video) []discovery.Video {
	seen := make(map[string]bool, len(videos))
	out := make([]discovery.Video, 0, len(videos))
	for _, video := range videos {
		if seen[video.VideoID] {
			continue
		}
		seen[video.VideoID] = true
		out = append(out, video)
	}
	return out
}

func mergeVideo(base, detail discovery.Video) discovery.Video {
	if detail.Title != "" {
		base.Title = detail.Title
	}
	if detail.ChannelID != "" {
		base.ChannelID = detail.ChannelID
	}
	if detail.Description != "" {
		base.Description = detail.Description
	}
	if !detail.PublishedAt.IsZero() {
		base.PublishedAt = detail.PublishedAt
	}
	if detail.ViewCount != 0 {
		base.ViewCount = detail.ViewCount
	}
	if detail.Duration != "" {
		base.Duration = detail.Duration
	}
	if len(detail.TopicTags) > 0 {
		base.TopicTags = append([]string(nil), detail.TopicTags...)
	}
	return base
}
