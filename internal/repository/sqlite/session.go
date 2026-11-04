package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/feed"
)

// SessionRepository stores browsing sessions.
type SessionRepository struct {
	db *sql.DB
}

// NewSessionRepository returns a repository over the given database.
func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// GetOpen returns the open session for one user.
func (r *SessionRepository) GetOpen(ctx context.Context, userID string) (feed.Session, error) {
	row := r.db.QueryRowContext(ctx, sessionSelect+` WHERE user_id = ? AND state = 'open' LIMIT 1`, userID)
	session, err := scanSession(row)
	if errors.Is(err, sql.ErrNoRows) {
		return feed.Session{}, feed.ErrNoOpenSession
	}
	if err != nil {
		return feed.Session{}, err
	}
	return session, nil
}

// GetLatest returns the newest session for one batch.
func (r *SessionRepository) GetLatest(ctx context.Context, userID, batchID string) (feed.Session, error) {
	row := r.db.QueryRowContext(ctx, sessionSelect+` WHERE user_id = ? AND batch_id = ? ORDER BY opened_at DESC LIMIT 1`, userID, batchID)
	session, err := scanSession(row)
	if errors.Is(err, sql.ErrNoRows) {
		return feed.Session{}, feed.ErrNoSession
	}
	if err != nil {
		return feed.Session{}, err
	}
	return session, nil
}

// Create inserts one session. A second open session is rejected.
func (r *SessionRepository) Create(ctx context.Context, session feed.Session) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO sessions
		(session_id, user_id, batch_id, state, current_card_id, furthest_card_id,
		 last_gesture_id, lease_owner, last_activity_at, expires_at, opened_at,
		 closed_at, resume_count, version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		session.ID, session.UserID, session.BatchID, string(session.State),
		session.CurrentCardID, session.FurthestCardID, session.LastGestureID,
		session.LeaseOwner, session.LastActivityAt, session.ExpiresAt, session.OpenedAt,
		nullableTime(session.ClosedAt), session.ResumeCount, session.Version,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return feed.ErrLiveLease
		}
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

// Save updates one session if its version still matches.
func (r *SessionRepository) Save(ctx context.Context, session feed.Session, expectedVersion int) (feed.Session, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE sessions SET
		state = ?, current_card_id = ?, furthest_card_id = ?, last_gesture_id = ?,
		lease_owner = ?, last_activity_at = ?, expires_at = ?, closed_at = ?,
		resume_count = ?, version = version + 1
		WHERE session_id = ? AND version = ?`,
		string(session.State), session.CurrentCardID, session.FurthestCardID, session.LastGestureID,
		session.LeaseOwner, session.LastActivityAt, session.ExpiresAt, nullableTime(session.ClosedAt),
		session.ResumeCount, session.ID, expectedVersion,
	)
	if err != nil {
		return feed.Session{}, fmt.Errorf("update session: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return feed.Session{}, fmt.Errorf("check session update: %w", err)
	}
	if rows == 0 {
		return feed.Session{}, feed.ErrStaleSession
	}
	return r.getByID(ctx, session.ID)
}

// Reopen reopens a closed session and keeps its cursor.
func (r *SessionRepository) Reopen(ctx context.Context, sessionID, leaseOwner string, now, expiresAt time.Time) (feed.Session, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE sessions SET
		state = 'open', lease_owner = ?, last_activity_at = ?, expires_at = ?,
		closed_at = NULL, resume_count = resume_count + 1, version = version + 1
		WHERE session_id = ? AND state = 'closed'`,
		leaseOwner, now, expiresAt, sessionID,
	)
	if err != nil {
		return feed.Session{}, fmt.Errorf("reopen session: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return feed.Session{}, fmt.Errorf("check session reopen: %w", err)
	}
	if rows == 0 {
		return feed.Session{}, feed.ErrSessionClosed
	}
	return r.getByID(ctx, sessionID)
}

// Close releases the lease and closes the session.
func (r *SessionRepository) Close(ctx context.Context, sessionID string, now time.Time) error {
	result, err := r.db.ExecContext(ctx, `UPDATE sessions SET
		state = 'closed', lease_owner = '', closed_at = ?, version = version + 1
		WHERE session_id = ? AND state = 'open'`,
		now, sessionID,
	)
	if err != nil {
		return fmt.Errorf("close session: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check session close: %w", err)
	}
	if rows == 0 {
		return feed.ErrNoOpenSession
	}
	return nil
}

func (r *SessionRepository) getByID(ctx context.Context, sessionID string) (feed.Session, error) {
	row := r.db.QueryRowContext(ctx, sessionSelect+` WHERE session_id = ?`, sessionID)
	session, err := scanSession(row)
	if errors.Is(err, sql.ErrNoRows) {
		return feed.Session{}, feed.ErrNoSession
	}
	if err != nil {
		return feed.Session{}, err
	}
	return session, nil
}

const sessionSelect = `SELECT session_id, user_id, batch_id, state, current_card_id,
	furthest_card_id, last_gesture_id, lease_owner, last_activity_at, expires_at,
	opened_at, closed_at, resume_count, version FROM sessions`

func scanSession(row scanner) (feed.Session, error) {
	var session feed.Session
	var state string
	var closed sql.NullTime
	err := row.Scan(&session.ID, &session.UserID, &session.BatchID, &state,
		&session.CurrentCardID, &session.FurthestCardID, &session.LastGestureID,
		&session.LeaseOwner, &session.LastActivityAt, &session.ExpiresAt,
		&session.OpenedAt, &closed, &session.ResumeCount, &session.Version)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return feed.Session{}, err
		}
		return feed.Session{}, fmt.Errorf("scan session: %w", err)
	}
	session.State = feed.SessionState(state)
	if closed.Valid {
		session.ClosedAt = closed.Time
	}
	return session, nil
}

func nullableTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}

var _ feed.SessionStore = (*SessionRepository)(nil)
