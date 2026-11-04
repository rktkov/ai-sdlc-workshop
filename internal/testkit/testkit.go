// Package testkit provides deterministic helpers for tests.
// It contains no product behavior.
package testkit

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// FixedClock is a clock that only moves when a test advances it.
type FixedClock struct {
	current time.Time
}

// NewFixedClock returns a clock at the given instant.
func NewFixedClock(at time.Time) *FixedClock {
	return &FixedClock{current: at}
}

// Now returns the current instant.
func (c *FixedClock) Now() time.Time {
	return c.current
}

// Advance moves the clock forward.
func (c *FixedClock) Advance(d time.Duration) {
	c.current = c.current.Add(d)
}

// TempDataDir returns an empty temporary data directory for one test.
func TempDataDir(t testing.TB) string {
	t.Helper()
	return t.TempDir()
}

// WriteScript writes an executable script and returns its path.
func WriteScript(t testing.TB, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o700); err != nil {
		t.Fatalf("write script %s: %v", name, err)
	}
	return path
}
