package feed

import (
	"context"
	"errors"
	"fmt"
	"sync"
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

// StreamHooks receives progressive batch and card updates.
type StreamHooks struct {
	Progress ProgressFunc
	Batch    func(Batch) error
	Card     func(Batch, Card)
}

// Pipeline runs discovery, selection and analysis in order.
type Pipeline struct {
	UserID              string
	AccessToken         string
	Interests           []string
	Size                int
	MicroTarget         float64
	RubricVersion       string
	ModelID             string
	BatchID             string
	Staging             bool
	Source              discovery.RefreshSource
	Exclusions          discovery.ExclusionSet
	Subscriptions       []identity.Subscription
	GraphChannelLimit   int
	GraphRelatedLimit   int
	ChannelCap          int
	AnalysisConcurrency int
	OnBatch             func(Batch) error
	OnCard              func(Batch, Card)

	reportMu sync.Mutex

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
	videos, state, shortfall, err := p.discover(ctx, queries, source)
	if err != nil {
		return Result{}, err
	}
	if shortfall == "quota" {
		return Result{Shortfall: shortfall}, nil
	}

	_, selection, err := p.expandPool(ctx, videos, state)
	if err != nil {
		return Result{}, err
	}
	if len(selection.Selected) < p.Size {
		p.report("shortfall", fmt.Sprintf("only %d of %d unseen cards are available", len(selection.Selected), p.Size), nil)
	}

	pending, err := ComposeBatch(p.batchID(), p.UserID, p.Size, selection.Selected, nil)
	if err != nil {
		return Result{}, err
	}
	p.degradeShortfall(&pending, selection)
	if p.OnBatch != nil {
		if err := p.OnBatch(cloneBatch(pending)); err != nil {
			return Result{}, fmt.Errorf("save pending batch: %w", err)
		}
	}
	analyses, err := p.analyze(ctx, selection.Selected, func(value analysis.Analysis) error {
		if p.OnCard == nil {
			return nil
		}
		cardID := "card-" + value.VideoID
		if err := ApplyAnalysis(&pending, cardID, value, -1); err != nil {
			return err
		}
		card, index := findCard(&pending, cardID)
		if index < 0 {
			return fmt.Errorf("card %q disappeared", cardID)
		}
		p.OnCard(cloneBatch(pending), card)
		return nil
	})
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
	p.degradeShortfall(&batch, selection)
	return Result{Selection: selection, Analyses: analyses, Batch: batch, Shortfall: selection.Shortfall}, nil
}

func cloneBatch(batch Batch) Batch {
	batch.Cards = append([]Card(nil), batch.Cards...)
	return batch
}

func (p *Pipeline) degradeShortfall(batch *Batch, selection discovery.Selection) {
	if len(selection.Selected) >= p.Size {
		return
	}
	batch.State = BatchDegraded
	batch.DegradationReason = DegradeShortfall
	batch.CompletedAt = p.now()
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

const maxExpansionRounds = 10

type searchCursor struct {
	Query         discovery.Query
	NextPageToken string
	Unseen        int
}

type sourceState struct {
	Search       []searchCursor
	GraphLoaded  map[string]bool
	GraphLimit   int
	RelatedLimit int
	Related      []string
	UploadOrder  []string
	UploadTokens map[string]string
}

type expansionResult struct {
	Videos   []discovery.Video
	Expanded bool
	Quota    bool
}

func (p *Pipeline) discover(ctx context.Context, queries []discovery.Query, source discovery.RefreshSource) ([]discovery.Video, *sourceState, string, error) {
	state := &sourceState{
		GraphLoaded:  map[string]bool{},
		GraphLimit:   p.graphChannelLimit(),
		RelatedLimit: p.graphRelatedLimit(),
		UploadTokens: map[string]string{},
	}
	var videos []discovery.Video
	if source.UsesSearch() {
		found, cursors, shortfall, err := p.search(ctx, queries)
		if err != nil {
			return nil, nil, "", err
		}
		if shortfall == "quota" {
			return found, state, "quota", nil
		}
		state.Search = cursors
		videos = append(videos, found...)
	}
	if source.UsesTrending() {
		found, err := p.trending(ctx)
		if err != nil {
			return nil, nil, "", err
		}
		videos = append(videos, found...)
	}
	state.Related = appendRelated(state.Related, videos)
	if source.UsesGraph() {
		found, err := p.graph(ctx, state)
		if err != nil {
			return nil, nil, "", err
		}
		videos = append(videos, found...)
	}
	if len(videos) == 0 {
		return nil, state, "", nil
	}
	videos = dedupeVideos(videos)
	enriched, shortfall, err := p.enrich(ctx, videos)
	if err != nil {
		return nil, nil, "", err
	}
	return enriched, state, shortfall, nil
}

func (p *Pipeline) search(ctx context.Context, queries []discovery.Query) ([]discovery.Video, []searchCursor, string, error) {
	var videos []discovery.Video
	cursors := make([]searchCursor, 0, len(queries))
	for index, query := range queries {
		p.report("search", "searching: "+query.Text, map[string]any{"current": index + 1, "total": len(queries)})
		reservation, err := p.Quota.Reserve(ctx, p.UserID, discovery.CallSearch)
		if errors.Is(err, discovery.ErrQuotaExhausted) {
			return videos, cursors, "quota", nil
		}
		if err != nil {
			return nil, nil, "", fmt.Errorf("reserve search quota: %w", err)
		}
		page, err := p.YouTube.SearchPage(ctx, p.AccessToken, query, "")
		if err != nil {
			_ = p.charge(ctx, reservation.ID)
			if errors.Is(err, identity.ErrUnauthorized) {
				return nil, nil, "", err
			}
			return videos, cursors, "quota", nil
		}
		if err := p.charge(ctx, reservation.ID); err != nil {
			return nil, nil, "", err
		}
		p.report("search_done", fmt.Sprintf("found %d results for %s", len(page.Videos), query.Text), map[string]any{"current": index + 1, "total": len(queries)})
		for i := range page.Videos {
			page.Videos[i].Source = discovery.SourceSearch
		}
		videos = append(videos, page.Videos...)
		if page.NextPageToken != "" {
			cursors = append(cursors, searchCursor{
				Query: query, NextPageToken: page.NextPageToken, Unseen: p.unseenCount(page.Videos),
			})
		}
	}
	return videos, cursors, "", nil
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

func (p *Pipeline) graph(ctx context.Context, state *sourceState) ([]discovery.Video, error) {
	channelIDs := discovery.GraphChannelIDs(p.Subscriptions, state.Related, state.GraphLimit, state.RelatedLimit)
	channelIDs = p.newGraphChannels(channelIDs, state)
	if len(channelIDs) == 0 {
		p.report("graph", "no graph channels to read", nil)
		return nil, nil
	}
	p.report("graph", fmt.Sprintf("reading uploads from %d graph channels", len(channelIDs)), nil)
	return p.readUploads(ctx, channelIDs, state)
}

func (p *Pipeline) expand(ctx context.Context, state *sourceState) (expansionResult, error) {
	if result, err := p.expandSearch(ctx, state); err != nil || result.Expanded {
		return result, err
	}
	if result, err := p.expandGraphChannels(ctx, state); err != nil || result.Expanded {
		return result, err
	}
	return p.expandUploads(ctx, state)
}

func (p *Pipeline) expandSearch(ctx context.Context, state *sourceState) (expansionResult, error) {
	best := -1
	for i := range state.Search {
		if state.Search[i].NextPageToken == "" {
			continue
		}
		if best == -1 || state.Search[i].Unseen > state.Search[best].Unseen {
			best = i
		}
	}
	if best == -1 {
		return expansionResult{}, nil
	}
	cursor := &state.Search[best]
	p.report("expand_search", "searching more: "+cursor.Query.Text, nil)
	reservation, err := p.Quota.Reserve(ctx, p.UserID, discovery.CallSearch)
	if errors.Is(err, discovery.ErrQuotaExhausted) {
		return expansionResult{Quota: true}, nil
	}
	if err != nil {
		return expansionResult{}, fmt.Errorf("reserve search quota: %w", err)
	}
	page, err := p.YouTube.SearchPage(ctx, p.AccessToken, cursor.Query, cursor.NextPageToken)
	if err != nil {
		_ = p.charge(ctx, reservation.ID)
		if errors.Is(err, identity.ErrUnauthorized) {
			return expansionResult{}, err
		}
		cursor.NextPageToken = ""
		return expansionResult{}, nil
	}
	if err := p.charge(ctx, reservation.ID); err != nil {
		return expansionResult{}, err
	}
	for i := range page.Videos {
		page.Videos[i].Source = discovery.SourceSearch
	}
	cursor.NextPageToken = page.NextPageToken
	cursor.Unseen = p.unseenCount(page.Videos)
	state.Related = appendRelated(state.Related, page.Videos)
	return expansionResult{Videos: page.Videos, Expanded: true}, nil
}

func (p *Pipeline) expandGraphChannels(ctx context.Context, state *sourceState) (expansionResult, error) {
	state.GraphLimit *= 2
	state.RelatedLimit *= 2
	channelIDs := discovery.GraphChannelIDs(p.Subscriptions, state.Related, state.GraphLimit, state.RelatedLimit)
	channelIDs = p.newGraphChannels(channelIDs, state)
	if len(channelIDs) == 0 {
		return expansionResult{}, nil
	}
	p.report("expand_graph", fmt.Sprintf("reading uploads from %d more graph channels", len(channelIDs)), nil)
	videos, err := p.readUploads(ctx, channelIDs, state)
	if err != nil {
		return expansionResult{}, err
	}
	return expansionResult{Videos: videos, Expanded: true}, nil
}

func (p *Pipeline) expandUploads(ctx context.Context, state *sourceState) (expansionResult, error) {
	for _, playlistID := range state.UploadOrder {
		pageToken := state.UploadTokens[playlistID]
		if pageToken == "" {
			continue
		}
		p.report("expand_uploads", "reading more uploads: "+playlistID, nil)
		page, shortfall, err := p.playlistItems(ctx, playlistID, pageToken, state)
		if err != nil {
			return expansionResult{}, err
		}
		if shortfall == "quota" {
			return expansionResult{Quota: true}, nil
		}
		return expansionResult{Videos: page.Videos, Expanded: true}, nil
	}
	return expansionResult{}, nil
}

func (p *Pipeline) newGraphChannels(channelIDs []string, state *sourceState) []string {
	out := make([]string, 0, len(channelIDs))
	for _, channelID := range channelIDs {
		if channelID == "" || state.GraphLoaded[channelID] {
			continue
		}
		state.GraphLoaded[channelID] = true
		out = append(out, channelID)
	}
	return out
}

func (p *Pipeline) readUploads(ctx context.Context, channelIDs []string, state *sourceState) ([]discovery.Video, error) {
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
		page, shortfall, err := p.playlistItems(ctx, playlistID, "", state)
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

func (p *Pipeline) playlistItems(ctx context.Context, playlistID, pageToken string, state *sourceState) (discovery.VideoPage, string, error) {
	reservation, err := p.Quota.Reserve(ctx, p.UserID, discovery.CallPlaylistItems)
	if errors.Is(err, discovery.ErrQuotaExhausted) {
		return discovery.VideoPage{}, "quota", nil
	}
	if err != nil {
		return discovery.VideoPage{}, "", fmt.Errorf("reserve playlist items quota: %w", err)
	}
	page, err := p.YouTube.PlaylistItems(ctx, p.AccessToken, playlistID, pageToken)
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
	if state != nil {
		if _, ok := state.UploadTokens[playlistID]; !ok {
			state.UploadOrder = append(state.UploadOrder, playlistID)
		}
		state.UploadTokens[playlistID] = page.NextPageToken
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

func (p *Pipeline) analyze(ctx context.Context, selected []discovery.SelectedCandidate, onResult func(analysis.Analysis) error) (out []analysis.Analysis, err error) {
	span := p.startSpan(ctx, "feed.analyze", map[string]any{"batch_id": p.BatchID, "count": len(selected)})
	defer func() { span.End(err) }()
	type result struct {
		index int
		value analysis.Analysis
		err   error
	}
	concurrency := p.analysisConcurrency()
	sem := make(chan struct{}, concurrency)
	results := make(chan result, len(selected))
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var wg sync.WaitGroup
	for index, candidate := range selected {
		wg.Add(1)
		go func(index int, candidate discovery.SelectedCandidate) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-runCtx.Done():
				results <- result{index: index, err: runCtx.Err()}
				return
			}
			value, err := p.analyzeCandidate(runCtx, candidate)
			results <- result{index: index, value: value, err: err}
		}(index, candidate)
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	out = make([]analysis.Analysis, 0, len(selected))
	values := make([]analysis.Analysis, len(selected))
	var firstErr error
	for item := range results {
		if item.err != nil {
			if firstErr == nil {
				firstErr = item.err
				cancel()
			}
			continue
		}
		values[item.index] = item.value
		out = append(out, item.value)
		if onResult != nil {
			if err := onResult(item.value); err != nil && firstErr == nil {
				firstErr = err
				cancel()
			}
		}
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return values, nil
}

func (p *Pipeline) analyzeCandidate(ctx context.Context, candidate discovery.SelectedCandidate) (analysis.Analysis, error) {
	existing, err := p.Analyses.FindByKey(ctx, p.UserID, candidate.VideoID, p.RubricVersion)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, analysis.ErrAnalysisNotFound) {
		return analysis.Analysis{}, fmt.Errorf("find analysis: %w", err)
	}
	return p.analyzeOne(ctx, candidate)
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
	if p.Progress == nil {
		return
	}
	p.reportMu.Lock()
	defer p.reportMu.Unlock()
	p.Progress(stage, message, attributes)
}

func (p *Pipeline) startSpan(ctx context.Context, name string, attributes map[string]any) *trace.Span {
	if p.Tracer == nil {
		return &trace.Span{}
	}
	return p.Tracer.Start(ctx, name, attributes)
}

func (p *Pipeline) analysisConcurrency() int {
	if p.AnalysisConcurrency <= 0 {
		return 2
	}
	if p.AnalysisConcurrency > 8 {
		return 8
	}
	return p.AnalysisConcurrency
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

func (p *Pipeline) expandPool(ctx context.Context, videos []discovery.Video, state *sourceState) ([]discovery.Video, discovery.Selection, error) {
	selection, err := p.selectVideos(videos)
	if err != nil {
		return nil, discovery.Selection{}, err
	}
	for round := 0; len(selection.Selected) < p.Size && round < maxExpansionRounds; round++ {
		expansion, err := p.expand(ctx, state)
		if err != nil {
			return nil, discovery.Selection{}, err
		}
		if expansion.Quota || !expansion.Expanded {
			break
		}
		added := newVideos(videos, expansion.Videos)
		if len(added) == 0 {
			continue
		}
		enriched, shortfall, err := p.enrich(ctx, added)
		if err != nil {
			return nil, discovery.Selection{}, err
		}
		if shortfall == "quota" {
			break
		}
		videos = dedupeVideos(append(videos, enriched...))
		selection, err = p.selectVideos(videos)
		if err != nil {
			return nil, discovery.Selection{}, err
		}
	}
	return videos, selection, nil
}

func (p *Pipeline) selectVideos(videos []discovery.Video) (discovery.Selection, error) {
	videos = orderVideos(videos, p.refreshSource())
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
	return discovery.Select(candidates, p.Exclusions, p.Size, p.MicroTarget, p.channelCap())
}

func (p *Pipeline) refreshSource() discovery.RefreshSource {
	if p.Source == "" {
		return discovery.RefreshMixed
	}
	return p.Source
}

func orderVideos(videos []discovery.Video, source discovery.RefreshSource) []discovery.Video {
	var order []discovery.CandidateSource
	switch source {
	case discovery.RefreshKeywords:
		order = []discovery.CandidateSource{discovery.SourceSearch}
	case discovery.RefreshTrending:
		order = []discovery.CandidateSource{discovery.SourceTrending, discovery.SourceSearch}
	case discovery.RefreshGraph:
		order = []discovery.CandidateSource{discovery.SourceGraph, discovery.SourceSearch}
	case discovery.RefreshFresh:
		order = []discovery.CandidateSource{discovery.SourceTrending, discovery.SourceGraph}
	default:
		order = []discovery.CandidateSource{discovery.SourceGraph, discovery.SourceTrending, discovery.SourceSearch}
	}
	groups := map[discovery.CandidateSource][]discovery.Video{}
	for _, video := range videos {
		groups[video.Source] = append(groups[video.Source], video)
	}
	indexes := map[discovery.CandidateSource]int{}
	out := make([]discovery.Video, 0, len(videos))
	for {
		added := false
		for _, candidateSource := range order {
			position := indexes[candidateSource]
			if position >= len(groups[candidateSource]) {
				continue
			}
			out = append(out, groups[candidateSource][position])
			indexes[candidateSource] = position + 1
			added = true
		}
		if !added {
			break
		}
	}
	listed := map[discovery.CandidateSource]bool{}
	for _, candidateSource := range order {
		listed[candidateSource] = true
	}
	for _, video := range videos {
		if !listed[video.Source] {
			out = append(out, video)
		}
	}
	return out
}

func (p *Pipeline) unseenCount(videos []discovery.Video) int {
	seen := map[string]bool{}
	for _, video := range videos {
		if video.VideoID == "" || seen[video.VideoID] || p.excluded(video.VideoID) {
			continue
		}
		seen[video.VideoID] = true
	}
	return len(seen)
}

func (p *Pipeline) excluded(videoID string) bool {
	return p.Exclusions.Watched[videoID] || p.Exclusions.Saved[videoID] || p.Exclusions.Shown7d[videoID]
}

func newVideos(known []discovery.Video, candidates []discovery.Video) []discovery.Video {
	seen := make(map[string]bool, len(known))
	for _, video := range known {
		seen[video.VideoID] = true
	}
	out := make([]discovery.Video, 0, len(candidates))
	for _, video := range candidates {
		if video.VideoID == "" || seen[video.VideoID] {
			continue
		}
		seen[video.VideoID] = true
		out = append(out, video)
	}
	return out
}

func appendRelated(related []string, videos []discovery.Video) []string {
	seen := make(map[string]bool, len(related))
	for _, channelID := range related {
		seen[channelID] = true
	}
	for _, video := range videos {
		channelID := video.ChannelID
		if channelID == "" || seen[channelID] {
			continue
		}
		seen[channelID] = true
		related = append(related, channelID)
	}
	return related
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

func (p *Pipeline) channelCap() int {
	if p.ChannelCap <= 0 {
		return 2
	}
	return p.ChannelCap
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
