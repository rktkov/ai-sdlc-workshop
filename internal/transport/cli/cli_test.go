package cli

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/rktkov/ytfeed/internal/transport/http"
)

func TestDoctorHappy(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "api-token"), []byte("secret"), 0o600); err != nil {
		t.Fatalf("write token: %v", err)
	}
	server := httptest.NewServer(httpapi.Handler("secret", "dev", 1))
	defer server.Close()

	var out, errOut bytes.Buffer
	code := Run(context.Background(), []string{"doctor"}, env(dir), &out, &errOut, server.URL)
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("ok")) {
		t.Fatalf("stdout = %q", out.String())
	}
}

func TestDoctorUnhappy(t *testing.T) {
	t.Run("wrong token", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "api-token"), []byte("wrong"), 0o600); err != nil {
			t.Fatalf("write token: %v", err)
		}
		server := httptest.NewServer(httpapi.Handler("secret", "dev", 1))
		defer server.Close()

		var out, errOut bytes.Buffer
		code := Run(context.Background(), []string{"doctor"}, env(dir), &out, &errOut, server.URL)
		if code != 2 {
			t.Fatalf("exit = %d, want 2; stderr = %s", code, errOut.String())
		}
	})

	t.Run("server down", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "api-token"), []byte("secret"), 0o600); err != nil {
			t.Fatalf("write token: %v", err)
		}

		var out, errOut bytes.Buffer
		code := Run(context.Background(), []string{"doctor"}, env(dir), &out, &errOut, "http://127.0.0.1:1")
		if code != 0 {
			t.Fatalf("exit = %d, want 0; stderr = %s", code, errOut.String())
		}
		if !bytes.Contains(out.Bytes(), []byte("server: unavailable")) {
			t.Fatalf("stdout = %q", out.String())
		}
	})
}

func TestVersionAndHelp(t *testing.T) {
	for _, args := range [][]string{{"--version"}, {"help"}} {
		var out, errOut bytes.Buffer
		if code := Run(context.Background(), args, env(t.TempDir()), &out, &errOut, ""); code != 0 {
			t.Fatalf("%v exit = %d; stderr = %s", args, code, errOut.String())
		}
		if out.Len() == 0 {
			t.Fatalf("%v produced no output", args)
		}
	}
}

func TestUnknownCommand(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := Run(context.Background(), []string{"nope"}, env(t.TempDir()), &out, &errOut, ""); code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
}

func env(dir string) func(string) string {
	return func(key string) string {
		if key == "YTFEED_DATA_DIR" {
			return dir
		}
		return ""
	}
}

func TestServeHappy(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "api-token"), []byte("secret"), 0o600); err != nil {
		t.Fatalf("write token: %v", err)
	}

	var errOut bytes.Buffer
	code := runServeWith(context.Background(), nil, env(dir), &errOut, func(*http.Server) error { return nil })
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut.String())
	}
}

func TestServeAllowRemote(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "api-token"), []byte("secret"), 0o600); err != nil {
		t.Fatalf("write token: %v", err)
	}

	var errOut bytes.Buffer
	code := runServeWith(context.Background(), []string{"--allow-remote"}, env(dir), &errOut, func(*http.Server) error { return nil })
	if code != 0 {
		t.Fatalf("exit = %d, stderr = %s", code, errOut.String())
	}
	if !bytes.Contains(errOut.Bytes(), []byte("warning")) {
		t.Fatalf("stderr = %q, want a warning", errOut.String())
	}
}

func TestServeUnhappy(t *testing.T) {
	t.Run("unknown argument", func(t *testing.T) {
		var errOut bytes.Buffer
		if code := runServeWith(context.Background(), []string{"--bad"}, env(t.TempDir()), &errOut, func(*http.Server) error { return nil }); code != 2 {
			t.Fatalf("exit = %d, want 2", code)
		}
	})

	t.Run("invalid config", func(t *testing.T) {
		var errOut bytes.Buffer
		code := runServeWith(context.Background(), nil, func(key string) string {
			if key == "LOG_ENV" {
				return "staging"
			}
			return ""
		}, &errOut, func(*http.Server) error { return nil })
		if code != 2 {
			t.Fatalf("exit = %d, want 2", code)
		}
	})

	t.Run("unwritable data dir", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "not-a-dir")
		if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
			t.Fatalf("write file: %v", err)
		}

		var errOut bytes.Buffer
		code := runServeWith(context.Background(), nil, env(file), &errOut, func(*http.Server) error { return nil })
		if code != 1 {
			t.Fatalf("exit = %d, want 1", code)
		}
	})
}

func TestDoctorInvalidConfig(t *testing.T) {
	var out, errOut bytes.Buffer
	code := Run(context.Background(), []string{"doctor"}, func(key string) string {
		if key == "LOG_ENV" {
			return "staging"
		}
		return ""
	}, &out, &errOut, "http://127.0.0.1:1")
	if code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
}

func TestDoctorBadBaseURL(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "api-token"), []byte("secret"), 0o600); err != nil {
		t.Fatalf("write token: %v", err)
	}

	var out, errOut bytes.Buffer
	if code := Run(context.Background(), []string{"doctor"}, env(dir), &out, &errOut, "://bad"); code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
}

func TestDoctorServerError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "api-token"), []byte("secret"), 0o600); err != nil {
		t.Fatalf("write token: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	var out, errOut bytes.Buffer
	if code := Run(context.Background(), []string{"doctor"}, env(dir), &out, &errOut, server.URL); code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
}

func TestDoctorInvalidHealth(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "api-token"), []byte("secret"), 0o600); err != nil {
		t.Fatalf("write token: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	var out, errOut bytes.Buffer
	if code := Run(context.Background(), []string{"doctor"}, env(dir), &out, &errOut, server.URL); code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
}

func TestDoctorTokenError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "api-token"), []byte("secret"), 0o644); err != nil {
		t.Fatalf("write token: %v", err)
	}

	var out, errOut bytes.Buffer
	if code := Run(context.Background(), []string{"doctor"}, env(dir), &out, &errOut, "http://127.0.0.1:1"); code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
}

func TestRunNoArgs(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := Run(context.Background(), nil, env(t.TempDir()), &out, &errOut, ""); code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
}
