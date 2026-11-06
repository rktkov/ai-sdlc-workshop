package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/taste"
	"github.com/rktkov/ytfeed/internal/transport/contract"
)

func renderFeedInteractive(ctx context.Context, token, baseURL string, body io.Reader, stdin io.Reader, stdout, stderr io.Writer) int {
	var payload contract.Feed
	if err := json.NewDecoder(body).Decode(&payload); err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed feed: decode feed: %v\n", err)
		return 1
	}
	if err := payload.Validate(); err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed feed: invalid feed: %v\n", err)
		return 1
	}
	if len(payload.Cards) == 0 {
		_, _ = fmt.Fprintln(stdout, "no cards")
		return 0
	}

	raw := isTerminal(stdin) && isTerminal(stdout)
	restore := enableRaw(stdin)
	defer restore()
	reader := bufio.NewReader(stdin)
	picker := &ReasonPicker{}
	index := 0
	for {
		if len(payload.Cards) == 0 {
			_, _ = fmt.Fprintln(stdout, "no cards; press r to refresh or q to quit")
			key, err := readInteractiveKey(reader, raw, stdout)
			if err != nil && key == "" {
				return 0
			}
			if strings.EqualFold(key, "q") || strings.EqualFold(key, "esc") {
				return 0
			}
			if strings.EqualFold(key, "r") || strings.EqualFold(key, "refresh") {
				payload, index = refreshInteractiveFeed(ctx, token, baseURL, stdout, stderr, payload)
			}
			continue
		}
		renderInteractiveCard(stdout, payload.Cards[index], index, len(payload.Cards), raw)
		key, err := readInteractiveKey(reader, raw, stdout)
		if err != nil && key == "" {
			return 0
		}
		if strings.EqualFold(key, "r") || strings.EqualFold(key, "refresh") {
			payload, index = refreshInteractiveFeed(ctx, token, baseURL, stdout, stderr, payload)
			continue
		}
		next, quit := handleInteractiveKey(ctx, token, baseURL, &payload.Cards, index, key, reader, raw, stdout, stderr, picker)
		if quit {
			return 0
		}
		index = next
	}
}

func refreshInteractiveFeed(ctx context.Context, token, baseURL string, stdout, stderr io.Writer, current contract.Feed) (contract.Feed, int) {
	_, _ = fmt.Fprintln(stdout, "composing a new batch...")
	value, err := fetchInteractiveFeed(ctx, token, baseURL)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed feed: refresh failed: %v\n", err)
		return current, 0
	}
	return value, 0
}

func fetchInteractiveFeed(ctx context.Context, token, baseURL string) (contract.Feed, error) {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/feed", nil)
	if err != nil {
		return contract.Feed{}, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("X-Ytfeed-Refresh", "true")
	resp, err := (&http.Client{Timeout: 0}).Do(request)
	if err != nil {
		return contract.Feed{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return contract.Feed{}, fmt.Errorf("server returned %d", resp.StatusCode)
	}
	var payload contract.Feed
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return contract.Feed{}, err
	}
	if err := payload.Validate(); err != nil {
		return contract.Feed{}, err
	}
	return payload, nil
}

func handleInteractiveKey(ctx context.Context, token, baseURL string, cards *[]contract.FeedCard, index int, key string, reader *bufio.Reader, raw bool, stdout, stderr io.Writer, picker *ReasonPicker) (int, bool) {
	switch strings.ToLower(key) {
	case "q", "esc":
		return index, true
	case "right", "l":
		if applyGesture(ctx, token, baseURL, cards, index, "right", "", stderr) {
			return nextCard(*cards, index), false
		}
	case "left", "h":
		reason, ok := askReason(reader, raw, stdout, picker, (*cards)[index])
		if !ok {
			return index, true
		}
		if applyGesture(ctx, token, baseURL, cards, index, "left", reason, stderr) {
			return nextCard(*cards, index), false
		}
	case "save", "s":
		applyGesture(ctx, token, baseURL, cards, index, "save", "", stderr)
	case "down", "j", "enter":
		return nextCard(*cards, index), false
	case "up", "k":
		return previousCard(*cards, index), false
	case "open", "o":
		openVideo((*cards)[index].VideoID, stdout, stderr)
	case "?", "help":
		printInteractiveHelp(stdout)
		_, _ = readInteractiveKey(reader, raw, stdout)
	default:
		_, _ = fmt.Fprintf(stdout, "unknown key %q\n", key)
	}
	return index, false
}

func applyGesture(ctx context.Context, token, baseURL string, cards *[]contract.FeedCard, index int, gesture, reason string, stderr io.Writer) bool {
	card := (*cards)[index]
	applied, err := sendGesture(ctx, token, baseURL, card, gesture, reason)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed feed: %v\n", err)
		return false
	}
	if !applied {
		return false
	}
	switch gesture {
	case "right":
		(*cards)[index].Decision = "accepted"
	case "left":
		(*cards)[index].Decision = "rejected"
	case "save":
		(*cards)[index].Decision = "saved"
	}
	return true
}

