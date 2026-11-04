package taste

import (
	"errors"
	"fmt"
	"strings"
)

// ErrProfileNotFound means no profile exists for the user.
var (
	ErrProfileNotFound    = errors.New("taste profile not found")
	ErrColdPeriodNotFound = errors.New("cold period not found")
)

// Axis is one evaluative dimension of a profile.
type Axis struct {
	Name             string
	Weight           float64
	PositiveExamples []string
	NegativeExamples []string
}

// Profile is the durable taste profile.
type Profile struct {
	UserID        string
	Axes          []Axis
	Interests     []string
	Note          string
	RubricVersion string
}

// Validate checks the profile invariants from F4 and F3.
func (p Profile) Validate() error {
	if strings.TrimSpace(p.UserID) == "" {
		return fmt.Errorf("profile has no user id")
	}
	if strings.TrimSpace(p.RubricVersion) == "" {
		return fmt.Errorf("profile has no rubric version")
	}
	if len(p.Axes) < 3 {
		return fmt.Errorf("profile needs at least 3 axes")
	}

	seen := map[string]bool{}
	for _, axis := range p.Axes {
		name := normalizeName(axis.Name)
		if name == "" {
			return fmt.Errorf("axis name must not be empty")
		}
		if seen[name] {
			return fmt.Errorf("axis %q is duplicated", axis.Name)
		}
		seen[name] = true
		if axis.Weight < 0 || axis.Weight > 1 {
			return fmt.Errorf("axis %q weight is outside [0,1]", axis.Name)
		}
		if len(axis.PositiveExamples) == 0 {
			return fmt.Errorf("axis %q has no positive example", axis.Name)
		}
		if len(axis.NegativeExamples) == 0 {
			return fmt.Errorf("axis %q has no negative example", axis.Name)
		}
		for _, example := range append(append([]string{}, axis.PositiveExamples...), axis.NegativeExamples...) {
			if strings.TrimSpace(example) == "" {
				return fmt.Errorf("axis %q has an empty example", axis.Name)
			}
		}
	}

	if _, err := NormalizeInterests(p.Interests); err != nil {
		return err
	}
	return nil
}

// NormalizeInterests validates and normalizes 1 to 10 interest slugs.
func NormalizeInterests(values []string) ([]string, error) {
	if len(values) < 1 || len(values) > 10 {
		return nil, fmt.Errorf("profile needs 1 to 10 interests")
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		slug := normalizeInterest(value)
		if slug == "" {
			return nil, fmt.Errorf("interest must not be empty")
		}
		if len(slug) > 64 {
			return nil, fmt.Errorf("interest %q is longer than 64 characters", value)
		}
		if seen[slug] {
			return nil, fmt.Errorf("interest %q is duplicated", value)
		}
		seen[slug] = true
		out = append(out, slug)
	}
	return out, nil
}

func normalizeName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeInterest(value string) string {
	slug := strings.ToLower(strings.TrimSpace(value))
	slug = strings.ReplaceAll(slug, "_", "-")
	slug = strings.Join(strings.Fields(slug), "-")
	return slug
}
