package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/discovery"
)

// QuotaRepository stores daily quota ledgers and reservations.
type QuotaRepository struct {
	db *sql.DB
}

// NewQuotaRepository returns a repository over the given database.
func NewQuotaRepository(db *sql.DB) *QuotaRepository {
	return &QuotaRepository{db: db}
}

// GetOrCreate returns the ledger for one user and UTC day.
func (r *QuotaRepository) GetOrCreate(ctx context.Context, userID string, day time.Time, budget int) (discovery.Ledger, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return discovery.Ledger{}, fmt.Errorf("begin quota ledger: %w", err)
	}
	ledger, err := r.getOrCreateTx(ctx, tx, userID, day, budget)
	if err != nil {
		_ = tx.Rollback()
		return discovery.Ledger{}, err
	}
	if err := tx.Commit(); err != nil {
		return discovery.Ledger{}, fmt.Errorf("commit quota ledger: %w", err)
	}
	return ledger, nil
}

// Reserve holds units for one call before the request is sent.
func (r *QuotaRepository) Reserve(ctx context.Context, userID string, day time.Time, reservation discovery.Reservation, budget int) (discovery.Ledger, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return discovery.Ledger{}, fmt.Errorf("begin quota reservation: %w", err)
	}
	if _, _, err := r.releaseStaleTx(ctx, tx, userID, day, reservation.CreatedAt); err != nil {
		_ = tx.Rollback()
		return discovery.Ledger{}, err
	}
	ledger, err := r.getOrCreateTx(ctx, tx, userID, day, budget)
	if err != nil {
		_ = tx.Rollback()
		return discovery.Ledger{}, err
	}
	if ledger.Reserved+ledger.Charged+reservation.Units > ledger.Budget {
		_ = tx.Rollback()
		return discovery.Ledger{}, discovery.ErrQuotaExhausted
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO quota_reservations
		(reservation_id, user_id, accounting_day, units, call, state, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		reservation.ID, userID, dayKey(day), reservation.Units, string(reservation.Call),
		string(discovery.ReservationReserved), reservation.CreatedAt, reservation.ExpiresAt,
	); err != nil {
		_ = tx.Rollback()
		return discovery.Ledger{}, fmt.Errorf("insert quota reservation: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE quota_ledgers SET
		reserved = reserved + ?, version = version + 1, updated_at = ?,
		exhausted = CASE WHEN reserved + charged + ? >= budget THEN 1 ELSE 0 END
		WHERE user_id = ? AND accounting_day = ?`,
		reservation.Units, reservation.CreatedAt, reservation.Units, userID, dayKey(day),
	); err != nil {
		_ = tx.Rollback()
		return discovery.Ledger{}, fmt.Errorf("update quota ledger: %w", err)
	}
	ledger, err = r.readLedgerTx(ctx, tx, userID, day)
	if err != nil {
		_ = tx.Rollback()
		return discovery.Ledger{}, err
	}
	if err := tx.Commit(); err != nil {
		return discovery.Ledger{}, fmt.Errorf("commit quota reservation: %w", err)
	}
	return ledger, nil
}

// Charge moves reserved units to charged. A repeated charge is idempotent.
func (r *QuotaRepository) Charge(ctx context.Context, reservationID string, now time.Time) (discovery.Ledger, bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return discovery.Ledger{}, false, fmt.Errorf("begin quota charge: %w", err)
	}
	var userID, day string
	var units int
	var state string
	err = tx.QueryRowContext(ctx, `SELECT user_id, accounting_day, units, state
		FROM quota_reservations WHERE reservation_id = ?`, reservationID,
	).Scan(&userID, &day, &units, &state)
	if errors.Is(err, sql.ErrNoRows) {
		_ = tx.Rollback()
		return discovery.Ledger{}, false, discovery.ErrReservationNotFound
	}
	if err != nil {
		_ = tx.Rollback()
		return discovery.Ledger{}, false, fmt.Errorf("read quota reservation: %w", err)
	}
	if state == string(discovery.ReservationReleased) {
		_ = tx.Rollback()
		return discovery.Ledger{}, false, discovery.ErrReservationFinal
	}

	ledger, err := r.readLedgerByKeyTx(ctx, tx, userID, day)
	if err != nil {
		_ = tx.Rollback()
		return discovery.Ledger{}, false, err
	}
	if state == string(discovery.ReservationCharged) {
		if err := tx.Commit(); err != nil {
			return discovery.Ledger{}, false, fmt.Errorf("commit quota charge: %w", err)
		}
		return ledger, false, nil
	}

	if _, err := tx.ExecContext(ctx, `UPDATE quota_reservations SET state = ?
		WHERE reservation_id = ?`, string(discovery.ReservationCharged), reservationID); err != nil {
		_ = tx.Rollback()
		return discovery.Ledger{}, false, fmt.Errorf("mark quota charged: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE quota_ledgers SET
		reserved = reserved - ?, charged = charged + ?, version = version + 1, updated_at = ?
		WHERE user_id = ? AND accounting_day = ?`,
		units, units, now, userID, day,
	); err != nil {
		_ = tx.Rollback()
		return discovery.Ledger{}, false, fmt.Errorf("update quota ledger: %w", err)
	}
	ledger, err = r.readLedgerByKeyTx(ctx, tx, userID, day)
	if err != nil {
		_ = tx.Rollback()
		return discovery.Ledger{}, false, err
	}
	warn := !ledger.WarnedAt80 && (ledger.Charged+ledger.Reserved)*100 >= ledger.Budget*80
	exhausted := ledger.Charged+ledger.Reserved >= ledger.Budget
	if warn || exhausted {
		if _, err := tx.ExecContext(ctx, `UPDATE quota_ledgers SET warned_at_80 = ?, exhausted = ?
			WHERE user_id = ? AND accounting_day = ?`,
			boolInt(warn || ledger.WarnedAt80), boolInt(exhausted), userID, day,
		); err != nil {
			_ = tx.Rollback()
			return discovery.Ledger{}, false, fmt.Errorf("update quota flags: %w", err)
		}
		ledger.WarnedAt80 = warn || ledger.WarnedAt80
		ledger.Exhausted = exhausted
	}
	if err := tx.Commit(); err != nil {
		return discovery.Ledger{}, false, fmt.Errorf("commit quota charge: %w", err)
	}
	return ledger, warn, nil
}

// Release returns units for a call that was never sent.
func (r *QuotaRepository) Release(ctx context.Context, reservationID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin quota release: %w", err)
	}
	var userID, day string
	var units int
	var state string
	err = tx.QueryRowContext(ctx, `SELECT user_id, accounting_day, units, state
		FROM quota_reservations WHERE reservation_id = ?`, reservationID,
	).Scan(&userID, &day, &units, &state)
	if errors.Is(err, sql.ErrNoRows) {
		_ = tx.Rollback()
		return discovery.ErrReservationNotFound
	}
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("read quota reservation: %w", err)
	}
	if state == string(discovery.ReservationReleased) {
		return tx.Commit()
	}
	if state == string(discovery.ReservationCharged) {
		_ = tx.Rollback()
		return discovery.ErrReservationFinal
	}
	if _, err := tx.ExecContext(ctx, `UPDATE quota_reservations SET state = ?
		WHERE reservation_id = ?`, string(discovery.ReservationReleased), reservationID); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("mark quota released: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE quota_ledgers SET
		reserved = CASE WHEN reserved >= ? THEN reserved - ? ELSE 0 END,
		version = version + 1, updated_at = ?
		WHERE user_id = ? AND accounting_day = ?`,
		units, units, time.Now().UTC(), userID, day,
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("update quota ledger: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit quota release: %w", err)
	}
	return nil
}

// ReleaseStale releases reservations older than the instant.
func (r *QuotaRepository) ReleaseStale(ctx context.Context, userID string, day time.Time, now time.Time) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin stale quota release: %w", err)
	}
	count, _, err := r.releaseStaleTx(ctx, tx, userID, day, now)
	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit stale quota release: %w", err)
	}
	return count, nil
}

func (r *QuotaRepository) releaseStaleTx(ctx context.Context, tx *sql.Tx, userID string, day time.Time, now time.Time) (int, int, error) {
	var count, units int
	if err := tx.QueryRowContext(ctx, `SELECT count(*), COALESCE(SUM(units), 0)
		FROM quota_reservations
		WHERE user_id = ? AND accounting_day = ? AND state = ? AND expires_at <= ?`,
		userID, dayKey(day), string(discovery.ReservationReserved), now,
	).Scan(&count, &units); err != nil {
		return 0, 0, fmt.Errorf("count stale reservations: %w", err)
	}
	if count == 0 {
		return 0, 0, nil
	}
	if _, err := tx.ExecContext(ctx, `UPDATE quota_reservations SET state = ?
		WHERE user_id = ? AND accounting_day = ? AND state = ? AND expires_at <= ?`,
		string(discovery.ReservationReleased), userID, dayKey(day), string(discovery.ReservationReserved), now,
	); err != nil {
		return 0, 0, fmt.Errorf("release stale reservations: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE quota_ledgers SET
		reserved = CASE WHEN reserved >= ? THEN reserved - ? ELSE 0 END,
		version = version + 1, updated_at = ?
		WHERE user_id = ? AND accounting_day = ?`,
		units, units, now, userID, dayKey(day),
	); err != nil {
		return 0, 0, fmt.Errorf("update stale quota ledger: %w", err)
	}
	return count, units, nil
}

