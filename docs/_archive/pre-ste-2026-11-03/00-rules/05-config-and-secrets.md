# 05 — Configuration and secrets rules

> **Scope:** `internal/config/`, `.env.example`, any adapter that needs
> credentials.
> **Derived from:** `sales-funnel-bot/internal/config/config.go` and
> `.env.example`.

---

## Source of truth

**C-1 — Configuration comes from environment variables only.** Not from a
committed YAML, not from flags that shadow env, not from defaults buried in a
handler. Flags may *override* env for one run; env is the base.

**C-2 — Secrets live only in the environment.** Never in code, never in
`go.mod`, never in a committed file, never in a log line, never in an error
message.

**C-3 — `.env.example` is the documentation.** Every variable the app reads
appears there with a comment: what it does, its default, and its valid range.
It is the contract; if a variable is missing there, it does not exist.

```bash
# Max videos in one composed feed. Default 20.
FEED_SIZE=20

# Cold period applied after a left-swipe, per reason.
# Format: 24h, 72h, 168h. Default 24h.
COLD_PERIOD_DEFAULT=24h

# Any runtime that speaks OpenAI-compatible /v1/chat/completions:
# Ollama, llama-server, Llama desktop app, LM Studio, opencode.
# Default points at Ollama; llama-server default is http://localhost:8080/v1.
LLM_BASE_URL=http://localhost:11434/v1
# Exact model id or GGUF repo:quant. Pinned by the model-evaluation spike.
LLM_MODEL=lfm2.5-2.6b-instruct
LLM_API_KEY=not-needed
# Schema-constrained JSON output. Required for analysis; false only for debugging.
LLM_JSON_MODE=true
```

---

## Structure

**C-4 — Config is grouped by usage, and each group is passed as a whole.**

```go
type Config struct {
	App       AppConfig
	Feed      FeedConfig
	YouTube   YouTubeConfig
	LLM       LLMConfig
	Storage   StorageConfig
	Terminal  TerminalConfig
}
```

Passing `cfg.LLM` into the analyzer keeps the analyzer's dependency surface
small and makes it obvious what the adapter may read.

**C-5 — Parsing lives in `internal/config` only.** No `os.Getenv` anywhere
else in the codebase. One place to read, one place to test.

**C-6 — Defaults are in code and documented in the struct field comment**, not
only in `.env.example`.

```go
type FeedConfig struct {
	Size int // FEED_SIZE, default 20
}
```

**C-7 — Durations are parsed as durations, not integers.** `24h`, not `86400`.

**C-8 — Booleans are parsed strictly.** Accept `true`/`false`/`1`/`0`; reject
anything else at startup rather than guessing.

---

## Validation

**C-9 — Validate at startup and fail fast, with one message that lists every
problem.** Not one error per restart.

```
✗ invalid configuration:
  - FEED_SIZE must be between 5 and 100 (got 3)
  - LLM_PROVIDER must be one of: ollama, opencode-agent (got openai)
  - COLD_PERIOD_DEFAULT must be a duration like 24h (got 24)
```

**C-10 — Refuse to start on a missing required secret**, with a message naming
the variable and where to get it. Never start in a half-configured state.

**C-11 — Optional integrations degrade, they do not crash.** If
`YOUTUBE_API_KEY` is absent, the app starts with discovery disabled and says
so in `ytfeed doctor`.

---

## Environment-specific behaviour

**C-12 — One explicit environment variable controls log format and verbosity**
(`LOG_ENV=development|production`), not a build tag. The same binary behaves
differently in dev and prod.

**C-13 — Config is immutable after startup.** No `os.Setenv`, no live reload
in v1. Restart to change config.

**C-14 — `ytfeed doctor` prints the effective config with secrets redacted**
and the status of each integration (YouTube, yt-dlp, LLM, terminal image
support). It is the first debugging step and must always work offline.

---

## Files and permissions

**C-15 — Local data lives in `./data` relative to the current working
directory**, resolved as: `$YTFEED_DATA_DIR` → `./data`.

The resolved absolute path is printed by `ytfeed doctor` and logged at debug
level on every start. This is mandatory, not cosmetic: a relative path means
launching ytfeed from another directory silently creates a second data
directory. The CLI must surface the path rather than hide it.

Nothing is written outside `./data`. No `$HOME` dotfolder, no temp files in the
repo root.

**C-16 — `.env` is git-ignored; `.env.example` is committed.** This is the
reference project's pattern and it is correct.

**C-17 — Tokens are stored with user-only permissions (`0600`)** and the code
checks it on read, warning if the file is world-readable.
