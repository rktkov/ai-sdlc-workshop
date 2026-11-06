package analysis

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/taste"
)

func validProfile() taste.ProfileSnapshot {
	return taste.ProfileSnapshot{
		UserID:        "owner",
		Note:          "prefer depth",
		RubricVersion: "v1",
		Axes: []taste.Axis{
			{Name: "depth", Weight: 0.7, PositiveExamples: []string{"long explanation"}, NegativeExamples: []string{"shallow"}},
			{Name: "novelty", Weight: 0.5, PositiveExamples: []string{"new angle"}, NegativeExamples: []string{"repeat"}},
			{Name: "clarity", Weight: 0.6, PositiveExamples: []string{"clear"}, NegativeExamples: []string{"vague"}},
		},
	}
}

func validJSON() string {
	return `{
		"topic":"ai-agents",
		"tldr":"A clear explanation of agent loops.",
		"axis_scores":[
			{"axis":"depth","score":0.8,"justification":"Explains the loop in detail."},
			{"axis":"novelty","score":0.4,"justification":"Mostly familiar material."},
			{"axis":"clarity","score":0.9,"justification":"Short and structured."}
		],
		"marketing_score":0.1,
		"ai_slop_score":0.2,
		"similarity":"new territory",
		"new_territory":true,
		"why_this_card":"It explains the trade-offs with a concrete example."
	}`
}

type fakeLLM struct {
	response string
	err      error
	called   bool
	system   string
	user     string
	schema   json.RawMessage
}

func (f *fakeLLM) Complete(_ context.Context, system, user string, schema json.RawMessage) (string, error) {
	f.called = true
	f.system = system
	f.user = user
	f.schema = schema
	return f.response, f.err
}

