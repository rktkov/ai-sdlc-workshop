package feed

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/taste"
)

type fakeTaste struct {
	calls  int
	cmd    taste.Command
	result taste.Result
	err    error
}

func (f *fakeTaste) Record(_ context.Context, cmd taste.Command) (taste.Result, error) {
	f.calls++
	f.cmd = cmd
	return f.result, f.err
}

func gestureBatch() Batch {
	return Batch{
		ID: "b", UserID: "owner", Size: 3, State: BatchStreaming,
		Cards: []Card{
			{CardID: "c1", VideoID: "v1", ChannelID: "ch1", Topic: "ai", Position: 0, DiscoveryRank: 0, State: StateAnalyzed, Title: "t", Channel: "ch", AnalysisID: "a1", Score: 0.5, Decision: DecisionNone},
			{CardID: "c2", VideoID: "v2", ChannelID: "ch2", Topic: "ai", Position: 1, DiscoveryRank: 1, State: StateAnalyzed, Title: "t", Channel: "ch", AnalysisID: "a2", Score: 0.5, Decision: DecisionNone},
			{CardID: "c3", VideoID: "v3", ChannelID: "ch3", Topic: "ai", Position: 2, DiscoveryRank: 2, State: StateAnalyzed, Title: "t", Channel: "ch", AnalysisID: "a3", Score: 0.5, Decision: DecisionNone},
		},
	}
}

func gestureSession() Session {
	return Session{
		ID: "s", UserID: "owner", BatchID: "b", State: SessionOpen,
		CurrentCardID: "c1", FurthestCardID: "c1", LeaseOwner: "lease",
		LastActivityAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Hour),
	}
}

func TestGestureLeft(t *testing.T) {
	tasteRecorder := &fakeTaste{result: taste.Result{Applied: true}}
	store := &fakeSessionStore{}
	service := &GestureService{Sessions: store, Taste: tasteRecorder}
	batch := gestureBatch()
	session := gestureSession()

	result, err := service.Apply(context.Background(), session, &batch, GestureCommand{
		GestureID: "g1", CardID: "c1", Gesture: GestureLeft, Reason: taste.ReasonMarketing,
		ScoredAxes: []taste.AxisScore{{Axis: "depth", Score: 0.8}},
	})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !result.Applied || result.Duplicate {
		t.Fatalf("result = %+v", result)
	}
	if tasteRecorder.calls != 1 || tasteRecorder.cmd.Reason != taste.ReasonMarketing {
		t.Fatalf("taste = %+v", tasteRecorder.cmd)
	}
	if batch.Cards[0].Decision != DecisionRejected {
		t.Fatalf("decision = %q", batch.Cards[0].Decision)
	}
	if result.Session.CurrentCardID != "c2" || result.Session.FurthestCardID != "c2" {
		t.Fatalf("session = %+v", result.Session)
	}
}

func TestGestureRightAndSave(t *testing.T) {
	tasteRecorder := &fakeTaste{result: taste.Result{Applied: true}}
	service := &GestureService{Sessions: &fakeSessionStore{}, Taste: tasteRecorder}
	batch := gestureBatch()
	session := gestureSession()

	result, err := service.Apply(context.Background(), session, &batch, GestureCommand{GestureID: "g1", CardID: "c1", Gesture: GestureRight})
	if err != nil {
		t.Fatalf("Apply right: %v", err)
	}
	if batch.Cards[0].Decision != DecisionAccepted || result.Session.CurrentCardID != "c2" {
		t.Fatalf("batch = %+v, session = %+v", batch.Cards[0], result.Session)
	}

	result, err = service.Apply(context.Background(), result.Session, &batch, GestureCommand{GestureID: "g2", CardID: "c2", Gesture: GestureSave})
	if err != nil {
		t.Fatalf("Apply save: %v", err)
	}
	if batch.Cards[1].Decision != DecisionSaved || result.Session.CurrentCardID != "c2" {
		t.Fatalf("batch = %+v, session = %+v", batch.Cards[1], result.Session)
	}
}

