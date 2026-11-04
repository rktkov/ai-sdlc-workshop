package taste

import (
	"context"
	"fmt"
)

// ProfileSnapshot is the read-only profile that analysis receives.
type ProfileSnapshot struct {
	UserID        string
	Axes          []Axis
	Note          string
	RubricVersion string
}

// Snapshot returns the profile without interests or other write-side state.
func (s *Service) Snapshot(ctx context.Context, userID string) (ProfileSnapshot, error) {
	if s.Profiles == nil {
		return ProfileSnapshot{}, fmt.Errorf("profile reader is nil")
	}
	profile, err := s.Profiles.Get(ctx, userID)
	if err != nil {
		return ProfileSnapshot{}, err
	}
	axes := make([]Axis, len(profile.Axes))
	copy(axes, profile.Axes)
	return ProfileSnapshot{
		UserID:        profile.UserID,
		Axes:          axes,
		Note:          profile.Note,
		RubricVersion: profile.RubricVersion,
	}, nil
}
