// Package identity owns the Google credential and the subscription graph.
package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	// ScopeYouTubeReadOnly is the only scope that ytfeed requests.
	ScopeYouTubeReadOnly = "https://www.googleapis.com/auth/youtube.readonly"
	// DefaultAuthURL is the Google authorization endpoint.
	DefaultAuthURL = "https://accounts.google.com/o/oauth2/v2/auth"
	// DefaultTokenURL is the Google token endpoint.
	DefaultTokenURL = "https://oauth2.googleapis.com/token"
	// DefaultTimeout is the consent window.
	DefaultTimeout = 2 * time.Minute
)

var (
	// ErrConsentDenied means the owner refused the consent screen.
	ErrConsentDenied = errors.New("oauth consent denied")
	// ErrStateMismatch means the callback state did not match the request.
	ErrStateMismatch = errors.New("oauth state mismatch")
	// ErrUnauthorized means the token endpoint rejected the grant.
	ErrUnauthorized = errors.New("oauth token rejected")
	// ErrTimeout means no callback arrived before the consent window closed.
	ErrTimeout = errors.New("oauth authorization timed out")
)

// Client is one installed OAuth client.
type Client struct {
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
}

// Token is the result of a consent flow. The caller decides how to store it.
type Token struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	Scope        string
	ExpiresAt    time.Time
}

// Fingerprint returns a short, non-reversible identifier for diagnostics.
func (t Token) Fingerprint() string {
	if t.AccessToken == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(t.AccessToken))
	return hex.EncodeToString(sum[:])[:12]
}

// Service runs one authorization-code flow with PKCE.
type Service struct {
	Client      Client
	HTTPClient  *http.Client
	Now         func() time.Time
	Rand        io.Reader
	Timeout     time.Duration
	Listen      func(network, address string) (net.Listener, error)
	OpenBrowser func(url string) error
	OnAuthURL   func(url string)
}

type clientFile struct {
	Installed *clientJSON `json:"installed"`
	Web       *clientJSON `json:"web"`
}

type clientJSON struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	AuthURI      string `json:"auth_uri"`
	TokenURI     string `json:"token_uri"`
}

// LoadClient reads an installed OAuth client file.
func LoadClient(path string) (Client, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Client{}, fmt.Errorf("read oauth client file: %w", err)
	}
	var file clientFile
	if err := json.Unmarshal(data, &file); err != nil {
		return Client{}, fmt.Errorf("decode oauth client file: %w", err)
	}
	raw := file.Installed
	if raw == nil {
		raw = file.Web
	}
	if raw == nil {
		return Client{}, errors.New("oauth client file has no installed or web client")
	}
	if strings.TrimSpace(raw.ClientID) == "" {
		return Client{}, errors.New("oauth client file has no client id")
	}

	authURL := strings.TrimSpace(raw.AuthURI)
	if authURL == "" {
		authURL = DefaultAuthURL
	}
	tokenURL := strings.TrimSpace(raw.TokenURI)
	if tokenURL == "" {
		tokenURL = DefaultTokenURL
	}
	if !validEndpoint(authURL) {
		return Client{}, errors.New("oauth client file has an invalid auth uri")
	}
	if !validEndpoint(tokenURL) {
		return Client{}, errors.New("oauth client file has an invalid token uri")
	}
	return Client{
		ClientID:     strings.TrimSpace(raw.ClientID),
		ClientSecret: raw.ClientSecret,
		AuthURL:      authURL,
		TokenURL:     tokenURL,
	}, nil
}

func validEndpoint(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return false
	}
	return parsed.Host != ""
}

type callbackResult struct {
	code string
	err  error
}

// Authorize runs the consent flow and returns the token. It does not store it.
func (s *Service) Authorize(ctx context.Context) (Token, error) {
	if err := s.prepare(); err != nil {
		return Token{}, err
	}
	listener, err := s.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return Token{}, fmt.Errorf("listen for oauth callback: %w", err)
	}
	defer func() { _ = listener.Close() }()

	redirectURI := "http://" + listener.Addr().String()
	verifier, challenge, state, err := newPKCE(s.Rand)
	if err != nil {
		return Token{}, err
	}

	results := make(chan callbackResult, 1)
	server := s.startCallbackServer(listener, state, results)
	defer func() { _ = server.Close() }()

	authURL := s.authURL(redirectURI, state, challenge)
	if s.OnAuthURL != nil {
		s.OnAuthURL(authURL)
	}
	_ = s.OpenBrowser(authURL)

	waitCtx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()
	select {
	case result := <-results:
		if result.err != nil {
			return Token{}, result.err
		}
		return s.exchange(waitCtx, result.code, verifier, redirectURI)
	case <-waitCtx.Done():
		if errors.Is(waitCtx.Err(), context.DeadlineExceeded) {
			return Token{}, ErrTimeout
		}
		return Token{}, waitCtx.Err()
	}
}

