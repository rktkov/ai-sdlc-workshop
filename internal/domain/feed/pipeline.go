package feed

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/analysis"
	"github.com/rktkov/ytfeed/internal/domain/discovery"
	"github.com/rktkov/ytfeed/internal/domain/identity"
	"github.com/rktkov/ytfeed/internal/pkg/trace"
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

// ProgressFunc reports one composition step.
type ProgressFunc func(stage, message string, attributes map[string]any)

// Pipeline runs discovery, selection and analysis in order.
type Pipeline struct {
	UserID            string
	AccessToken       string
	Interests         []string
	Size              int
	MicroTarget       float64
	RubricVersion     string
	ModelID           string
	BatchID           string
	Staging           bool
	Source            discovery.RefreshSource
	Exclusions        discovery.ExclusionSet
	Subscriptions     []identity.Subscription
	GraphChannelLimit int
	GraphRelatedLimit int

	Planner     Planner
	YouTube     discovery.YouTube
	Quota       Quota
	Fetcher     analysis.Fetcher
	Transcripts analysis.TranscriptStore
	Analyzer    Analyzer
	Analyses    Analyses

	Now      func() time.Time
	NewID    func(videoID, rubricVersion string) string
	Tracer   *trace.Logger
	Progress ProgressFunc
}

// Result is one completed batch pipeline run.
type Result struct {
	Selection discovery.Selection
	Analyses  []analysis.Analysis
	Batch     Batch
	Shortfall string
}

