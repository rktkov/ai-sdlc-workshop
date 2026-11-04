package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/rktkov/ytfeed/internal/domain/taste"
)

func TestReasonPickerHappy(t *testing.T) {
	tests := []struct {
		key  string
		want taste.Reason
	}{
		{key: "1", want: taste.ReasonMarketing},
		{key: "2", want: taste.ReasonAISlop},
		{key: "3", want: taste.ReasonValueMisaligned},
		{key: "4", want: taste.ReasonOther},
		{key: "esc", want: taste.ReasonOther},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			picker := &ReasonPicker{}
			got, err := picker.Key(tt.key)
			if err != nil {
				t.Fatalf("Key: %v", err)
			}
			if got != tt.want {
				t.Fatalf("Key(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}

	picker := &ReasonPicker{Last: taste.ReasonAISlop}
	got, err := picker.Key("enter")
	if err != nil {
		t.Fatalf("Key(enter): %v", err)
	}
	if got != taste.ReasonAISlop {
		t.Fatalf("Key(enter) = %q", got)
	}
}

func TestReasonPickerUnhappy(t *testing.T) {
	picker := &ReasonPicker{Last: taste.ReasonMarketing}
	if _, err := picker.Key("9"); !errors.Is(err, ErrInvalidReasonKey) {
		t.Fatalf("Key(9) = %v, want ErrInvalidReasonKey", err)
	}
	if picker.Last != taste.ReasonMarketing {
		t.Fatalf("Last changed to %q", picker.Last)
	}
}

func TestReasonPickerRender(t *testing.T) {
	var buf bytes.Buffer
	picker := &ReasonPicker{Last: taste.ReasonMarketing}
	if err := picker.Render(&buf, "channel-1", "ai-agents"); err != nil {
		t.Fatalf("Render: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"mute channel: channel-1", "mute topic: ai-agents", "Enter accepts marketing"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output is missing %q:\n%s", want, out)
		}
	}
}
