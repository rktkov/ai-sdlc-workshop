// Package taste holds the durable preference model.
package taste

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Gesture is a signal-producing owner action.
type Gesture string

const (
	GestureLeft  Gesture = "left"
	GestureRight Gesture = "right"
	GestureSave  Gesture = "save"
)

// Reason is the rejection reason for a left swipe.
type Reason string

const (
	ReasonMarketing       Reason = "marketing"
	ReasonAISlop          Reason = "ai-slop"
	ReasonValueMisaligned Reason = "value-misaligned"
	ReasonOther           Reason = "other"
)

// Scope is the target type of a cold period.
type Scope string

const (
	ScopeTopic   Scope = "topic"
	ScopeChannel Scope = "channel"
)

// Command records one gesture.
type Command struct {
	GestureID string
	UserID    string
	CardID    string
	VideoID   string
	ChannelID string
	Topic     string
	Gesture   Gesture
	Reason    Reason
}

// Signal is one immutable preference record.
type Signal struct {
	SignalID   string
	GestureID  string
	UserID     string
	CardID     string
	VideoID    string
	ChannelID  string
	Topic      string
	Gesture    Gesture
	Reason     Reason
	OccurredAt time.Time
}

// ColdPeriod suppresses a topic or channel until it expires.
type ColdPeriod struct {
	ColdPeriodID    string
	UserID          string
	Scope           Scope
	Target          string
	Reason          Reason
	CreatedAt       time.Time
	ExpiresAt       time.Time
	LastRefreshedAt time.Time
}

// Result is the outcome of a gesture.
type Result struct {
	Applied   bool
	Duplicate bool
	SignalID  string
}

// Store persists a signal and an optional cold period in one transaction.
type Store interface {
	RecordGesture(ctx context.Context, signal Signal, cold *ColdPeriod) (bool, error)
}

// Service records gestures.
type Service struct {
	Store    Store
	Now      func() time.Time
	Duration time.Duration
}

// Record validates and records one gesture.
func (s *Service) Record(ctx context.Context, cmd Command) (Result, error) {
	if err := cmd.validate(); err != nil {
		return Result{}, err
	}
	if s.Store == nil {
		return Result{}, errors.New("taste store is nil")
	}

	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	duration := s.Duration
	if duration <= 0 {
		duration = 24 * time.Hour
	}

	signal := Signal{
		SignalID:   "sig-" + cmd.GestureID,
		GestureID:  cmd.GestureID,
		UserID:     cmd.UserID,
		CardID:     cmd.CardID,
		VideoID:    cmd.VideoID,
		ChannelID:  cmd.ChannelID,
		Topic:      cmd.Topic,
		Gesture:    cmd.Gesture,
		Reason:     cmd.Reason,
		OccurredAt: now,
	}

	var cold *ColdPeriod
	if cmd.Gesture == GestureLeft {
		scope, target := scopeFor(cmd.Reason, cmd)
		cold = &ColdPeriod{
			ColdPeriodID:    "cold-" + cmd.GestureID,
			UserID:          cmd.UserID,
			Scope:           scope,
			Target:          target,
			Reason:          cmd.Reason,
			CreatedAt:       now,
			ExpiresAt:       now.Add(duration),
			LastRefreshedAt: now,
		}
	}

	created, err := s.Store.RecordGesture(ctx, signal, cold)
	if err != nil {
		return Result{}, err
	}
	return Result{Applied: true, Duplicate: !created, SignalID: signal.SignalID}, nil
}

func (c Command) validate() error {
	for name, value := range map[string]string{
		"gesture_id": c.GestureID,
		"user_id":    c.UserID,
		"card_id":    c.CardID,
		"video_id":   c.VideoID,
		"channel_id": c.ChannelID,
		"topic":      c.Topic,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s must not be empty", name)
		}
	}
	switch c.Gesture {
	case GestureLeft:
		if !validReason(c.Reason) {
			return fmt.Errorf("left swipe needs a valid reason")
		}
	case GestureRight, GestureSave:
		if c.Reason != "" {
			return fmt.Errorf("%s swipe must not have a reason", c.Gesture)
		}
	default:
		return fmt.Errorf("gesture must be left, right or save")
	}
	return nil
}

func validReason(reason Reason) bool {
	switch reason {
	case ReasonMarketing, ReasonAISlop, ReasonValueMisaligned, ReasonOther:
		return true
	default:
		return false
	}
}

func scopeFor(reason Reason, cmd Command) (Scope, string) {
	switch reason {
	case ReasonValueMisaligned, ReasonMarketing:
		return ScopeChannel, cmd.ChannelID
	default:
		return ScopeTopic, cmd.Topic
	}
}
