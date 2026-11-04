package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// Config is the complete runtime configuration.
// It is immutable after Load returns.
type Config struct {
	App     AppConfig
	Feed    FeedConfig
	YouTube YouTubeConfig
	LLM     LLMConfig
	Storage StorageConfig
}

// AppConfig controls logging and the runtime mode.
type AppConfig struct {
	LogEnv   string
	LogLevel string
}

// FeedConfig controls feed composition.
type FeedConfig struct {
	Size int
}

// YouTubeConfig controls the optional Google integrations.
type YouTubeConfig struct {
	OAuthClientFile string
	APIKey          string
}

// LLMConfig controls the OpenAI-compatible endpoint.
type LLMConfig struct {
	BaseURL  string
	Model    string
	APIKey   string
	JSONMode bool
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

	cfg := Config{
		App: AppConfig{
			LogEnv:   valueOr(getenv, "LOG_ENV", "development"),
			LogLevel: valueOr(getenv, "LOG_LEVEL", "info"),
		},
		Feed: FeedConfig{
			Size: 20,
		},
		YouTube: YouTubeConfig{
			OAuthClientFile: strings.TrimSpace(getenv("GOOGLE_OAUTH_CLIENT_FILE")),
			APIKey:          strings.TrimSpace(getenv("YOUTUBE_API_KEY")),
		},
		LLM: LLMConfig{
			BaseURL:  valueOr(getenv, "LLM_BASE_URL", "http://localhost:11434/v1"),
			Model:    valueOr(getenv, "LLM_MODEL", "lfm2.5-1.2b-instruct"),
			APIKey:   valueOr(getenv, "LLM_API_KEY", "not-needed"),
			JSONMode: true,
		},
		Storage: StorageConfig{
			DataDir: valueOr(getenv, "YTFEED_DATA_DIR", "./data"),
		},
	}

	var problems []string

	if cfg.App.LogEnv != "development" && cfg.App.LogEnv != "production" {
		problems = append(problems, "LOG_ENV must be development or production")
	}
	if !validLogLevel(cfg.App.LogLevel) {
		problems = append(problems, "LOG_LEVEL must be debug, info, warn or error")
	}

	if raw := strings.TrimSpace(getenv("FEED_SIZE")); raw != "" {
		size, err := strconv.Atoi(raw)
		if err != nil || size < 1 || size > 20 {
			problems = append(problems, "FEED_SIZE must be an integer between 1 and 20")
		} else {
			cfg.Feed.Size = size
		}
	}

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

	if strings.TrimSpace(cfg.Storage.DataDir) == "" {
		problems = append(problems, "YTFEED_DATA_DIR must not be empty")
	}

	if len(problems) > 0 {
		return Config{}, fmt.Errorf("invalid configuration:\n- %s", strings.Join(problems, "\n- "))
	}
	return cfg, nil
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
		"llm_base_url":              c.LLM.BaseURL,
		"llm_model":                 c.LLM.Model,
		"llm_api_key":               key,
		"youtube_oauth_client_file": c.YouTube.OAuthClientFile,
		"youtube_api_key":           redactSecret(c.YouTube.APIKey),
		"data_dir":                  c.Storage.DataDir,
	}
}

func valueOr(getenv func(string) string, key, fallback string) string {
	if value := strings.TrimSpace(getenv(key)); value != "" {
		return value
	}
	return fallback
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
