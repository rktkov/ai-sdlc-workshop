package trace

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSpanWritesJSONL(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	logger := &Logger{Dir: filepath.Join(dir, "traces"), Now: func() time.Time { return now }}
	span := logger.Start(context.Background(), "feed.compose", map[string]any{
		"batch_id": "b1", "token": "secret", "transcript": "text", "prompt": "hidden",
	})
	now = now.Add(250 * time.Millisecond)
	logger.Now = func() time.Time { return now }
	span.End(nil)

	path := filepath.Join(dir, "traces", "2026-11-05.jsonl")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = file.Close() }()
	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		t.Fatal("no span line")
	}
	var value Span
	if err := json.Unmarshal(scanner.Bytes(), &value); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if value.Name != "feed.compose" || value.DurationMS != 250 {
		t.Fatalf("span = %+v", value)
	}
	if _, ok := value.Attributes["token"]; ok {
		t.Fatal("token attribute was written")
	}
	if _, ok := value.Attributes["transcript"]; ok {
		t.Fatal("transcript attribute was written")
	}
	if value.Attributes["batch_id"] != "b1" {
		t.Fatalf("attributes = %+v", value.Attributes)
	}
}

func TestSpanError(t *testing.T) {
	logger := &Logger{Dir: filepath.Join(t.TempDir(), "traces")}
	span := logger.Start(context.Background(), "op", nil)
	span.End(os.ErrNotExist)
	if span.Error == "" {
		t.Fatal("error was not recorded")
	}
	(&Span{}).End(nil)
}

func TestSweep(t *testing.T) {
	dir := t.TempDir()
	logger := &Logger{Dir: filepath.Join(dir, "traces"), Retention: 24 * time.Hour}
	if err := os.MkdirAll(logger.Dir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	old := filepath.Join(logger.Dir, "2026-10-01.jsonl")
	if err := os.WriteFile(old, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("write old: %v", err)
	}
	oldTime := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(old, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}
	recent := filepath.Join(logger.Dir, "2026-11-05.jsonl")
	if err := os.WriteFile(recent, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("write recent: %v", err)
	}
	removed, err := logger.Sweep(context.Background())
	if err != nil {
		t.Fatalf("Sweep: %v", err)
	}
	if removed != 1 {
		t.Fatalf("removed = %d, want 1", removed)
	}
	if _, err := os.Stat(recent); err != nil {
		t.Fatalf("recent file missing: %v", err)
	}
}

func TestSweepMissing(t *testing.T) {
	logger := &Logger{Dir: filepath.Join(t.TempDir(), "missing")}
	removed, err := logger.Sweep(context.Background())
	if err != nil || removed != 0 {
		t.Fatalf("Sweep = %d, %v", removed, err)
	}
}

func TestTraceIDContext(t *testing.T) {
	ctx := WithTraceID(context.Background(), "trace-1")
	logger := &Logger{Dir: filepath.Join(t.TempDir(), "traces")}
	span := logger.Start(ctx, "op", nil)
	if span.TraceID != "trace-1" {
		t.Fatalf("trace id = %q", span.TraceID)
	}
}
