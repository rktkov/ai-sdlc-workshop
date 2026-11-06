package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/rktkov/ytfeed/internal/transport/contract"
)

func interactiveFeed() contract.Feed {
	return contract.Feed{
		BatchID: "b1", State: "completed",
		Cards: []contract.FeedCard{
			{
				CardID: "c1", VideoID: "v1", ChannelID: "ch1", Title: "First card",
				Channel: "Channel", State: "analyzed", Topic: "ai", TLDR: "tldr",
				Similarity: "new territory", WhyThisCard: "why", Position: 0,
				ScoredAxes: []contract.AxisScore{{Axis: "depth", Score: 0.8}},
			},
			{
				CardID: "c2", VideoID: "v2", ChannelID: "ch2", Title: "Second card",
				Channel: "Channel", State: "analyzed", Topic: "", TLDR: "tldr",
				Similarity: "new territory", WhyThisCard: "why", Position: 1,
			},
		},
	}
}

func TestInteractiveFeed(t *testing.T) {
	var gestures []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/feed":
			_ = json.NewEncoder(w).Encode(interactiveFeed())
		case "/v1/rating":
			http.NotFound(w, r)
		case "/v1/gestures":
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			gestures = append(gestures, body)
			_, _ = fmt.Fprint(w, `{"applied":true,"duplicate":false}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var out, errOut bytes.Buffer
	input := strings.NewReader("right\nsave\ndown\nup\nleft\n2\nq\n")
	code := RunWithInput(context.Background(), []string{"feed", "--interactive"}, feedTestEnv(t), input, &out, &errOut, server.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut.String())
	}
	if len(gestures) != 3 {
		t.Fatalf("gestures = %d, want 3: %+v", len(gestures), gestures)
	}
	if gestures[0]["gesture"] != "right" || gestures[1]["gesture"] != "save" || gestures[2]["gesture"] != "left" {
		t.Fatalf("gestures = %+v", gestures)
	}
	if gestures[2]["reason"] != "ai-slop" {
		t.Fatalf("left reason = %v", gestures[2]["reason"])
	}
	if gestures[1]["topic"] != "unanalyzed" {
		t.Fatalf("save topic = %v", gestures[1]["topic"])
	}
	axes, ok := gestures[0]["scored_axes"].([]any)
	if !ok || len(axes) != 1 {
		t.Fatalf("scored axes = %+v", gestures[0]["scored_axes"])
	}
	if !strings.Contains(out.String(), "Card 1/2") || !strings.Contains(out.String(), "Card 2/2") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestInteractiveEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"batch_id":"b1","state":"completed","cards":[]}`)
	}))
	defer server.Close()
	var out, errOut bytes.Buffer
	code := RunWithInput(context.Background(), []string{"feed", "--interactive"}, feedTestEnv(t), strings.NewReader("q\n"), &out, &errOut, server.URL)
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "no cards") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestInteractiveGestureError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/feed":
			_ = json.NewEncoder(w).Encode(interactiveFeed())
		case "/v1/rating":
			http.NotFound(w, r)
		case "/v1/gestures":
			http.Error(w, "boom", http.StatusInternalServerError)
		}
	}))
	defer server.Close()
	var out, errOut bytes.Buffer
	code := RunWithInput(context.Background(), []string{"feed", "--interactive"}, feedTestEnv(t), strings.NewReader("right\nq\n"), &out, &errOut, server.URL)
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(errOut.String(), "gesture returned 500") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestInteractiveInvalidAndHelp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/feed":
			_ = json.NewEncoder(w).Encode(interactiveFeed())
		case "/v1/rating":
			http.NotFound(w, r)
		case "/v1/gestures":
			_, _ = fmt.Fprint(w, `{"applied":true,"duplicate":false}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	var out, errOut bytes.Buffer
	input := strings.NewReader("z\n?\nx\nleft\n9\n1\nq\n")
	if code := RunWithInput(context.Background(), []string{"feed", "--interactive"}, feedTestEnv(t), input, &out, &errOut, server.URL); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	for _, want := range []string{"unknown key", "reject and pick a reason", "choose 1, 2, 3, 4 or Esc"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, out.String())
		}
	}
}

func TestNextPreviousCard(t *testing.T) {
	cards := []contract.FeedCard{{CardID: "c1"}, {CardID: "c2", Decision: "rejected"}, {CardID: "c3"}}
	if got := nextCard(cards, 0); got != 2 {
		t.Fatalf("nextCard = %d, want 2", got)
	}
	if got := nextCard(cards, 2); got != 2 {
		t.Fatalf("nextCard at end = %d, want 2", got)
	}
	if got := previousCard(cards, 2); got != 0 {
		t.Fatalf("previousCard = %d, want 0", got)
	}
	if got := previousCard(cards, 0); got != 0 {
		t.Fatalf("previousCard at start = %d, want 0", got)
	}
}

func TestEnableRawNonTerminal(t *testing.T) {
	restore := enableRaw(strings.NewReader(""))
	restore()
}

func TestInteractiveQuitEsc(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/feed" {
			_ = json.NewEncoder(w).Encode(interactiveFeed())
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	var out, errOut bytes.Buffer
	if code := RunWithInput(context.Background(), []string{"feed", "--interactive"}, feedTestEnv(t), strings.NewReader("esc\n"), &out, &errOut, server.URL); code != 0 {
		t.Fatalf("exit = %d", code)
	}
}

func TestInteractiveNavigationBounds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/feed":
			_ = json.NewEncoder(w).Encode(interactiveFeed())
		case "/v1/rating":
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	var out, errOut bytes.Buffer
	input := strings.NewReader("up\ndown\ndown\nup\nup\nq\n")
	if code := RunWithInput(context.Background(), []string{"feed", "--interactive"}, feedTestEnv(t), input, &out, &errOut, server.URL); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if !strings.Contains(out.String(), "Card 2/2") || !strings.Contains(out.String(), "Card 1/2") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestInteractiveDuplicateGesture(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/feed":
			_ = json.NewEncoder(w).Encode(interactiveFeed())
		case "/v1/rating":
			http.NotFound(w, r)
		case "/v1/gestures":
			_, _ = fmt.Fprint(w, `{"applied":true,"duplicate":true}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	var out, errOut bytes.Buffer
	if code := RunWithInput(context.Background(), []string{"feed", "--interactive"}, feedTestEnv(t), strings.NewReader("right\nq\n"), &out, &errOut, server.URL); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if strings.Count(out.String(), "Card 2/2") != 0 {
		t.Fatalf("duplicate gesture advanced: %q", out.String())
	}
}

