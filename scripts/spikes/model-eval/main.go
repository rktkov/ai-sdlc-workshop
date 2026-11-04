// Command model-eval runs the S-01 model comparison against an OpenAI-compatible server.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

type axis struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type rubric struct {
	Axes []axis `json:"axes"`
}

type fixture struct {
	ID            string             `json:"id"`
	Kind          string             `json:"kind"`
	WorthWatching bool               `json:"worth_watching"`
	Labels        map[string]float64 `json:"labels"`
	Transcript    string             `json:"transcript"`
}

type axisScore struct {
	Axis          string  `json:"axis"`
	Score         float64 `json:"score"`
	Justification string  `json:"justification"`
}

type result struct {
	Topic          string      `json:"topic"`
	TLDR           string      `json:"tldr"`
	AxisScores     []axisScore `json:"axis_scores"`
	MarketingScore float64     `json:"marketing_score"`
	AISlopScore    float64     `json:"ai_slop_score"`
	Similarity     string      `json:"similarity"`
	NewTerritory   bool        `json:"new_territory"`
	WhyThisCard    string      `json:"why_this_card"`
}

type record struct {
	Model      string      `json:"model"`
	Fixture    string      `json:"fixture"`
	Kind       string      `json:"kind"`
	Run        int         `json:"run"`
	LatencyMS  int64       `json:"latency_ms"`
	JSONMode   bool        `json:"json_mode"`
	Valid      bool        `json:"valid"`
	Error      string      `json:"error,omitempty"`
	Scores     []axisScore `json:"scores,omitempty"`
	Worth      bool        `json:"worth,omitempty"`
	LabelWorth bool        `json:"label_worth,omitempty"`
	MAE        float64     `json:"mae,omitempty"`
	Topic      string      `json:"topic,omitempty"`
	Expected   string      `json:"expected_topic,omitempty"`
}

func main() {
	baseURL := flag.String("base-url", "http://127.0.0.1:8081", "OpenAI-compatible base URL")
	model := flag.String("model", "", "model id")
	fixturesPath := flag.String("fixtures", "fixtures.json", "fixtures file")
	rubricPath := flag.String("rubric", "rubric.json", "rubric file")
	outPath := flag.String("out", "results.jsonl", "output JSONL")
	repeat := flag.Int("repeat", 1, "runs per fixture")
	timeout := flag.Duration("timeout", 120*time.Second, "request timeout")
	flag.Parse()

	if *model == "" {
		fmt.Fprintln(os.Stderr, "model is required")
		os.Exit(2)
	}

	fixtures, err := loadFixtures(*fixturesPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	rb, err := loadRubric(*rubricPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	out, err := os.OpenFile(*outPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() { _ = out.Close() }()

	client := &http.Client{Timeout: *timeout}
	var records []record
	for _, f := range fixtures {
		if f.Kind == "empty" {
			continue
		}
		for run := 1; run <= *repeat; run++ {
			rec := evaluate(client, *baseURL, *model, rb, f, run)
			records = append(records, rec)
			if err := json.NewEncoder(out).Encode(rec); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Printf("%s %s run=%d valid=%v latency=%dms %s\n", rec.Model, rec.Fixture, rec.Run, rec.Valid, rec.LatencyMS, rec.Error)
		}
	}
	summarize(records)
}

func loadFixtures(path string) ([]fixture, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read fixtures: %w", err)
	}
	var fixtures []fixture
	if err := json.Unmarshal(raw, &fixtures); err != nil {
		return nil, fmt.Errorf("decode fixtures: %w", err)
	}
	return fixtures, nil
}

func loadRubric(path string) (rubric, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return rubric{}, fmt.Errorf("read rubric: %w", err)
	}
	var rb rubric
	if err := json.Unmarshal(raw, &rb); err != nil {
		return rubric{}, fmt.Errorf("decode rubric: %w", err)
	}
	return rb, nil
}

func evaluate(client *http.Client, baseURL, model string, rb rubric, f fixture, run int) record {
	rec := record{Model: model, Fixture: f.ID, Kind: f.Kind, Run: run, LabelWorth: f.WorthWatching}
	start := time.Now()
	content, usedJSON, err := call(client, baseURL, model, rb, f)
	rec.LatencyMS = time.Since(start).Milliseconds()
	rec.JSONMode = usedJSON
	if err != nil {
		rec.Error = err.Error()
		return rec
	}
	var parsed result
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		rec.Error = "invalid json: " + err.Error()
		return rec
	}
	if err := validate(parsed, rb); err != nil {
		rec.Error = err.Error()
		return rec
	}
	rec.Valid = true
	rec.Scores = parsed.AxisScores
	rec.Topic = parsed.Topic
	rec.Worth = average(parsed.AxisScores) >= 0.6
	rec.MAE = mae(parsed.AxisScores, f.Labels)
	return rec
}

func call(client *http.Client, baseURL, model string, rb rubric, f fixture) (string, bool, error) {
	body := map[string]any{
		"model":       model,
		"temperature": 0.1,
		"max_tokens":  800,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt(rb)},
			{"role": "user", "content": userPrompt(rb, f)},
		},
	}
	body["response_format"] = map[string]any{
		"type": "json_schema",
		"json_schema": map[string]any{
			"name":   "analysis",
			"strict": true,
			"schema": responseSchema(rb),
		},
	}
	content, err := post(client, baseURL+"/chat/completions", body)
	if err == nil {
		return content, true, nil
	}
	body["response_format"] = map[string]string{"type": "json_object"}
	content, err = post(client, baseURL+"/chat/completions", body)
	if err == nil {
		return content, true, nil
	}
	delete(body, "response_format")
	content, err = post(client, baseURL+"/chat/completions", body)
	return content, false, err
}

