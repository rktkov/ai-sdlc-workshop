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

	"github.com/rktkov/ytfeed/internal/domain/discovery"
	"github.com/rktkov/ytfeed/internal/domain/taste"
	"github.com/rktkov/ytfeed/internal/transport/contract"
)

func renderFeedInteractive(ctx context.Context, token, baseURL, defaultSource string, body io.Reader, stdin io.Reader, stdout, stderr io.Writer) int {
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
	lastSource := discovery.RefreshSource(defaultSource)
	if lastSource == "" {
		lastSource = discovery.RefreshMixed
	}
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
				if chosen, ok := chooseRefreshSource(reader, raw, stdout, lastSource); ok {
					lastSource = chosen
					payload, index = refreshInteractiveFeed(ctx, token, baseURL, lastSource, stdout, stderr, payload)
				}
			}
			continue
		}
		renderInteractiveCard(stdout, payload.Cards[index], index, len(payload.Cards), raw)
		key, err := readInteractiveKey(reader, raw, stdout)
		if err != nil && key == "" {
			return 0
		}
		if strings.EqualFold(key, "r") || strings.EqualFold(key, "refresh") {
			if chosen, ok := chooseRefreshSource(reader, raw, stdout, lastSource); ok {
				lastSource = chosen
				payload, index = refreshInteractiveFeed(ctx, token, baseURL, lastSource, stdout, stderr, payload)
			}
			continue
		}
		next, quit := handleInteractiveKey(ctx, token, baseURL, &payload.Cards, index, key, reader, raw, stdout, stderr, picker)
		if quit {
			return 0
		}
		index = next
	}
}

func refreshInteractiveFeed(ctx context.Context, token, baseURL string, source discovery.RefreshSource, stdout, stderr io.Writer, current contract.Feed) (contract.Feed, int) {
	_, _ = fmt.Fprintf(stdout, "composing a new batch with source %s...\n", source)
	value, err := fetchInteractiveFeed(ctx, token, baseURL, source, stderr)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed feed: refresh failed: %v\n", err)
		return current, 0
	}
	return value, 0
}

func fetchInteractiveFeed(ctx context.Context, token, baseURL string, source discovery.RefreshSource, stderr io.Writer) (contract.Feed, error) {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/feed", nil)
	if err != nil {
		return contract.Feed{}, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "text/event-stream")
	request.Header.Set("X-Ytfeed-Refresh", "true")
	if source != "" {
		request.Header.Set("X-Ytfeed-Source", string(source))
	}
	resp, err := (&http.Client{Timeout: 0}).Do(request)
	if err != nil {
		return contract.Feed{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return contract.Feed{}, fmt.Errorf("server returned %d", resp.StatusCode)
	}

	var payload contract.Feed
	scanner := bufio.NewScanner(resp.Body)
	event := ""
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "event: "):
			event = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			done, err := applyInteractiveEvent(event, []byte(strings.TrimPrefix(line, "data: ")), &payload, stderr)
			if err != nil {
				return contract.Feed{}, err
			}
			if done {
				return payload, nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return contract.Feed{}, err
	}
	return contract.Feed{}, fmt.Errorf("stream ended before batch.done")
}

func applyInteractiveEvent(event string, data []byte, payload *contract.Feed, stderr io.Writer) (bool, error) {
	switch event {
	case "progress":
		var progress struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(data, &progress); err == nil && progress.Message != "" {
			_, _ = fmt.Fprintf(stderr, "· %s\n", progress.Message)
		}
	case "batch.started":
		var start struct {
			BatchID string `json:"batch_id"`
			State   string `json:"state"`
		}
		if err := json.Unmarshal(data, &start); err == nil {
			payload.BatchID = start.BatchID
			payload.State = start.State
		}
	case "card.upsert":
		var card contract.FeedCard
		if err := json.Unmarshal(data, &card); err != nil {
			return false, err
		}
		payload.Cards = append(payload.Cards, card)
	case "error":
		var problem struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(data, &problem)
		return false, fmt.Errorf("%s", feedErrorText(problem.Error))
	case "batch.done":
		if err := payload.Validate(); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func chooseRefreshSource(reader *bufio.Reader, raw bool, stdout io.Writer, current discovery.RefreshSource) (discovery.RefreshSource, bool) {
	if current == "" {
		current = discovery.RefreshMixed
	}
	_, _ = fmt.Fprintln(stdout, "Refresh source:")
	_, _ = fmt.Fprintln(stdout, "  1 mixed     trends + subscriptions + search (default)")
	_, _ = fmt.Fprintln(stdout, "  2 trending  trends + search")
	_, _ = fmt.Fprintln(stdout, "  3 graph     subscriptions + search")
	_, _ = fmt.Fprintln(stdout, "  4 fresh     trends + subscriptions")
	_, _ = fmt.Fprintln(stdout, "  5 keywords  search only")
	_, _ = fmt.Fprintf(stdout, "Enter uses %s. Esc cancels.\n", current)
	key, err := readInteractiveKey(reader, raw, stdout)
	if err != nil && key == "" {
		return "", false
	}
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "enter", "":
		return current, true
	case "1":
		return discovery.RefreshMixed, true
	case "2":
		return discovery.RefreshTrending, true
	case "3":
		return discovery.RefreshGraph, true
	case "4":
		return discovery.RefreshFresh, true
	case "5":
		return discovery.RefreshKeywords, true
	case "esc", "q":
		return "", false
	default:
		_, _ = fmt.Fprintln(stdout, "choose 1, 2, 3, 4, 5 or Enter")
		return "", false
	}
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
