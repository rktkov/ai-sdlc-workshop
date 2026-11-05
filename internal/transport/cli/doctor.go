package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rktkov/ytfeed/internal/config"
	"github.com/rktkov/ytfeed/internal/domain/discovery"
	"github.com/rktkov/ytfeed/internal/domain/identity"
	"github.com/rktkov/ytfeed/internal/repository/sqlite"
	"github.com/rktkov/ytfeed/internal/repository/tokenfile"
	"github.com/rktkov/ytfeed/internal/storage"
	"github.com/rktkov/ytfeed/internal/transport/contract"
	httpapi "github.com/rktkov/ytfeed/internal/transport/http"
)

type doctorDeps struct {
	Now            func() time.Time
	YtdlpVersion   func() (string, error)
	LLMReachable   func(context.Context, config.Config) error
	TerminalImages func() bool
	Health         func(context.Context, string, string) (contract.Health, int, error)
}

func runDoctorWith(ctx context.Context, getenv func(string) string, stdout, stderr io.Writer, baseURL string, deps doctorDeps) int {
	cfg, err := config.Load(getenv)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	now := time.Now().UTC()
	if deps.Now != nil {
		now = deps.Now().UTC()
	}
	dataDir, err := filepath.Abs(cfg.Storage.DataDir)
	if err != nil {
		dataDir = cfg.Storage.DataDir
	}
	dbPath := filepath.Join(dataDir, "ytfeed.db")
	_, _ = fmt.Fprintf(stdout, "data_dir: %s\n", dataDir)
	_, _ = fmt.Fprintf(stdout, "db: %s\n", dbPath)

	db, err := storage.Open(ctx, dbPath)
	if err != nil {
		_, _ = fmt.Fprintln(stdout, "db_status: error")
		_, _ = fmt.Fprintf(stderr, "ytfeed doctor: %v\n", err)
		return 1
	}
	defer func() { _ = db.Close() }()
	schema, err := storage.SchemaVersion(ctx, db)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ytfeed doctor: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "schema: %d\n", schema)

	printAPIToken(stdout, dataDir)
	printOAuthToken(stdout, dataDir)
	printQuota(ctx, stdout, db, now)
	printYTDLP(stdout, deps)
	printLLM(ctx, stdout, cfg, deps)
	printTerminal(stdout, deps)
	return printServer(ctx, stdout, baseURL, dataDir, schema, deps)
}

func printAPIToken(stdout io.Writer, dataDir string) {
	if _, err := httpapi.LoadOrCreateToken(dataDir); err != nil {
		_, _ = fmt.Fprintln(stdout, "api_token: error")
		return
	}
	_, _ = fmt.Fprintln(stdout, "api_token: present")
}

func printOAuthToken(stdout io.Writer, dataDir string) {
	store := tokenfile.New(dataDir)
	token, err := store.Read(context.Background())
	if errors.Is(err, identity.ErrTokenNotFound) {
		_, _ = fmt.Fprintln(stdout, "token: missing")
		return
	}
	if err != nil {
		_, _ = fmt.Fprintln(stdout, "token: error")
		return
	}
	status := "active"
	if !token.ExpiresAt.After(time.Now().UTC()) {
		status = "expired"
	}
	if err := store.CheckPermissions(); err != nil {
		status += " (permissions too open)"
	}
	_, _ = fmt.Fprintf(stdout, "token: %s expires=%s fingerprint=%s\n",
		status, token.ExpiresAt.UTC().Format(time.RFC3339), token.Fingerprint())
}

func printQuota(ctx context.Context, stdout io.Writer, db *sql.DB, now time.Time) {
	ledger, err := sqlite.NewQuotaRepository(db).GetOrCreate(ctx, "owner", discovery.AccountingDay(now), 10000)
	if err != nil {
		_, _ = fmt.Fprintln(stdout, "quota: error")
		return
	}
	_, _ = fmt.Fprintf(stdout, "quota: charged=%d budget=%d\n", ledger.Charged, ledger.Budget)
}

func printYTDLP(stdout io.Writer, deps doctorDeps) {
	getVersion := deps.YtdlpVersion
	if getVersion == nil {
		getVersion = defaultYtdlpVersion
	}
	version, err := getVersion()
	if err != nil {
		_, _ = fmt.Fprintln(stdout, "yt-dlp: missing")
		return
	}
	_, _ = fmt.Fprintf(stdout, "yt-dlp: %s\n", version)
}

func printLLM(ctx context.Context, stdout io.Writer, cfg config.Config, deps doctorDeps) {
	check := deps.LLMReachable
	if check == nil {
		check = defaultLLMReachable
	}
	if err := check(ctx, cfg); err != nil {
		_, _ = fmt.Fprintln(stdout, "llm: unreachable")
		return
	}
	_, _ = fmt.Fprintf(stdout, "llm: reachable model=%s\n", cfg.LLM.Model)
}

func printTerminal(stdout io.Writer, deps doctorDeps) {
	detect := deps.TerminalImages
	if detect == nil {
		detect = defaultTerminalImages
	}
	if detect() {
		_, _ = fmt.Fprintln(stdout, "terminal_images: yes")
	} else {
		_, _ = fmt.Fprintln(stdout, "terminal_images: no")
	}
}

func printServer(ctx context.Context, stdout io.Writer, baseURL, dataDir string, schema int, deps doctorDeps) int {
	token, err := httpapi.LoadOrCreateToken(dataDir)
	if err != nil {
		_, _ = fmt.Fprintln(stdout, "server: unavailable")
		return 0
	}
	check := deps.Health
	if check == nil {
		check = defaultHealth
	}
	health, status, err := check(ctx, baseURL, token)
	if err != nil || status != http.StatusOK {
		_, _ = fmt.Fprintln(stdout, "server: unavailable")
		if status == http.StatusUnauthorized {
			return 2
		}
		return 0
	}
	if err := health.Validate(); err != nil {
		_, _ = fmt.Fprintln(stdout, "server: invalid")
		return 1
	}
	if health.SchemaVersion != schema {
		_, _ = fmt.Fprintf(stdout, "server: ok version=%s schema=%d (local schema mismatch)\n", health.Version, health.SchemaVersion)
		return 0
	}
	_, _ = fmt.Fprintf(stdout, "server: ok version=%s schema=%d\n", health.Version, health.SchemaVersion)
	return 0
}

func defaultYtdlpVersion() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "yt-dlp", "--version").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func defaultLLMReachable(ctx context.Context, cfg config.Config) error {
	client := &http.Client{Timeout: 2 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(cfg.LLM.BaseURL, "/")+"/models", nil)
	if err != nil {
		return err
	}
	if key := strings.TrimSpace(cfg.LLM.APIKey); key != "" && key != "not-needed" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("llm returned %d", resp.StatusCode)
	}
	return nil
}

func defaultTerminalImages() bool {
	program := strings.ToLower(os.Getenv("TERM_PROGRAM"))
	if strings.Contains(program, "iterm") || strings.Contains(program, "wezterm") || strings.Contains(program, "ghostty") {
		return true
	}
	return os.Getenv("KITTY_WINDOW_ID") != "" || os.Getenv("WEZTERM_PANE") != ""
}

func defaultHealth(ctx context.Context, baseURL, token string) (contract.Health, int, error) {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/health", nil)
	if err != nil {
		return contract.Health{}, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return contract.Health{}, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	var health contract.Health
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return contract.Health{}, resp.StatusCode, err
	}
	return health, resp.StatusCode, nil
}
