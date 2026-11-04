// Package logger builds the structured logger for the application.
package logger

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// New returns a logger for the given environment and level.
// Production writes JSON. Development writes text.
func New(env, level string, w io.Writer) (*slog.Logger, error) {
	if w == nil {
		return nil, fmt.Errorf("logger writer must not be nil")
	}

	parsedLevel, err := parseLevel(level)
	if err != nil {
		return nil, err
	}

	var handler slog.Handler
	options := &slog.HandlerOptions{Level: parsedLevel}
	switch strings.TrimSpace(env) {
	case "production":
		handler = slog.NewJSONHandler(w, options)
	case "development":
		handler = slog.NewTextHandler(w, options)
	default:
		return nil, fmt.Errorf("LOG_ENV must be development or production")
	}
	return slog.New(handler), nil
}

func parseLevel(level string) (slog.Level, error) {
	switch strings.TrimSpace(level) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("LOG_LEVEL must be debug, info, warn or error")
	}
}
