package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/discovery"
	"github.com/rktkov/ytfeed/internal/storage"
)

func newQuota(t *testing.T) (*QuotaRepository, *discovery.QuotaService) {
	t.Helper()
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repo := NewQuotaRepository(db)
	return repo, &discovery.QuotaService{Store: repo}
}

func TestQuotaReserveCharge(t *testing.T) {
	_, service := newQuota(t)
	ctx := context.Background()

	reservation, err := service.Reserve(ctx, "owner", discovery.CallSearch)
	if err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	ledger, warn, err := service.Charge(ctx, reservation.ID)
	if err != nil {
		t.Fatalf("Charge: %v", err)
	}
	if ledger.Charged != 100 || ledger.Reserved != 0 {
		t.Fatalf("ledger = %+v", ledger)
	}
	if warn {
		t.Fatal("warn = true at 1% usage")
	}
}

func TestQuotaWarnAt80(t *testing.T) {
	_, service := newQuota(t)
	service.Budget = 1000
	ctx := context.Background()

	for i := 0; i < 9; i++ {
		reservation, err := service.Reserve(ctx, "owner", discovery.CallSearch)
		if err != nil {
			t.Fatalf("Reserve %d: %v", i, err)
		}
		_, warn, err := service.Charge(ctx, reservation.ID)
		if err != nil {
			t.Fatalf("Charge %d: %v", i, err)
		}
		if i == 7 && !warn {
			t.Fatal("no warning at 80%")
		}
		if i > 7 && warn {
			t.Fatalf("warning repeated at %d%%", (i+1)*10)
		}
	}
}

func TestQuotaRefuseAt100(t *testing.T) {
	_, service := newQuota(t)
	service.Budget = 1000
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		if _, err := service.Reserve(ctx, "owner", discovery.CallSearch); err != nil {
			t.Fatalf("Reserve %d: %v", i, err)
		}
	}
	if _, err := service.Reserve(ctx, "owner", discovery.CallSearch); !errors.Is(err, discovery.ErrQuotaExhausted) {
		t.Fatalf("Reserve at 100%% = %v, want ErrQuotaExhausted", err)
	}
}

func TestQuotaRelease(t *testing.T) {
	_, service := newQuota(t)
	ctx := context.Background()

	reservation, err := service.Reserve(ctx, "owner", discovery.CallSearch)
	if err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	if err := service.Release(ctx, reservation.ID); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if _, _, err := service.Charge(ctx, reservation.ID); !errors.Is(err, discovery.ErrReservationFinal) {
		t.Fatalf("Charge after release = %v, want ErrReservationFinal", err)
	}
}

func TestQuotaDoubleCharge(t *testing.T) {
	_, service := newQuota(t)
	ctx := context.Background()

	reservation, err := service.Reserve(ctx, "owner", discovery.CallVideos)
	if err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	if _, _, err := service.Charge(ctx, reservation.ID); err != nil {
		t.Fatalf("Charge: %v", err)
	}
	ledger, warn, err := service.Charge(ctx, reservation.ID)
	if err != nil {
		t.Fatalf("Charge twice: %v", err)
	}
	if warn || ledger.Charged != 1 || ledger.Reserved != 0 {
		t.Fatalf("ledger = %+v, warn = %t", ledger, warn)
	}
}

func TestQuotaMissingReservation(t *testing.T) {
	_, service := newQuota(t)
	if _, _, err := service.Charge(context.Background(), "missing"); !errors.Is(err, discovery.ErrReservationNotFound) {
		t.Fatalf("Charge = %v, want ErrReservationNotFound", err)
	}
	if err := service.Release(context.Background(), "missing"); !errors.Is(err, discovery.ErrReservationNotFound) {
		t.Fatalf("Release = %v, want ErrReservationNotFound", err)
	}
}

func TestQuotaStale(t *testing.T) {
	repo, service := newQuota(t)
	now := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	service.TTL = time.Minute

	first, err := service.Reserve(context.Background(), "owner", discovery.CallSearch)
	if err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	now = now.Add(2 * time.Minute)
	service.Now = func() time.Time { return now }
	second, err := service.Reserve(context.Background(), "owner", discovery.CallSearch)
	if err != nil {
		t.Fatalf("Reserve after stale: %v", err)
	}
	if second.ID == first.ID {
		t.Fatal("reservation id was reused")
	}
	ledger, err := repo.GetOrCreate(context.Background(), "owner", discovery.AccountingDay(now), 10000)
	if err != nil {
		t.Fatalf("GetOrCreate: %v", err)
	}
	if ledger.Reserved != 100 {
		t.Fatalf("reserved = %d, want 100", ledger.Reserved)
	}
	if _, _, err := service.Charge(context.Background(), first.ID); !errors.Is(err, discovery.ErrReservationFinal) {
		t.Fatalf("Charge stale = %v, want ErrReservationFinal", err)
	}
}

