package feed

import (
	"context"
	"errors"
	"time"
)

// QuotaPoint is one day of quota usage.
type QuotaPoint struct {
	Day     string
	Charged int
}

// Stats is the read-only feed projection.
type Stats struct {
	Swipes      map[string]int
	Reasons     map[string]int
	ColdActive  int
	ColdExpired int
	Quota       []QuotaPoint
	Topics      map[string]int
}

// StatsRepository reads persisted feed data.
type StatsRepository interface {
	Swipes(ctx context.Context, userID string) (map[string]int, error)
	Reasons(ctx context.Context, userID string) (map[string]int, error)
	ColdPeriods(ctx context.Context, userID string, now time.Time) (int, int, error)
	QuotaTrend(ctx context.Context, userID string, since time.Time) ([]QuotaPoint, error)
	Topics(ctx context.Context, userID string) (map[string]int, error)
}

// StatsService computes the stats projection.
type StatsService struct {
	Repo StatsRepository
	Now  func() time.Time
}

// Get returns one read-only stats snapshot.
func (s *StatsService) Get(ctx context.Context, userID string, days int) (Stats, error) {
	if s.Repo == nil {
		return Stats{}, errors.New("stats repository is nil")
	}
	if userID == "" {
		return Stats{}, errors.New("stats has no user")
	}
	if days <= 0 {
		days = 7
	}
	if days > 90 {
		days = 90
	}
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	swipes, err := s.Repo.Swipes(ctx, userID)
	if err != nil {
		return Stats{}, err
	}
	reasons, err := s.Repo.Reasons(ctx, userID)
	if err != nil {
		return Stats{}, err
	}
	active, expired, err := s.Repo.ColdPeriods(ctx, userID, now)
	if err != nil {
		return Stats{}, err
	}
	quota, err := s.Repo.QuotaTrend(ctx, userID, now.AddDate(0, 0, -days))
	if err != nil {
		return Stats{}, err
	}
	topics, err := s.Repo.Topics(ctx, userID)
	if err != nil {
		return Stats{}, err
	}
	return Stats{
		Swipes: swipes, Reasons: reasons, ColdActive: active, ColdExpired: expired,
		Quota: quota, Topics: topics,
	}, nil
}
