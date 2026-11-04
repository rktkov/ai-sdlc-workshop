package logger

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestNewProductionJSON(t *testing.T) {
	var buf bytes.Buffer
	log, err := New("production", "info", &buf)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	log.Info("hello", "key", "value")

	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("JSON log: %v: %s", err, buf.String())
	}
	if payload["msg"] != "hello" || payload["key"] != "value" {
		t.Fatalf("payload = %v", payload)
	}
}

func TestNewDevelopmentText(t *testing.T) {
	var buf bytes.Buffer
	log, err := New("development", "debug", &buf)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	log.Debug("hello", "key", "value")

	out := buf.String()
	if !strings.Contains(out, "msg=hello") || !strings.Contains(out, "key=value") {
		t.Fatalf("text log = %q", out)
	}
}

func TestNewRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		env   string
		level string
		want  string
	}{
		{name: "env", env: "staging", level: "info", want: "LOG_ENV"},
		{name: "level", env: "development", level: "trace", want: "LOG_LEVEL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := New(tt.env, tt.level, &bytes.Buffer{}); err == nil {
				t.Fatalf("New(%q, %q) = nil error", tt.env, tt.level)
			} else if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error %q does not mention %s", err, tt.want)
			}
		})
	}
}
