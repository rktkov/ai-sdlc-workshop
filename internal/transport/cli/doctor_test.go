package cli

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rktkov/ytfeed/internal/config"
	"github.com/rktkov/ytfeed/internal/domain/identity"
	"github.com/rktkov/ytfeed/internal/repository/tokenfile"
	"github.com/rktkov/ytfeed/internal/transport/contract"
)

func TestDoctorLocalReport(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	if err := tokenfile.New(dir).Write(context.Background(), identity.Token{
		AccessToken: "access", ExpiresAt: now.Add(time.Hour),
	}); err != nil {
		t.Fatalf("write token: %v", err)
	}
	deps := doctorDeps{
		Now:            func() time.Time { return now },
		YtdlpVersion:   func() (string, error) { return "2026.01.01", nil },
		LLMReachable:   func(context.Context, config.Config) error { return nil },
		TerminalImages: func() bool { return true },
		Health: func(context.Context, string, string) (contract.Health, int, error) {
			return contract.Health{Status: "ok", Version: "dev", SchemaVersion: 13}, http.StatusOK, nil
		},
	}
	var out, errOut bytes.Buffer
	code := runDoctorWith(context.Background(), env(dir), &out, &errOut, "", deps)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut.String())
	}
	for _, want := range []string{
		"data_dir:", "db:", "schema: 13", "api_token: present", "token: active",
		"quota: charged=0 budget=10000", "yt-dlp: 2026.01.01",
		"llm: reachable model=", "terminal_images: yes", "server: ok",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, out.String())
		}
	}
	if strings.Contains(out.String(), "access") {
		t.Fatalf("stdout leaks the token: %s", out.String())
	}
}

func TestDoctorLocalUnhappy(t *testing.T) {
	dir := t.TempDir()
	deps := doctorDeps{
		YtdlpVersion:   func() (string, error) { return "", errors.New("missing") },
		LLMReachable:   func(context.Context, config.Config) error { return errors.New("down") },
		TerminalImages: func() bool { return false },
		Health: func(context.Context, string, string) (contract.Health, int, error) {
			return contract.Health{}, 0, errors.New("down")
		},
	}
	var out, errOut bytes.Buffer
	if code := runDoctorWith(context.Background(), env(dir), &out, &errOut, "", deps); code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	for _, want := range []string{"token: missing", "yt-dlp: missing", "llm: unreachable", "terminal_images: no", "server: unavailable"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, out.String())
		}
	}
}

func TestDoctorUnreadableDB(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	var out, errOut bytes.Buffer
	if code := runDoctorWith(context.Background(), func(key string) string {
		if key == "YTFEED_DATA_DIR" {
			return filepath.Join(file, "data")
		}
		return ""
	}, &out, &errOut, "", doctorDeps{}); code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
}

func TestDoctorServerStates(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name   string
		health contract.Health
		status int
		want   int
		text   string
	}{
		{name: "unauthorized", status: http.StatusUnauthorized, want: 2, text: "server: unavailable"},
		{name: "invalid", health: contract.Health{}, status: http.StatusOK, want: 1, text: "server: invalid"},
		{name: "schema mismatch", health: contract.Health{Status: "ok", Version: "dev", SchemaVersion: 9}, status: http.StatusOK, want: 0, text: "local schema mismatch"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := doctorDeps{Health: func(context.Context, string, string) (contract.Health, int, error) {
				return tt.health, tt.status, nil
			}}
			var out, errOut bytes.Buffer
			code := runDoctorWith(context.Background(), env(dir), &out, &errOut, "", deps)
			if code != tt.want {
				t.Fatalf("exit = %d, want %d", code, tt.want)
			}
			if !strings.Contains(out.String(), tt.text) {
				t.Fatalf("stdout = %q, missing %q", out.String(), tt.text)
			}
		})
	}
}
