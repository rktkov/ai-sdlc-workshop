package feed

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// SessionState is open or closed.
type SessionState string

const (
	SessionOpen   SessionState = "open"
	SessionClosed SessionState = "closed"
)

var (
	// ErrNoOpenSession means the user has no open session.
	ErrNoOpenSession = errors.New("no open session")
	// ErrNoSession means no session exists for the batch.
	ErrNoSession = errors.New("session not found")
	// ErrLiveLease means another UI holds the session.
	ErrLiveLease = errors.New("session lease is live")
	// ErrStaleSession means the session version changed.
	ErrStaleSession = errors.New("session version is stale")
	// ErrSessionClosed means the session is closed.
	ErrSessionClosed = errors.New("session is closed")
)

// Session is one browsing session over a batch.
type Session struct {
	ID             string
	UserID         string
	BatchID        string
	State          SessionState
	CurrentCardID  string
	FurthestCardID string
	LastGestureID  string
	LeaseOwner     string
	LastActivityAt time.Time
	ExpiresAt      time.Time
	OpenedAt       time.Time
	ClosedAt       time.Time
	ResumeCount    int
	Version        int
}

// SessionStore persists sessions.
type SessionStore interface {
	GetOpen(ctx context.Context, userID string) (Session, error)
	GetLatest(ctx context.Context, userID, batchID string) (Session, error)
	Create(ctx context.Context, session Session) error
	Save(ctx context.Context, session Session, expectedVersion int) (Session, error)
	Reopen(ctx context.Context, sessionID, leaseOwner string, now, expiresAt time.Time) (Session, error)
	Close(ctx context.Context, sessionID string, now time.Time) error
}

// SessionService opens, resumes and closes sessions.
type SessionService struct {
	Store       SessionStore
	Now         func() time.Time
	IdleTimeout time.Duration
	ID          func() string
}

// Open returns an open session or a resumed session.
func (s *SessionService) Open(ctx context.Context, userID, batchID, firstCardID, leaseOwner string) (Session, bool, error) {
	if s.Store == nil {
		return Session{}, false, errors.New("session store is nil")
	}
	if userID == "" || batchID == "" || firstCardID == "" || leaseOwner == "" {
		return Session{}, false, errors.New("session has no user, batch, card or lease")
	}
	now := s.now()
	existing, err := s.Store.GetOpen(ctx, userID)
	if err == nil {
		if existing.LeaseOwner == leaseOwner {
			return existing, false, nil
		}
		if existing.ExpiresAt.After(now) {
			return Session{}, false, ErrLiveLease
		}
		existing.LeaseOwner = leaseOwner
		existing.LastActivityAt = now
		existing.ExpiresAt = now.Add(s.idle())
		updated, err := s.Store.Save(ctx, existing, existing.Version)
		if err != nil {
			return Session{}, false, err
		}
		return updated, true, nil
	}
	if !errors.Is(err, ErrNoOpenSession) {
		return Session{}, false, err
	}

	latest, err := s.Store.GetLatest(ctx, userID, batchID)
	if err == nil && latest.State == SessionClosed {
		reopened, err := s.Store.Reopen(ctx, latest.ID, leaseOwner, now, now.Add(s.idle()))
		if err != nil {
			return Session{}, false, err
		}
		return reopened, true, nil
	}
	if err != nil && !errors.Is(err, ErrNoSession) {
		return Session{}, false, err
	}

	session := Session{
		ID: s.newID(), UserID: userID, BatchID: batchID, State: SessionOpen,
		CurrentCardID: firstCardID, FurthestCardID: firstCardID, LeaseOwner: leaseOwner,
		LastActivityAt: now, ExpiresAt: now.Add(s.idle()), OpenedAt: now,
	}
	if err := s.Store.Create(ctx, session); err != nil {
		return Session{}, false, err
	}
	return session, false, nil
}

// Touch updates the idle timer after an interaction.
func (s *SessionService) Touch(ctx context.Context, session Session) (Session, error) {
	if session.State != SessionOpen {
		return Session{}, ErrSessionClosed
	}
	now := s.now()
	session.LastActivityAt = now
	session.ExpiresAt = now.Add(s.idle())
	return s.Store.Save(ctx, session, session.Version)
}

// Close releases the lease and closes the session.
func (s *SessionService) Close(ctx context.Context, sessionID string) error {
	if s.Store == nil {
		return errors.New("session store is nil")
	}
	return s.Store.Close(ctx, sessionID, s.now())
}

func (s *SessionService) now() time.Time {
	if s.Now == nil {
		return time.Now().UTC()
	}
	return s.Now().UTC()
}

func (s *SessionService) idle() time.Duration {
	if s.IdleTimeout <= 0 {
		return time.Hour
	}
	return s.IdleTimeout
}

func (s *SessionService) newID() string {
	if s.ID != nil {
		return s.ID()
	}
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("session-%d", time.Now().UnixNano())
	}
	return "session-" + hex.EncodeToString(buf)
}

// Validate checks the session invariants.
func (s Session) Validate() error {
	if s.ID == "" || s.UserID == "" || s.BatchID == "" {
		return errors.New("session has no id, user or batch")
	}
	if s.CurrentCardID == "" || s.FurthestCardID == "" {
		return errors.New("session has no cursor cards")
	}
	switch s.State {
	case SessionOpen, SessionClosed:
	default:
		return fmt.Errorf("session has an invalid state %q", s.State)
	}
	if s.ResumeCount < 0 {
		return errors.New("session has a negative resume count")
	}
	if s.ExpiresAt.Before(s.LastActivityAt) {
		return errors.New("session expires before its last activity")
	}
	if s.State == SessionClosed && s.ClosedAt.IsZero() {
		return errors.New("closed session has no closed time")
	}
	return nil
}
