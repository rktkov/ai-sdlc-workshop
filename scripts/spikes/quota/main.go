// Command quota measures the YouTube Data API cost of one planned batch.
//
// It is spike code, not part of the product. It runs the discovery calls in the
// same shape as the planned batch and reports the billable units. The harness
// never prints the API key.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	costSearch        = 100
	costVideos        = 1
	costChannels      = 1
	costSubscriptions = 1
	costMostPopular   = 1

	dailyBudget       = 10000
	usableBudget      = 8000 // 80% warning threshold from F7
	strictBatchTarget = 100
	maxIDsPerCall     = 50
)

type plannedBatch struct {
	Interests  []string
	MaxResults int
	Days       int
	Region     string
}

type call struct {
	Name   string `json:"name"`
	Units  int    `json:"units"`
	Status int    `json:"status"`
	Items  int    `json:"items"`
	Error  string `json:"error,omitempty"`
}

type report struct {
	StartedAt         string   `json:"started_at"`
	Interests         []string `json:"interests"`
	MaxResults        int      `json:"max_results"`
	Calls             []call   `json:"calls"`
	TotalUnits        int      `json:"total_units"`
	UniqueVideos      int      `json:"unique_videos"`
	UniqueChannels    int      `json:"unique_channels"`
	QuotaExhausted    bool     `json:"quota_exhausted"`
	MeetsStrictTarget bool     `json:"meets_strict_target"`
	BatchesPerDayAt80 int      `json:"batches_per_day_at_80_percent"`
	Feasible20Cards   bool     `json:"feasible_20_cards"`
	Verdict           string   `json:"verdict"`
	Note              string   `json:"note"`
}

type runner struct {
	apiKey      string
	accessToken string
	baseURL     string
	client      *http.Client
}

type httpError struct {
	Status  int
	Reason  string
	Message string
}

func (e *httpError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("http %d: %s", e.Status, e.Reason)
	}
	return fmt.Sprintf("http %d: %s", e.Status, e.Message)
}

func isQuotaError(err error) bool {
	var herr *httpError
	if !errors.As(err, &herr) {
		return false
	}
	switch herr.Reason {
	case "quotaExceeded", "dailyLimitExceeded":
		return true
	default:
		return false
	}
}

type errorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Errors  []struct {
			Reason string `json:"reason"`
		} `json:"errors"`
	} `json:"error"`
}

func (e errorResponse) reason() string {
	if len(e.Error.Errors) == 0 {
		return ""
	}
	return e.Error.Errors[0].Reason
}

type searchResponse struct {
	Items []struct {
		ID struct {
			VideoID string `json:"videoId"`
		} `json:"id"`
		Snippet struct {
			ChannelID string `json:"channelId"`
		} `json:"snippet"`
	} `json:"items"`
}

type videosResponse struct {
	Items []struct {
		ID      string `json:"id"`
		Snippet struct {
			ChannelID string `json:"channelId"`
		} `json:"snippet"`
	} `json:"items"`
}

type channelsResponse struct {
	Items []struct {
		ID string `json:"id"`
	} `json:"items"`
}

func (r *runner) get(ctx context.Context, endpoint string, params url.Values) ([]byte, int, error) {
	if r.accessToken == "" {
		params.Set("key", r.apiKey)
	}
	requestURL := strings.TrimRight(r.baseURL, "/") + "/" + endpoint + "?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}
	if r.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+r.accessToken)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("call %s: %s", endpoint, r.redact(err.Error()))
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read %s: %w", endpoint, err)
	}
	if resp.StatusCode != http.StatusOK {
		var apiErr errorResponse
		_ = json.Unmarshal(body, &apiErr)
		return body, resp.StatusCode, &httpError{
			Status:  resp.StatusCode,
			Reason:  apiErr.reason(),
			Message: r.redact(apiErr.Error.Message),
		}
	}
	return body, resp.StatusCode, nil
}

func (r *runner) redact(value string) string {
	if r.apiKey != "" {
		value = strings.ReplaceAll(value, r.apiKey, "REDACTED")
	}
	if r.accessToken != "" {
		value = strings.ReplaceAll(value, r.accessToken, "REDACTED")
	}
	return value
}

func addCall(rep *report, item call) {
	rep.Calls = append(rep.Calls, item)
	if item.Error == "" {
		rep.TotalUnits += item.Units
	}
}

