package config

import (
	"strings"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(func(string) string { return "" })
	if err != nil {
		t.Fatalf("Load defaults: %v", err)
	}

	if cfg.App.LogEnv != "development" {
		t.Fatalf("LogEnv = %q", cfg.App.LogEnv)
	}
	if cfg.App.LogLevel != "info" {
		t.Fatalf("LogLevel = %q", cfg.App.LogLevel)
	}
	if cfg.Feed.Size != 20 {
		t.Fatalf("Feed.Size = %d", cfg.Feed.Size)
	}
	if cfg.LLM.BaseURL != "http://localhost:11434/v1" {
		t.Fatalf("LLM.BaseURL = %q", cfg.LLM.BaseURL)
	}
	if cfg.LLM.Model != "lfm2.5-1.2b-instruct" {
		t.Fatalf("LLM.Model = %q", cfg.LLM.Model)
	}
	if !cfg.LLM.JSONMode {
		t.Fatal("LLM.JSONMode = false, want true")
	}
	if cfg.Storage.DataDir != "./data" {
		t.Fatalf("Storage.DataDir = %q", cfg.Storage.DataDir)
	}
	if cfg.YouTube.OAuthClientFile != "" || cfg.YouTube.APIKey != "" {
		t.Fatalf("YouTube = %+v, want empty optional values", cfg.YouTube)
	}
}

func TestLoadOverrides(t *testing.T) {
	env := map[string]string{
		"LOG_ENV":                  "production",
		"LOG_LEVEL":                "debug",
		"FEED_SIZE":                "12",
		"LLM_BASE_URL":             "http://127.0.0.1:8080/v1",
		"LLM_MODEL":                "test-model",
		"LLM_API_KEY":              "secret",
		"LLM_JSON_MODE":            "false",
		"YTFEED_DATA_DIR":          "/tmp/ytfeed-test",
		"GOOGLE_OAUTH_CLIENT_FILE": "/tmp/client.json",
		"YOUTUBE_API_KEY":          "youtube-key",
	}

	cfg, err := Load(func(key string) string { return env[key] })
	if err != nil {
		t.Fatalf("Load overrides: %v", err)
	}

	if cfg.App.LogEnv != "production" || cfg.App.LogLevel != "debug" {
		t.Fatalf("App = %+v", cfg.App)
	}
	if cfg.Feed.Size != 12 {
		t.Fatalf("Feed.Size = %d", cfg.Feed.Size)
	}
	if cfg.LLM.JSONMode {
		t.Fatal("LLM.JSONMode = true, want false")
	}
	if cfg.Storage.DataDir != "/tmp/ytfeed-test" {
		t.Fatalf("Storage.DataDir = %q", cfg.Storage.DataDir)
	}
	if cfg.YouTube.OAuthClientFile != "/tmp/client.json" || cfg.YouTube.APIKey != "youtube-key" {
		t.Fatalf("YouTube = %+v", cfg.YouTube)
	}
}

func TestLoadRejectsInvalidConfig(t *testing.T) {
	env := map[string]string{
		"LOG_ENV":       "staging",
		"LOG_LEVEL":     "trace",
		"FEED_SIZE":     "0",
		"LLM_BASE_URL":  "ftp://example.com",
		"LLM_JSON_MODE": "maybe",
	}

	_, err := Load(func(key string) string { return env[key] })
	if err == nil {
		t.Fatal("Load() = nil, want an error")
	}

	for _, want := range []string{"LOG_ENV", "LOG_LEVEL", "FEED_SIZE", "LLM_BASE_URL", "LLM_JSON_MODE"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not mention %s", err, want)
		}
	}
}

func TestRedacted(t *testing.T) {
	cfg, err := Load(func(key string) string {
		if key == "LLM_API_KEY" {
			return "super-secret"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := cfg.Redacted()
	if got["llm_api_key"] != "[redacted]" {
		t.Fatalf("llm_api_key = %q", got["llm_api_key"])
	}
	for _, value := range got {
		if strings.Contains(value, "super-secret") {
			t.Fatal("redacted output contains the secret")
		}
	}
}

func TestRedactedYouTube(t *testing.T) {
	cfg, err := Load(func(key string) string {
		switch key {
		case "GOOGLE_OAUTH_CLIENT_FILE":
			return "/tmp/client.json"
		case "YOUTUBE_API_KEY":
			return "super-secret-youtube"
		default:
			return ""
		}
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := cfg.Redacted()
	if got["youtube_oauth_client_file"] != "/tmp/client.json" {
		t.Fatalf("youtube_oauth_client_file = %q", got["youtube_oauth_client_file"])
	}
	if got["youtube_api_key"] != "[redacted]" {
		t.Fatalf("youtube_api_key = %q", got["youtube_api_key"])
	}
	for _, value := range got {
		if strings.Contains(value, "super-secret-youtube") {
			t.Fatal("redacted output contains the YouTube key")
		}
	}
}