func post(client *http.Client, url string, body map[string]any) (string, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	resp, err := client.Post(url, "application/json", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var envelope struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if len(envelope.Choices) == 0 {
		return "", fmt.Errorf("no choices")
	}
	return envelope.Choices[0].Message.Content, nil
}

func systemPrompt(rb rubric) string {
	return "You are a strict content judge. Return one JSON object only. Do not write prose outside the JSON. " + schemaDescription(rb)
}

func userPrompt(rb rubric, f fixture) string {
	var b strings.Builder
	b.WriteString("Rubric axes:\n")
	for _, a := range rb.Axes {
		b.WriteString("- " + a.Name + ": " + a.Description + "\n")
	}
	b.WriteString("\nReturn this schema: ")
	b.WriteString(schemaDescription(rb))
	b.WriteString("\nTranscript:\n")
	if f.Kind == "long" {
		b.WriteString(strings.Repeat(f.Transcript+"\n", 40))
	} else {
		b.WriteString(f.Transcript)
	}
	return b.String()
}

func schemaDescription(rb rubric) string {
	names := make([]string, 0, len(rb.Axes))
	for _, a := range rb.Axes {
		names = append(names, `"`+a.Name+`"`)
	}
	return `{"topic":string,"tldr":string,"axis_scores":[{"axis":one of [` + strings.Join(names, ",") + `],"score":number 0..1,"justification":string}],"marketing_score":number 0..1,"ai_slop_score":number 0..1,"similarity":string,"new_territory":boolean,"why_this_card":string}`
}

func responseSchema(rb rubric) map[string]any {
	names := make([]string, 0, len(rb.Axes))
	for _, a := range rb.Axes {
		names = append(names, a.Name)
	}
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             []string{"topic", "tldr", "axis_scores", "marketing_score", "ai_slop_score", "similarity", "new_territory", "why_this_card"},
		"properties": map[string]any{
			"topic": map[string]any{"type": "string"},
			"tldr":  map[string]any{"type": "string"},
			"axis_scores": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"required":             []string{"axis", "score", "justification"},
					"properties": map[string]any{
						"axis":          map[string]any{"type": "string", "enum": names},
						"score":         map[string]any{"type": "number", "minimum": 0, "maximum": 1},
						"justification": map[string]any{"type": "string"},
					},
				},
			},
			"marketing_score": map[string]any{"type": "number", "minimum": 0, "maximum": 1},
			"ai_slop_score":   map[string]any{"type": "number", "minimum": 0, "maximum": 1},
			"similarity":      map[string]any{"type": "string"},
			"new_territory":   map[string]any{"type": "boolean"},
			"why_this_card":   map[string]any{"type": "string"},
		},
	}
}

func validate(r result, rb rubric) error {
	if r.Topic == "" || r.TLDR == "" || r.WhyThisCard == "" {
		return fmt.Errorf("missing required string field")
	}
	if len(r.AxisScores) != len(rb.Axes) {
		return fmt.Errorf("axis count %d, want %d", len(r.AxisScores), len(rb.Axes))
	}
	for _, s := range r.AxisScores {
		if s.Score < 0 || s.Score > 1 {
			return fmt.Errorf("axis %s score out of range", s.Axis)
		}
		if s.Justification == "" {
			return fmt.Errorf("axis %s has no justification", s.Axis)
		}
	}
	if r.MarketingScore < 0 || r.MarketingScore > 1 || r.AISlopScore < 0 || r.AISlopScore > 1 {
		return fmt.Errorf("score out of range")
	}
	return nil
}

func average(scores []axisScore) float64 {
	if len(scores) == 0 {
		return 0
	}
	var sum float64
	for _, s := range scores {
		sum += s.Score
	}
	return sum / float64(len(scores))
}

func mae(scores []axisScore, labels map[string]float64) float64 {
	if len(labels) == 0 {
		return 0
	}
	var sum float64
	var count int
	for _, s := range scores {
		label, ok := labels[s.Axis]
		if !ok {
			continue
		}
		diff := s.Score - label
		if diff < 0 {
			diff = -diff
		}
		sum += diff
		count++
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

func summarize(records []record) {
	if len(records) == 0 {
		return
	}
	var valid int
	var latencies []int64
	var maes []float64
	var worthOK, worthTotal int
	for _, r := range records {
		if r.Valid {
			valid++
		}
		latencies = append(latencies, r.LatencyMS)
		if r.MAE > 0 {
			maes = append(maes, r.MAE)
		}
		if r.Kind == "core" {
			worthTotal++
			if r.Worth == r.LabelWorth {
				worthOK++
			}
		}
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	var maeSum float64
	for _, m := range maes {
		maeSum += m
	}
	fmt.Printf("SUMMARY model=%s calls=%d valid=%d schema=%.1f%% p50=%dms p95=%dms mae=%.3f worth=%d/%d\n",
		records[0].Model, len(records), valid, 100*float64(valid)/float64(len(records)),
		latencies[len(latencies)/2], latencies[(len(latencies)*95)/100],
		maeSum/float64(len(maes)), worthOK, worthTotal)
}
