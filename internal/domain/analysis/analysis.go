package analysis

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// State is the terminal analysis state.
type State string

const (
	StateAnalyzed State = "analyzed"
	StateFailed   State = "failed"
)

// Reason is the failure classification.
type Reason string

const (
	ReasonNoTranscript       Reason = "no_transcript"
	ReasonAnalysisSlow       Reason = "analysis_slow"
	ReasonInvalidModelOutput Reason = "invalid_model_output"
)

// ErrLLMTimeout means the model request exceeded its deadline.
var ErrLLMTimeout = errors.New("llm timeout")

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// AxisScore is one scored profile axis.
type AxisScore struct {
	Axis          string
	Score         float64
	Justification string
}

// Analysis is one terminal judgment for a video and rubric version.
type Analysis struct {
	ID              string
	UserID          string
	VideoID         string
	ProfileVersion  int
	RubricVersion   string
	ModelID         string
	State           State
	Reason          Reason
	Topic           string
	TLDR            string
	AxisScores      []AxisScore
	MarketingScore  float64
	AISlopScore     float64
	Similarity      string
	NewTerritory    bool
	WhyThisCard     string
	TranscriptHash  string
	TranscriptBytes int64
	CreatedAt       time.Time
	CompletedAt     time.Time
}

// Validate checks the terminal result invariants.
func (a Analysis) Validate() error {
	if strings.TrimSpace(a.RubricVersion) == "" {
		return errors.New("analysis has no rubric version")
	}
	if strings.TrimSpace(a.ModelID) == "" {
		return errors.New("analysis has no model id")
	}
	if a.CompletedAt.IsZero() {
		return errors.New("analysis has no completion time")
	}

	switch a.State {
	case StateAnalyzed:
		return a.validateAnalyzed()
	case StateFailed:
		return a.validateFailed()
	default:
		return fmt.Errorf("analysis has an invalid state %q", a.State)
	}
}

func (a Analysis) validateAnalyzed() error {
	if a.Reason != "" {
		return errors.New("analyzed result has a failure reason")
	}
	if len(a.AxisScores) == 0 {
		return errors.New("analyzed result has no axis scores")
	}
	if !validSlug(a.Topic) {
		return fmt.Errorf("analysis has an invalid topic %q", a.Topic)
	}
	if length := len(strings.TrimSpace(a.TLDR)); length < 1 || length > 500 {
		return errors.New("analysis tldr must be 1 to 500 characters")
	}
	if length := len(strings.TrimSpace(a.Similarity)); length < 1 || length > 200 {
		return errors.New("analysis similarity must be 1 to 200 characters")
	}
	if length := len(strings.TrimSpace(a.WhyThisCard)); length < 1 || length > 500 {
		return errors.New("analysis why-this-card must be 1 to 500 characters")
	}
	if a.MarketingScore < 0 || a.MarketingScore > 1 {
		return errors.New("analysis marketing score is outside [0,1]")
	}
	if a.AISlopScore < 0 || a.AISlopScore > 1 {
		return errors.New("analysis ai-slop score is outside [0,1]")
	}

	seen := map[string]bool{}
	for _, score := range a.AxisScores {
		if !validSlug(score.Axis) {
			return fmt.Errorf("axis score has an invalid axis %q", score.Axis)
		}
		if seen[score.Axis] {
			return fmt.Errorf("axis %q is scored twice", score.Axis)
		}
		seen[score.Axis] = true
		if score.Score < 0 || score.Score > 1 {
			return fmt.Errorf("axis %q score is outside [0,1]", score.Axis)
		}
		if length := len(strings.TrimSpace(score.Justification)); length < 1 || length > 300 {
			return fmt.Errorf("axis %q justification must be 1 to 300 characters", score.Axis)
		}
	}
	return nil
}

func (a Analysis) validateFailed() error {
	switch a.Reason {
	case ReasonNoTranscript, ReasonAnalysisSlow, ReasonInvalidModelOutput:
	default:
		return fmt.Errorf("analysis has an invalid failure reason %q", a.Reason)
	}
	if len(a.AxisScores) != 0 || a.Topic != "" || a.TLDR != "" || a.Similarity != "" || a.WhyThisCard != "" {
		return errors.New("failed analysis carries result fields")
	}
	return nil
}

func validSlug(value string) bool {
	if len(value) < 1 || len(value) > 64 {
		return false
	}
	return slugPattern.MatchString(value)
}

func normalizeTopic(value string) string {
	topic := strings.ToLower(strings.TrimSpace(value))
	topic = strings.ReplaceAll(topic, "_", "-")
	topic = strings.ReplaceAll(topic, " ", "-")
	for strings.Contains(topic, "--") {
		topic = strings.ReplaceAll(topic, "--", "-")
	}
	return strings.Trim(topic, "-")
}
