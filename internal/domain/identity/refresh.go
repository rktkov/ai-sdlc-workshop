package identity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Manager returns a usable token and refreshes it once when needed.
type Manager struct {
	Store      Store
	Client     Client
	HTTPClient *http.Client
	Now        func() time.Time

	mu       sync.Mutex
	inflight *refreshCall
}

type refreshCall struct {
	done  chan struct{}
	token Token
	err   error
}

// ValidToken returns a stored token. It refreshes the token when it expires soon.
func (m *Manager) ValidToken(ctx context.Context) (Token, error) {
	if m.Store == nil {
		return Token{}, errors.New("token store is nil")
	}
	token, err := m.Store.Read(ctx)
	if err != nil {
		return Token{}, err
	}
	if token.ExpiresAt.After(m.now().Add(30 * time.Second)) {
		return token, nil
	}
	return m.Refresh(ctx, token)
}

// Refresh runs one refresh for all concurrent callers.
func (m *Manager) Refresh(ctx context.Context, token Token) (Token, error) {
	m.mu.Lock()
	if m.inflight != nil {
		call := m.inflight
		m.mu.Unlock()
		select {
		case <-call.done:
			return call.token, call.err
		case <-ctx.Done():
			return Token{}, ctx.Err()
		}
	}
	call := &refreshCall{done: make(chan struct{})}
	m.inflight = call
	m.mu.Unlock()

	refreshed, err := m.refreshOnce(ctx, token)
	call.token = refreshed
	call.err = err
	close(call.done)

	m.mu.Lock()
	m.inflight = nil
	m.mu.Unlock()
	return refreshed, err
}

func (m *Manager) refreshOnce(ctx context.Context, token Token) (Token, error) {
	if strings.TrimSpace(token.RefreshToken) == "" {
		return Token{}, ErrUnauthorized
	}
	if strings.TrimSpace(m.Client.ClientID) == "" || strings.TrimSpace(m.Client.TokenURL) == "" {
		return Token{}, ErrUnauthorized
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", token.RefreshToken)
	form.Set("client_id", m.Client.ClientID)
	if m.Client.ClientSecret != "" {
		form.Set("client_secret", m.Client.ClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.Client.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return Token{}, fmt.Errorf("create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := m.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return Token{}, fmt.Errorf("refresh oauth token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Token{}, fmt.Errorf("read refresh response: %w", err)
	}
	var payload tokenResponse
	decodeErr := json.Unmarshal(body, &payload)
	if resp.StatusCode != http.StatusOK {
		if decodeErr == nil && (payload.Error == "invalid_grant" || payload.Error == "invalid_client") {
			return Token{}, ErrUnauthorized
		}
		return Token{}, fmt.Errorf("oauth refresh endpoint returned %d", resp.StatusCode)
	}
	if decodeErr != nil {
		return Token{}, fmt.Errorf("decode refresh response: %w", decodeErr)
	}
	if payload.AccessToken == "" {
		return Token{}, errors.New("oauth refresh response has no access token")
	}

	token.AccessToken = payload.AccessToken
	if payload.RefreshToken != "" {
		token.RefreshToken = payload.RefreshToken
	}
	if payload.TokenType != "" {
		token.TokenType = payload.TokenType
	}
	if payload.Scope != "" {
		token.Scope = payload.Scope
	}
	if payload.ExpiresIn > 0 {
		token.ExpiresAt = m.now().Add(time.Duration(payload.ExpiresIn) * time.Second)
	}
	if err := m.Store.Write(ctx, token); err != nil {
		return Token{}, err
	}
	return token, nil
}

func (m *Manager) now() time.Time {
	if m.Now == nil {
		return time.Now()
	}
	return m.Now()
}
