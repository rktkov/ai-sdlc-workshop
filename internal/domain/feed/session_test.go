package feed

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeSessionStore struct {
	open      *Session
	latest    *Session
	openErr   error
	latestErr error
	createErr error
	saveErr   error
	reopenErr error
	closeErr  error
	created   *Session
	saved     *Session
	reopened  bool
	closed    bool
}

func (f *fakeSessionStore) GetOpen(context.Context, string) (Session, error) {
	if f.openErr != nil {
		return Session{}, f.openErr
	}
	if f.open == nil {
		return Session{}, ErrNoOpenSession
	}
	return *f.open, nil
}

func (f *fakeSessionStore) GetLatest(context.Context, string, string) (Session, error) {
	if f.latestErr != nil {
		return Session{}, f.latestErr
	}
	if f.latest == nil {
		return Session{}, ErrNoSession
	}
	return *f.latest, nil
}

func (f *fakeSessionStore) Create(_ context.Context, session Session) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.created = &session
	return nil
}

func (f *fakeSessionStore) Save(_ context.Context, session Session, _ int) (Session, error) {
	if f.saveErr != nil {
		return Session{}, f.saveErr
	}
	session.Version++
	f.saved = &session
	return session, nil
}

func (f *fakeSessionStore) Reopen(_ context.Context, _, _ string, now, expiresAt time.Time) (Session, error) {
	if f.reopenErr != nil {
		return Session{}, f.reopenErr
	}
	f.reopened = true
	return Session{
		ID: "session-1", State: SessionOpen, LastActivityAt: now, ExpiresAt: expiresAt,
		ResumeCount: 1,
	}, nil
}

func (f *fakeSessionStore) Close(context.Context, string, time.Time) error {
	if f.closeErr != nil {
		return f.closeErr
	}
	f.closed = true
	return nil
}

func TestSessionValidate(t *testing.T) {
	valid := Session{
		ID: "s", UserID: "owner", BatchID: "b", State: SessionOpen,
		CurrentCardID: "c1", FurthestCardID: "c1",
		LastActivityAt: time.Now().UTC(), ExpiresAt: time.Now().UTC().Add(time.Hour),
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	tests := []struct {
		name string
		edit func(*Session)
	}{
		{name: "no id", edit: func(s *Session) { s.ID = "" }},
		{name: "no user", edit: func(s *Session) { s.UserID = "" }},
		{name: "no batch", edit: func(s *Session) { s.BatchID = "" }},
		{name: "no cursor", edit: func(s *Session) { s.CurrentCardID = "" }},
		{name: "bad state", edit: func(s *Session) { s.State = "bad" }},
		{name: "negative resume", edit: func(s *Session) { s.ResumeCount = -1 }},
		{name: "bad expiry", edit: func(s *Session) { s.ExpiresAt = s.LastActivityAt.Add(-time.Hour) }},
		{name: "closed no time", edit: func(s *Session) { s.State = SessionClosed }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := valid
			tt.edit(&session)
			if err := session.Validate(); err == nil {
				t.Fatal("Validate = nil, want an error")
			}
		})
	}
}

func TestSessionServiceWithFakeStore(t *testing.T) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	store := &fakeSessionStore{}
	service := &SessionService{Store: store, Now: func() time.Time { return now }, ID: func() string { return "session-1" }}
	session, resumed, err := service.Open(context.Background(), "owner", "b", "c1", "lease")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if resumed || store.created == nil || session.ID != "session-1" {
		t.Fatalf("session = %+v, created = %+v", session, store.created)
	}
	if !session.ExpiresAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("expiry = %v", session.ExpiresAt)
	}
}

func TestSessionServiceFakeErrors(t *testing.T) {
	tests := []struct {
		name  string
		store *fakeSessionStore
	}{
		{name: "open error", store: &fakeSessionStore{openErr: errors.New("db")}},
		{name: "latest error", store: &fakeSessionStore{latestErr: errors.New("db")}},
		{name: "create error", store: &fakeSessionStore{createErr: errors.New("db")}},
		{name: "reopen error", store: &fakeSessionStore{latest: &Session{State: SessionClosed}, reopenErr: errors.New("db")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &SessionService{Store: tt.store}
			if _, _, err := service.Open(context.Background(), "owner", "b", "c1", "lease"); err == nil {
				t.Fatal("Open = nil, want an error")
			}
		})
	}
	if _, _, err := (&SessionService{}).Open(context.Background(), "owner", "b", "c1", "lease"); err == nil {
		t.Fatal("Open(nil store) = nil, want an error")
	}
}

func TestSessionServiceReopenFake(t *testing.T) {
	store := &fakeSessionStore{latest: &Session{ID: "old", State: SessionClosed}}
	service := &SessionService{Store: store}
	session, resumed, err := service.Open(context.Background(), "owner", "b", "c1", "lease")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if !resumed || !store.reopened || session.ResumeCount != 1 {
		t.Fatalf("session = %+v, resumed = %t", session, resumed)
	}
}

func TestSessionServiceCloseFake(t *testing.T) {
	store := &fakeSessionStore{}
	service := &SessionService{Store: store}
	if err := service.Close(context.Background(), "s"); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !store.closed {
		t.Fatal("store was not closed")
	}
	if err := (&SessionService{}).Close(context.Background(), "s"); err == nil {
		t.Fatal("Close(nil store) = nil, want an error")
	}
}