func TestGestureDownUp(t *testing.T) {
	tasteRecorder := &fakeTaste{}
	service := &GestureService{Sessions: &fakeSessionStore{}, Taste: tasteRecorder}
	batch := gestureBatch()
	session := gestureSession()

	result, err := service.Apply(context.Background(), session, &batch, GestureCommand{GestureID: "g1", CardID: "c1", Gesture: GestureDown})
	if err != nil {
		t.Fatalf("Apply down: %v", err)
	}
	if tasteRecorder.calls != 0 || result.Session.CurrentCardID != "c2" {
		t.Fatalf("taste calls = %d, session = %+v", tasteRecorder.calls, result.Session)
	}
	result, err = service.Apply(context.Background(), result.Session, &batch, GestureCommand{GestureID: "g2", CardID: "c2", Gesture: GestureUp})
	if err != nil {
		t.Fatalf("Apply up: %v", err)
	}
	if result.Session.CurrentCardID != "c1" || result.Session.FurthestCardID != "c2" {
		t.Fatalf("session = %+v", result.Session)
	}
}

func TestGestureUpSkipsRejected(t *testing.T) {
	service := &GestureService{Sessions: &fakeSessionStore{}, Taste: &fakeTaste{result: taste.Result{Applied: true}}}
	batch := gestureBatch()
	session := gestureSession()
	session.CurrentCardID = "c3"
	session.FurthestCardID = "c3"
	batch.Cards[1].Decision = DecisionRejected

	result, err := service.Apply(context.Background(), session, &batch, GestureCommand{GestureID: "g1", CardID: "c3", Gesture: GestureUp})
	if err != nil {
		t.Fatalf("Apply up: %v", err)
	}
	if result.Session.CurrentCardID != "c1" {
		t.Fatalf("cursor = %q, want c1", result.Session.CurrentCardID)
	}
}

func TestGestureIdempotent(t *testing.T) {
	tasteRecorder := &fakeTaste{result: taste.Result{Applied: true}}
	service := &GestureService{Sessions: &fakeSessionStore{}, Taste: tasteRecorder}
	batch := gestureBatch()
	session := gestureSession()
	cmd := GestureCommand{GestureID: "g1", CardID: "c1", Gesture: GestureLeft, Reason: taste.ReasonOther}

	first, err := service.Apply(context.Background(), session, &batch, cmd)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	second, err := service.Apply(context.Background(), first.Session, &batch, cmd)
	if err != nil {
		t.Fatalf("Apply duplicate: %v", err)
	}
	if !second.Duplicate || tasteRecorder.calls != 1 {
		t.Fatalf("result = %+v, taste calls = %d", second, tasteRecorder.calls)
	}
	if second.Session.CurrentCardID != first.Session.CurrentCardID {
		t.Fatalf("cursor moved: %q -> %q", first.Session.CurrentCardID, second.Session.CurrentCardID)
	}
}

func TestGestureTasteFailure(t *testing.T) {
	service := &GestureService{Sessions: &fakeSessionStore{}, Taste: &fakeTaste{err: errors.New("taste down")}}
	batch := gestureBatch()
	session := gestureSession()
	_, err := service.Apply(context.Background(), session, &batch, GestureCommand{GestureID: "g1", CardID: "c1", Gesture: GestureLeft, Reason: taste.ReasonOther})
	if err == nil {
		t.Fatal("Apply = nil, want an error")
	}
	if batch.Cards[0].Decision != DecisionNone || session.CurrentCardID != "c1" {
		t.Fatalf("state changed after a taste failure")
	}
}

func TestGestureUnhappy(t *testing.T) {
	service := &GestureService{Sessions: &fakeSessionStore{}, Taste: &fakeTaste{result: taste.Result{Applied: true}}}
	batch := gestureBatch()
	session := gestureSession()

	if _, err := service.Apply(context.Background(), session, &batch, GestureCommand{GestureID: "g1", CardID: "c1", Gesture: GestureLeft}); err == nil {
		t.Fatal("left without reason = nil, want an error")
	}
	if _, err := service.Apply(context.Background(), session, &batch, GestureCommand{GestureID: "g1", CardID: "missing", Gesture: GestureRight}); err == nil {
		t.Fatal("unknown card = nil, want an error")
	}
	closed := session
	closed.State = SessionClosed
	if _, err := service.Apply(context.Background(), closed, &batch, GestureCommand{GestureID: "g1", CardID: "c1", Gesture: GestureRight}); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("closed session = %v, want ErrSessionClosed", err)
	}
	if _, err := service.Apply(context.Background(), session, &batch, GestureCommand{GestureID: "g1", CardID: "c1", Gesture: "bad"}); err == nil {
		t.Fatal("bad gesture = nil, want an error")
	}
}
