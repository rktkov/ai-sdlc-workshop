package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/rktkov/ytfeed/internal/transport/contract"
)

func TestHealthHappy(t *testing.T) {
	handler := Handler("secret", "dev", 1)
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var health contract.Health
	if err := json.Unmarshal(rec.Body.Bytes(), &health); err != nil {
		t.Fatalf("decode health: %v", err)
	}
	if err := health.Validate(); err != nil {
		t.Fatalf("health contract: %v", err)
	}
}

func TestHealthUnhappy(t *testing.T) {
	handler := Handler("secret", "dev", 1)

	t.Run("missing token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("wrong token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		req.Header.Set("Authorization", "Bearer wrong")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})

	t.Run("non loopback", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		req.Header.Set("Authorization", "Bearer secret")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
		}
	})

	t.Run("wrong method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/health", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		req.Header.Set("Authorization", "Bearer secret")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})
}

func TestLoadOrCreateToken(t *testing.T) {
	dir := t.TempDir()
	token, err := LoadOrCreateToken(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateToken: %v", err)
	}
	if token == "" {
		t.Fatal("token is empty")
	}

	info, err := os.Stat(filepath.Join(dir, "api-token"))
	if err != nil {
		t.Fatalf("stat token: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("token mode = %o, want 600", info.Mode().Perm())
	}

	again, err := LoadOrCreateToken(dir)
	if err != nil {
		t.Fatalf("LoadOrCreateToken second: %v", err)
	}
	if again != token {
		t.Fatal("second call returned a different token")
	}
}

func TestAllowRemote(t *testing.T) {
	handler := HandlerWithRemote("secret", "dev", 1, true)
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
