package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rktkov/ytfeed/internal/domain/identity"
)

func TestAuthLoginHappy(t *testing.T) {
	dir := t.TempDir()
	clientPath := writeAuthClient(t, dir)
	env := authEnv(clientPath, dir)

	var out, errOut bytes.Buffer
	code := runAuthLoginWith(context.Background(), nil, env, &out, &errOut, func(context.Context, identity.Service) (identity.Token, error) {
		return identity.Token{
			AccessToken:  "access-secret",
			RefreshToken: "refresh-secret",
			TokenType:    "Bearer",
			Scope:        identity.ScopeYouTubeReadOnly,
			ExpiresAt:    time.Date(2026, 11, 4, 13, 0, 0, 0, time.UTC),
		}, nil
	})
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "authorized") {
		t.Fatalf("stdout = %q", out.String())
	}
	if strings.Contains(out.String(), "access-secret") || strings.Contains(out.String(), "refresh-secret") {
		t.Fatalf("stdout leaks the token: %q", out.String())
	}
}

func TestAuthLoginRefusal(t *testing.T) {
	dir := t.TempDir()
	clientPath := writeAuthClient(t, dir)
	env := authEnv(clientPath, dir)

	tests := []struct {
		name string
		err  error
		want int
		text string
	}{
		{name: "consent denied", err: identity.ErrConsentDenied, want: 2, text: "denied"},
		{name: "state mismatch", err: identity.ErrStateMismatch, want: 2, text: "state"},
		{name: "token rejected", err: identity.ErrUnauthorized, want: 2, text: "rejected"},
		{name: "timeout", err: identity.ErrTimeout, want: 2, text: "timed out"},
		{name: "unexpected", err: errors.New("boom"), want: 1, text: "failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			code := runAuthLoginWith(context.Background(), nil, env, &out, &errOut, func(context.Context, identity.Service) (identity.Token, error) {
				return identity.Token{}, tt.err
			})
			if code != tt.want {
				t.Fatalf("exit = %d, want %d; stderr = %s", code, tt.want, errOut.String())
			}
			if !strings.Contains(errOut.String(), tt.text) {
				t.Fatalf("stderr = %q, want %q", errOut.String(), tt.text)
			}
			if strings.Contains(out.String(), "authorized") {
				t.Fatalf("stdout = %q", out.String())
			}
		})
	}
}

func TestAuthLoginMissingClientFile(t *testing.T) {
	var out, errOut bytes.Buffer
	code := runAuthLoginWith(context.Background(), nil, func(string) string { return "" }, &out, &errOut, func(context.Context, identity.Service) (identity.Token, error) {
		t.Fatal("authorize was called without a client file")
		return identity.Token{}, nil
	})
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	if !strings.Contains(errOut.String(), "GOOGLE_OAUTH_CLIENT_FILE") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestAuthLoginInvalidClientFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "client.json")
	if err := os.WriteFile(path, []byte(`{`), 0o600); err != nil {
		t.Fatalf("write client file: %v", err)
	}

	var out, errOut bytes.Buffer
	code := runAuthLoginWith(context.Background(), nil, authEnv(path, dir), &out, &errOut, func(context.Context, identity.Service) (identity.Token, error) {
		t.Fatal("authorize was called with an invalid client file")
		return identity.Token{}, nil
	})
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
}

func TestAuthUsage(t *testing.T) {
	for _, args := range [][]string{{"auth"}, {"auth", "nope"}} {
		var out, errOut bytes.Buffer
		if code := Run(context.Background(), args, func(string) string { return "" }, &out, &errOut, ""); code != 2 {
			t.Fatalf("%v exit = %d, want 2", args, code)
		}
	}
}

func writeAuthClient(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "client.json")
	body := `{"installed":{"client_id":"id","client_secret":"secret","auth_uri":"https://auth.example.test/auth","token_uri":"https://token.example.test/token"}}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write client file: %v", err)
	}
	return path
}

func authEnv(clientPath, dir string) func(string) string {
	return func(key string) string {
		switch key {
		case "GOOGLE_OAUTH_CLIENT_FILE":
			return clientPath
		case "YTFEED_DATA_DIR":
			return dir
		default:
			return ""
		}
	}
}