func (r *QuotaRepository) getOrCreateTx(ctx context.Context, tx *sql.Tx, userID string, day time.Time, budget int) (discovery.Ledger, error) {
	now := time.Now().UTC()
	if _, err := tx.ExecContext(ctx, `INSERT INTO quota_ledgers
		(user_id, accounting_day, budget, reserved, charged, warned_at_80, exhausted, version, updated_at)
		VALUES (?, ?, ?, 0, 0, 0, 0, 0, ?)
		ON CONFLICT(user_id, accounting_day) DO NOTHING`,
		userID, dayKey(day), budget, now,
	); err != nil {
		return discovery.Ledger{}, fmt.Errorf("create quota ledger: %w", err)
	}
	return r.readLedgerTx(ctx, tx, userID, day)
}

func (r *QuotaRepository) readLedgerTx(ctx context.Context, tx *sql.Tx, userID string, day time.Time) (discovery.Ledger, error) {
	return r.readLedgerByKeyTx(ctx, tx, userID, dayKey(day))
}

func (r *QuotaRepository) readLedgerByKeyTx(ctx context.Context, tx *sql.Tx, userID, day string) (discovery.Ledger, error) {
	var ledger discovery.Ledger
	var dayValue string
	var warned, exhausted int
	err := tx.QueryRowContext(ctx, `SELECT user_id, accounting_day, budget, reserved, charged,
		warned_at_80, exhausted, version, updated_at
		FROM quota_ledgers WHERE user_id = ? AND accounting_day = ?`, userID, day,
	).Scan(&ledger.UserID, &dayValue, &ledger.Budget, &ledger.Reserved, &ledger.Charged,
		&warned, &exhausted, &ledger.Version, &ledger.UpdatedAt)
	if err != nil {
		return discovery.Ledger{}, fmt.Errorf("read quota ledger: %w", err)
	}
	parsed, err := time.Parse("2006-01-02", dayValue)
	if err != nil {
		return discovery.Ledger{}, fmt.Errorf("parse accounting day: %w", err)
	}
	ledger.Day = parsed
	ledger.WarnedAt80 = warned != 0
	ledger.Exhausted = exhausted != 0
	return ledger, nil
}

func dayKey(day time.Time) string {
	return day.UTC().Format("2006-01-02")
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
