package feed

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/analysis"
	"github.com/rktkov/ytfeed/internal/domain/discovery"
	"github.com/rktkov/ytfeed/internal/domain/identity"
	"github.com/rktkov/ytfeed/internal/pkg/trace"
)

type fakePlanner struct {
	queries []discovery.Query
	err     error
	calls   *[]string
}

func (f *fakePlanner) Plan([]string) ([]discovery.Query, error) {
	f.record("plan")
	return f.queries, f.err
}

func (f *fakePlanner) record(value string) {
	if f.calls != nil {
		*f.calls = append(*f.calls, value)
	}
}

type fakeYouTube struct {
	videos      []discovery.Video
	trending    []discovery.Video
	trendingErr error
	metas       map[string]identity.Subscription
	playlists   map[string]discovery.VideoPage
	playlistErr map[string]error
	playlistIDs []string
	err         error
	videosErr   error
	chErr       error
	calls       *[]string
	searchN     int
}

func (f *fakeYouTube) Search(context.Context, string, discovery.Query) ([]discovery.Video, error) {
	f.record("search")
	f.searchN++
	return f.videos, f.err
}

func (f *fakeYouTube) SearchPage(context.Context, string, discovery.Query, string) (discovery.VideoPage, error) {
	f.record("search")
	f.searchN++
	return discovery.VideoPage{Videos: f.videos}, f.err
}

func (f *fakeYouTube) PlaylistItems(_ context.Context, _ string, playlistID, _ string) (discovery.VideoPage, error) {
	f.record("playlist_items")
	f.playlistIDs = append(f.playlistIDs, playlistID)
	if err := f.playlistErr[playlistID]; err != nil {
		return discovery.VideoPage{}, err
	}
	return f.playlists[playlistID], nil
}

func (f *fakeYouTube) Channels(context.Context, string, []string) (map[string]identity.Subscription, error) {
	f.record("channels")
	return f.metas, f.chErr
}

func (f *fakeYouTube) Videos(context.Context, string, []string) ([]discovery.Video, error) {
	f.record("videos")
	if f.videosErr != nil {
		return nil, f.videosErr
	}
	return f.videos, nil
}

func (f *fakeYouTube) SubscriptionPage(context.Context, string, string) (identity.SubscriptionPage, error) {
	return identity.SubscriptionPage{}, nil
}

func (f *fakeYouTube) MostPopular(context.Context, string, string) ([]discovery.Video, error) {
	f.record("most_popular")
	return f.trending, f.trendingErr
}

func (f *fakeYouTube) record(value string) {
	if f.calls != nil {
		*f.calls = append(*f.calls, value)
	}
}

type fakeQuota struct {
	refuseAt   int
	reserved   int
	reserveErr error
	chargeErr  error
	calls      *[]string
}

func (f *fakeQuota) Reserve(context.Context, string, discovery.QuotaCall) (discovery.Reservation, error) {
	f.record("reserve")
	if f.reserveErr != nil {
		return discovery.Reservation{}, f.reserveErr
	}
	if f.refuseAt > 0 && f.reserved >= f.refuseAt {
		return discovery.Reservation{}, discovery.ErrQuotaExhausted
	}
	f.reserved++
	return discovery.Reservation{ID: "res-" + string(rune('0'+f.reserved))}, nil
}

func (f *fakeQuota) Charge(context.Context, string) (discovery.Ledger, bool, error) {
	f.record("charge")
	return discovery.Ledger{}, false, f.chargeErr
}

func (f *fakeQuota) Release(context.Context, string) error {
	f.record("release")
	return nil
}

func (f *fakeQuota) record(value string) {
	if f.calls != nil {
		*f.calls = append(*f.calls, value)
	}
}

type fakeFetcher struct {
	err   error
	calls *[]string
}

func (f *fakeFetcher) Fetch(_ context.Context, videoID string) (string, error) {
	f.record("fetch")
	if f.err != nil {
		return "", f.err
	}
	return "transcript " + videoID, nil
}

