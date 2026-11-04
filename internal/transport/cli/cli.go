// Package cli implements the command-line entry points.
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/rktkov/ytfeed/internal/config"
	"github.com/rktkov/ytfeed/internal/domain/identity"
	"github.com/rktkov/ytfeed/internal/domain/taste"
	"github.com/rktkov/ytfeed/internal/repository/sqlite"
	"github.com/rktkov/ytfeed/internal/repository/tokenfile"
	"github.com/rktkov/ytfeed/internal/storage"
	"github.com/rktkov/ytfeed/internal/transport/contract"
	httpapi "github.com/rktkov/ytfeed/internal/transport/http"
)

const (
	defaultBaseURL = "http://127.0.0.1:7420"
	defaultAddress = "127.0.0.1:7420"
	remoteAddress  = "0.0.0.0:7420"
)

// Version and Commit are set at build time.
var (
	Version = "dev"
	Commit  = "none"
)

// Run executes one command and returns the process exit code.
func Run(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer, baseURL string) int {
	return RunWithInput(ctx, args, getenv, os.Stdin, stdout, stderr, baseURL)
}

// RunWithInput executes one command with an explicit standard input.
func RunWithInput(ctx context.Context, args []string, getenv func(string) string, stdin io.Reader, stdout, stderr io.Writer, baseURL string) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "ytfeed: command required")
		return 2
	}

	switch args[0] {
	case "feed":
		return runFeed(ctx, args[1:], getenv, stdout, stderr, baseURL)
	case "onboard":
		return runOnboard(ctx, args[1:], getenv, stdin, stdout, stderr)
	case "auth":
		return runAuth(ctx, args[1:], getenv, stdout, stderr)
	case "doctor":
		return runDoctor(ctx, getenv, stdout, stderr, baseURL)
	case "serve":
		return runServe(ctx, args[1:], getenv, stderr)
	case "help", "-h", "--help":
		printHelp(stdout)
		return 0
	case "--version", "version":
		_, _ = fmt.Fprintf(stdout, "ytfeed %s (%s)\n", Version, Commit)
		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "ytfeed: unknown command %q\n", args[0])
		return 2
	}
}

func runAuth(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "ytfeed auth: command required (login)")
		return 2
	}
	switch args[0] {
	case "login":
		return runAuthLogin(ctx, args[1:], getenv, stdout, stderr)
	case "status":
		return runAuthStatus(ctx, args[1:], getenv, stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "ytfeed auth: unknown command %q\n", args[0])
		return 2
	}
}

func runAuthLogin(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	return runAuthLoginWith(ctx, args, getenv, stdout, stderr, func(ctx context.Context, service identity.Service) (identity.Token, error) {
		return service.Authorize(ctx)
	})
}

func runAuthLoginWith(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer, authorize func(context.Context, identity.Service) (identity.Token, error)) int {
	flags := flag.NewFlagSet("auth login", flag.ContinueOnError)
	flags.SetOutput(stderr)
	noBrowser := flags.Bool("no-browser", false, "print the URL instead of opening the browser")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() > 0 {
		_, _ = fmt.Fprintf(stderr, "ytfeed auth login: unexpected argument %q\n", flags.Arg(0))
		return 2
	}

	cfg, err := config.Load(getenv)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	if cfg.YouTube.OAuthClientFile == "" {
		_, _ = fmt.Fprintln(stderr, "ytfeed auth login: GOOGLE_OAUTH_CLIENT_FILE is not set")
		return 2
	}
	client, err := identity.LoadClient(cfg.YouTube.OAuthClientFile)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed auth login: %v\n", err)
		return 2
	}

	service := identity.Service{
		Client: client,
		OnAuthURL: func(url string) {
			_, _ = fmt.Fprintf(stderr, "open this URL to authorize ytfeed:\n%s\n", url)
		},
	}
	if *noBrowser {
		service.OpenBrowser = func(string) error { return nil }
	}

	token, err := authorize(ctx, service)
	if err != nil {
		return authError("ytfeed auth login", stderr, err)
	}
	store := tokenfile.New(cfg.Storage.DataDir)
	if err := store.Write(ctx, token); err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed auth login: store token: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "authorized scope=%s expires_at=%s fingerprint=%s token=%s\n",
		token.Scope, token.ExpiresAt.UTC().Format(time.RFC3339), token.Fingerprint(), store.Path())
	return 0
}

func runAuthStatus(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		_, _ = fmt.Fprintf(stderr, "ytfeed auth status: unexpected argument %q\n", args[0])
		return 2
	}
	cfg, err := config.Load(getenv)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	store := tokenfile.New(cfg.Storage.DataDir)
	token, err := store.Read(ctx)
	if err != nil {
		if errors.Is(err, identity.ErrTokenNotFound) {
			_, _ = fmt.Fprintln(stderr, "ytfeed auth status: not authenticated; run `ytfeed auth login`")
			return 2
		}
		_, _ = fmt.Fprintf(stderr, "ytfeed auth status: %v\n", err)
		return 1
	}
	if !token.ExpiresAt.After(time.Now().Add(30 * time.Second)) {
		if cfg.YouTube.OAuthClientFile == "" {
			_, _ = fmt.Fprintln(stderr, "ytfeed auth status: token expired; run `ytfeed auth login`")
			return 2
		}
		client, err := identity.LoadClient(cfg.YouTube.OAuthClientFile)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "ytfeed auth status: %v\n", err)
			return 2
		}
		manager := &identity.Manager{Store: store, Client: client}
		token, err = manager.ValidToken(ctx)
		if err != nil {
			if errors.Is(err, identity.ErrUnauthorized) {
				_, _ = fmt.Fprintln(stderr, "ytfeed auth status: authorization revoked; run `ytfeed auth login`")
				return 2
			}
			_, _ = fmt.Fprintf(stderr, "ytfeed auth status: %v\n", err)
			return 1
		}
	}
	_, _ = fmt.Fprintf(stdout, "status=active scope=%s expires_at=%s fingerprint=%s\n",
		token.Scope, token.ExpiresAt.UTC().Format(time.RFC3339), token.Fingerprint())
	return 0
}

