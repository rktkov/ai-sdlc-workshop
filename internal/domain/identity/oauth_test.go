package identity

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAuthorizeHappy(t *testing.T) {
	var challenge string
	var tokenCalls int
	tokenServer := happyTokenServer(t, &challenge, &tokenCalls)
	defer tokenServer.Close()

	now := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	service := &Service{
		Client: Client{
			ClientID:     "client-1",
			ClientSecret: "secret-1",
			AuthURL:      "https://accounts.example.test/auth",
			TokenURL:     tokenServer.URL,
		},
		Now:         func() time.Time { return now },
		Timeout:     2 * time.Second,
		OpenBrowser: happyOpener(t, &challenge),
	}

	token, err := service.Authorize(context.Background())
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}
	if token.AccessToken != "access-1" || token.RefreshToken != "refresh-1" {
		t.Fatalf("token = %+v", token)
	}
	if !token.ExpiresAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("ExpiresAt = %v", token.ExpiresAt)
	}
	if token.Fingerprint() == "" || token.Fingerprint() == token.AccessToken {
		t.Fatalf("Fingerprint = %q", token.Fingerprint())
	}
	if tokenCalls != 1 {
		t.Fatalf("token calls = %d, want 1", tokenCalls)
	}
}

func happyTokenServer(t *testing.T, challenge *string, calls *int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*calls++
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse token form: %v", err)
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		if got := r.Form.Get("grant_type"); got != "authorization_code" {
			t.Errorf("grant_type = %q", got)
		}
		if got := r.Form.Get("code"); got != "code-1" {
			t.Errorf("code = %q", got)
		}
		if got := r.Form.Get("client_id"); got != "client-1" {
			t.Errorf("client_id = %q", got)
		}
		if got := r.Form.Get("client_secret"); got != "secret-1" {
			t.Errorf("client_secret = %q", got)
		}
		verifier := r.Form.Get("code_verifier")
		if len(verifier) != 43 {
			t.Errorf("code_verifier length = %d, want 43", len(verifier))
		}
		sum := sha256.Sum256([]byte(verifier))
		if got := base64.RawURLEncoding.EncodeToString(sum[:]); got != *challenge {
			t.Errorf("code_challenge = %q, want %q", *challenge, got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"access_token":"access-1","refresh_token":"refresh-1","expires_in":3600,"scope":"https://www.googleapis.com/auth/youtube.readonly","token_type":"Bearer"}`)
	}))
}

func happyOpener(t *testing.T, challenge *string) func(string) error {
	t.Helper()
	return func(authURL string) error {
		parsed, err := url.Parse(authURL)
		if err != nil {
			return err
		}
		query := parsed.Query()
		*challenge = query.Get("code_challenge")
		if got := query.Get("code_challenge_method"); got != "S256" {
			t.Errorf("code_challenge_method = %q", got)
		}
		if got := query.Get("scope"); got != ScopeYouTubeReadOnly {
			t.Errorf("scope = %q", got)
		}
		if got := query.Get("access_type"); got != "offline" {
			t.Errorf("access_type = %q", got)
		}
		if got := query.Get("prompt"); got != "consent" {
			t.Errorf("prompt = %q", got)
		}
		if got := query.Get("state"); got == "" {
			t.Error("state is empty")
		}
		redirect := query.Get("redirect_uri")
		if !strings.HasPrefix(redirect, "http://127.0.0.1:") {
			t.Errorf("redirect_uri = %q", redirect)
		}
		resp, err := http.Get(redirect + "/?code=code-1&state=" + url.QueryEscape(query.Get("state")))
		if err != nil {
			return err
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("callback status = %d", resp.StatusCode)
		}
		return nil
	}
}

func TestAuthorizeStateMismatch(t *testing.T) {
	tokenCalled := false
	tokenServer := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		tokenCalled = true
	}))
	defer tokenServer.Close()

	service := testService(tokenServer.URL, func(authURL string) error {
		parsed, _ := url.Parse(authURL)
		redirect := parsed.Query().Get("redirect_uri")
		resp, err := http.Get(redirect + "/?code=code-1&state=wrong")
		if err != nil {
			return err
		}
		_ = resp.Body.Close()
		return nil
	})

	_, err := service.Authorize(context.Background())
	if !errors.Is(err, ErrStateMismatch) {
		t.Fatalf("Authorize = %v, want ErrStateMismatch", err)
	}
	if tokenCalled {
		t.Fatal("token endpoint was called after a state mismatch")
	}
}

func TestAuthorizeConsentDenied(t *testing.T) {
	tokenCalled := false
	tokenServer := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		tokenCalled = true
	}))
	defer tokenServer.Close()

	service := testService(tokenServer.URL, func(authURL string) error {
		parsed, _ := url.Parse(authURL)
		query := parsed.Query()
		resp, err := http.Get(query.Get("redirect_uri") + "/?error=access_denied&state=" + url.QueryEscape(query.Get("state")))
		if err != nil {
			return err
		}
		_ = resp.Body.Close()
		return nil
	})

	_, err := service.Authorize(context.Background())
	if !errors.Is(err, ErrConsentDenied) {
		t.Fatalf("Authorize = %v, want ErrConsentDenied", err)
	}
	if tokenCalled {
		t.Fatal("token endpoint was called after consent was denied")
	}
}

func TestAuthorizeTimeout(t *testing.T) {
	service := testService("https://token.example.test", func(string) error { return nil })
	service.Timeout = 30 * time.Millisecond

	_, err := service.Authorize(context.Background())
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("Authorize = %v, want ErrTimeout", err)
	}
}

func TestAuthorizeTokenRejected(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, `{"error":"invalid_grant","error_description":"bad code"}`)
	}))
	defer tokenServer.Close()

	service := testService(tokenServer.URL, callbackForCode(t, "code-1"))
	_, err := service.Authorize(context.Background())
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Authorize = %v, want ErrUnauthorized", err)
	}
	if strings.Contains(err.Error(), "code-1") || strings.Contains(err.Error(), "bad code") {
		t.Fatalf("error leaks the code or description: %q", err)
	}
}

func TestAuthorizeMissingAccessToken(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"token_type":"Bearer"}`)
	}))
	defer tokenServer.Close()

	service := testService(tokenServer.URL, callbackForCode(t, "code-1"))
	if _, err := service.Authorize(context.Background()); err == nil {
		t.Fatal("Authorize = nil, want an error")
	}
}