func (f *fakeFetcher) record(value string) {
	if f.calls != nil {
		*f.calls = append(*f.calls, value)
	}
}

type fakeTranscripts struct {
	staged map[string]analysis.Staged
	putErr error
	getErr error
	calls  *[]string
}

func (f *fakeTranscripts) Put(_ context.Context, videoID, text string) (analysis.Staged, error) {
	f.record("stage")
	if f.putErr != nil {
		return analysis.Staged{}, f.putErr
	}
	if f.staged == nil {
		f.staged = map[string]analysis.Staged{}
	}
	staged := analysis.Staged{Hash: "hash-" + videoID, Bytes: int64(len(text)), Text: text}
	f.staged[videoID] = staged
	return staged, nil
}

func (f *fakeTranscripts) Get(_ context.Context, videoID, _ string) (analysis.Staged, error) {
	f.record("get")
	if f.getErr != nil {
		return analysis.Staged{}, f.getErr
	}
	if staged, ok := f.staged[videoID]; ok {
		return staged, nil
	}
	return analysis.Staged{}, analysis.ErrStageMiss
}

func (f *fakeTranscripts) Delete(context.Context, string) error {
	f.record("delete")
	return nil
}

func (f *fakeTranscripts) Sweep(context.Context, time.Time) (int, error) {
	return 0, nil
}

func (f *fakeTranscripts) record(value string) {
	if f.calls != nil {
		*f.calls = append(*f.calls, value)
	}
}

type fakeAnalyzer struct {
	err   error
	calls *[]string
}

func (f *fakeAnalyzer) Analyze(_ context.Context, transcript string) (analysis.Analysis, error) {
	f.record("analyze")
	if f.err != nil {
		return analysis.Analysis{}, f.err
	}
	return analysis.Analysis{
		State:         analysis.StateAnalyzed,
		Topic:         "ai-agents",
		TLDR:          "tldr",
		AxisScores:    []analysis.AxisScore{{Axis: "depth", Score: 0.8, Justification: "good"}},
		Similarity:    "new territory",
		WhyThisCard:   "why",
		RubricVersion: "v1",
		ModelID:       "model",
		CompletedAt:   time.Now().UTC(),
	}, nil
}

func (f *fakeAnalyzer) record(value string) {
	if f.calls != nil {
		*f.calls = append(*f.calls, value)
	}
}

type fakeAnalyses struct {
	items      map[string]analysis.Analysis
	calls      *[]string
	findErr    error
	saveErr    error
	saveExists bool
}

func (f *fakeAnalyses) FindByKey(_ context.Context, userID, videoID, rubricVersion string) (analysis.Analysis, error) {
	f.record("find")
	if f.findErr != nil {
		return analysis.Analysis{}, f.findErr
	}
	if value, ok := f.items[userID+"|"+videoID+"|"+rubricVersion]; ok {
		return value, nil
	}
	return analysis.Analysis{}, analysis.ErrAnalysisNotFound
}

func (f *fakeAnalyses) Save(_ context.Context, value analysis.Analysis) error {
	f.record("save")
	if f.saveErr != nil {
		return f.saveErr
	}
	if f.saveExists {
		return analysis.ErrAnalysisExists
	}
	if f.items == nil {
		f.items = map[string]analysis.Analysis{}
	}
	f.items[value.UserID+"|"+value.VideoID+"|"+value.RubricVersion] = value
	return nil
}

func (f *fakeAnalyses) record(value string) {
	if f.calls != nil {
		*f.calls = append(*f.calls, value)
	}
}

