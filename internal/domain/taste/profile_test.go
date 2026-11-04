package taste

import "testing"

func validProfile() Profile {
	return Profile{
		UserID:        "owner",
		RubricVersion: "v1",
		Axes: []Axis{
			{Name: "depth", Weight: 0.7, PositiveExamples: []string{"long"}, NegativeExamples: []string{"shallow"}},
			{Name: "novelty", Weight: 0.5, PositiveExamples: []string{"new"}, NegativeExamples: []string{"repeat"}},
			{Name: "clarity", Weight: 0.6, PositiveExamples: []string{"clear"}, NegativeExamples: []string{"vague"}},
		},
		Interests: []string{"ai-agents", "local-models"},
	}
}

func TestProfileValidateHappy(t *testing.T) {
	if err := validProfile().Validate(); err != nil {
		t.Fatalf("Validate() = %v", err)
	}
}

func TestProfileValidateUnhappy(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Profile)
	}{
		{name: "no user", edit: func(p *Profile) { p.UserID = "" }},
		{name: "no rubric", edit: func(p *Profile) { p.RubricVersion = "" }},
		{name: "two axes", edit: func(p *Profile) { p.Axes = p.Axes[:2] }},
		{name: "duplicate axis", edit: func(p *Profile) { p.Axes[1].Name = "Depth" }},
		{name: "weight low", edit: func(p *Profile) { p.Axes[0].Weight = -0.1 }},
		{name: "weight high", edit: func(p *Profile) { p.Axes[0].Weight = 1.1 }},
		{name: "no positive", edit: func(p *Profile) { p.Axes[0].PositiveExamples = nil }},
		{name: "no negative", edit: func(p *Profile) { p.Axes[0].NegativeExamples = nil }},
		{name: "empty example", edit: func(p *Profile) { p.Axes[0].PositiveExamples = []string{" "} }},
		{name: "no interests", edit: func(p *Profile) { p.Interests = nil }},
		{name: "eleven interests", edit: func(p *Profile) {
			p.Interests = []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}
		}},
		{name: "duplicate interest", edit: func(p *Profile) { p.Interests = []string{"AI Agents", "ai-agents"} }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := validProfile()
			tt.edit(&p)
			if err := p.Validate(); err == nil {
				t.Fatal("Validate() = nil, want an error")
			}
		})
	}
}