func run(ctx context.Context, r *runner, batch plannedBatch) report {
	rep := report{
		StartedAt:  time.Now().UTC().Format(time.RFC3339),
		Interests:  append([]string(nil), batch.Interests...),
		MaxResults: batch.MaxResults,
	}

	videos, channels, stopped := runSearches(ctx, r, batch, &rep)
	if stopped {
		rep.QuotaExhausted = true
		finalize(&rep)
		return rep
	}
	channels, stopped = runVideos(ctx, r, videos, channels, &rep)
	if stopped {
		rep.QuotaExhausted = true
		finalize(&rep)
		return rep
	}
	if runChannels(ctx, r, channels, &rep) {
		rep.QuotaExhausted = true
	}
	finalize(&rep)
	return rep
}

func runSearches(ctx context.Context, r *runner, batch plannedBatch, rep *report) ([]string, []string, bool) {
	var videos []string
	var channels []string
	for _, interest := range batch.Interests {
		params := url.Values{}
		params.Set("part", "snippet")
		params.Set("type", "video")
		params.Set("maxResults", strconv.Itoa(batch.MaxResults))
		params.Set("order", "relevance")
		params.Set("q", interest)
		params.Set("publishedAfter", time.Now().UTC().AddDate(0, 0, -batch.Days).Format(time.RFC3339))
		if batch.Region != "" {
			params.Set("regionCode", batch.Region)
		}

		body, status, err := r.get(ctx, "search", params)
		item := call{Name: "search.list", Units: costSearch, Status: status}
		if err != nil {
			item.Error = err.Error()
			addCall(rep, item)
			if isQuotaError(err) {
				return nil, nil, true
			}
			continue
		}
		var sr searchResponse
		if err := json.Unmarshal(body, &sr); err != nil {
			item.Error = "decode response: " + err.Error()
			addCall(rep, item)
			continue
		}
		item.Items = len(sr.Items)
		addCall(rep, item)
		for _, result := range sr.Items {
			if result.ID.VideoID != "" {
				videos = append(videos, result.ID.VideoID)
			}
			if result.Snippet.ChannelID != "" {
				channels = append(channels, result.Snippet.ChannelID)
			}
		}
	}
	return videos, channels, false
}

func runVideos(ctx context.Context, r *runner, videos, channels []string, rep *report) ([]string, bool) {
	videos = dedupe(videos)
	rep.UniqueVideos = len(videos)
	for _, chunk := range chunkStrings(videos, maxIDsPerCall) {
		params := url.Values{}
		params.Set("part", "snippet,contentDetails,statistics,topicDetails")
		params.Set("id", strings.Join(chunk, ","))

		body, status, err := r.get(ctx, "videos", params)
		item := call{Name: "videos.list", Units: costVideos, Status: status}
		if err != nil {
			item.Error = err.Error()
			addCall(rep, item)
			if isQuotaError(err) {
				return channels, true
			}
			continue
		}
		var vr videosResponse
		if err := json.Unmarshal(body, &vr); err != nil {
			item.Error = "decode response: " + err.Error()
			addCall(rep, item)
			continue
		}
		item.Items = len(vr.Items)
		addCall(rep, item)
		for _, video := range vr.Items {
			if video.Snippet.ChannelID != "" {
				channels = append(channels, video.Snippet.ChannelID)
			}
		}
	}
	return channels, false
}

func runChannels(ctx context.Context, r *runner, channels []string, rep *report) bool {
	channels = dedupe(channels)
	rep.UniqueChannels = len(channels)
	for _, chunk := range chunkStrings(channels, maxIDsPerCall) {
		params := url.Values{}
		params.Set("part", "snippet,statistics,topicDetails")
		params.Set("id", strings.Join(chunk, ","))

		body, status, err := r.get(ctx, "channels", params)
		item := call{Name: "channels.list", Units: costChannels, Status: status}
		if err != nil {
			item.Error = err.Error()
			addCall(rep, item)
			if isQuotaError(err) {
				return true
			}
			continue
		}
		var cr channelsResponse
		if err := json.Unmarshal(body, &cr); err != nil {
			item.Error = "decode response: " + err.Error()
			addCall(rep, item)
			continue
		}
		item.Items = len(cr.Items)
		addCall(rep, item)
	}
	return false
}

func runMostPopular(ctx context.Context, r *runner, maxResults int, region string, rep *report) {
	if rep.QuotaExhausted {
		return
	}
	params := url.Values{}
	params.Set("part", "snippet,contentDetails,statistics")
	params.Set("chart", "mostPopular")
	params.Set("maxResults", strconv.Itoa(maxResults))
	if region != "" {
		params.Set("regionCode", region)
	}
	item := call{Name: "mostPopular", Units: costMostPopular}
	body, status, err := r.get(ctx, "videos", params)
	item.Status = status
	if err != nil {
		item.Error = err.Error()
		addCall(rep, item)
		if isQuotaError(err) {
			rep.QuotaExhausted = true
		}
		return
	}
	var vr videosResponse
	if err := json.Unmarshal(body, &vr); err != nil {
		item.Error = "decode response: " + err.Error()
		addCall(rep, item)
		return
	}
	item.Items = len(vr.Items)
	addCall(rep, item)
}

