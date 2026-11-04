package testkit

import (
	"os/exec"
	"testing"
	"time"
)

func TestFixedClock(t *testing.T) {
	start := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	clock := NewFixedClock(start)

	if got := clock.Now(); !got.Equal(start) {
		t.Fatalf("Now() = %v, want %v", got, start)
	}

	clock.Advance(2 * time.Hour)
	want := start.Add(2 * time.Hour)
	if got := clock.Now(); !got.Equal(want) {
		t.Fatalf("Now() after advance = %v, want %v", got, want)
	}
}

func TestTempDataDir(t *testing.T) {
	dir := TempDataDir(t)
	if dir == "" {
		t.Fatal("TempDataDir returned an empty path")
	}
}

func TestWriteScript(t *testing.T) {
	dir := t.TempDir()
	path := WriteScript(t, dir, "fake.sh", "#!/bin/sh\nprintf 'ok'\n")

	out, err := exec.Command(path).CombinedOutput()
	if err != nil {
		t.Fatalf("run script: %v: %s", err, out)
	}
	if string(out) != "ok" {
		t.Fatalf("script output = %q, want %q", out, "ok")
	}
}