func authError(command string, stderr io.Writer, err error) int {
	switch {
	case errors.Is(err, identity.ErrConsentDenied):
		_, _ = fmt.Fprintf(stderr, "%s: authorization denied\n", command)
		return 2
	case errors.Is(err, identity.ErrStateMismatch):
		_, _ = fmt.Fprintf(stderr, "%s: authorization failed: state mismatch\n", command)
		return 2
	case errors.Is(err, identity.ErrUnauthorized):
		_, _ = fmt.Fprintf(stderr, "%s: authorization rejected; run `ytfeed auth login` again\n", command)
		return 2
	case errors.Is(err, identity.ErrTimeout):
		_, _ = fmt.Fprintf(stderr, "%s: authorization timed out\n", command)
		return 2
	case errors.Is(err, context.Canceled):
		return 130
	default:
		_, _ = fmt.Fprintf(stderr, "%s: authorization failed: %v\n", command, err)
		return 1
	}
}

func runDoctor(ctx context.Context, getenv func(string) string, stdout, stderr io.Writer, baseURL string) int {
	cfg, err := config.Load(getenv)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}

	token, err := httpapi.LoadOrCreateToken(cfg.Storage.DataDir)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/health", nil)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed doctor: server unavailable: %v\n", err)
		return 1
	}
	defer func() { _ = resp.Body.Close() }() // response body, no recovery path

	if resp.StatusCode == http.StatusUnauthorized {
		_, _ = fmt.Fprintln(stderr, "ytfeed doctor: unauthorized; check the API token")
		return 2
	}
	if resp.StatusCode != http.StatusOK {
		_, _ = fmt.Fprintf(stderr, "ytfeed doctor: server returned %d\n", resp.StatusCode)
		return 1
	}

	var health contract.Health
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed doctor: decode health: %v\n", err)
		return 1
	}
	if err := health.Validate(); err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed doctor: invalid health: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "ok version=%s schema=%d\n", health.Version, health.SchemaVersion)
	return 0
}

func runServe(ctx context.Context, args []string, getenv func(string) string, stderr io.Writer) int {
	return runServeWith(ctx, args, getenv, stderr, func(server *http.Server) error {
		return server.ListenAndServe()
	})
}

func runServeWith(ctx context.Context, args []string, getenv func(string) string, stderr io.Writer, serve func(*http.Server) error) int {
	allowRemote := false
	for _, arg := range args {
		if arg != "--allow-remote" {
			_, _ = fmt.Fprintf(stderr, "ytfeed serve: unknown argument %q\n", arg)
			return 2
		}
		allowRemote = true
	}

	cfg, err := config.Load(getenv)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}

	dbPath := filepath.Join(cfg.Storage.DataDir, "ytfeed.db")
	db, err := storage.Open(ctx, dbPath)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	defer func() { _ = db.Close() }()

	schemaVersion, err := storage.SchemaVersion(ctx, db)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	token, err := httpapi.LoadOrCreateToken(cfg.Storage.DataDir)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	address := defaultAddress
	if allowRemote {
		address = remoteAddress
		_, _ = fmt.Fprintln(stderr, "warning: binding to a non-loopback address; the API token is the only control")
	}

	tasteService := &taste.Service{
		Store:    sqlite.NewTasteRepository(db),
		Profiles: sqlite.NewProfileRepository(db),
	}
	handler := httpapi.HandlerWithGestureRemote(token, Version, schemaVersion, httpapi.GestureHandler(tasteService, "owner"), allowRemote)

	server := &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	if err := serve(server); err != nil && !errors.Is(err, http.ErrServerClosed) {
		_, _ = fmt.Fprintf(stderr, "ytfeed serve: %v\n", err)
		return 1
	}
	return 0
}

func printHelp(w io.Writer) {
	var buf bytes.Buffer
	buf.WriteString("ytfeed — local YouTube feed composer\n\n")
	buf.WriteString("Usage:\n")
	buf.WriteString("  ytfeed feed              Show today's cards\n")
	buf.WriteString("  ytfeed feed --json       Print one JSON feed object\n")
	buf.WriteString("  ytfeed onboard           Connect Google and choose interests\n")
	buf.WriteString("  ytfeed auth login        Authorize YouTube access\n")
	buf.WriteString("  ytfeed auth status       Show the stored credential\n")
	buf.WriteString("  ytfeed doctor            Check the local server\n")
	buf.WriteString("  ytfeed serve             Run the local API server\n")
	buf.WriteString("  ytfeed serve --allow-remote  Bind to a non-loopback address\n")
	buf.WriteString("  ytfeed help              Show this help\n")
	buf.WriteString("  ytfeed --version         Show the version\n")
	_, _ = w.Write(buf.Bytes())
}