func runSubscriptions(ctx context.Context, r *runner, rep *report) {
	pageToken := ""
	for pages := 0; pages < 20; pages++ {
		if rep.QuotaExhausted {
			return
		}
		params := url.Values{}
		params.Set("part", "snippet,contentDetails")
		params.Set("mine", "true")
		params.Set("maxResults", "50")
		if pageToken != "" {
			params.Set("pageToken", pageToken)
		}
		item := call{Name: "subscriptions.list", Units: costSubscriptions}
		body, status, err := r.get(ctx, "subscriptions", params)
		item.Status = status
		if err != nil {
			item.Error = err.Error()
			addCall(rep, item)
			if isQuotaError(err) {
				rep.QuotaExhausted = true
			}
			return
		}
		var page struct {
			Items         []json.RawMessage `json:"items"`
			NextPageToken string            `json:"nextPageToken"`
		}
		if err := json.Unmarshal(body, &page); err != nil {
			item.Error = "decode response: " + err.Error()
			addCall(rep, item)
			return
		}
		item.Items = len(page.Items)
		addCall(rep, item)
		if page.NextPageToken == "" {
			return
		}
		pageToken = page.NextPageToken
	}
}

func finalize(rep *report) {
	rep.MeetsStrictTarget = rep.TotalUnits <= strictBatchTarget
	rep.BatchesPerDayAt80 = 0
	if rep.TotalUnits > 0 {
		rep.BatchesPerDayAt80 = usableBudget / rep.TotalUnits
	}
	rep.Feasible20Cards = rep.BatchesPerDayAt80 >= 3
	switch {
	case rep.QuotaExhausted:
		rep.Verdict = "INCOMPLETE — quota exhausted"
		rep.Note = "The project quota is used up. Re-run on a fresh quota day."
	case rep.TotalUnits == 0:
		rep.Verdict = "INCOMPLETE — no billable call"
		rep.Note = "Check the API key, the API enablement and the network."
	case rep.MeetsStrictTarget && rep.Feasible20Cards:
		rep.Verdict = "GO"
		rep.Note = "The batch meets the ~100-unit target and supports at least 3 batches per day at 80% budget."
	case rep.Feasible20Cards:
		rep.Verdict = "GO within the daily budget, NO-GO for the ~100-unit target"
		rep.Note = "The batch fits the 10,000-unit day with at least 3 batches at 80% budget, but it exceeds 100 units."
	default:
		rep.Verdict = "NO-GO"
		rep.Note = "One batch does not fit the 10,000-unit day at 3 batches and 80% budget."
	}
}

