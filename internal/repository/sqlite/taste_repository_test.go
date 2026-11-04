package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/taste"
	"github.com/rktkov/ytfeed/internal/storage"
)

func TestRecordGestureEndToEnd(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	now := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	repo := NewTasteRepository(db)
	service := &taste.Service{Store: repo, Now: func() time.Time { return now }, Duration: 24 * time.Hour}

	cmd := taste.Command{
		GestureID: "g-1",
		UserID:    "owner",
		CardID:    "card-1",
		VideoID:   "video-1",
		ChannelID: "channel-1",
		Topic:     "ai-agents",
		Gesture:   taste.GestureLeft,
		Reason:    taste.ReasonMarketing,
	}
	result, err := service.Record(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}
	if !result.Applied || result.Duplicate {
		t.Fatalf("result = %+v", result)
	}

	var signals, colds int
	if err := db.QueryRow("SELECT count(*) FROM signals").Scan(&signals); err != nil {
		t.Fatalf("count signals: %v", err)
	}
	if err := db.QueryRow("SELECT count(*) FROM cold_periods").Scan(&colds); err != nil {
		t.Fatalf("count cold periods: %v", err)
	}
	if signals != 1 || colds != 1 {
		t.Fatalf("signals = %d, cold periods = %d", signals, colds)
	}

	var scope, target string
	var expires time.Time
	if err := db.QueryRow("SELECT scope, target, expires_at FROM cold_periods").Scan(&scope, &target, &expires); err != nil {
		t.Fatalf("read cold period: %v", err)
	}
	if scope != string(taste.ScopeChannel) || target != cmd.ChannelID {
		t.Fatalf("scope = %q, target = %q", scope, target)
	}
	if !expires.Equal(now.Add(24 * time.Hour)) {
		t.Fatalf("expires = %v", expires)
	}

	again, err := service.Record(context.Background(), cmd)
	if err != nil {
		t.Fatalf("Record duplicate: %v", err)
	}
	if !again.Duplicate {
		t.Fatalf("duplicate result = %+v", again)
	}
	if err := db.QueryRow("SELECT count(*) FROM signals").Scan(&signals); err != nil {
		t.Fatalf("count signals: %v", err)
	}
	if signals != 1 {
		t.Fatalf("signals after duplicate = %d, want 1", signals)
	}

	refreshed := &taste.Service{Store: repo, Now: func() time.Time { return now.Add(6 * time.Hour) }, Duration: 48 * time.Hour}
	refreshCmd := cmd
	refreshCmd.GestureID = "g-2"
	refreshCmd.Gesture = taste.GestureLeft
	refreshCmd.Reason = taste.ReasonMarketing
	if _, err := refreshed.Record(context.Background(), refreshCmd); err != nil {
		t.Fatalf("Record refresh: %v", err)
	}
	if err := db.QueryRow("SELECT count(*) FROM cold_periods").Scan(&colds); err != nil {
		t.Fatalf("count cold periods: %v", err)
	}
	if colds != 1 {
		t.Fatalf("cold periods after refresh = %d, want 1", colds)
	}
}

func TestRecordGestureUnhappy(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	repo := NewTasteRepository(db)
	service := &taste.Service{Store: repo}
	_ = db.Close()

	cmd := taste.Command{
		GestureID: "g-1",
		UserID:    "owner",
		CardID:    "card-1",
		VideoID:   "video-1",
		ChannelID: "channel-1",
		Topic:     "ai-agents",
		Gesture:   taste.GestureRight,
	}
	if _, err := service.Record(context.Background(), cmd); err == nil {
		t.Fatal("Record(closed db) = nil, want an error")
	}
}

func TestRecordGestureRight(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	repo := NewTasteRepository(db)
	service := &taste.Service{Store: repo, Duration: 24 * time.Hour}
	cmd := taste.Command{
		GestureID: "g-right",
		UserID:    "owner",
		CardID:    "card-1",
		VideoID:   "video-1",
		ChannelID: "channel-1",
		Topic:     "ai-agents",
		Gesture:   taste.GestureRight,
	}
	if _, err := service.Record(context.Background(), cmd); err != nil {
		t.Fatalf("Record: %v", err)
	}

	var colds int
	if err := db.QueryRow("SELECT count(*) FROM cold_periods").Scan(&colds); err != nil {
		t.Fatalf("count cold periods: %v", err)
	}
	if colds != 0 {
		t.Fatalf("cold periods = %d, want 0", colds)
	}
}

func TestRecordGestureStoreErrors(t *testing.T) {
	t.Run("missing signals table", func(t *testing.T) {
		db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		defer func() { _ = db.Close() }()
		if _, err := db.Exec("DROP TABLE signals"); err != nil {
			t.Fatalf("drop signals: %v", err)
		}

		service := &taste.Service{Store: NewTasteRepository(db)}
		cmd := taste.Command{GestureID: "g", UserID: "u", CardID: "c", VideoID: "v", ChannelID: "ch", Topic: "t", Gesture: taste.GestureRight}
		if _, err := service.Record(context.Background(), cmd); err == nil {
			t.Fatal("Record(missing signals) = nil, want an error")
		}
	})

	t.Run("missing cold periods table", func(t *testing.T) {
		db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
		if err != nil {
			t.Fatalf("Open: %v", err)
		}
		defer func() { _ = db.Close() }()
		if _, err := db.Exec("DROP TABLE cold_periods"); err != nil {
			t.Fatalf("drop cold periods: %v", err)
		}

		service := &taste.Service{Store: NewTasteRepository(db)}
		cmd := taste.Command{GestureID: "g", UserID: "u", CardID: "c", VideoID: "v", ChannelID: "ch", Topic: "t", Gesture: taste.GestureLeft, Reason: taste.ReasonMarketing}
		if _, err := service.Record(context.Background(), cmd); err == nil {
			t.Fatal("Record(missing cold periods) = nil, want an error")
		}
	})
}

