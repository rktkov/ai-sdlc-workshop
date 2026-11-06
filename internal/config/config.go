package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the complete runtime configuration.
// It is immutable after Load returns.
type Config struct {
	App      AppConfig
	Feed     FeedConfig
	YouTube  YouTubeConfig
	Analysis AnalysisConfig
	LLM      LLMConfig
	Playback PlaybackConfig
	Storage  StorageConfig
}

// AppConfig controls logging and the runtime mode.
type AppConfig struct {
	LogEnv   string
	LogLevel string
}

// FeedConfig controls feed composition.
type FeedConfig struct {
	Size          int
	RefreshSource string // FEED_REFRESH_SOURCE, default mixed
}

// YouTubeConfig controls the optional Google integrations.
type YouTubeConfig struct {
	OAuthClientFile string
	APIKey          string
	Region          string
}

// AnalysisConfig controls transcript staging and analysis concurrency.
type AnalysisConfig struct {
	Concurrency int
	Staging     bool
	StagingTTL  time.Duration
	MaxBytes    int64
	YTDLP       string
}

// LLMConfig controls the OpenAI-compatible endpoint.
type LLMConfig struct {
	BaseURL  string
	Model    string
	APIKey   string
	JSONMode bool
	Timeout  time.Duration
}

// PlaybackConfig controls local playback.
type PlaybackConfig struct {
	PlayerCmd string
}

// StorageConfig controls local state.
type StorageConfig struct {
	DataDir string
}

// Load reads configuration from environment variables.
// It returns one error that lists every invalid variable.
func Load(getenv func(string) string) (Config, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	cfg := defaultConfig(getenv)
	problems := validateApp(cfg)
	problems = append(problems, parseFeed(getenv, &cfg)...)
	problems = append(problems, parseAnalysis(getenv, &cfg)...)
	problems = append(problems, parseLLM(getenv, &cfg)...)
	if strings.TrimSpace(cfg.Storage.DataDir) == "" {
		problems = append(problems, "YTFEED_DATA_DIR must not be empty")
	}
	if len(problems) > 0 {
		return Config{}, fmt.Errorf("invalid configuration:\n- %s", strings.Join(problems, "\n- "))
	}
	return cfg, nil
}

func defaultConfig(getenv func(string) string) Config {
	return Config{
		App: AppConfig{
			LogEnv:   valueOr(getenv, "LOG_ENV", "development"),
			LogLevel: valueOr(getenv, "LOG_LEVEL", "info"),
		},
		Feed: FeedConfig{
			Size:          20,
			RefreshSource: valueOr(getenv, "FEED_REFRESH_SOURCE", "mixed"),
		},
		YouTube: YouTubeConfig{
			OAuthClientFile: strings.TrimSpace(getenv("GOOGLE_OAUTH_CLIENT_FILE")),
			APIKey:          strings.TrimSpace(getenv("YOUTUBE_API_KEY")),
			Region:          valueOr(getenv, "YOUTUBE_REGION", "US"),
		},
		Analysis: AnalysisConfig{
			Concurrency: 2,
			Staging:     true,
			StagingTTL:  24 * time.Hour,
			MaxBytes:    8 << 20,
			YTDLP:       valueOr(getenv, "YTDLP_BIN", "yt-dlp"),
		},
		LLM: LLMConfig{
			BaseURL:  valueOr(getenv, "LLM_BASE_URL", "http://localhost:11434/v1"),
			Model:    valueOr(getenv, "LLM_MODEL", "lfm2.5-1.2b-instruct"),
			APIKey:   valueOr(getenv, "LLM_API_KEY", "not-needed"),
			JSONMode: true,
			Timeout:  60 * time.Second,
		},
		Playback: PlaybackConfig{PlayerCmd: strings.TrimSpace(getenv("PLAYER_CMD"))},
		Storage:  StorageConfig{DataDir: valueOr(getenv, "YTFEED_DATA_DIR", "./data")},
	}
}

func validateApp(cfg Config) []string {
	var problems []string
	if cfg.App.LogEnv != "development" && cfg.App.LogEnv != "production" {
		problems = append(problems, "LOG_ENV must be development or production")
	}
	if !validLogLevel(cfg.App.LogLevel) {
		problems = append(problems, "LOG_LEVEL must be debug, info, warn or error")
	}
	if !validRefreshSource(cfg.Feed.RefreshSource) {
		problems = append(problems, "FEED_REFRESH_SOURCE must be mixed, trending, graph, fresh or keywords")
	}
	return problems
}

func parseFeed(getenv func(string) string, cfg *Config) []string {
	var problems []string
	if raw := strings.TrimSpace(getenv("FEED_SIZE")); raw != "" {
		size, err := strconv.Atoi(raw)
		if err != nil || size < 1 || size > 20 {
			problems = append(problems, "FEED_SIZE must be an integer between 1 and 20")
		} else {
			cfg.Feed.Size = size
		}
	}
	return problems
}

