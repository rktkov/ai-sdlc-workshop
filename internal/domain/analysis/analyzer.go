package analysis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/taste"
)

// LLM returns one raw JSON response for a prompt and schema.
type LLM interface {
	Complete(ctx context.Context, system, user string, schema json.RawMessage) (string, error)
}

// Analyzer validates one model response against the profile.
type Analyzer struct {
	LLM     LLM
	Profile taste.ProfileSnapshot
	ModelID string
	Now     func() time.Time
}

type modelResult struct {
	Topic      string `json:"topic"`
	TLDR       string `json:"tldr"`
	AxisScores []struct {
		Axis          string  `json:"axis"`
		Score         float64 `json:"score"`
		Justification string  `json:"justification"`
	} `json:"axis_scores"`
	MarketingScore *float64 `json:"marketing_score"`
	AISlopScore    *float64 `json:"ai_slop_score"`
	Similarity     string   `json:"similarity"`
	NewTerritory   *bool    `json:"new_territory"`
	WhyThisCard    string   `json:"why_this_card"`
}

// Analyze returns a terminal result. Validation failures are explicit states.
func (a *Analyzer) Analyze(ctx context.Context, transcript string) (Analysis, error) {
	if a.LLM == nil {
		return Analysis{}, errors.New("analysis llm is nil")
	}
	now := time.Now().UTC()
	if a.Now != nil {
		now = a.Now().UTC()
	}
	if strings.TrimSpace(transcript) == "" {
		return a.failed(ReasonNoTranscript, now), nil
	}
	schema, err := BuildSchema(a.Profile)
	if err != nil {
		return Analysis{}, err
	}
	system, user := BuildPrompt(a.Profile, transcript)
	raw, err := a.LLM.Complete(ctx, system, user, schema)
	if err != nil {
		return a.failed(ReasonAnalysisSlow, now), nil
	}

	var result modelResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return a.failed(ReasonInvalidModelOutput, now), nil
	}
	analysis, err := result.toAnalysis(a.Profile, a.ModelID, now)
	if err != nil {
		return a.failed(ReasonInvalidModelOutput, now), nil
	}
	return analysis, nil
}

func (a *Analyzer) failed(reason Reason, now time.Time) Analysis {
	return Analysis{
		State:         StateFailed,
		Reason:        reason,
		RubricVersion: a.Profile.RubricVersion,
		ModelID:       a.ModelID,
		CreatedAt:     now,
		CompletedAt:   now,
	}
}

func (r modelResult) toAnalysis(profile taste.ProfileSnapshot, modelID string, now time.Time) (Analysis, error) {
	if r.MarketingScore == nil || r.AISlopScore == nil || r.NewTerritory == nil {
		return Analysis{}, errors.New("model result is missing a required field")
	}
	provided := make(map[string]AxisScore, len(r.AxisScores))
	for _, score := range r.AxisScores {
		axis := strings.ToLower(strings.TrimSpace(score.Axis))
		if _, ok := provided[axis]; ok {
			return Analysis{}, fmt.Errorf("axis %q is scored twice", axis)
		}
		provided[axis] = AxisScore{
			Axis:          axis,
			Score:         score.Score,
			Justification: strings.TrimSpace(score.Justification),
		}
	}
	ordered := make([]AxisScore, 0, len(profile.Axes))
	for _, axis := range profile.Axes {
		score, ok := provided[axis.Name]
		if !ok {
			return Analysis{}, fmt.Errorf("axis %q is missing", axis.Name)
		}
		ordered = append(ordered, score)
		delete(provided, axis.Name)
	}
	if len(provided) != 0 {
		return Analysis{}, errors.New("model returned an unknown axis")
	}

	analysis := Analysis{
		ProfileVersion: 1,
		RubricVersion:  profile.RubricVersion,
		ModelID:        modelID,
		State:          StateAnalyzed,
		Topic:          normalizeTopic(r.Topic),
		TLDR:           strings.TrimSpace(r.TLDR),
		AxisScores:     ordered,
		MarketingScore: *r.MarketingScore,
		AISlopScore:    *r.AISlopScore,
		Similarity:     strings.TrimSpace(r.Similarity),
		NewTerritory:   *r.NewTerritory,
		WhyThisCard:    strings.TrimSpace(r.WhyThisCard),
		CreatedAt:      now,
		CompletedAt:    now,
	}
	if err := analysis.Validate(); err != nil {
		return Analysis{}, err
	}
	return analysis, nil
}
