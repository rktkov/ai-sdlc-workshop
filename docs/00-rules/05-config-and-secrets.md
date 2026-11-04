# 05 — Configuration and secrets rules

> **Scope:** `internal/config/`, `.env.example`, any adapter that needs
> credentials.
> **Derived from:** `sales-funnel-bot/internal/config/config.go` and
> `.env.example`.

---

## Source of truth

**C-1 — Configuration comes from environment variables only.** Do not use a
committed YAML file, flags that shadow env, or defaults buried in a handler. A
flag may *override* env for one run. Env is the base.

**C-2 — Secrets live only in the environment.** Never in code, never in
`go.mod`, never in a committed file, never in a log line, never in an error
message.

**C-3 — `.env.example` is the documentation.** Every variable that the app reads
appears there with a comment: what it does, its default, and its valid range.
It is the contract. If a variable is missing there, it does not exist.

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

**C-4 — Group config by usage, and pass each group as a whole.**

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
small. It also makes clear what the adapter may read.

**C-5 — Parsing lives in `internal/config` only.** Do not use `os.Getenv`
anywhere else in the codebase. One place to read, one place to test.

**C-6 — Put defaults in code, and document them in the struct field comment**,
not only in `.env.example`.

```go
type FeedConfig struct {
	Size int // FEED_SIZE, default 20
}
```

**C-7 — Parse durations as durations, not integers.** Use `24h`, not `86400`.

**C-8 — Parse booleans strictly.** Accept `true`, `false`, `1` and `0`. Reject
anything else at startup. Do not guess.

---

## Validation

**C-9 — Validate at startup and fail fast.** Use one message that lists every
problem. Do not report one error per restart.

```
✗ invalid configuration:
  - FEED_SIZE must be between 5 and 100 (got 3)
  - LLM_PROVIDER must be one of: ollama, opencode-agent (got openai)
  - COLD_PERIOD_DEFAULT must be a duration like 24h (got 24)
```

**C-10 — Refuse to start when a required secret is missing.** Name the variable
in the message and say where to get it. Never start in a half-configured state.

**C-11 — An optional integration degrades. It does not crash.** If
`YOUTUBE_API_KEY` is absent, the app starts with discovery disabled and says so
in `ytfeed doctor`.

---

## Environment-specific behaviour

**C-12 — Use one explicit environment variable for log format and verbosity**
(`LOG_ENV=development|production`), not a build tag. The same binary behaves
differently in dev and prod.

**C-13 — Config is immutable after startup.** No `os.Setenv`. No live reload in
v1. Restart to change config.

**C-14 — `ytfeed doctor` prints the effective config with secrets redacted**
and the status of each integration (YouTube, yt-dlp, LLM, terminal image
support). It is the first debugging step and must always work offline.

---

## Files and permissions

**C-15 — Local data lives in `./data` relative to the current working
directory**, resolved as: `$YTFEED_DATA_DIR` → `./data`.

`ytfeed doctor` prints the resolved absolute path. The code also logs it at
debug level on every start. This step is mandatory, not cosmetic. A relative
path means that a launch from another directory silently creates a second data
directory. The CLI must surface the path, not hide it.

The app writes nothing outside `./data`. It creates no `$HOME` dotfolder and no
temp files in the repo root.

**C-16 — `.env` is git-ignored. `.env.example` is committed.** This is the
reference project's pattern, and it is correct.

**C-17 — Store tokens with user-only permissions (`0600`).** The code checks the
permissions on read and warns if the file is world-readable.