func testPipeline(calls *[]string) *Pipeline {
	return &Pipeline{
		UserID: "owner", AccessToken: "token", Interests: []string{"ai-agents"},
		Size: 3, MicroTarget: 0.2, RubricVersion: "v1", ModelID: "model", Staging: true,
		Planner: &fakePlanner{queries: []discovery.Query{{Text: "ai agents", MaxResults: 3, Days: 30}}, calls: calls},
		YouTube: &fakeYouTube{
			videos: []discovery.Video{
				{VideoID: "v1", ChannelID: "c1", Title: "One"},
				{VideoID: "v2", ChannelID: "c2", Title: "Two"},
				{VideoID: "v3", ChannelID: "c3", Title: "Three"},
			},
			metas: map[string]identity.Subscription{
				"c1": {ChannelID: "c1", SubscriberCount: 1000},
				"c2": {ChannelID: "c2", SubscriberCount: 200000},
				"c3": {ChannelID: "c3", SubscriberCount: 200000},
			},
			calls: calls,
		},
		Quota:       &fakeQuota{calls: calls},
		Fetcher:     &fakeFetcher{calls: calls},
		Transcripts: &fakeTranscripts{calls: calls},
		Analyzer:    &fakeAnalyzer{calls: calls},
		Analyses:    &fakeAnalyses{calls: calls},
		Now:         func() time.Time { return time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC) },
	}
}

func TestPipelineHappy(t *testing.T) {
	var calls []string
	pipeline := testPipeline(&calls)
	result, err := pipeline.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(result.Analyses) != 3 {
		t.Fatalf("analyses = %d, want 3", len(result.Analyses))
	}
	if result.Shortfall != "" {
		t.Fatalf("shortfall = %q", result.Shortfall)
	}
	for i, analysis := range result.Analyses {
		if analysis.UserID != "owner" || analysis.VideoID == "" || analysis.ID == "" {
			t.Fatalf("analysis %d = %+v", i, analysis)
		}
	}
	if len(calls) == 0 {
		t.Fatal("no stages recorded")
	}
	// The stages must run in order. Graph channel reads come before enrichment.
	order := []string{"plan", "reserve", "search", "channels", "videos", "fetch", "stage", "analyze", "save", "delete"}
	position := map[string]int{}
	for i, call := range calls {
		if _, ok := position[call]; !ok {
			position[call] = i
		}
	}
	for i := 1; i < len(order); i++ {
		if position[order[i-1]] >= position[order[i]] {
			t.Fatalf("stage order = %v, want %v", calls, order)
		}
	}
}

func TestPipelineResume(t *testing.T) {
	var calls []string
	pipeline := testPipeline(&calls)
	pipeline.Size = 1
	pipeline.Analyses = &fakeAnalyses{
		calls: &calls,
		items: map[string]analysis.Analysis{
			"owner|v1|v1": {ID: "existing", UserID: "owner", VideoID: "v1", RubricVersion: "v1", State: analysis.StateAnalyzed},
		},
	}
	result, err := pipeline.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(result.Analyses) != 1 || result.Analyses[0].ID != "existing" {
		t.Fatalf("analyses = %+v", result.Analyses)
	}
	for _, call := range calls {
		if call == "fetch" || call == "analyze" || call == "save" {
			t.Fatalf("resume repeated %q: %v", call, calls)
		}
	}
}

func TestPipelineNoTranscript(t *testing.T) {
	var calls []string
	pipeline := testPipeline(&calls)
	pipeline.Size = 1
	pipeline.Fetcher = &fakeFetcher{err: analysis.ErrNoTranscript, calls: &calls}
	result, err := pipeline.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(result.Analyses) != 1 || result.Analyses[0].State != analysis.StateFailed || result.Analyses[0].Reason != analysis.ReasonNoTranscript {
		t.Fatalf("analyses = %+v", result.Analyses)
	}
	for _, call := range calls {
		if call == "analyze" {
			t.Fatalf("analyze ran without a transcript: %v", calls)
		}
	}
}

