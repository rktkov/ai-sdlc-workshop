package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rktkov/ytfeed/internal/config"
	"github.com/rktkov/ytfeed/internal/transport/contract"
	httpapi "github.com/rktkov/ytfeed/internal/transport/http"
)

func runStats(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer, baseURL string) int {
	flags := flag.NewFlagSet("stats", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print one JSON object")
	days := flags.Int("days", 7, "quota trend days")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *days < 1 || *days > 90 {
		_, _ = fmt.Fprintln(stderr, "ytfeed stats: days must be between 1 and 90")
		return 2
	}
	body, code := getJSON(ctx, getenv, baseURL, "/v1/stats?days="+strconv.Itoa(*days), stderr)
	if code != 0 {
		return code
	}
	var stats contract.Stats
	if err := json.Unmarshal(body, &stats); err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed stats: decode: %v\n", err)
		return 1
	}
	if *jsonOut {
		return writeJSON(stdout, stats, stderr)
	}
	printStats(stdout, stats)
	return 0
}

func printStats(w io.Writer, stats contract.Stats) {
	_, _ = fmt.Fprintf(w, "Swipes: %s\n", formatCounts(stats.Swipes))
	_, _ = fmt.Fprintf(w, "Reasons: %s\n", formatCounts(stats.Reasons))
	_, _ = fmt.Fprintf(w, "Cold periods: active=%d expired=%d\n", stats.ColdActive, stats.ColdExpired)
	_, _ = fmt.Fprintln(w, "Quota:")
	for _, point := range stats.Quota {
		_, _ = fmt.Fprintf(w, "  %s %d\n", point.Day, point.Charged)
	}
	_, _ = fmt.Fprintln(w, "Topics:")
	for _, topic := range sortedKeys(stats.Topics) {
		_, _ = fmt.Fprintf(w, "  %s %d\n", topic, stats.Topics[topic])
	}
}

func formatCounts(values map[string]int) string {
	if len(values) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(values))
	for _, key := range sortedKeys(values) {
		parts = append(parts, fmt.Sprintf("%s=%d", key, values[key]))
	}
	return strings.Join(parts, " ")
}

func sortedKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func runRate(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer, baseURL string) int {
	flags := flag.NewFlagSet("rate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	value := flags.String("value", "", "-1, 0 or 1")
	dismiss := flags.Bool("dismiss", false, "record unrated")
	batchIDs := flags.String("batch-id", "", "comma-separated batch IDs")
	jsonOut := flags.Bool("json", false, "print one JSON object")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *dismiss && *value != "" {
		_, _ = fmt.Fprintln(stderr, "ytfeed rate: use --value or --dismiss, not both")
		return 2
	}
	if *value == "" && !*dismiss {
		return showRatingStatus(ctx, getenv, stdout, stderr, baseURL, *jsonOut)
	}

	ratingValue := *value
	if *dismiss {
		ratingValue = "unrated"
	}
	switch ratingValue {
	case "-1", "0", "1", "unrated":
	default:
		_, _ = fmt.Fprintln(stderr, "ytfeed rate: value must be -1, 0, 1 or unrated")
		return 2
	}
	ids := splitCSV(*batchIDs)
	if len(ids) == 0 {
		feedBody, code := getJSON(ctx, getenv, baseURL, "/v1/feed", stderr)
		if code != 0 {
			return code
		}
		var feedValue contract.Feed
		if err := json.Unmarshal(feedBody, &feedValue); err != nil {
			_, _ = fmt.Fprintf(stderr, "ytfeed rate: decode feed: %v\n", err)
			return 1
		}
		if feedValue.BatchID == "" {
			_, _ = fmt.Fprintln(stderr, "ytfeed rate: no batch to rate")
			return 1
		}
		ids = []string{feedValue.BatchID}
	}

	payload, _ := json.Marshal(map[string]any{"value": ratingValue, "batch_ids": ids})
	body, code := postJSON(ctx, getenv, baseURL, "/v1/rating", payload, stderr)
	if code != 0 {
		return code
	}
	var rating contract.Rating
	if err := json.Unmarshal(body, &rating); err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed rate: decode: %v\n", err)
		return 1
	}
	if *jsonOut {
		return writeJSON(stdout, rating, stderr)
	}
	_, _ = fmt.Fprintf(stdout, "rating recorded: value=%s week=%s\n", rating.Value, rating.WeekStart)
	return 0
}

func showRatingStatus(ctx context.Context, getenv func(string) string, stdout, stderr io.Writer, baseURL string, jsonOut bool) int {
	body, code := getJSON(ctx, getenv, baseURL, "/v1/rating", stderr)
	if code != 0 {
		return code
	}
	var status contract.RatingStatus
	if err := json.Unmarshal(body, &status); err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed rate: decode: %v\n", err)
		return 1
	}
	if jsonOut {
		return writeJSON(stdout, status, stderr)
	}
	state := "not due"
	if status.ShouldPrompt {
		state = "due"
	}
	_, _ = fmt.Fprintf(stdout, "weekly rating: %s week=%s\n", state, status.WeekStart)
	if status.Last != nil {
		_, _ = fmt.Fprintf(stdout, "last: value=%s week=%s\n", status.Last.Value, status.Last.WeekStart)
	}
	return 0
}

func maybeRatingPrompt(ctx context.Context, token, baseURL string, stderr io.Writer) {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/rating", nil)
	if err != nil {
		return
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: time.Second}
	resp, err := client.Do(request)
	if err != nil {
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return
	}
	var status contract.RatingStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return
	}
	if status.ShouldPrompt {
		_, _ = fmt.Fprintln(stderr, "weekly rating due: run `ytfeed rate --value -1|0|1` or `ytfeed rate --dismiss`")
	}
}

func getJSON(ctx context.Context, getenv func(string) string, baseURL, path string, stderr io.Writer) ([]byte, int) {
	return apiJSON(ctx, getenv, baseURL, http.MethodGet, path, nil, stderr)
}

func postJSON(ctx context.Context, getenv func(string) string, baseURL, path string, payload []byte, stderr io.Writer) ([]byte, int) {
	return apiJSON(ctx, getenv, baseURL, http.MethodPost, path, payload, stderr)
}

func apiJSON(ctx context.Context, getenv func(string) string, baseURL, method, path string, payload []byte, stderr io.Writer) ([]byte, int) {
	cfg, err := config.Load(getenv)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return nil, 2
	}
	token, err := httpapi.LoadOrCreateToken(cfg.Storage.DataDir)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return nil, 1
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(ctx, method, baseURL+path, body)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return nil, 1
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/json")
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed: server unavailable: %v\n", err)
		return nil, 1
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusUnauthorized {
		_, _ = fmt.Fprintln(stderr, "ytfeed: unauthorized; check the API token")
		return nil, 2
	}
	if resp.StatusCode == http.StatusBadRequest {
		_, _ = fmt.Fprintln(stderr, "ytfeed: invalid request")
		return nil, 2
	}
	if resp.StatusCode != http.StatusOK {
		_, _ = fmt.Fprintf(stderr, "ytfeed: server returned %d\n", resp.StatusCode)
		return nil, 1
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed: read response: %v\n", err)
		return nil, 1
	}
	return data, 0
}

func writeJSON(w io.Writer, value any, stderr io.Writer) int {
	if err := json.NewEncoder(w).Encode(value); err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed: write json: %v\n", err)
		return 1
	}
	return 0
}