func TestQuotaReleaseStale(t *testing.T) {
	repo, service := newQuota(t)
	now := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }
	service.TTL = time.Minute
	if _, err := service.Reserve(context.Background(), "owner", discovery.CallSearch); err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	now = now.Add(2 * time.Minute)
	service.Now = func() time.Time { return now }
	count, err := service.ReleaseStale(context.Background(), "owner")
	if err != nil {
		t.Fatalf("ReleaseStale: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	ledger, err := repo.GetOrCreate(context.Background(), "owner", discovery.AccountingDay(now), 10000)
	if err != nil {
		t.Fatalf("GetOrCreate: %v", err)
	}
	if ledger.Reserved != 0 {
		t.Fatalf("reserved = %d, want 0", ledger.Reserved)
	}
}

func TestQuotaReleaseCharged(t *testing.T) {
	_, service := newQuota(t)
	reservation, err := service.Reserve(context.Background(), "owner", discovery.CallVideos)
	if err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	if _, _, err := service.Charge(context.Background(), reservation.ID); err != nil {
		t.Fatalf("Charge: %v", err)
	}
	if err := service.Release(context.Background(), reservation.ID); !errors.Is(err, discovery.ErrReservationFinal) {
		t.Fatalf("Release = %v, want ErrReservationFinal", err)
	}
}

func TestQuotaReleaseTwice(t *testing.T) {
	_, service := newQuota(t)
	reservation, err := service.Reserve(context.Background(), "owner", discovery.CallVideos)
	if err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	if err := service.Release(context.Background(), reservation.ID); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if err := service.Release(context.Background(), reservation.ID); err != nil {
		t.Fatalf("Release twice: %v", err)
	}
}

func TestQuotaGetOrCreateIdempotent(t *testing.T) {
	repo, _ := newQuota(t)
	day := time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)
	first, err := repo.GetOrCreate(context.Background(), "owner", day, 10000)
	if err != nil {
		t.Fatalf("GetOrCreate: %v", err)
	}
	second, err := repo.GetOrCreate(context.Background(), "owner", day, 10000)
	if err != nil {
		t.Fatalf("GetOrCreate twice: %v", err)
	}
	if first.Version != second.Version || first.Budget != second.Budget {
		t.Fatalf("ledgers = %+v, %+v", first, second)
	}
}

func TestQuotaExhaustedFlag(t *testing.T) {
	repo, service := newQuota(t)
	service.Budget = 100
	reservation, err := service.Reserve(context.Background(), "owner", discovery.CallSearch)
	if err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	if _, _, err := service.Charge(context.Background(), reservation.ID); err != nil {
		t.Fatalf("Charge: %v", err)
	}
	day := discovery.AccountingDay(time.Now().UTC())
	ledger, err := repo.GetOrCreate(context.Background(), "owner", day, 100)
	if err != nil {
		t.Fatalf("GetOrCreate: %v", err)
	}
	if !ledger.Exhausted {
		t.Fatalf("ledger = %+v, want exhausted", ledger)
	}
}

func TestQuotaRepositoryErrors(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec("DROP TABLE quota_reservations"); err != nil {
		t.Fatalf("drop reservations: %v", err)
	}
	if _, err := db.Exec("DROP TABLE quota_ledgers"); err != nil {
		t.Fatalf("drop ledgers: %v", err)
	}
	repo := NewQuotaRepository(db)
	ctx := context.Background()
	day := time.Now().UTC()
	if _, err := repo.GetOrCreate(ctx, "owner", day, 10000); err == nil {
		t.Fatal("GetOrCreate = nil, want an error")
	}
	if _, err := repo.Reserve(ctx, "owner", day, discovery.Reservation{}, 10000); err == nil {
		t.Fatal("Reserve = nil, want an error")
	}
	if _, _, err := repo.Charge(ctx, "x", day); err == nil {
		t.Fatal("Charge = nil, want an error")
	}
	if err := repo.Release(ctx, "x"); err == nil {
		t.Fatal("Release = nil, want an error")
	}
	if _, err := repo.ReleaseStale(ctx, "owner", day, day); err == nil {
		t.Fatal("ReleaseStale = nil, want an error")
	}
}

func TestQuotaReleaseStaleEmpty(t *testing.T) {
	repo, _ := newQuota(t)
	now := time.Now().UTC()
	count, err := repo.ReleaseStale(context.Background(), "owner", discovery.AccountingDay(now), now)
	if err != nil {
		t.Fatalf("ReleaseStale: %v", err)
	}
	if count != 0 {
		t.Fatalf("count = %d, want 0", count)
	}
}

func TestQuotaDuplicateReservation(t *testing.T) {
	repo, _ := newQuota(t)
	now := time.Now().UTC()
	reservation := discovery.Reservation{
		ID: "dup", UserID: "owner", Day: discovery.AccountingDay(now), Units: 1,
		Call: discovery.CallVideos, State: discovery.ReservationReserved,
		CreatedAt: now, ExpiresAt: now.Add(time.Minute),
	}
	if _, err := repo.Reserve(context.Background(), "owner", reservation.Day, reservation, 10000); err != nil {
		t.Fatalf("Reserve first: %v", err)
	}
	if _, err := repo.Reserve(context.Background(), "owner", reservation.Day, reservation, 10000); err == nil {
		t.Fatal("Reserve duplicate = nil, want an error")
	}
}

func TestQuotaClosedDB(t *testing.T) {
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "ytfeed.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	_ = db.Close()
	repo := NewQuotaRepository(db)
	ctx := context.Background()
	day := time.Now().UTC()
	if _, err := repo.GetOrCreate(ctx, "owner", day, 10000); err == nil {
		t.Fatal("GetOrCreate(closed) = nil, want an error")
	}
	if _, err := repo.Reserve(ctx, "owner", day, discovery.Reservation{}, 10000); err == nil {
		t.Fatal("Reserve(closed) = nil, want an error")
	}
	if _, _, err := repo.Charge(ctx, "x", day); err == nil {
		t.Fatal("Charge(closed) = nil, want an error")
	}
	if err := repo.Release(ctx, "x"); err == nil {
		t.Fatal("Release(closed) = nil, want an error")
	}
	if _, err := repo.ReleaseStale(ctx, "owner", day, day); err == nil {
		t.Fatal("ReleaseStale(closed) = nil, want an error")
	}
}
