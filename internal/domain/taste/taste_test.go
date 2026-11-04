package taste

import (
	"context"
	"testing"
	"time"
)

type fakeStore struct {
	signal  Signal
	cold    *ColdPeriod
	updates []WeightUpdate
	create  bool
	err     error
}

func (f *fakeStore) RecordGesture(_ context.Context, signal Signal, cold *ColdPeriod, updates []WeightUpdate) (bool, error) {
	f.signal = signal
	f.cold = cold
	f.updates = updates
	return f.create, f.err
}

type fakeProfiles struct {
	profile Profile
	err     error
}

func (f *fakeProfiles) Get(context.Context, string) (Profile, error) {
	return f.profile, f.err
}

func TestRecordLeftHappy(t *testing.T) {
	now := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	store := &fakeStore{create: true}
	service := &Service{Store: store, Now: func() time.Time { return now }, Duration: 24 * time.Hour}

	cmd := Command{
		GestureID: "g-1",
		UserID:    "owner",
		CardID:    "card-1",
		VideoID:   "video-1",
		ChannelID: "channel-1",
		Topic:     "ai-agents",
		Gesture:   GestureLeft,
		Reason:    ReasonMarketing,
	}
	result, err := service.Record(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if !result.Applied || result.Duplicate {
		t.Fatalf("result = %+v", result)
	}
	if store.cold == nil {
		t.Fatal("left swipe did not create a cold period")
	}
	if store.cold.Scope != ScopeChannel || store.cold.Target != cmd.ChannelID {
		t.Fatalf("cold = %+v", store.cold)
	}
	if !store.cold.ExpiresAt.Equal(now.Add(24 * time.Hour)) {
		t.Fatalf("expiry = %v", store.cold.ExpiresAt)
	}
}

func TestRecordRightHappy(t *testing.T) {
	store := &fakeStore{create: true}
	service := &Service{Store: store, Duration: 24 * time.Hour}

	cmd := Command{
		GestureID: "g-2",
		UserID:    "owner",
		CardID:    "card-1",
		VideoID:   "video-1",
		ChannelID: "channel-1",
		Topic:     "ai-agents",
		Gesture:   GestureRight,
	}
	if _, err := service.Record(context.Background(), cmd); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if store.cold != nil {
		t.Fatal("right swipe created a cold period")
	}
}

func TestRecordUnhappy(t *testing.T) {
	base := Command{
		GestureID: "g-1",
		UserID:    "owner",
		CardID:    "card-1",
		VideoID:   "video-1",
		ChannelID: "channel-1",
		Topic:     "ai-agents",
		Gesture:   GestureLeft,
		Reason:    ReasonMarketing,
	}

	tests := []struct {
		name string
		cmd  Command
	}{
		{name: "missing gesture id", cmd: func() Command { c := base; c.GestureID = ""; return c }()},
		{name: "left without reason", cmd: func() Command { c := base; c.Reason = ""; return c }()},
		{name: "right with reason", cmd: func() Command { c := base; c.Gesture = GestureRight; return c }()},
		{name: "bad gesture", cmd: func() Command { c := base; c.Gesture = Gesture("up"); return c }()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &Service{Store: &fakeStore{create: true}}
			if _, err := service.Record(context.Background(), tt.cmd); err == nil {
				t.Fatal("Record() = nil, want an error")
			}
		})
	}

	t.Run("nil store", func(t *testing.T) {
		service := &Service{}
		if _, err := service.Record(context.Background(), base); err == nil {
			t.Fatal("Record(nil store) = nil, want an error")
		}
	})
}