func TestPipelineQuota(t *testing.T) {
	var calls []string
	pipeline := testPipeline(&calls)
	pipeline.Quota = &fakeQuota{refuseAt: 1, calls: &calls}
	result, err := pipeline.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Shortfall != "quota" {
		t.Fatalf("shortfall = %q", result.Shortfall)
	}
	if len(result.Analyses) != 0 {
		t.Fatalf("analyses = %+v", result.Analyses)
	}
}

func TestPipelineStagingFailure(t *testing.T) {
	var calls []string
	pipeline := testPipeline(&calls)
	pipeline.Size = 1
	pipeline.Transcripts = &fakeTranscripts{putErr: errors.New("disk full"), calls: &calls}
	result, err := pipeline.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(result.Analyses) != 1 {
		t.Fatalf("analyses = %+v", result.Analyses)
	}
	for _, call := range calls {
		if call == "delete" {
			t.Fatalf("delete ran after a staging failure: %v", calls)
		}
	}
}

func TestPipelineAnalyzerError(t *testing.T) {
	var calls []string
	pipeline := testPipeline(&calls)
	pipeline.Size = 1
	pipeline.Analyzer = &fakeAnalyzer{err: errors.New("analyzer down"), calls: &calls}
	result, err := pipeline.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(result.Analyses) != 1 || result.Analyses[0].State != analysis.StateFailed || result.Analyses[0].Reason != analysis.ReasonInvalidModelOutput {
		t.Fatalf("analyses = %+v", result.Analyses)
	}
}

func TestPipelineInvalidConfig(t *testing.T) {
	if _, err := (&Pipeline{}).Run(context.Background()); err == nil {
		t.Fatal("Run(empty) = nil, want an error")
	}
	pipeline := testPipeline(nil)
	pipeline.Size = 0
	if _, err := pipeline.Run(context.Background()); err == nil {
		t.Fatal("Run(size 0) = nil, want an error")
	}
	pipeline = testPipeline(nil)
	pipeline.Interests = nil
	if _, err := pipeline.Run(context.Background()); err == nil {
		t.Fatal("Run(no interests) = nil, want an error")
	}
}

func TestPipelineSearchError(t *testing.T) {
	var calls []string
	pipeline := testPipeline(&calls)
	pipeline.YouTube.(*fakeYouTube).err = errors.New("youtube down")
	result, err := pipeline.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Shortfall != "quota" {
		t.Fatalf("shortfall = %q", result.Shortfall)
	}
}

func TestPipelineUnauthorized(t *testing.T) {
	var calls []string
	pipeline := testPipeline(&calls)
	pipeline.YouTube.(*fakeYouTube).err = identity.ErrUnauthorized
	if _, err := pipeline.Run(context.Background()); !errors.Is(err, identity.ErrUnauthorized) {
		t.Fatalf("Run = %v, want ErrUnauthorized", err)
	}
}

func TestPipelineVideosError(t *testing.T) {
	var calls []string
	pipeline := testPipeline(&calls)
	pipeline.YouTube.(*fakeYouTube).videosErr = errors.New("videos down")
	result, err := pipeline.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(result.Analyses) != 3 {
		t.Fatalf("analyses = %d, want 3", len(result.Analyses))
	}
}

func TestPipelineChannelsError(t *testing.T) {
	var calls []string
	pipeline := testPipeline(&calls)
	pipeline.YouTube.(*fakeYouTube).chErr = errors.New("channels down")
	result, err := pipeline.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(result.Analyses) != 3 {
		t.Fatalf("analyses = %d, want 3", len(result.Analyses))
	}
}

func TestPipelineReserveError(t *testing.T) {
	var calls []string
	pipeline := testPipeline(&calls)
	pipeline.Quota = &fakeQuota{reserveErr: errors.New("quota down"), calls: &calls}
	if _, err := pipeline.Run(context.Background()); err == nil {
		t.Fatal("Run = nil, want an error")
	}
}

