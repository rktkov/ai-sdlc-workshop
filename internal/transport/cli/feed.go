package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rktkov/ytfeed/internal/config"
	"github.com/rktkov/ytfeed/internal/domain/feed"
	"github.com/rktkov/ytfeed/internal/transport/contract"
	httpapi "github.com/rktkov/ytfeed/internal/transport/http"
)

func runFeed(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer, baseURL string) int {
	jsonOut, code := parseFeedFlags(args, stderr)
	if code != 0 {
		return code
	}
	cfg, err := config.Load(getenv)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	token, err := httpapi.LoadOrCreateToken(cfg.Storage.DataDir)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/feed", nil)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	request.Header.Set("Authorization", "Bearer "+token)
	if jsonOut {
		request.Header.Set("Accept", "application/json")
	} else {
		request.Header.Set("Accept", "text/event-stream")
	}

	resp, err := (&http.Client{Timeout: 0}).Do(request)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed feed: server unavailable: %v\n", err)
		return 1
	}
	defer func() { _ = resp.Body.Close() }()
	if code := feedHTTPError(resp, stderr); code >= 0 {
		return code
	}
	if jsonOut || strings.Contains(resp.Header.Get("Content-Type"), "application/json") {
		return renderFeedJSON(resp.Body, stdout, stderr)
	}
	return renderFeedSSE(resp.Body, stdout, stderr)
}

func parseFeedFlags(args []string, stderr io.Writer) (bool, int) {
	flags := flag.NewFlagSet("feed", flag.ContinueOnError)
	flags.SetOutput(stderr)
	jsonOut := flags.Bool("json", false, "print one JSON object")
	_ = flags.Bool("quiet", false, "hide the footer")
	_ = flags.Bool("no-color", false, "disable color")
	if err := flags.Parse(args); err != nil {
		return false, 2
	}
	if flags.NArg() > 0 {
		_, _ = fmt.Fprintf(stderr, "ytfeed feed: unexpected argument %q\n", flags.Arg(0))
		return false, 2
	}
	return *jsonOut, 0
}

func feedHTTPError(resp *http.Response, stderr io.Writer) int {
	if resp.StatusCode == http.StatusOK {
		return -1
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusConflict {
		var problem struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&problem)
		switch problem.Error {
		case "not_authenticated":
			_, _ = fmt.Fprintln(stderr, "ytfeed feed: not authenticated; run `ytfeed onboard`")
		case "interests_not_set":
			_, _ = fmt.Fprintln(stderr, "ytfeed feed: interests not set; run `ytfeed onboard`")
		case "profile_not_set":
			_, _ = fmt.Fprintln(stderr, "ytfeed feed: taste profile not set; run `ytfeed onboard`")
		default:
			if resp.StatusCode == http.StatusUnauthorized {
				_, _ = fmt.Fprintln(stderr, "ytfeed feed: unauthorized; check the API token")
			} else {
				_, _ = fmt.Fprintf(stderr, "ytfeed feed: server returned %d\n", resp.StatusCode)
				return 1
			}
		}
		return 2
	}
	if resp.StatusCode == http.StatusNotFound {
		_, _ = fmt.Fprintln(stderr, "ytfeed feed: no feed available")
		return 6
	}
	_, _ = fmt.Fprintf(stderr, "ytfeed feed: server returned %d\n", resp.StatusCode)
	return 1
}

func renderFeedJSON(body io.Reader, stdout, stderr io.Writer) int {
	var payload contract.Feed
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed feed: decode feed: %v\n", err)
		return 1
	}
	if err := payload.Validate(); err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed feed: invalid feed: %v\n", err)
		return 1
	}
	if err := json.NewEncoder(stdout).Encode(payload); err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed feed: write feed: %v\n", err)
		return 1
	}
	return 0
}

func renderFeedSSE(body io.Reader, stdout, stderr io.Writer) int {
	scanner := bufio.NewScanner(body)
	var event string
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "event: "):
			event = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			data := []byte(strings.TrimPrefix(line, "data: "))
			switch event {
			case "batch.started":
				var start struct {
					Banner string `json:"banner"`
				}
				if err := json.Unmarshal(data, &start); err != nil {
					_, _ = fmt.Fprintf(stderr, "ytfeed feed: decode batch: %v\n", err)
					return 1
				}
				if start.Banner != "" {
					_, _ = fmt.Fprintf(stdout, "banner: %s\n", start.Banner)
				}
			case "card.upsert":
				var wire contract.FeedCard
				if err := json.Unmarshal(data, &wire); err != nil {
					_, _ = fmt.Fprintf(stderr, "ytfeed feed: decode card: %v\n", err)
					return 1
				}
				card, err := cardFromWire(wire)
				if err != nil {
					_, _ = fmt.Fprintf(stderr, "ytfeed feed: invalid card: %v\n", err)
					return 1
				}
				if err := RenderCard(stdout, card); err != nil {
					_, _ = fmt.Fprintf(stderr, "ytfeed feed: render card: %v\n", err)
					return 1
				}
				_, _ = fmt.Fprintln(stdout)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed feed: read stream: %v\n", err)
		return 1
	}
	return 0
}

func cardFromWire(wire contract.FeedCard) (feed.Card, error) {
	published, err := time.Parse(time.RFC3339, wire.PublishedAt)
	if err != nil && wire.PublishedAt != "" {
		return feed.Card{}, fmt.Errorf("published time: %w", err)
	}
	return feed.Card{
		CardID: wire.CardID, VideoID: wire.VideoID, ChannelID: wire.ChannelID,
		Title: wire.Title, Channel: wire.Channel, SubscriberCount: wire.SubscriberCount,
		Views: wire.Views, Duration: time.Duration(wire.DurationSeconds) * time.Second,
		PublishedAt: published, Topic: wire.Topic, TLDR: wire.TLDR,
		Similarity: wire.Similarity, WhyThisCard: wire.WhyThisCard,
		State: feed.State(wire.State), Reason: wire.Reason,
		AnalysisID: wire.AnalysisID, Score: wire.Score, Position: wire.Position,
		Decision: feed.CardDecision(wire.Decision), AnalyzedLate: wire.AnalyzedLate,
	}, nil
}
