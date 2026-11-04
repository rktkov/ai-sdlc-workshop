package identity

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type memoryStore struct {
	mu    sync.Mutex
	token Token
	err   error
}

func (m *memoryStore) Read(context.Context) (Token, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return Token{}, m.err
	}
	return m.token, nil
}

func (m *memoryStore) Write(_ context.Context, token Token) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.token = token
	return nil
}

func (m *memoryStore) Delete(context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.token = Token{}
	return nil
}

func (m *memoryStore) Path() string { return "memory" }

func TestManagerValidTokenRefreshes(t *testing.T) {
	now := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	store := &memoryStore{token: Token{
		AccessToken:  "old-access",
		RefreshToken: "refresh-1",
		ExpiresAt:    now.Add(-time.Minute),
	}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		if got := r.Form.Get("grant_type"); got != "refresh_token" {
			t.Errorf("grant_type = %q", got)
		}
		if got := r.Form.Get("refresh_token"); got != "refresh-1" {
			t.Errorf("refresh_token = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"access_token":"new-access","expires_in":3600,"token_type":"Bearer","scope":"https://www.googleapis.com/auth/youtube.readonly"}`)
	}))
	defer server.Close()

	manager := &Manager{
		Store:  store,
		Client: Client{ClientID: "client-1", ClientSecret: "secret-1", TokenURL: server.URL},
		Now:    func() time.Time { return now },
	}
	token, err := manager.ValidToken(context.Background())
	if err != nil {
		t.Fatalf("ValidToken: %v", err)
	}
	if token.AccessToken != "new-access" {
		t.Fatalf("access token = %q", token.AccessToken)
	}
	if token.RefreshToken != "refresh-1" {
		t.Fatalf("refresh token = %q, want the stored token", token.RefreshToken)
	}
	if !token.ExpiresAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("expiry = %v", token.ExpiresAt)
	}
}

func TestManagerValidTokenUsesFreshToken(t *testing.T) {
	now := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	store := &memoryStore{token: Token{AccessToken: "fresh", ExpiresAt: now.Add(time.Hour)}}
	manager := &Manager{
		Store:  store,
		Client: Client{ClientID: "client-1", TokenURL: "https://unused.example.test"},
		Now:    func() time.Time { return now },
	}
	token, err := manager.ValidToken(context.Background())
	if err != nil {
		t.Fatalf("ValidToken: %v", err)
	}
	if token.AccessToken != "fresh" {
		t.Fatalf("access token = %q", token.AccessToken)
	}
}

func TestManagerSingleFlight(t *testing.T) {
	now := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"access_token":"new-access","expires_in":3600,"token_type":"Bearer"}`)
	}))
	defer server.Close()

	store := &memoryStore{token: Token{AccessToken: "old", RefreshToken: "refresh-1", ExpiresAt: now.Add(-time.Minute)}}
	manager := &Manager{
		Store:  store,
		Client: Client{ClientID: "client-1", ClientSecret: "secret-1", TokenURL: server.URL},
		Now:    func() time.Time { return now },
	}

	const workers = 10
	start := make(chan struct{})
	var ready sync.WaitGroup
	var done sync.WaitGroup
	ready.Add(workers)
	done.Add(workers)
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer done.Done()
			ready.Done()
			<-start
			token, err := manager.Refresh(context.Background(), store.token)
			if err != nil {
				errs <- err
				return
			}
			if token.AccessToken != "new-access" {
				errs <- fmt.Errorf("access token = %q", token.AccessToken)
			}
		}()
	}
	ready.Wait()
	close(start)
	done.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("Refresh: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("token endpoint calls = %d, want 1", got)
	}
}

func TestManagerRefreshRejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, `{"error":"invalid_grant"}`)
	}))
	defer server.Close()

	store := &memoryStore{token: Token{AccessToken: "old", RefreshToken: "refresh-1"}}
	manager := &Manager{
		Store:  store,
		Client: Client{ClientID: "client-1", TokenURL: server.URL},
		Now:    time.Now,
	}
	_, err := manager.Refresh(context.Background(), store.token)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Refresh = %v, want ErrUnauthorized", err)
	}
}

func TestManagerRefreshWithoutRefreshToken(t *testing.T) {
	store := &memoryStore{token: Token{AccessToken: "old"}}
	manager := &Manager{Store: store, Client: Client{ClientID: "client-1"}, Now: time.Now}
	_, err := manager.Refresh(context.Background(), store.token)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Refresh = %v, want ErrUnauthorized", err)
	}
}

func TestManagerRefreshServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	store := &memoryStore{token: Token{AccessToken: "old", RefreshToken: "refresh-1"}}
	manager := &Manager{
		Store:  store,
		Client: Client{ClientID: "client-1", TokenURL: server.URL},
		Now:    time.Now,
	}
	_, err := manager.Refresh(context.Background(), store.token)
	if err == nil || errors.Is(err, ErrUnauthorized) {
		t.Fatalf("Refresh = %v, want a generic error", err)
	}
}