func TestPipelineChargeError(t *testing.T) {
	var calls []string
	pipeline := testPipeline(&calls)
	pipeline.Quota = &fakeQuota{chargeErr: errors.New("charge down"), calls: &calls}
	if _, err := pipeline.Run(context.Background()); err == nil {
		t.Fatal("Run = nil, want an error")
	}
}

func TestPipelineQuotaChannels(t *testing.T) {
	var calls []string
	pipeline := testPipeline(&calls)
	pipeline.Quota = &fakeQuota{refuseAt: 2, calls: &calls}
	result, err := pipeline.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Shortfall != "quota" {
		t.Fatalf("shortfall = %q", result.Shortfall)
	}
}

func TestPipelineFindError(t *testing.T) {
	var calls []string
	pipeline := testPipeline(&calls)
	pipeline.Analyses = &fakeAnalyses{findErr: errors.New("db down"), calls: &calls}
	if _, err := pipeline.Run(context.Background()); err == nil {
		t.Fatal("Run = nil, want an error")
	}
}

func TestPipelineSaveError(t *testing.T) {
	var calls []string
	pipeline := testPipeline(&calls)
	pipeline.Size = 1
	pipeline.Analyses = &fakeAnalyses{saveErr: errors.New("write down"), calls: &calls}
	if _, err := pipeline.Run(context.Background()); err == nil {
		t.Fatal("Run = nil, want an error")
	}
}

func TestPipelineNoCandidates(t *testing.T) {
	var calls []string
	pipeline := testPipeline(&calls)
	pipeline.YouTube.(*fakeYouTube).videos = nil
	pipeline.YouTube.(*fakeYouTube).videosErr = errors.New("none")
	result, err := pipeline.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Shortfall != "no_candidates" {
		t.Fatalf("shortfall = %q", result.Shortfall)
	}
}

func TestPipelineStagingDisabled(t *testing.T) {
	var calls []string
	pipeline := testPipeline(&calls)
	pipeline.Staging = false
	pipeline.Transcripts = nil
	pipeline.Size = 1
	result, err := pipeline.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(result.Analyses) != 1 {
		t.Fatalf("analyses = %d, want 1", len(result.Analyses))
	}
	for _, call := range calls {
		if call == "stage" || call == "delete" {
			t.Fatalf("staging ran while disabled: %v", calls)
		}
	}
}

func TestPipelineValidate(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Pipeline)
	}{
		{name: "no planner", edit: func(p *Pipeline) { p.Planner = nil }},
		{name: "no youtube", edit: func(p *Pipeline) { p.YouTube = nil }},
		{name: "no quota", edit: func(p *Pipeline) { p.Quota = nil }},
		{name: "no fetcher", edit: func(p *Pipeline) { p.Fetcher = nil }},
		{name: "no analyzer", edit: func(p *Pipeline) { p.Analyzer = nil }},
		{name: "no analyses", edit: func(p *Pipeline) { p.Analyses = nil }},
		{name: "no transcripts", edit: func(p *Pipeline) { p.Transcripts = nil }},
		{name: "no user", edit: func(p *Pipeline) { p.UserID = "" }},
		{name: "no token", edit: func(p *Pipeline) { p.AccessToken = "" }},
		{name: "too many interests", edit: func(p *Pipeline) { p.Interests = []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11"} }},
		{name: "no rubric", edit: func(p *Pipeline) { p.RubricVersion = "" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline := testPipeline(nil)
			tt.edit(pipeline)
			if _, err := pipeline.Run(context.Background()); err == nil {
				t.Fatal("Run = nil, want an error")
			}
		})
	}
}

func TestPipelineWritesSpans(t *testing.T) {
	dir := t.TempDir()
	logger := trace.New(dir)
	pipeline := testPipeline(nil)
	pipeline.Tracer = logger
	pipeline.Size = 1
	pipeline.BatchID = "b1"
	if _, err := pipeline.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(dir, "traces"))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("trace files = %d", len(entries))
	}
	data, err := os.ReadFile(filepath.Join(dir, "traces", entries[0].Name()))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	names := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var span map[string]any
		if err := json.Unmarshal([]byte(line), &span); err != nil {
			t.Fatalf("decode span: %v", err)
		}
		name, _ := span["name"].(string)
		names[name] = true
	}
	for _, want := range []string{"feed.compose", "feed.analyze"} {
		if !names[want] {
			t.Fatalf("missing span %q in %v", want, names)
		}
	}
}