func dedupe(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	var out []string
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func chunkStrings(values []string, size int) [][]string {
	if size <= 0 {
		size = maxIDsPerCall
	}
	var chunks [][]string
	for len(values) > 0 {
		n := size
		if len(values) < n {
			n = len(values)
		}
		chunks = append(chunks, values[:n])
		values = values[n:]
	}
	return chunks
}

func (p plannedBatch) maxUnits() int {
	searches := len(p.Interests)
	if searches == 0 {
		return 0
	}
	ids := searches * p.MaxResults
	videoCalls := (ids + maxIDsPerCall - 1) / maxIDsPerCall
	return searches*costSearch + videoCalls*costVideos + videoCalls*costChannels
}

func loadKey(path string) (string, error) {
	if value := strings.TrimSpace(os.Getenv("YOUTUBE_API_KEY")); value != "" {
		return value, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read key file: %w", err)
	}
	value := strings.TrimSpace(string(data))
	if value == "" {
		return "", fmt.Errorf("key file %s is empty", path)
	}
	return value, nil
}

func printSummary(w io.Writer, rep report) {
	_, _ = fmt.Fprintf(w, "S-03 quota measurement\n")
	_, _ = fmt.Fprintf(w, "Interests: %d, max results per search: %d\n", len(rep.Interests), rep.MaxResults)
	_, _ = fmt.Fprintf(w, "\n%-16s %6s %6s %6s %s\n", "CALL", "UNITS", "STATUS", "ITEMS", "ERROR")
	for _, item := range rep.Calls {
		_, _ = fmt.Fprintf(w, "%-16s %6d %6d %6d %s\n", item.Name, item.Units, item.Status, item.Items, item.Error)
	}
	_, _ = fmt.Fprintf(w, "\nTotal units:        %d\n", rep.TotalUnits)
	_, _ = fmt.Fprintf(w, "Unique videos:      %d\n", rep.UniqueVideos)
	_, _ = fmt.Fprintf(w, "Unique channels:    %d\n", rep.UniqueChannels)
	_, _ = fmt.Fprintf(w, "Batches/day at 80%%: %d\n", rep.BatchesPerDayAt80)
	_, _ = fmt.Fprintf(w, "Strict <=100 units: %t\n", rep.MeetsStrictTarget)
	_, _ = fmt.Fprintf(w, "Verdict:            %s\n", rep.Verdict)
	_, _ = fmt.Fprintf(w, "Note:               %s\n", rep.Note)
}

type options struct {
	batch         plannedBatch
	keyFile       string
	baseURL       string
	out           string
	dryRun        bool
	mostPopular   bool
	subscriptions bool
	accessToken   string
}

func parseOptions() options {
	var (
		interests     = flag.String("interests", "AI agents,local LLM,distributed systems,developer productivity,machine learning", "comma-separated interests")
		maxResults    = flag.Int("max-results", 50, "results per search, 1-50")
		days          = flag.Int("days", 30, "publishedAfter window in days")
		region        = flag.String("region", "US", "region code for search and mostPopular")
		keyFile       = flag.String("key-file", ".secrets/youtube/api_key", "API key file")
		baseURL       = flag.String("base-url", "https://www.googleapis.com/youtube/v3", "API base URL")
		out           = flag.String("out", "", "write the JSON report to this path")
		dryRun        = flag.Bool("dry-run", false, "print the planned cost without calling the API")
		mostPopular   = flag.Bool("most-popular", false, "also call videos.list?chart=mostPopular")
		accessToken   = flag.String("access-token", "", "OAuth access token for subscriptions")
		subscriptions = flag.Bool("subscriptions", false, "also page subscriptions.list; needs -access-token")
	)
	flag.Parse()

	var list []string
	for _, value := range strings.Split(*interests, ",") {
		value = strings.TrimSpace(value)
		if value != "" {
			list = append(list, value)
		}
	}
	if len(list) == 0 {
		fail("no interests")
	}
	if *maxResults < 1 || *maxResults > 50 {
		fail("max-results must be 1-50")
	}
	if *subscriptions && *accessToken == "" {
		fail("subscriptions needs -access-token")
	}
	return options{
		batch:         plannedBatch{Interests: list, MaxResults: *maxResults, Days: *days, Region: *region},
		keyFile:       *keyFile,
		baseURL:       *baseURL,
		out:           *out,
		dryRun:        *dryRun,
		mostPopular:   *mostPopular,
		subscriptions: *subscriptions,
		accessToken:   *accessToken,
	}
}

func fail(message string) {
	_, _ = fmt.Fprintln(os.Stderr, message)
	os.Exit(2)
}

func printDryRun(opts options) {
	_, _ = fmt.Fprintf(os.Stdout, "Dry run: %d searches, worst case %d units.\n", len(opts.batch.Interests), opts.batch.maxUnits())
	_, _ = fmt.Fprintf(os.Stdout, "Searches: %d x %d = %d units.\n", len(opts.batch.Interests), costSearch, len(opts.batch.Interests)*costSearch)
	_, _ = fmt.Fprintf(os.Stdout, "videos.list and channels.list are 1 unit per 50 IDs.\n")
	if opts.mostPopular {
		_, _ = fmt.Fprintf(os.Stdout, "mostPopular: +%d unit.\n", costMostPopular)
	}
	if opts.subscriptions {
		_, _ = fmt.Fprintf(os.Stdout, "subscriptions: +%d unit per page.\n", costSubscriptions)
	}
}

func writeReport(path string, rep report) {
	if path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "create output directory: %v\n", err)
		os.Exit(1)
	}
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "encode report: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "write report: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	opts := parseOptions()
	if opts.dryRun {
		printDryRun(opts)
		return
	}

	key, err := loadKey(opts.keyFile)
	if err != nil && opts.accessToken == "" {
		_, _ = fmt.Fprintf(os.Stderr, "missing credentials: %v\n", err)
		_, _ = fmt.Fprintf(os.Stderr, "Set YOUTUBE_API_KEY or write the key to %s.\n", opts.keyFile)
		os.Exit(2)
	}

	r := &runner{
		apiKey:      key,
		accessToken: opts.accessToken,
		baseURL:     opts.baseURL,
		client:      &http.Client{Timeout: 30 * time.Second},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	rep := run(ctx, r, opts.batch)
	if opts.mostPopular {
		runMostPopular(ctx, r, opts.batch.MaxResults, opts.batch.Region, &rep)
	}
	if opts.subscriptions {
		runSubscriptions(ctx, r, &rep)
	}
	finalize(&rep)

	printSummary(os.Stdout, rep)
	writeReport(opts.out, rep)
	if rep.QuotaExhausted {
		os.Exit(3)
	}
}