func TestRecordUpdatesWeights(t *testing.T) {
	store := &fakeStore{create: true}
	service := &Service{
		Store: store,
		Profiles: &fakeProfiles{profile: Profile{Axes: []Axis{
			{Name: "depth", Weight: 0.5},
			{Name: "novelty", Weight: 0.98},
			{Name: "clarity", Weight: 0.5},
		}}},
	}
	cmd := Command{
		GestureID: "g", UserID: "owner", CardID: "c", VideoID: "v", ChannelID: "ch", Topic: "t",
		Gesture: GestureRight,
		ScoredAxes: []AxisScore{
			{Axis: "depth", Score: 0.8},
			{Axis: "novelty", Score: 0.9},
			{Axis: "missing", Score: 0.7},
		},
	}
	if _, err := service.Record(context.Background(), cmd); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if len(store.updates) != 2 {
		t.Fatalf("updates = %+v, want 2", store.updates)
	}
	if store.updates[0].Weight != 0.55 || store.updates[1].Weight != 1.0 {
		t.Fatalf("updates = %+v", store.updates)
	}

	cmd.Gesture = GestureLeft
	cmd.Reason = ReasonOther
	cmd.GestureID = "g2"
	if _, err := service.Record(context.Background(), cmd); err != nil {
		t.Fatalf("Record left: %v", err)
	}
	if store.updates[0].Weight != 0.45 {
		t.Fatalf("left update = %+v", store.updates)
	}
}

func TestColdPeriodValidate(t *testing.T) {
	now := time.Now().UTC()
	valid := ColdPeriod{
		ColdPeriodID: "c", UserID: "owner", Scope: ScopeChannel, Target: "ch",
		Reason: ReasonMarketing, CreatedAt: now, ExpiresAt: now.Add(time.Hour), LastRefreshedAt: now,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() = %v", err)
	}

	tests := []struct {
		name string
		edit func(*ColdPeriod)
	}{
		{name: "no user", edit: func(c *ColdPeriod) { c.UserID = "" }},
		{name: "no target", edit: func(c *ColdPeriod) { c.Target = "" }},
		{name: "bad reason", edit: func(c *ColdPeriod) { c.Reason = Reason("bad") }},
		{name: "bad scope", edit: func(c *ColdPeriod) { c.Scope = Scope("bad") }},
		{name: "scope mismatch", edit: func(c *ColdPeriod) { c.Scope = ScopeTopic }},
		{name: "bad expiry", edit: func(c *ColdPeriod) { c.ExpiresAt = c.CreatedAt }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := valid
			tt.edit(&c)
			if err := c.Validate(); err == nil {
				t.Fatal("Validate() = nil, want an error")
			}
		})
	}
}

func TestSnapshotHappy(t *testing.T) {
	profile := Profile{
		UserID:        "owner",
		Note:          "prefer depth",
		RubricVersion: "v1",
		Axes: []Axis{
			{Name: "depth", Weight: 0.7, PositiveExamples: []string{"p"}, NegativeExamples: []string{"n"}},
			{Name: "novelty", Weight: 0.5, PositiveExamples: []string{"p"}, NegativeExamples: []string{"n"}},
			{Name: "clarity", Weight: 0.6, PositiveExamples: []string{"p"}, NegativeExamples: []string{"n"}},
		},
		Interests: []string{"ai-agents"},
	}
	service := &Service{Profiles: &fakeProfiles{profile: profile}}

	got, err := service.Snapshot(context.Background(), "owner")
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if got.UserID != profile.UserID || got.Note != profile.Note || got.RubricVersion != profile.RubricVersion {
		t.Fatalf("snapshot = %+v", got)
	}
	if len(got.Axes) != 3 || got.Axes[0].Name != "depth" {
		t.Fatalf("axes = %+v", got.Axes)
	}
	got.Axes[0].Name = "mutated"
	if service.Profiles.(*fakeProfiles).profile.Axes[0].Name != "depth" {
		t.Fatal("snapshot shares the profile axis slice")
	}
}

func TestSnapshotUnhappy(t *testing.T) {
	if _, err := (&Service{}).Snapshot(context.Background(), "owner"); err == nil {
		t.Fatal("Snapshot(nil reader) = nil, want an error")
	}
	if _, err := (&Service{Profiles: &fakeProfiles{err: tasteError{}}}).Snapshot(context.Background(), "owner"); err == nil {
		t.Fatal("Snapshot(error reader) = nil, want an error")
	}
}

type tasteError struct{}

func (tasteError) Error() string { return "boom" }
