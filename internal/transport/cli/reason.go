package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/rktkov/ytfeed/internal/domain/taste"
)

// ErrInvalidReasonKey means the key is not part of the picker.
var ErrInvalidReasonKey = errors.New("invalid reason key")

// ReasonPicker maps a key to a rejection reason.
type ReasonPicker struct {
	Last taste.Reason
}

// Key returns the reason for a key. Enter accepts the last reason. Esc records other.
func (p *ReasonPicker) Key(key string) (taste.Reason, error) {
	var reason taste.Reason
	switch strings.TrimSpace(strings.ToLower(key)) {
	case "1":
		reason = taste.ReasonMarketing
	case "2":
		reason = taste.ReasonAISlop
	case "3":
		reason = taste.ReasonValueMisaligned
	case "4", "esc":
		reason = taste.ReasonOther
	case "", "enter":
		if p.Last == "" {
			reason = taste.ReasonOther
		} else {
			reason = p.Last
		}
	default:
		return "", ErrInvalidReasonKey
	}
	p.Last = reason
	return reason, nil
}

// Render writes the picker and the scope for each option.
func (p *ReasonPicker) Render(w io.Writer, channel, topic string) error {
	last := p.Last
	if last == "" {
		last = taste.ReasonOther
	}
	var b strings.Builder
	b.WriteString("Why reject this card?\n")
	b.WriteString(fmt.Sprintf("  1 marketing        → mute channel: %s\n", channel))
	b.WriteString(fmt.Sprintf("  2 ai-slop          → mute topic: %s\n", topic))
	b.WriteString(fmt.Sprintf("  3 value-misaligned → mute channel: %s\n", channel))
	b.WriteString(fmt.Sprintf("  4 other            → mute topic: %s\n", topic))
	b.WriteString(fmt.Sprintf("Enter accepts %s. Esc records other.\n", last))
	_, err := io.WriteString(w, b.String())
	return err
}
