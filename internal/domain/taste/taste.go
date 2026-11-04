// Package taste holds the durable preference model.
package taste

import (
	"context"
	"errors"
	"fmt"
	"math"
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

// AxisScore is one analysis score used for the weight update.
type AxisScore struct {
	Axis  string
	Score float64
}

// Command records one gesture.
type Command struct {
	GestureID  string
	UserID     string
	CardID     string
	VideoID    string
	ChannelID  string
	Topic      string
	Gesture    Gesture
	Reason     Reason
	ScoredAxes []AxisScore
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
	ScoredAxes []AxisScore
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

// Validate checks the cold-period invariants.
func (c ColdPeriod) Validate() error {
	if strings.TrimSpace(c.UserID) == "" {
		return fmt.Errorf("cold period has no user id")
	}
	if strings.TrimSpace(c.Target) == "" {
		return fmt.Errorf("cold period has no target")
	}
	if !validReason(c.Reason) {
		return fmt.Errorf("cold period has an invalid reason")
	}
	switch c.Scope {
	case ScopeTopic, ScopeChannel:
	default:
		return fmt.Errorf("cold period has an invalid scope")
	}
	wantScope, _ := scopeFor(c.Reason, Command{ChannelID: c.Target, Topic: c.Target})
	if c.Scope != wantScope {
		return fmt.Errorf("cold period scope does not match its reason")
	}
	if c.CreatedAt.IsZero() || c.LastRefreshedAt.IsZero() {
		return fmt.Errorf("cold period has no timestamps")
	}
	if !c.ExpiresAt.After(c.CreatedAt) {
		return fmt.Errorf("cold period expires before it starts")
	}
	if c.LastRefreshedAt.Before(c.CreatedAt) {
		return fmt.Errorf("cold period was refreshed before it started")
	}
	return nil
}

// Result is the outcome of a gesture.
type Result struct {
	Applied   bool
	Duplicate bool
	SignalID  string
}

// WeightUpdate is one axis change calculated by the service.
type WeightUpdate struct {
	Axis   string
	Weight float64
}

// Store persists a signal, its scores, the weight updates and an optional cold period.
type Store interface {
	RecordGesture(ctx context.Context, signal Signal, cold *ColdPeriod, updates []WeightUpdate) (bool, error)
}

// ProfileStore loads and updates the profile.
type ProfileStore interface {
	Get(ctx context.Context, userID string) (Profile, error)
	SetInterests(ctx context.Context, userID string, interests []string) error
}

// Service records gestures.
type Service struct {
	Store    Store
	Profiles ProfileStore
	Now      func() time.Time
	Duration time.Duration
}

// SetInterests replaces the interest set after validation.
func (s *Service) SetInterests(ctx context.Context, userID string, interests []string) error {
	if s.Profiles == nil {
		return errors.New("profile store is nil")
	}
	normalized, err := NormalizeInterests(interests)
	if err != nil {
		return err
	}
	return s.Profiles.SetInterests(ctx, userID, normalized)
}

// Interests returns the stored interest set.
func (s *Service) Interests(ctx context.Context, userID string) ([]string, error) {
	if s.Profiles == nil {
		return nil, errors.New("profile store is nil")
	}
	profile, err := s.Profiles.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	return append([]string(nil), profile.Interests...), nil
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
		ScoredAxes: cmd.ScoredAxes,
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
		if err := cold.Validate(); err != nil {
			return Result{}, err
		}
	}

	updates, err := s.weightUpdates(ctx, cmd)
	if err != nil {
		return Result{}, err
	}

	created, err := s.Store.RecordGesture(ctx, signal, cold, updates)
	if err != nil {
		return Result{}, err
	}
	return Result{Applied: true, Duplicate: !created, SignalID: signal.SignalID}, nil
}

func (s *Service) weightUpdates(ctx context.Context, cmd Command) ([]WeightUpdate, error) {
	if s.Profiles == nil || len(cmd.ScoredAxes) == 0 {
		return nil, nil
	}
	profile, err := s.Profiles.Get(ctx, cmd.UserID)
	if err != nil {
		return nil, err
	}
	scores := map[string]float64{}
	for _, score := range cmd.ScoredAxes {
		if score.Score >= 0 && score.Score <= 1 {
			scores[normalizeName(score.Axis)] = score.Score
		}
	}

	delta := 0.05
	if cmd.Gesture == GestureLeft {
		delta = -0.05
	}

	var updates []WeightUpdate
	for _, axis := range profile.Axes {
		score, ok := scores[normalizeName(axis.Name)]
		if !ok || score == 0 {
			continue
		}
		weight := math.Round(clamp(axis.Weight+delta)*10000) / 10000
		if weight != axis.Weight {
			updates = append(updates, WeightUpdate{Axis: axis.Name, Weight: weight})
		}
	}
	return updates, nil
}

func clamp(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
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

// ScopeForReason returns the cold-period scope for a rejection reason.
func ScopeForReason(reason Reason) Scope {
	switch reason {
	case ReasonValueMisaligned, ReasonMarketing:
		return ScopeChannel
	default:
		return ScopeTopic
	}
}

func scopeFor(reason Reason, cmd Command) (Scope, string) {
	if ScopeForReason(reason) == ScopeChannel {
		return ScopeChannel, cmd.ChannelID
	}
	return ScopeTopic, cmd.Topic
}
