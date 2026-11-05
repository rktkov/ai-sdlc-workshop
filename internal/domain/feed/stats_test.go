package feed

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeStatsRepo struct {
	swipes  map[string]int
	reasons map[string]int
	active  int
	expired int
	quota   []QuotaPoint
	topics  map[string]int
	err     error
}

func (f *fakeStatsRepo) Swipes(context.Context, string) (map[string]int, error) {
	return f.swipes, f.err
}
func (f *fakeStatsRepo) Reasons(context.Context, string) (map[string]int, error) {
	return f.reasons, f.err
}
func (f *fakeStatsRepo) ColdPeriods(context.Context, string, time.Time) (int, int, error) {
	return f.active, f.expired, f.err
}
func (f *fakeStatsRepo) QuotaTrend(context.Context, string, time.Time) ([]QuotaPoint, error) {
	return f.quota, f.err
}
func (f *fakeStatsRepo) Topics(context.Context, string) (map[string]int, error) {
	return f.topics, f.err
}

func TestStatsGet(t *testing.T) {
	repo := &fakeStatsRepo{
		swipes: map[string]int{"left": 2}, reasons: map[string]int{"marketing": 1},
		active: 1, expired: 2, quota: []QuotaPoint{{Day: "2026-11-05", Charged: 100}},
		topics: map[string]int{"ai": 3},
	}
	service := &StatsService{Repo: repo}
	stats, err := service.Get(context.Background(), "owner", 0)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if stats.Swipes["left"] != 2 || stats.ColdActive != 1 || stats.Topics["ai"] != 3 {
		t.Fatalf("stats = %+v", stats)
	}
}

func TestStatsUnhappy(t *testing.T) {
	if _, err := (&StatsService{}).Get(context.Background(), "owner", 7); err == nil {
		t.Fatal("Get(nil repo) = nil, want an error")
	}
	if _, err := (&StatsService{Repo: &fakeStatsRepo{}}).Get(context.Background(), "", 7); err == nil {
		t.Fatal("Get(no user) = nil, want an error")
	}
	if _, err := (&StatsService{Repo: &fakeStatsRepo{err: errors.New("db")}}).Get(context.Background(), "owner", 7); err == nil {
		t.Fatal("Get(error) = nil, want an error")
	}
}
