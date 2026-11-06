package feed

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/taste"
)

// Gesture is one feed action.
type Gesture string

const (
	GestureLeft  Gesture = "left"
	GestureRight Gesture = "right"
	GestureSave  Gesture = "save"
	GestureDown  Gesture = "down"
	GestureUp    Gesture = "up"
)

// GestureCommand is one user action on one card.
type GestureCommand struct {
	GestureID  string
	CardID     string
	Gesture    Gesture
	Reason     taste.Reason
	ScoredAxes []taste.AxisScore
}

// GestureResult is the applied action and the updated cursor.
type GestureResult struct {
	Applied   bool
	Duplicate bool
	Decision  CardDecision
	Session   Session
	Card      Card
}

// TasteRecorder records a signal in taste.
type TasteRecorder interface {
	Record(ctx context.Context, cmd taste.Command) (taste.Result, error)
}

// GestureService applies gestures to a session and a batch.
type GestureService struct {
	Sessions SessionStore
	Taste    TasteRecorder
	Now      func() time.Time
}

// Apply records or applies one gesture. A duplicate gesture is idempotent.
func (s *GestureService) Apply(ctx context.Context, session Session, batch *Batch, cmd GestureCommand) (GestureResult, error) {
	if s.Sessions == nil {
		return GestureResult{}, errors.New("session store is nil")
	}
	if batch == nil {
		return GestureResult{}, errors.New("batch is nil")
	}
	if session.State != SessionOpen {
		return GestureResult{}, ErrSessionClosed
	}
	card, index := findCard(batch, cmd.CardID)
	if index == -1 {
		return GestureResult{}, fmt.Errorf("card %q is not in the batch", cmd.CardID)
	}
	if cmd.GestureID == "" {
		return GestureResult{}, errors.New("gesture has no id")
	}
	if session.LastGestureID == cmd.GestureID {
		return GestureResult{Applied: true, Duplicate: true, Decision: card.Decision, Session: session, Card: card}, nil
	}

	switch cmd.Gesture {
	case GestureLeft, GestureRight, GestureSave:
		if s.Taste == nil {
			return GestureResult{}, errors.New("taste recorder is nil")
		}
		if cmd.Gesture == GestureLeft && !validRejectionReason(cmd.Reason) {
			return GestureResult{}, errors.New("left gesture has no valid reason")
		}
		if cmd.Gesture != GestureLeft && cmd.Reason != "" {
			return GestureResult{}, errors.New("non-left gesture has a reason")
		}
		scoredAxes := cmd.ScoredAxes
		if len(scoredAxes) == 0 {
			scoredAxes = card.ScoredAxes
		}
		topic := strings.TrimSpace(card.Topic)
		if topic == "" {
			topic = "unanalyzed"
		}
		result, err := s.Taste.Record(ctx, taste.Command{
			GestureID:  cmd.GestureID,
			UserID:     session.UserID,
			CardID:     card.CardID,
			VideoID:    card.VideoID,
			ChannelID:  card.ChannelID,
			Topic:      topic,
			Gesture:    tasteGesture(cmd.Gesture),
			Reason:     cmd.Reason,
			ScoredAxes: scoredAxes,
		})
		if err != nil {
			return GestureResult{}, err
		}
		if !result.Applied {
			return GestureResult{}, errors.New("taste did not apply the gesture")
		}
		batch.Cards[index].Decision = decisionFor(cmd.Gesture)
	case GestureDown:
	case GestureUp:
	default:
		return GestureResult{}, fmt.Errorf("unknown gesture %q", cmd.Gesture)
	}

	session.CurrentCardID = cursorAfter(batch, session.CurrentCardID, cmd.Gesture)
	if positionOf(batch, session.CurrentCardID) > positionOf(batch, session.FurthestCardID) {
		session.FurthestCardID = session.CurrentCardID
	}
	session.LastGestureID = cmd.GestureID
	session.LastActivityAt = s.now()
	updated, err := s.Sessions.Save(ctx, session, session.Version)
	if err != nil {
		return GestureResult{}, err
	}
	updatedCard, _ := findCard(batch, session.CurrentCardID)
	return GestureResult{
		Applied: true, Decision: batch.Cards[index].Decision,
		Session: updated, Card: updatedCard,
	}, nil
}

func findCard(batch *Batch, cardID string) (Card, int) {
	for i, card := range batch.Cards {
		if card.CardID == cardID {
			return card, i
		}
	}
	return Card{}, -1
}

func positionOf(batch *Batch, cardID string) int {
	for _, card := range batch.Cards {
		if card.CardID == cardID {
			return card.Position
		}
	}
	return -1
}

func cursorAfter(batch *Batch, currentCardID string, gesture Gesture) string {
	current := positionOf(batch, currentCardID)
	if gesture == GestureSave || current < 0 {
		return currentCardID
	}
	if gesture == GestureUp {
		for position := current - 1; position >= 0; position-- {
			for _, card := range batch.Cards {
				if card.Position == position && card.Decision != DecisionRejected {
					return card.CardID
				}
			}
		}
		return currentCardID
	}
	for position := current + 1; position < len(batch.Cards); position++ {
		for _, card := range batch.Cards {
			if card.Position == position && card.Decision != DecisionRejected {
				return card.CardID
			}
		}
	}
	return currentCardID
}

func tasteGesture(gesture Gesture) taste.Gesture {
	switch gesture {
	case GestureLeft:
		return taste.GestureLeft
	case GestureRight:
		return taste.GestureRight
	default:
		return taste.GestureSave
	}
}

func decisionFor(gesture Gesture) CardDecision {
	switch gesture {
	case GestureLeft:
		return DecisionRejected
	case GestureRight:
		return DecisionAccepted
	case GestureSave:
		return DecisionSaved
	default:
		return DecisionNone
	}
}

func validRejectionReason(reason taste.Reason) bool {
	switch reason {
	case taste.ReasonMarketing, taste.ReasonAISlop, taste.ReasonValueMisaligned, taste.ReasonOther:
		return true
	default:
		return false
	}
}

func (s *GestureService) now() time.Time {
	if s.Now == nil {
		return time.Now().UTC()
	}
	return s.Now().UTC()
}
