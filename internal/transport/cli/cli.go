// Package cli implements the command-line entry points.
package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/rktkov/ytfeed/internal/config"
	"github.com/rktkov/ytfeed/internal/domain/taste"
	"github.com/rktkov/ytfeed/internal/repository/sqlite"
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
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "ytfeed: command required")
		return 2
	}

	switch args[0] {
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
	buf.WriteString("  ytfeed doctor            Check the local server\n")
	buf.WriteString("  ytfeed serve             Run the local API server\n")
	buf.WriteString("  ytfeed serve --allow-remote  Bind to a non-loopback address\n")
	buf.WriteString("  ytfeed help              Show this help\n")
	buf.WriteString("  ytfeed --version         Show the version\n")
	_, _ = w.Write(buf.Bytes())
}