func TestValidateAnalyzed(t *testing.T) {
	analysis := Analysis{
		State: StateAnalyzed, RubricVersion: "v1", ModelID: "model",
		Topic: "ai-agents", TLDR: "tldr", Similarity: "new territory", WhyThisCard: "why",
		AxisScores: []AxisScore{
			{Axis: "depth", Score: 0.8, Justification: "good"},
			{Axis: "novelty", Score: 0.5, Justification: "ok"},
			{Axis: "clarity", Score: 0.9, Justification: "clear"},
		},
		MarketingScore: 0.1, AISlopScore: 0.2, NewTerritory: true,
		CompletedAt: time.Now().UTC(),
	}
	if err := analysis.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestValidateFailed(t *testing.T) {
	analysis := Analysis{
		State: StateFailed, Reason: ReasonNoTranscript,
		RubricVersion: "v1", ModelID: "model", CompletedAt: time.Now().UTC(),
	}
	if err := analysis.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestValidateUnhappy(t *testing.T) {
	valid := Analysis{
		State: StateAnalyzed, RubricVersion: "v1", ModelID: "model",
		Topic: "ai-agents", TLDR: "tldr", Similarity: "new territory", WhyThisCard: "why",
		AxisScores: []AxisScore{
			{Axis: "depth", Score: 0.8, Justification: "good"},
			{Axis: "novelty", Score: 0.5, Justification: "ok"},
		},
		MarketingScore: 0.1, AISlopScore: 0.2,
		CompletedAt: time.Now().UTC(),
	}
	tests := []struct {
		name string
		edit func(*Analysis)
	}{
		{name: "bad state", edit: func(a *Analysis) { a.State = "pending" }},
		{name: "failed reason", edit: func(a *Analysis) { a.State = StateFailed; a.Reason = "bad" }},
		{name: "failed with scores", edit: func(a *Analysis) { a.State = StateFailed; a.Reason = ReasonNoTranscript }},
		{name: "no rubric", edit: func(a *Analysis) { a.RubricVersion = "" }},
		{name: "no model", edit: func(a *Analysis) { a.ModelID = "" }},
		{name: "no topic", edit: func(a *Analysis) { a.Topic = "" }},
		{name: "bad topic", edit: func(a *Analysis) { a.Topic = "AI Agents!" }},
		{name: "no tldr", edit: func(a *Analysis) { a.TLDR = "" }},
		{name: "no scores", edit: func(a *Analysis) { a.AxisScores = nil }},
		{name: "bad score", edit: func(a *Analysis) { a.AxisScores[0].Score = 2 }},
		{name: "no justification", edit: func(a *Analysis) { a.AxisScores[0].Justification = "" }},
		{name: "duplicate axis", edit: func(a *Analysis) { a.AxisScores[1].Axis = "depth" }},
		{name: "bad marketing", edit: func(a *Analysis) { a.MarketingScore = -1 }},
		{name: "bad slop", edit: func(a *Analysis) { a.AISlopScore = 2 }},
		{name: "no similarity", edit: func(a *Analysis) { a.Similarity = "" }},
		{name: "no why", edit: func(a *Analysis) { a.WhyThisCard = "" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := valid
			a.AxisScores = append([]AxisScore(nil), valid.AxisScores...)
			tt.edit(&a)
			if err := a.Validate(); err == nil {
				t.Fatal("Validate = nil, want an error")
			}
		})
	}
}

func TestBuildPrompt(t *testing.T) {
	system, user := BuildPrompt(validProfile(), "the transcript")
	for _, want := range []string{"json", "transcript", "untrusted"} {
		if !strings.Contains(strings.ToLower(system), want) {
			t.Fatalf("system = %q, missing %q", system, want)
		}
	}
	for _, want := range []string{"depth", "long explanation", "shallow", "prefer depth", "the transcript"} {
		if !strings.Contains(user, want) {
			t.Fatalf("user = %q, missing %q", user, want)
		}
	}
}

func TestBuildSchema(t *testing.T) {
	schema, err := BuildSchema(validProfile())
	if err != nil {
		t.Fatalf("BuildSchema: %v", err)
	}
	var value map[string]any
	if err := json.Unmarshal(schema, &value); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	required, ok := value["required"].([]any)
	if !ok || len(required) == 0 {
		t.Fatalf("required = %v", value["required"])
	}
	properties, ok := value["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties = %v", value["properties"])
	}
	if _, ok := properties["axis_scores"]; !ok {
		t.Fatal("axis_scores is missing")
	}
	if !strings.Contains(string(schema), `"depth"`) {
		t.Fatalf("schema does not name the axes: %s", schema)
	}
}

func TestAnalyzeHappy(t *testing.T) {
	now := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	llm := &fakeLLM{response: validJSON()}
	analyzer := &Analyzer{LLM: llm, Profile: validProfile(), ModelID: "model-1", Now: func() time.Time { return now }}
	analysis, err := analyzer.Analyze(context.Background(), "transcript")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if analysis.State != StateAnalyzed {
		t.Fatalf("state = %q, reason = %q", analysis.State, analysis.Reason)
	}
	if analysis.Topic != "ai-agents" || analysis.RubricVersion != "v1" || analysis.ModelID != "model-1" {
		t.Fatalf("analysis = %+v", analysis)
	}
	if len(analysis.AxisScores) != 3 || analysis.AxisScores[0].Axis != "depth" {
		t.Fatalf("scores = %+v", analysis.AxisScores)
	}
	if !analysis.CompletedAt.Equal(now) {
		t.Fatalf("completed = %v", analysis.CompletedAt)
	}
	if !llm.called || len(llm.schema) == 0 {
		t.Fatal("LLM was not called with a schema")
	}
}

func TestAnalyzeNoTranscript(t *testing.T) {
	llm := &fakeLLM{response: validJSON()}
	analyzer := &Analyzer{LLM: llm, Profile: validProfile(), ModelID: "model"}
	analysis, err := analyzer.Analyze(context.Background(), "  ")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if analysis.State != StateFailed || analysis.Reason != ReasonNoTranscript {
		t.Fatalf("analysis = %+v", analysis)
	}
	if llm.called {
		t.Fatal("LLM was called without a transcript")
	}
}

func TestAnalyzeInvalidOutput(t *testing.T) {
	tests := []string{
		`not json`,
		`{"topic":"ai-agents"}`,
		strings.Replace(validJSON(), `"axis":"depth"`, `"axis":"unknown"`, 1),
		strings.Replace(validJSON(), `"score":0.8`, `"score":2`, 1),
		strings.Replace(validJSON(), `"marketing_score":0.1`, `"marketing_score":-1`, 1),
		strings.Replace(validJSON(), `"new_territory":true`, ``, 1),
	}
	for _, response := range tests {
		analyzer := &Analyzer{LLM: &fakeLLM{response: response}, Profile: validProfile(), ModelID: "model"}
		analysis, err := analyzer.Analyze(context.Background(), "transcript")
		if err != nil {
			t.Fatalf("Analyze: %v", err)
		}
		if analysis.State != StateFailed || analysis.Reason != ReasonInvalidModelOutput {
			t.Fatalf("response %q -> %+v", response, analysis)
		}
	}
}

func TestAnalyzeTimeout(t *testing.T) {
	analyzer := &Analyzer{LLM: &fakeLLM{err: ErrLLMTimeout}, Profile: validProfile(), ModelID: "model"}
	analysis, err := analyzer.Analyze(context.Background(), "transcript")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if analysis.State != StateFailed || analysis.Reason != ReasonAnalysisSlow {
		t.Fatalf("analysis = %+v", analysis)
	}
}

func TestAnalyzeLLMError(t *testing.T) {
	analyzer := &Analyzer{LLM: &fakeLLM{err: errors.New("down")}, Profile: validProfile(), ModelID: "model"}
	analysis, err := analyzer.Analyze(context.Background(), "transcript")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if analysis.State != StateFailed || analysis.Reason != ReasonAnalysisSlow {
		t.Fatalf("analysis = %+v", analysis)
	}
}

func TestAnalyzeNoAxes(t *testing.T) {
	analyzer := &Analyzer{LLM: &fakeLLM{response: validJSON()}, Profile: taste.ProfileSnapshot{}, ModelID: "model"}
	if _, err := analyzer.Analyze(context.Background(), "transcript"); err == nil {
		t.Fatal("Analyze(no axes) = nil, want an error")
	}
}