func TestRecordGestureUpdatesWeights(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	profile := taste.Profile{
		UserID:        "owner",
		RubricVersion: "v1",
		Axes: []taste.Axis{
			{Name: "depth", Weight: 0.5, PositiveExamples: []string{"p"}, NegativeExamples: []string{"n"}},
			{Name: "novelty", Weight: 0.98, PositiveExamples: []string{"p"}, NegativeExamples: []string{"n"}},
			{Name: "clarity", Weight: 0.5, PositiveExamples: []string{"p"}, NegativeExamples: []string{"n"}},
		},
		Interests: []string{"ai-agents"},
	}
	if err := NewProfileRepository(db).Save(context.Background(), profile); err != nil {
		t.Fatalf("Save profile: %v", err)
	}

	service := &taste.Service{
		Store:    NewTasteRepository(db),
		Profiles: NewProfileRepository(db),
	}
	cmd := taste.Command{
		GestureID: "g", UserID: "owner", CardID: "c", VideoID: "v", ChannelID: "ch", Topic: "t",
		Gesture: taste.GestureRight,
		ScoredAxes: []taste.AxisScore{
			{Axis: "depth", Score: 0.8},
			{Axis: "novelty", Score: 0.9},
		},
	}
	if _, err := service.Record(context.Background(), cmd); err != nil {
		t.Fatalf("Record: %v", err)
	}

	got, err := NewProfileRepository(db).Get(context.Background(), "owner")
	if err != nil {
		t.Fatalf("Get profile: %v", err)
	}
	if got.Axes[0].Weight != 0.55 || got.Axes[1].Weight != 1.0 {
		t.Fatalf("weights = %v, want 0.55 and 1.0", got.Axes)
	}

	var scores int
	if err := db.QueryRow("SELECT count(*) FROM signal_axis_scores").Scan(&scores); err != nil {
		t.Fatalf("count scores: %v", err)
	}
	if scores != 2 {
		t.Fatalf("scores = %d, want 2", scores)
	}
}

func TestRecordGestureScoreError(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec("DROP TABLE signal_axis_scores"); err != nil {
		t.Fatalf("drop scores: %v", err)
	}

	service := &taste.Service{Store: NewTasteRepository(db)}
	cmd := taste.Command{
		GestureID: "g", UserID: "owner", CardID: "c", VideoID: "v", ChannelID: "ch", Topic: "t",
		Gesture:    taste.GestureRight,
		ScoredAxes: []taste.AxisScore{{Axis: "depth", Score: 0.8}},
	}
	if _, err := service.Record(context.Background(), cmd); err == nil {
		t.Fatal("Record(dropped scores) = nil, want an error")
	}
}

type ghostProfile struct{}

func (ghostProfile) SetInterests(context.Context, string, []string) error { return nil }

func (ghostProfile) Get(context.Context, string) (taste.Profile, error) {
	return taste.Profile{
		UserID: "owner",
		Axes: []taste.Axis{
			{Name: "ghost", Weight: 0.5, PositiveExamples: []string{"p"}, NegativeExamples: []string{"n"}},
		},
	}, nil
}

func TestRecordGestureMissingAxis(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	service := &taste.Service{Store: NewTasteRepository(db), Profiles: ghostProfile{}}
	cmd := taste.Command{
		GestureID: "g", UserID: "owner", CardID: "c", VideoID: "v", ChannelID: "ch", Topic: "t",
		Gesture:    taste.GestureRight,
		ScoredAxes: []taste.AxisScore{{Axis: "ghost", Score: 0.8}},
	}
	if _, err := service.Record(context.Background(), cmd); err == nil {
		t.Fatal("Record(missing axis) = nil, want an error")
	}
}

func TestColdPeriodLifecycle(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	now := time.Now().UTC()
	repo := NewTasteRepository(db)
	service := &taste.Service{Store: repo, Now: func() time.Time { return now }, Duration: 24 * time.Hour}
	cmd := taste.Command{
		GestureID: "g1", UserID: "owner", CardID: "c", VideoID: "v", ChannelID: "ch", Topic: "t",
		Gesture: taste.GestureLeft, Reason: taste.ReasonMarketing,
	}
	if _, err := service.Record(context.Background(), cmd); err != nil {
		t.Fatalf("Record: %v", err)
	}

	active, err := repo.ListActiveColdPeriods(context.Background(), "owner")
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("active = %d, want 1", len(active))
	}

	found, err := repo.FindColdPeriod(context.Background(), "owner", taste.ScopeChannel, "ch")
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if found.ExpiresAt.Before(now) {
		t.Fatalf("expiry = %v", found.ExpiresAt)
	}

	refresh := &taste.Service{Store: repo, Now: func() time.Time { return now.Add(2 * time.Hour) }, Duration: 48 * time.Hour}
	cmd.GestureID = "g2"
	if _, err := refresh.Record(context.Background(), cmd); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	refreshed, err := repo.FindColdPeriod(context.Background(), "owner", taste.ScopeChannel, "ch")
	if err != nil {
		t.Fatalf("Find refreshed: %v", err)
	}
	if !refreshed.ExpiresAt.After(found.ExpiresAt) {
		t.Fatalf("expiry did not move: %v -> %v", found.ExpiresAt, refreshed.ExpiresAt)
	}

	if _, err := repo.FindColdPeriod(context.Background(), "owner", taste.ScopeTopic, "missing"); !errors.Is(err, taste.ErrColdPeriodNotFound) {
		t.Fatalf("Find(missing) = %v", err)
	}
}