// Run executes the stages in order.
func (p *Pipeline) Run(ctx context.Context) (result Result, err error) {
	span := p.startSpan(ctx, "feed.compose", map[string]any{
		"batch_id": p.BatchID, "size": p.Size, "interests": len(p.Interests),
	})
	defer func() { span.End(err) }()
	if err := p.validate(); err != nil {
		return Result{}, err
	}
	source := p.Source
	if source == "" {
		source = discovery.RefreshMixed
	}
	p.report("source", "refresh source: "+string(source), nil)
	var queries []discovery.Query
	if source.UsesSearch() {
		p.report("plan", fmt.Sprintf("planning %d queries", len(p.Interests)), nil)
		queries, err = p.Planner.Plan(p.Interests)
		if err != nil {
			return Result{}, fmt.Errorf("plan queries: %w", err)
		}
	} else {
		p.report("plan", "keyword search is off for this source", nil)
	}
	videos, shortfall, err := p.discover(ctx, queries, source)
	if err != nil {
		return Result{}, err
	}
	if shortfall == "quota" {
		return Result{Shortfall: shortfall}, nil
	}

	candidates := make([]discovery.Candidate, 0, len(videos))
	for position, video := range videos {
		candidateSource := video.Source
		if candidateSource == "" {
			candidateSource = discovery.SourceSearch
		}
		candidates = append(candidates, discovery.Candidate{
			VideoID:         video.VideoID,
			ChannelID:       video.ChannelID,
			Title:           video.Title,
			SubscriberCount: video.SubscriberCount,
			TopicTags:       video.TopicTags,
			Source:          candidateSource,
			DiscoveryRank:   position,
		})
	}
	p.report("select", fmt.Sprintf("selecting up to %d cards from %d candidates", p.Size, len(candidates)), nil)
	selection, err := discovery.Select(candidates, p.Exclusions, p.Size, p.MicroTarget)
	if err != nil {
		return Result{}, err
	}
	analyses, err := p.analyze(ctx, selection.Selected)
	if err != nil {
		return Result{}, err
	}
	byVideo := make(map[string]analysis.Analysis, len(analyses))
	for _, value := range analyses {
		byVideo[value.VideoID] = value
	}
	batch, err := ComposeBatch(p.batchID(), p.UserID, p.Size, selection.Selected, byVideo)
	if err != nil {
		return Result{}, err
	}
	return Result{Selection: selection, Analyses: analyses, Batch: batch, Shortfall: selection.Shortfall}, nil
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

func (p *Pipeline) discover(ctx context.Context, queries []discovery.Query, source discovery.RefreshSource) ([]discovery.Video, string, error) {
	var videos []discovery.Video
	if source.UsesSearch() {
		found, shortfall, err := p.search(ctx, queries)
		if err != nil {
			return nil, "", err
		}
		if shortfall == "quota" {
			return found, "quota", nil
		}
		videos = append(videos, found...)
	}
	if source.UsesTrending() {
		found, err := p.trending(ctx)
		if err != nil {
			return nil, "", err
		}
		videos = append(videos, found...)
	}
	if source.UsesGraph() {
		found, err := p.graph(ctx, videos)
		if err != nil {
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

func (p *Pipeline) search(ctx context.Context, queries []discovery.Query) ([]discovery.Video, string, error) {
	var videos []discovery.Video
	for index, query := range queries {
		p.report("search", "searching: "+query.Text, map[string]any{"current": index + 1, "total": len(queries)})
		reservation, err := p.Quota.Reserve(ctx, p.UserID, discovery.CallSearch)
		if errors.Is(err, discovery.ErrQuotaExhausted) {
			return videos, "quota", nil
		}
		if err != nil {
			return nil, "", fmt.Errorf("reserve search quota: %w", err)
		}
		page, err := p.YouTube.SearchPage(ctx, p.AccessToken, query, "")
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
		p.report("search_done", fmt.Sprintf("found %d results for %s", len(page.Videos), query.Text), map[string]any{"current": index + 1, "total": len(queries)})
		for i := range page.Videos {
			page.Videos[i].Source = discovery.SourceSearch
		}
		videos = append(videos, page.Videos...)
	}
	return videos, "", nil
}

func (p *Pipeline) trending(ctx context.Context) ([]discovery.Video, error) {
	p.report("trending", "loading trending videos", nil)
	reservation, err := p.Quota.Reserve(ctx, p.UserID, discovery.CallMostPopular)
	if errors.Is(err, discovery.ErrQuotaExhausted) {
		p.report("trending_skip", "quota exhausted before trending", nil)
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reserve trending quota: %w", err)
	}
	videos, err := p.YouTube.MostPopular(ctx, p.AccessToken, "")
	if err != nil {
		_ = p.charge(ctx, reservation.ID)
		if errors.Is(err, identity.ErrUnauthorized) {
			return nil, err
		}
		p.report("trending_skip", "trending unavailable", nil)
		return nil, nil
	}
	if err := p.charge(ctx, reservation.ID); err != nil {
		return nil, err
	}
	for i := range videos {
		videos[i].Source = discovery.SourceTrending
	}
	p.report("trending_done", fmt.Sprintf("found %d trending videos", len(videos)), nil)
	return videos, nil
}

func (p *Pipeline) graph(ctx context.Context, seeds []discovery.Video) ([]discovery.Video, error) {
	related := make([]string, 0, len(seeds))
	for _, video := range seeds {
		related = append(related, video.ChannelID)
	}
	channelIDs := discovery.GraphChannelIDs(p.Subscriptions, related, p.graphChannelLimit(), p.graphRelatedLimit())
	if len(channelIDs) == 0 {
		p.report("graph", "no graph channels to read", nil)
		return nil, nil
	}
	p.report("graph", fmt.Sprintf("reading uploads from %d graph channels", len(channelIDs)), nil)
	channels, shortfall, err := p.channels(ctx, channelIDs)
	if err != nil {
		return nil, err
	}
	if shortfall == "quota" {
		p.report("graph_skip", "quota exhausted before graph metadata", nil)
		return nil, nil
	}
	var videos []discovery.Video
	for _, channelID := range channelIDs {
		playlistID := channels[channelID].UploadsPlaylistID
		if playlistID == "" {
			p.report("graph_skip", "no uploads playlist: "+channelID, nil)
			continue
		}
		p.report("graph_uploads", "reading uploads: "+channelID, nil)
		page, shortfall, err := p.playlistItems(ctx, playlistID)
		if err != nil {
			return nil, err
		}
		if shortfall == "quota" {
			p.report("graph_skip", "quota exhausted during graph reads", nil)
			break
		}
		p.report("graph_done", fmt.Sprintf("found %d uploads for %s", len(page.Videos), channelID), nil)
		videos = append(videos, page.Videos...)
	}
	return videos, nil
}

func (p *Pipeline) playlistItems(ctx context.Context, playlistID string) (discovery.VideoPage, string, error) {
	reservation, err := p.Quota.Reserve(ctx, p.UserID, discovery.CallPlaylistItems)
	if errors.Is(err, discovery.ErrQuotaExhausted) {
		return discovery.VideoPage{}, "quota", nil
	}
	if err != nil {
		return discovery.VideoPage{}, "", fmt.Errorf("reserve playlist items quota: %w", err)
	}
	page, err := p.YouTube.PlaylistItems(ctx, p.AccessToken, playlistID, "")
	if err != nil {
		_ = p.charge(ctx, reservation.ID)
		if errors.Is(err, identity.ErrUnauthorized) {
			return discovery.VideoPage{}, "", err
		}
		p.report("graph_skip", "uploads unavailable: "+playlistID, nil)
		return discovery.VideoPage{}, "", nil
	}
	if err := p.charge(ctx, reservation.ID); err != nil {
		return discovery.VideoPage{}, "", err
	}
	for i := range page.Videos {
		page.Videos[i].Source = discovery.SourceGraph
	}
	return page, "", nil
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
	p.report("videos", fmt.Sprintf("loading metadata for %d videos", len(ids)), nil)
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
	p.report("channels", fmt.Sprintf("loading metadata for %d channels", len(ids)), nil)
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

func (p *Pipeline) analyze(ctx context.Context, selected []discovery.SelectedCandidate) (out []analysis.Analysis, err error) {
	span := p.startSpan(ctx, "feed.analyze", map[string]any{"batch_id": p.BatchID, "count": len(selected)})
	defer func() { span.End(err) }()
	out = make([]analysis.Analysis, 0, len(selected))
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
	p.report("fetch", "fetching transcript: "+candidate.VideoID, nil)
	transcript, err := p.Fetcher.Fetch(ctx, candidate.VideoID)
	if err != nil {
		p.report("no_transcript", "no transcript: "+candidate.VideoID, nil)
		return p.saveFailed(ctx, candidate, analysis.ReasonNoTranscript)
	}
	p.report("fetch_done", "transcript ready: "+candidate.VideoID, nil)

	var staged analysis.Staged
	hasStage := false
	if p.Staging {
		p.report("stage", "staging transcript: "+candidate.VideoID, nil)
		staged, err = p.Transcripts.Put(ctx, candidate.VideoID, transcript)
		if err == nil {
			hasStage = true
		}
	}

	p.report("analyze", "analyzing: "+candidate.VideoID, nil)
	value, err := p.Analyzer.Analyze(ctx, transcript)
	if err != nil {
		value = analysis.Analysis{State: analysis.StateFailed, Reason: analysis.ReasonInvalidModelOutput}
	}
	if value.State == analysis.StateAnalyzed {
		p.report("analyzed", "analyzed: "+value.Topic, map[string]any{"video_id": candidate.VideoID})
	} else {
		p.report("analysis_failed", "analysis failed: "+string(value.Reason), map[string]any{"video_id": candidate.VideoID})
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
	p.report("save", "saving card: "+candidate.VideoID, nil)
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
	p.report("save", "saving card: "+candidate.VideoID, nil)
	if err := p.Analyses.Save(ctx, value); err != nil {
		return analysis.Analysis{}, fmt.Errorf("save failed analysis: %w", err)
	}
	return value, nil
}

func (p *Pipeline) report(stage, message string, attributes map[string]any) {
	if p.Progress != nil {
		p.Progress(stage, message, attributes)
	}
}

func (p *Pipeline) startSpan(ctx context.Context, name string, attributes map[string]any) *trace.Span {
	if p.Tracer == nil {
		return &trace.Span{}
	}
	return p.Tracer.Start(ctx, name, attributes)
}

func (p *Pipeline) now() time.Time {
	if p.Now == nil {
		return time.Now().UTC()
	}
	return p.Now().UTC()
}

func (p *Pipeline) batchID() string {
	if p.BatchID != "" {
		return p.BatchID
	}
	return fmt.Sprintf("batch-%d", p.now().UnixNano())
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
	index := make(map[string]int, len(videos))
	out := make([]discovery.Video, 0, len(videos))
	for _, video := range videos {
		if position, ok := index[video.VideoID]; ok {
			if discovery.SourceRank(video.Source) < discovery.SourceRank(out[position].Source) {
				out[position].Source = video.Source
			}
			continue
		}
		index[video.VideoID] = len(out)
		out = append(out, video)
	}
	return out
}

func (p *Pipeline) graphChannelLimit() int {
	if p.GraphChannelLimit <= 0 {
		return 20
	}
	return p.GraphChannelLimit
}

func (p *Pipeline) graphRelatedLimit() int {
	if p.GraphRelatedLimit <= 0 {
		return 10
	}
	return p.GraphRelatedLimit
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