func parseAnalysis(getenv func(string) string, cfg *Config) []string {
	var problems []string
	if raw := strings.TrimSpace(getenv("ANALYSIS_CONCURRENCY")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 8 {
			problems = append(problems, "ANALYSIS_CONCURRENCY must be an integer between 1 and 8")
		} else {
			cfg.Analysis.Concurrency = value
		}
	}
	if raw := strings.TrimSpace(getenv("TRANSCRIPT_STAGING")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			problems = append(problems, "TRANSCRIPT_STAGING must be true, false, 1 or 0")
		} else {
			cfg.Analysis.Staging = value
		}
	}
	if raw := strings.TrimSpace(getenv("TRANSCRIPT_STAGING_TTL")); raw != "" {
		value, err := time.ParseDuration(raw)
		if err != nil || value <= 0 {
			problems = append(problems, "TRANSCRIPT_STAGING_TTL must be a positive duration like 24h")
		} else {
			cfg.Analysis.StagingTTL = value
		}
	}
	if raw := strings.TrimSpace(getenv("TRANSCRIPT_MAX_BYTES")); raw != "" {
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value <= 0 || value > 64<<20 {
			problems = append(problems, "TRANSCRIPT_MAX_BYTES must be between 1 and 67108864")
		} else {
			cfg.Analysis.MaxBytes = value
		}
	}
	if strings.TrimSpace(cfg.Analysis.YTDLP) == "" {
		problems = append(problems, "YTDLP_BIN must not be empty")
	}
	return problems
}

func parseLLM(getenv func(string) string, cfg *Config) []string {
	var problems []string
	if !validURL(cfg.LLM.BaseURL) {
		problems = append(problems, "LLM_BASE_URL must be an http or https URL")
	}
	if strings.TrimSpace(cfg.LLM.Model) == "" {
		problems = append(problems, "LLM_MODEL must not be empty")
	}
	if raw := strings.TrimSpace(getenv("LLM_JSON_MODE")); raw != "" {
		mode, err := strconv.ParseBool(raw)
		if err != nil {
			problems = append(problems, "LLM_JSON_MODE must be true, false, 1 or 0")
		} else {
			cfg.LLM.JSONMode = mode
		}
	}
	if raw := strings.TrimSpace(getenv("LLM_TIMEOUT")); raw != "" {
		value, err := time.ParseDuration(raw)
		if err != nil || value <= 0 {
			problems = append(problems, "LLM_TIMEOUT must be a positive duration like 60s")
		} else {
			cfg.LLM.Timeout = value
		}
	}
	return problems
}

// Redacted returns a copy safe to print in diagnostics.
func (c Config) Redacted() map[string]string {
	key := "[redacted]"
	if c.LLM.APIKey == "" || c.LLM.APIKey == "not-needed" {
		key = c.LLM.APIKey
	}
	return map[string]string{
		"log_env":                   c.App.LogEnv,
		"log_level":                 c.App.LogLevel,
		"feed_size":                 strconv.Itoa(c.Feed.Size),
		"feed_refresh_source":       c.Feed.RefreshSource,
		"llm_base_url":              c.LLM.BaseURL,
		"llm_model":                 c.LLM.Model,
		"llm_api_key":               key,
		"youtube_oauth_client_file": c.YouTube.OAuthClientFile,
		"youtube_api_key":           redactSecret(c.YouTube.APIKey),
		"youtube_region":            c.YouTube.Region,
		"analysis_concurrency":      strconv.Itoa(c.Analysis.Concurrency),
		"transcript_staging":        strconv.FormatBool(c.Analysis.Staging),
		"transcript_staging_ttl":    c.Analysis.StagingTTL.String(),
		"transcript_max_bytes":      strconv.FormatInt(c.Analysis.MaxBytes, 10),
		"ytdlp_bin":                 c.Analysis.YTDLP,
		"llm_timeout":               c.LLM.Timeout.String(),
		"player_cmd":                c.Playback.PlayerCmd,
		"data_dir":                  c.Storage.DataDir,
	}
}

func valueOr(getenv func(string) string, key, fallback string) string {
	if value := strings.TrimSpace(getenv(key)); value != "" {
		return value
	}
	return fallback
}

func validRefreshSource(value string) bool {
	switch value {
	case "mixed", "trending", "graph", "fresh", "keywords":
		return true
	default:
		return false
	}
}

func validLogLevel(level string) bool {
	switch level {
	case "debug", "info", "warn", "error":
		return true
	default:
		return false
	}
}

func validURL(raw string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return parsed.Host != ""
}

func redactSecret(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return "[redacted]"
}
