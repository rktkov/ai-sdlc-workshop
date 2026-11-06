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
	if cfg.Feed.Size != 20 || cfg.Feed.RefreshSource != "mixed" {
		t.Fatalf("Feed = %+v", cfg.Feed)
	}
	if cfg.Feed.GraphChannelLimit != 20 || cfg.Feed.GraphRelatedLimit != 10 || cfg.Feed.ChannelCap != 2 {
		t.Fatalf("Feed graph limits = %+v", cfg.Feed)
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
		"FEED_REFRESH_SOURCE":      "trending",
		"FEED_GRAPH_CHANNEL_LIMIT": "5",
		"FEED_GRAPH_RELATED_LIMIT": "3",
		"FEED_CHANNEL_CAP":         "4",
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
	if cfg.Feed.Size != 12 || cfg.Feed.RefreshSource != "trending" || cfg.Feed.GraphChannelLimit != 5 || cfg.Feed.GraphRelatedLimit != 3 || cfg.Feed.ChannelCap != 4 {
		t.Fatalf("Feed = %+v", cfg.Feed)
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
		"LOG_ENV":                  "staging",
		"LOG_LEVEL":                "trace",
		"FEED_SIZE":                "0",
		"FEED_REFRESH_SOURCE":      "random",
		"FEED_GRAPH_CHANNEL_LIMIT": "0",
		"FEED_GRAPH_RELATED_LIMIT": "-1",
		"FEED_CHANNEL_CAP":         "0",
		"LLM_BASE_URL":             "ftp://example.com",
		"LLM_JSON_MODE":            "maybe",
	}

	_, err := Load(func(key string) string { return env[key] })
	if err == nil {
		t.Fatal("Load() = nil, want an error")
	}

	for _, want := range []string{"LOG_ENV", "LOG_LEVEL", "FEED_SIZE", "FEED_REFRESH_SOURCE", "FEED_GRAPH_CHANNEL_LIMIT", "FEED_GRAPH_RELATED_LIMIT", "FEED_CHANNEL_CAP", "LLM_BASE_URL", "LLM_JSON_MODE"} {
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

func TestLoadAnalysisDefaults(t *testing.T) {
	cfg, err := Load(func(string) string { return "" })
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Analysis.Concurrency != 2 || !cfg.Analysis.Staging {
		t.Fatalf("Analysis = %+v", cfg.Analysis)
	}
	if cfg.Analysis.StagingTTL.Hours() != 24 || cfg.Analysis.MaxBytes != 8<<20 {
		t.Fatalf("Analysis = %+v", cfg.Analysis)
	}
	if cfg.Analysis.YTDLP != "yt-dlp" || cfg.LLM.Timeout.Seconds() != 60 {
		t.Fatalf("Analysis.YTDLP = %q, LLM.Timeout = %v", cfg.Analysis.YTDLP, cfg.LLM.Timeout)
	}
	if cfg.YouTube.Region != "US" {
		t.Fatalf("Region = %q", cfg.YouTube.Region)
	}
}

func TestLoadAnalysisOverrides(t *testing.T) {
	env := map[string]string{
		"ANALYSIS_CONCURRENCY":   "4",
		"TRANSCRIPT_STAGING":     "false",
		"TRANSCRIPT_STAGING_TTL": "48h",
		"TRANSCRIPT_MAX_BYTES":   "1024",
		"YTDLP_BIN":              "/usr/local/bin/yt-dlp",
		"LLM_TIMEOUT":            "30s",
		"YOUTUBE_REGION":         "GB",
	}
	cfg, err := Load(func(key string) string { return env[key] })
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Analysis.Concurrency != 4 || cfg.Analysis.Staging {
		t.Fatalf("Analysis = %+v", cfg.Analysis)
	}
	if cfg.Analysis.StagingTTL.Hours() != 48 || cfg.Analysis.MaxBytes != 1024 {
		t.Fatalf("Analysis = %+v", cfg.Analysis)
	}
	if cfg.Analysis.YTDLP != "/usr/local/bin/yt-dlp" || cfg.LLM.Timeout.Seconds() != 30 {
		t.Fatalf("Analysis.YTDLP = %q, LLM.Timeout = %v", cfg.Analysis.YTDLP, cfg.LLM.Timeout)
	}
	if cfg.YouTube.Region != "GB" {
		t.Fatalf("Region = %q", cfg.YouTube.Region)
	}
}

func TestLoadAnalysisInvalid(t *testing.T) {
	env := map[string]string{
		"ANALYSIS_CONCURRENCY":   "0",
		"TRANSCRIPT_STAGING":     "maybe",
		"TRANSCRIPT_STAGING_TTL": "24",
		"TRANSCRIPT_MAX_BYTES":   "0",
		"LLM_TIMEOUT":            "soon",
	}
	_, err := Load(func(key string) string { return env[key] })
	if err == nil {
		t.Fatal("Load() = nil, want an error")
	}
	for _, want := range []string{"ANALYSIS_CONCURRENCY", "TRANSCRIPT_STAGING", "TRANSCRIPT_STAGING_TTL", "TRANSCRIPT_MAX_BYTES", "LLM_TIMEOUT"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not mention %s", err, want)
		}
	}
}