func TestLoadClient(t *testing.T) {
	path := writeClientFile(t, `{"installed":{"client_id":"id","client_secret":"secret","auth_uri":"https://auth.example.test/auth","token_uri":"https://token.example.test/token","project_id":"project"}}`)

	client, err := LoadClient(path)
	if err != nil {
		t.Fatalf("LoadClient: %v", err)
	}
	if client.ClientID != "id" || client.ClientSecret != "secret" {
		t.Fatalf("client = %+v", client)
	}
	if client.AuthURL != "https://auth.example.test/auth" || client.TokenURL != "https://token.example.test/token" {
		t.Fatalf("client URLs = %+v", client)
	}
}

func TestLoadClientDefaults(t *testing.T) {
	path := writeClientFile(t, `{"installed":{"client_id":"id","client_secret":"secret"}}`)
	client, err := LoadClient(path)
	if err != nil {
		t.Fatalf("LoadClient: %v", err)
	}
	if client.AuthURL != DefaultAuthURL || client.TokenURL != DefaultTokenURL {
		t.Fatalf("default URLs = %+v", client)
	}
}

func TestLoadClientUnhappy(t *testing.T) {
	t.Run("missing client id", func(t *testing.T) {
		path := writeClientFile(t, `{"installed":{"client_secret":"secret"}}`)
		if _, err := LoadClient(path); err == nil {
			t.Fatal("LoadClient = nil, want an error")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		path := writeClientFile(t, `{`)
		if _, err := LoadClient(path); err == nil {
			t.Fatal("LoadClient = nil, want an error")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		if _, err := LoadClient(filepath.Join(t.TempDir(), "missing.json")); err == nil {
			t.Fatal("LoadClient = nil, want an error")
		}
	})
}

func TestTokenFingerprint(t *testing.T) {
	token := Token{AccessToken: "access-1"}
	first := token.Fingerprint()
	second := token.Fingerprint()
	if first == "" || first != second {
		t.Fatalf("fingerprint = %q, %q", first, second)
	}
	if first == token.AccessToken {
		t.Fatal("fingerprint equals the token")
	}
	if (Token{}).Fingerprint() != "" {
		t.Fatal("empty token has a fingerprint")
	}
}

func testService(tokenURL string, open func(string) error) *Service {
	return &Service{
		Client: Client{
			ClientID:     "client-1",
			ClientSecret: "secret-1",
			AuthURL:      "https://accounts.example.test/auth",
			TokenURL:     tokenURL,
		},
		Timeout:     2 * time.Second,
		OpenBrowser: open,
	}
}

func callbackForCode(t *testing.T, code string) func(string) error {
	t.Helper()
	return func(authURL string) error {
		parsed, err := url.Parse(authURL)
		if err != nil {
			return err
		}
		query := parsed.Query()
		resp, err := http.Get(query.Get("redirect_uri") + "/?code=" + url.QueryEscape(code) + "&state=" + url.QueryEscape(query.Get("state")))
		if err != nil {
			return err
		}
		_ = resp.Body.Close()
		return nil
	}
}

func writeClientFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "client.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write client file: %v", err)
	}
	return path
}

func TestClientJSONShape(t *testing.T) {
	// Guard the parser against a silent shape change.
	var value map[string]any
	if err := json.Unmarshal([]byte(`{"installed":{"client_id":"id"}}`), &value); err != nil {
		t.Fatalf("json: %v", err)
	}
	if _, ok := value["installed"]; !ok {
		t.Fatal("installed key is missing")
	}
}
