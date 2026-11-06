package discovery

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeQuotaStore struct {
	reservation Reservation
	ledger      Ledger
	warn        bool
	err         error
	released    string
	charged     string
}

func (f *fakeQuotaStore) GetOrCreate(context.Context, string, time.Time, int) (Ledger, error) {
	return f.ledger, f.err
}

func (f *fakeQuotaStore) Reserve(_ context.Context, _ string, _ time.Time, reservation Reservation, _ int) (Ledger, error) {
	f.reservation = reservation
	return f.ledger, f.err
}

func (f *fakeQuotaStore) Charge(_ context.Context, id string, _ time.Time) (Ledger, bool, error) {
	f.charged = id
	return f.ledger, f.warn, f.err
}

func (f *fakeQuotaStore) Release(_ context.Context, id string) error {
	f.released = id
	return f.err
}

func (f *fakeQuotaStore) ReleaseStale(context.Context, string, time.Time, time.Time) (int, error) {
	return 0, f.err
}

func TestQuotaUnits(t *testing.T) {
	tests := map[QuotaCall]int{
		CallSearch:        100,
		CallVideos:        1,
		CallChannels:      1,
		CallSubscriptions: 1,
		CallMostPopular:   1,
		CallPlaylistItems: 1,
		QuotaCall("bad"):  0,
	}
	for call, want := range tests {
		if got := Units(call); got != want {
			t.Fatalf("Units(%q) = %d, want %d", call, got, want)
		}
	}
}

func TestQuotaReserve(t *testing.T) {
	now := time.Date(2026, 11, 4, 22, 30, 0, 0, time.UTC)
	store := &fakeQuotaStore{}
	service := &QuotaService{
		Store: store,
		Now:   func() time.Time { return now },
		NewID: func() string { return "res-1" },
	}
	reservation, err := service.Reserve(context.Background(), "owner", CallSearch)
	if err != nil {
		t.Fatalf("Reserve: %v", err)
	}
	if reservation.ID != "res-1" || reservation.Units != 100 || reservation.Call != CallSearch {
		t.Fatalf("reservation = %+v", reservation)
	}
	if !reservation.Day.Equal(time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("day = %v", reservation.Day)
	}
	if !reservation.ExpiresAt.Equal(now.Add(10 * time.Minute)) {
		t.Fatalf("expiry = %v", reservation.ExpiresAt)
	}
	if store.reservation.ID != "res-1" {
		t.Fatalf("stored reservation = %+v", store.reservation)
	}
}

func TestQuotaReserveUnhappy(t *testing.T) {
	if _, err := (&QuotaService{}).Reserve(context.Background(), "owner", CallSearch); err == nil {
		t.Fatal("Reserve(nil store) = nil, want an error")
	}
	store := &fakeQuotaStore{err: ErrQuotaExhausted}
	service := &QuotaService{Store: store}
	if _, err := service.Reserve(context.Background(), "owner", CallSearch); !errors.Is(err, ErrQuotaExhausted) {
		t.Fatalf("Reserve = %v, want ErrQuotaExhausted", err)
	}
	if _, err := service.Reserve(context.Background(), "owner", QuotaCall("bad")); err == nil {
		t.Fatal("Reserve(bad call) = nil, want an error")
	}
}

func TestQuotaChargeRelease(t *testing.T) {
	store := &fakeQuotaStore{warn: true}
	service := &QuotaService{Store: store}
	if _, warn, err := service.Charge(context.Background(), "res-1"); err != nil || !warn {
		t.Fatalf("Charge = warn %t, err %v", warn, err)
	}
	if store.charged != "res-1" {
		t.Fatalf("charged = %q", store.charged)
	}
	if err := service.Release(context.Background(), "res-1"); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if store.released != "res-1" {
		t.Fatalf("released = %q", store.released)
	}
	if _, _, err := (&QuotaService{}).Charge(context.Background(), "x"); err == nil {
		t.Fatal("Charge(nil store) = nil, want an error")
	}
	if err := (&QuotaService{}).Release(context.Background(), "x"); err == nil {
		t.Fatal("Release(nil store) = nil, want an error")
	}
}

func TestAccountingDay(t *testing.T) {
	instant := time.Date(2026, 11, 4, 23, 59, 59, 0, time.FixedZone("CET", 3600))
	got := AccountingDay(instant)
	want := time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("AccountingDay = %v, want %v", got, want)
	}
}