func TestPipelineProgress(t *testing.T) {
	var stages []string
	pipeline := testPipeline(nil)
	pipeline.Size = 1
	pipeline.Progress = func(stage, _ string, _ map[string]any) { stages = append(stages, stage) }
	if _, err := pipeline.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := []string{"plan", "search", "search_done", "videos", "channels", "select", "fetch", "fetch_done", "stage", "analyze", "analyzed", "save"}
	for _, stage := range want {
		found := false
		for _, got := range stages {
			if got == stage {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing stage %q in %v", stage, stages)
		}
	}
}

func TestPipelineSources(t *testing.T) {
	tests := []struct {
		name     string
		source   discovery.RefreshSource
		search   bool
		trending bool
		graph    bool
	}{
		{name: "keywords", source: discovery.RefreshKeywords, search: true},
		{name: "trending", source: discovery.RefreshTrending, search: true, trending: true},
		{name: "graph", source: discovery.RefreshGraph, search: true, graph: true},
		{name: "fresh", source: discovery.RefreshFresh, trending: true, graph: true},
		{name: "mixed", source: discovery.RefreshMixed, search: true, trending: true, graph: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls []string
			pipeline := testPipeline(&calls)
			pipeline.Source = tt.source
			pipeline.Subscriptions = []identity.Subscription{{ChannelID: "sub1", SubscriberCount: 500000}}
			fake := pipeline.YouTube.(*fakeYouTube)
			fake.metas["sub1"] = identity.Subscription{ChannelID: "sub1", UploadsPlaylistID: "p-sub"}
			fake.playlists = map[string]discovery.VideoPage{
				"p-sub": {Videos: []discovery.Video{{VideoID: "g1", ChannelID: "sub1"}}},
			}
			fake.trending = []discovery.Video{{VideoID: "t1", ChannelID: "c2"}}
			result, err := pipeline.Run(context.Background())
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if len(result.Selection.Selected) == 0 && tt.source != discovery.RefreshFresh {
				t.Fatalf("no selected cards for %s", tt.name)
			}
			if hasCall(calls, "search") != tt.search {
				t.Fatalf("search = %t, calls = %v", hasCall(calls, "search"), calls)
			}
			if hasCall(calls, "most_popular") != tt.trending {
				t.Fatalf("most_popular = %t, calls = %v", hasCall(calls, "most_popular"), calls)
			}
			if hasCall(calls, "playlist_items") != tt.graph {
				t.Fatalf("playlist_items = %t, calls = %v", hasCall(calls, "playlist_items"), calls)
			}
			if tt.search && !hasCall(calls, "plan") {
				t.Fatalf("search source did not plan: %v", calls)
			}
			if !tt.search && hasCall(calls, "plan") {
				t.Fatalf("non-search source planned: %v", calls)
			}
		})
	}
}

