// Package trace writes local JSONL spans with daily rotation.
package trace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Span is one local trace span.
type Span struct {
	TraceID    string         `json:"trace_id"`
	SpanID     string         `json:"span_id"`
	Name       string         `json:"name"`
	StartedAt  time.Time      `json:"started_at"`
	EndedAt    time.Time      `json:"ended_at"`
	DurationMS int64          `json:"duration_ms"`
	Attributes map[string]any `json:"attributes,omitempty"`
	Error      string         `json:"error,omitempty"`

	logger *Logger
}

// Logger writes spans to a directory.
type Logger struct {
	Dir       string
	Now       func() time.Time
	Retention time.Duration
}

// New returns a trace logger under dataDir/traces.
func New(dataDir string) *Logger {
	return &Logger{Dir: filepath.Join(dataDir, "traces"), Retention: 7 * 24 * time.Hour}
}

// Start begins one span.
func (l *Logger) Start(ctx context.Context, name string, attributes map[string]any) *Span {
	now := l.now()
	return &Span{
		TraceID:    traceID(ctx),
		SpanID:     randomID(8),
		Name:       name,
		StartedAt:  now,
		Attributes: sanitize(attributes),
		logger:     l,
	}
}

// End writes the span. It does not return an error; tracing must not break the product.
func (s *Span) End(err error) {
	if s == nil || s.logger == nil {
		return
	}
	now := s.logger.now()
	s.EndedAt = now
	s.DurationMS = now.Sub(s.StartedAt).Milliseconds()
	if err != nil {
		s.Error = err.Error()
	}
	_ = s.logger.write(s)
}

func (l *Logger) write(span *Span) error {
	if err := os.MkdirAll(l.Dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(l.Dir, 0o700); err != nil {
		return err
	}
	path := filepath.Join(l.Dir, span.StartedAt.UTC().Format("2006-01-02")+".jsonl")
	data, err := json.Marshal(span)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	if _, err := file.Write(append(data, '\n')); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

// Sweep removes trace files older than the retention window.
func (l *Logger) Sweep(ctx context.Context) (int, error) {
	_ = ctx
	entries, err := os.ReadDir(l.Dir)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read trace directory: %w", err)
	}
	cutoff := l.now().Add(-l.retention())
	removed := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(filepath.Join(l.Dir, entry.Name())); err == nil {
				removed++
			}
		}
	}
	return removed, nil
}

func (l *Logger) now() time.Time {
	if l.Now == nil {
		return time.Now().UTC()
	}
	return l.Now().UTC()
}

func (l *Logger) retention() time.Duration {
	if l.Retention <= 0 {
		return 7 * 24 * time.Hour
	}
	return l.Retention
}

func sanitize(attributes map[string]any) map[string]any {
	if len(attributes) == 0 {
		return nil
	}
	out := make(map[string]any, len(attributes))
	for key, value := range attributes {
		switch strings.ToLower(key) {
		case "token", "access_token", "refresh_token", "api_key", "authorization", "prompt", "transcript", "content":
			continue
		default:
			out[key] = value
		}
	}
	return out
}

func traceID(ctx context.Context) string {
	if value, ok := ctx.Value(traceIDKey{}).(string); ok && value != "" {
		return value
	}
	return randomID(16)
}

type traceIDKey struct{}

// WithTraceID attaches a trace ID to a context.
func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceIDKey{}, id)
}

func randomID(bytes int) string {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