func (s *Service) prepare() error {
	if strings.TrimSpace(s.Client.ClientID) == "" {
		return errors.New("oauth client has no client id")
	}
	if s.Client.AuthURL == "" {
		s.Client.AuthURL = DefaultAuthURL
	}
	if s.Client.TokenURL == "" {
		s.Client.TokenURL = DefaultTokenURL
	}
	if s.HTTPClient == nil {
		s.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	if s.Now == nil {
		s.Now = time.Now
	}
	if s.Rand == nil {
		s.Rand = rand.Reader
	}
	if s.Timeout == 0 {
		s.Timeout = DefaultTimeout
	}
	if s.Listen == nil {
		s.Listen = net.Listen
	}
	if s.OpenBrowser == nil {
		s.OpenBrowser = openBrowser
	}
	return nil
}

func (s *Service) authURL(redirectURI, state, challenge string) string {
	params := url.Values{}
	params.Set("client_id", s.Client.ClientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("response_type", "code")
	params.Set("scope", ScopeYouTubeReadOnly)
	params.Set("state", state)
	params.Set("code_challenge", challenge)
	params.Set("code_challenge_method", "S256")
	params.Set("access_type", "offline")
	params.Set("prompt", "consent")

	separator := "?"
	if strings.Contains(s.Client.AuthURL, "?") {
		separator = "&"
	}
	return s.Client.AuthURL + separator + params.Encode()
}

func (s *Service) startCallbackServer(listener net.Listener, state string, results chan<- callbackResult) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		query := r.URL.Query()
		if code := query.Get("error"); code != "" {
			writeCallbackPage(w, http.StatusOK, "Authorization denied. You can close this window.")
			deliver(results, callbackResult{err: fmt.Errorf("%w: %s", ErrConsentDenied, code)})
			return
		}
		if subtle.ConstantTimeCompare([]byte(query.Get("state")), []byte(state)) != 1 {
			writeCallbackPage(w, http.StatusBadRequest, "Invalid state. Return to the terminal and try again.")
			deliver(results, callbackResult{err: ErrStateMismatch})
			return
		}
		code := query.Get("code")
		if code == "" {
			writeCallbackPage(w, http.StatusBadRequest, "Missing authorization code.")
			deliver(results, callbackResult{err: errors.New("oauth callback has no code")})
			return
		}
		writeCallbackPage(w, http.StatusOK, "Authorization complete. You can close this window.")
		deliver(results, callbackResult{code: code})
	})

	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = server.Serve(listener) }()
	return server
}

func deliver(results chan<- callbackResult, result callbackResult) {
	select {
	case results <- result:
	default:
	}
}

func writeCallbackPage(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, "<!doctype html><html><body><p>%s</p></body></html>", html.EscapeString(message))
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int64  `json:"expires_in"`
	Error        string `json:"error"`
}

func (s *Service) exchange(ctx context.Context, code, verifier, redirectURI string) (Token, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("client_id", s.Client.ClientID)
	if s.Client.ClientSecret != "" {
		form.Set("client_secret", s.Client.ClientSecret)
	}
	form.Set("code_verifier", verifier)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.Client.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return Token{}, fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return Token{}, fmt.Errorf("exchange oauth code: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Token{}, fmt.Errorf("read token response: %w", err)
	}
	var payload tokenResponse
	decodeErr := json.Unmarshal(body, &payload)
	if resp.StatusCode != http.StatusOK {
		if decodeErr == nil && (payload.Error == "invalid_grant" || payload.Error == "invalid_client") {
			return Token{}, ErrUnauthorized
		}
		return Token{}, fmt.Errorf("oauth token endpoint returned %d", resp.StatusCode)
	}
	if decodeErr != nil {
		return Token{}, fmt.Errorf("decode token response: %w", decodeErr)
	}
	if payload.AccessToken == "" {
		return Token{}, errors.New("oauth token response has no access token")
	}

	token := Token{
		AccessToken:  payload.AccessToken,
		RefreshToken: payload.RefreshToken,
		TokenType:    payload.TokenType,
		Scope:        payload.Scope,
	}
	if payload.ExpiresIn > 0 {
		token.ExpiresAt = s.Now().Add(time.Duration(payload.ExpiresIn) * time.Second)
	}
	return token, nil
}

func newPKCE(reader io.Reader) (verifier, challenge, state string, err error) {
	verifier, err = randomToken(reader)
	if err != nil {
		return "", "", "", err
	}
	state, err = randomToken(reader)
	if err != nil {
		return "", "", "", err
	}
	return verifier, codeChallenge(verifier), state, nil
}

func randomToken(reader io.Reader) (string, error) {
	buf := make([]byte, 32)
	if _, err := io.ReadFull(reader, buf); err != nil {
		return "", fmt.Errorf("generate oauth token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func codeChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func openBrowser(target string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", target).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", target).Start()
	default:
		return exec.Command("xdg-open", target).Start()
	}
}