func TestPipelineGraphSourcePriority(t *testing.T) {
	var calls []string
	pipeline := testPipeline(&calls)
	pipeline.Source = discovery.RefreshMixed
	pipeline.Size = 5
	pipeline.Subscriptions = []identity.Subscription{{ChannelID: "sub1", SubscriberCount: 500000}}
	fake := pipeline.YouTube.(*fakeYouTube)
	fake.metas["sub1"] = identity.Subscription{ChannelID: "sub1", UploadsPlaylistID: "p-sub"}
	fake.playlists = map[string]discovery.VideoPage{
		"p-sub": {Videos: []discovery.Video{
			{VideoID: "g1", ChannelID: "sub1"},
			{VideoID: "v1", ChannelID: "c1"},
		}},
	}
	fake.trending = []discovery.Video{{VideoID: "t1", ChannelID: "c2"}}
	result, err := pipeline.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(result.Selection.Selected) != 5 {
		t.Fatalf("selected = %d", len(result.Selection.Selected))
	}
	first := result.Selection.Selected[0]
	if first.Source != discovery.SourceGraph {
		t.Fatalf("first source = %q, want graph", first.Source)
	}
	for _, value := range result.Selection.Selected {
		if value.VideoID == "v1" && value.Source != discovery.SourceGraph {
			t.Fatalf("duplicate v1 source = %q, want graph", value.Source)
		}
	}
}

func TestPipelineGraphSkipsPrivateUploads(t *testing.T) {
	var calls []string
	pipeline := testPipeline(&calls)
	pipeline.Source = discovery.RefreshGraph
	pipeline.Subscriptions = []identity.Subscription{{ChannelID: "sub1", SubscriberCount: 500000}}
	fake := pipeline.YouTube.(*fakeYouTube)
	fake.metas["sub1"] = identity.Subscription{ChannelID: "sub1", UploadsPlaylistID: "p-private"}
	fake.playlistErr = map[string]error{"p-private": errors.New("private uploads")}
	result, err := pipeline.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(result.Selection.Selected) != 3 {
		t.Fatalf("selected = %d, want 3", len(result.Selection.Selected))
	}
}

func TestPipelineGraphChannelLimit(t *testing.T) {
	var calls []string
	pipeline := testPipeline(&calls)
	pipeline.Source = discovery.RefreshGraph
	pipeline.GraphChannelLimit = 1
	pipeline.Subscriptions = []identity.Subscription{
		{ChannelID: "sub1", SubscriberCount: 100},
		{ChannelID: "sub2", SubscriberCount: 500},
	}
	fake := pipeline.YouTube.(*fakeYouTube)
	fake.metas["sub1"] = identity.Subscription{ChannelID: "sub1", UploadsPlaylistID: "p1"}
	fake.metas["sub2"] = identity.Subscription{ChannelID: "sub2", UploadsPlaylistID: "p2"}
	fake.playlists = map[string]discovery.VideoPage{
		"p1": {Videos: []discovery.Video{{VideoID: "g1", ChannelID: "sub1"}}},
		"p2": {Videos: []discovery.Video{{VideoID: "g2", ChannelID: "sub2"}}},
	}
	if _, err := pipeline.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(fake.playlistIDs) == 0 || fake.playlistIDs[0] != "p2" {
		t.Fatalf("playlist ids = %v, want p2 first", fake.playlistIDs)
	}
}

func TestDedupeVideosPrefersGraph(t *testing.T) {
	videos := []discovery.Video{
		{VideoID: "v1", Source: discovery.SourceSearch},
		{VideoID: "v1", Source: discovery.SourceGraph},
	}
	got := dedupeVideos(videos)
	if len(got) != 1 || got[0].Source != discovery.SourceGraph {
		t.Fatalf("videos = %+v", got)
	}
}

func hasCall(calls []string, want string) bool {
	for _, call := range calls {
		if call == want {
			return true
		}
	}
	return false
}

func TestPipelineTrendingPriority(t *testing.T) {
	var calls []string
	pipeline := testPipeline(&calls)
	pipeline.Source = discovery.RefreshTrending
	fake := pipeline.YouTube.(*fakeYouTube)
	fake.trending = []discovery.Video{{VideoID: "t1", ChannelID: "c9"}}
	fake.metas["c9"] = identity.Subscription{ChannelID: "c9", SubscriberCount: 1000}
	result, err := pipeline.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(result.Selection.Selected) == 0 || result.Selection.Selected[0].VideoID != "t1" {
		t.Fatalf("selected = %+v", result.Selection.Selected)
	}
}
