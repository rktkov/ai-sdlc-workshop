package discovery

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// QuotaCall is one billable YouTube API call.
type QuotaCall string

const (
	CallSearch        QuotaCall = "search"
	CallVideos        QuotaCall = "videos"
	CallChannels      QuotaCall = "channels"
	CallSubscriptions QuotaCall = "subscriptions"
	CallMostPopular   QuotaCall = "most_popular"
)

// ReservationState is the terminal or open state of one reservation.
type ReservationState string

const (
	ReservationReserved ReservationState = "reserved"
	ReservationCharged  ReservationState = "charged"
	ReservationReleased ReservationState = "released"
)

var (
	// ErrQuotaExhausted means the daily budget cannot cover the call.
	ErrQuotaExhausted = errors.New("youtube quota exhausted")
	// ErrReservationNotFound means the reservation does not exist.
	ErrReservationNotFound = errors.New("quota reservation not found")
	// ErrReservationFinal means the reservation is already charged or released.
	ErrReservationFinal = errors.New("quota reservation is final")
)

// Units returns the documented cost of one call.
func Units(call QuotaCall) int {
	switch call {
	case CallSearch:
		return 100
	case CallVideos, CallChannels, CallSubscriptions, CallMostPopular:
		return 1
	default:
		return 0
	}
}

// Ledger is one accounting day for one user.
type Ledger struct {
	UserID     string
	Day        time.Time
	Budget     int
	Reserved   int
	Charged    int
	WarnedAt80 bool
	Exhausted  bool
	Version    int
	UpdatedAt  time.Time
}

// Reservation is one held or consumed call.
type Reservation struct {
	ID        string
	UserID    string
	Day       time.Time
	Units     int
	Call      QuotaCall
	State     ReservationState
	CreatedAt time.Time
	ExpiresAt time.Time
}

// QuotaStore persists the ledger and its reservations.
type QuotaStore interface {
	GetOrCreate(ctx context.Context, userID string, day time.Time, budget int) (Ledger, error)
	Reserve(ctx context.Context, userID string, day time.Time, reservation Reservation, budget int) (Ledger, error)
	Charge(ctx context.Context, reservationID string, now time.Time) (Ledger, bool, error)
	Release(ctx context.Context, reservationID string) error
	ReleaseStale(ctx context.Context, userID string, day time.Time, now time.Time) (int, error)
}

// QuotaService reserves before a call and charges after it.
type QuotaService struct {
	Store  QuotaStore
	Now    func() time.Time
	Budget int
	TTL    time.Duration
	NewID  func() string
}

// Reserve holds units for one call.
func (s *QuotaService) Reserve(ctx context.Context, userID string, call QuotaCall) (Reservation, error) {
	if s.Store == nil {
		return Reservation{}, errors.New("quota store is nil")
	}
	units := Units(call)
	if units <= 0 {
		return Reservation{}, fmt.Errorf("unknown quota call %q", call)
	}
	now := s.now()
	reservation := Reservation{
		ID:        s.newID(),
		UserID:    userID,
		Day:       AccountingDay(now),
		Units:     units,
		Call:      call,
		State:     ReservationReserved,
		CreatedAt: now,
		ExpiresAt: now.Add(s.ttl()),
	}
	if _, err := s.Store.Reserve(ctx, userID, reservation.Day, reservation, s.budget()); err != nil {
		return Reservation{}, err
	}
	return reservation, nil
}

// Charge records the units for a sent call.
func (s *QuotaService) Charge(ctx context.Context, reservationID string) (Ledger, bool, error) {
	if s.Store == nil {
		return Ledger{}, false, errors.New("quota store is nil")
	}
	return s.Store.Charge(ctx, reservationID, s.now())
}

// Release returns units for a call that was never sent.
func (s *QuotaService) Release(ctx context.Context, reservationID string) error {
	if s.Store == nil {
		return errors.New("quota store is nil")
	}
	return s.Store.Release(ctx, reservationID)
}

// ReleaseStale clears expired reservations for the current day.
func (s *QuotaService) ReleaseStale(ctx context.Context, userID string) (int, error) {
	if s.Store == nil {
		return 0, errors.New("quota store is nil")
	}
	now := s.now()
	return s.Store.ReleaseStale(ctx, userID, AccountingDay(now), now)
}

// AccountingDay returns the UTC calendar day for an instant.
func AccountingDay(instant time.Time) time.Time {
	utc := instant.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

func (s *QuotaService) now() time.Time {
	if s.Now == nil {
		return time.Now().UTC()
	}
	return s.Now().UTC()
}

func (s *QuotaService) budget() int {
	if s.Budget <= 0 {
		return 10000
	}
	return s.Budget
}

func (s *QuotaService) ttl() time.Duration {
	if s.TTL <= 0 {
		return 10 * time.Minute
	}
	return s.TTL
}

func (s *QuotaService) newID() string {
	if s.NewID != nil {
		return s.NewID()
	}
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "reservation-" + fmt.Sprint(time.Now().UnixNano())
	}
	return "reservation-" + hex.EncodeToString(buf)
}