func TestRenderInteractiveCardInvalid(t *testing.T) {
	var out bytes.Buffer
	renderInteractiveCard(&out, contract.FeedCard{CardID: "c", VideoID: "v", PublishedAt: "bad"}, 0, 1, false)
	if !strings.Contains(out.String(), "invalid card") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestSendGestureBadURL(t *testing.T) {
	if _, err := sendGesture(context.Background(), "token", "://bad", contract.FeedCard{CardID: "c"}, "right", ""); err == nil {
		t.Fatal("sendGesture = nil, want an error")
	}
}

func TestReadInteractiveKeyEOF(t *testing.T) {
	key, err := readInteractiveKey(bufio.NewReader(strings.NewReader("")), false, io.Discard)
	if key != "q" || err == nil {
		t.Fatalf("key = %q, err = %v", key, err)
	}
}

func TestInteractiveReasonQuit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/feed" {
			_ = json.NewEncoder(w).Encode(interactiveFeed())
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	var out, errOut bytes.Buffer
	if code := RunWithInput(context.Background(), []string{"feed", "--interactive"}, feedTestEnv(t), strings.NewReader("left\nq\n"), &out, &errOut, server.URL); code != 0 {
		t.Fatalf("exit = %d", code)
	}
}

func TestRenderInteractiveCardClear(t *testing.T) {
	var out bytes.Buffer
	renderInteractiveCard(&out, interactiveFeed().Cards[0], 0, 2, true)
	if !strings.Contains(out.String(), "\033[2J") {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestEnableRawRegularFile(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer func() { _ = file.Close() }()
	restore := enableRaw(file)
	restore()
}

func TestReadInteractiveKeyEmptyLine(t *testing.T) {
	key, err := readInteractiveKey(bufio.NewReader(strings.NewReader("\n")), false, io.Discard)
	if err != nil || key != "enter" {
		t.Fatalf("key = %q, err = %v", key, err)
	}
	if isTerminal(strings.NewReader("")) {
		t.Fatal("strings.Reader reported as a terminal")
	}
}