func renderInteractiveCard(w io.Writer, card contract.FeedCard, index, total int, clear bool) {
	if clear {
		_, _ = fmt.Fprint(w, "\033[2J\033[H")
	}
	_, _ = fmt.Fprintf(w, "Card %d/%d\n\n", index+1, total)
	value, err := cardFromWire(card)
	if err != nil {
		_, _ = fmt.Fprintf(w, "invalid card: %v\n", err)
		return
	}
	_ = RenderCard(w, value)
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "[←/h] reject  [→/l] accept  [s] save  [j] down  [k] up")
	_, _ = fmt.Fprintln(w, "[o] open      [r] refresh  [?] help     [q] quit")
}

func readInteractiveKey(reader *bufio.Reader, raw bool, stdout io.Writer) (string, error) {
	if raw {
		value, err := reader.ReadByte()
		if err != nil {
			return "", err
		}
		switch value {
		case '\r', '\n':
			return "enter", nil
		case 0x1b:
			return "esc", nil
		default:
			return string(value), nil
		}
	}
	_, _ = fmt.Fprint(stdout, "command> ")
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return "q", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return "enter", nil
	}
	return line, nil
}

func askReason(reader *bufio.Reader, raw bool, stdout io.Writer, picker *ReasonPicker, card contract.FeedCard) (string, bool) {
	for {
		if err := picker.Render(stdout, card.Channel, card.Topic); err != nil {
			return "", false
		}
		key, err := readInteractiveKey(reader, raw, stdout)
		if err != nil && key == "" {
			return "", false
		}
		if strings.EqualFold(key, "q") {
			return "", false
		}
		reason, err := picker.Key(key)
		if err != nil {
			_, _ = fmt.Fprintln(stdout, "choose 1, 2, 3, 4 or Esc")
			continue
		}
		return string(reason), true
	}
}

func sendGesture(ctx context.Context, token, baseURL string, card contract.FeedCard, gesture, reason string) (bool, error) {
	topic := strings.TrimSpace(card.Topic)
	if topic == "" {
		topic = "unanalyzed"
	}
	axes := make([]taste.AxisScore, 0, len(card.ScoredAxes))
	for _, score := range card.ScoredAxes {
		axes = append(axes, taste.AxisScore{Axis: score.Axis, Score: score.Score})
	}
	payload, err := json.Marshal(map[string]any{
		"gesture_id":  fmt.Sprintf("%s-%d", card.CardID, time.Now().UnixNano()),
		"card_id":     card.CardID,
		"video_id":    card.VideoID,
		"channel_id":  card.ChannelID,
		"topic":       topic,
		"gesture":     gesture,
		"reason":      reason,
		"scored_axes": axes,
	})
	if err != nil {
		return false, err
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/gestures", strings.NewReader(string(payload)))
	if err != nil {
		return false, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		return false, fmt.Errorf("gesture request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("gesture returned %d", resp.StatusCode)
	}
	var result struct {
		Applied   bool `json:"applied"`
		Duplicate bool `json:"duplicate"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}
	return result.Applied && !result.Duplicate, nil
}

func nextCard(cards []contract.FeedCard, current int) int {
	for index := current + 1; index < len(cards); index++ {
		if cards[index].Decision != "rejected" {
			return index
		}
	}
	return current
}

func previousCard(cards []contract.FeedCard, current int) int {
	for index := current - 1; index >= 0; index-- {
		if cards[index].Decision != "rejected" {
			return index
		}
	}
	return current
}

func printInteractiveHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "h / ←   reject and pick a reason")
	_, _ = fmt.Fprintln(w, "l / →   accept and move on")
	_, _ = fmt.Fprintln(w, "s       save without moving")
	_, _ = fmt.Fprintln(w, "j / ↓   next card")
	_, _ = fmt.Fprintln(w, "k / ↑   previous card")
	_, _ = fmt.Fprintln(w, "o       open the video in the browser")
	_, _ = fmt.Fprintln(w, "r       compose a new batch")
	_, _ = fmt.Fprintln(w, "q / Esc quit")
	_, _ = fmt.Fprintln(w, "\nPress any key to return.")
}

func openVideo(videoID string, stdout, stderr io.Writer) {
	url := "https://www.youtube.com/watch?v=" + videoID
	command := "open"
	if runtime.GOOS != "darwin" {
		command = "xdg-open"
	}
	if err := exec.Command(command, url).Start(); err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed feed: open %s: %v\n", url, err)
		return
	}
	_, _ = fmt.Fprintf(stdout, "opened %s\n", url)
}

func enableRaw(stdin io.Reader) func() {
	file, ok := stdin.(*os.File)
	if !ok || !isTerminal(file) {
		return func() {}
	}
	cmd := exec.Command("stty", "-echo", "-icanon", "min", "1", "time", "0")
	cmd.Stdin = file
	if err := cmd.Run(); err != nil {
		return func() {}
	}
	return func() {
		restore := exec.Command("stty", "echo", "icanon")
		restore.Stdin = file
		_ = restore.Run()
	}
}
