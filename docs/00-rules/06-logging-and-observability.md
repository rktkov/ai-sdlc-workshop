# 06 — Logging and observability rules

> **Scope:** all Go code.
> **Derived from:** `sales-funnel-bot/internal/pkg/logger/logger.go`,
> `internal/transport/middleware/logging.go`.

---

## Logger

**L-1 — `log/slog` is the only logging API.** No `log`, no `fmt.Println`, no
third-party logger.

**L-2 — Inject a `*slog.Logger`. Never fetch one from a global.** Services
and adapters receive it through their constructor.

```go
func NewFeedService(repo feed.Repository, log *slog.Logger) *FeedService
```

**L-3 — Use JSON in production and text in development**, selected by `LOG_ENV`
(see `05-config-and-secrets.md`).

**L-4 — Tests use `slog.New(slog.DiscardHandler)`** unless the test asserts on
log output.

---

## What to log

**L-5 — Log events, not narration.** One line per meaningful state change:
batch started, video analyzed, card shown, swipe recorded, quota warning,
degradation triggered.

**L-6 — Use structured attributes. Never interpolate values into the message.**

```go
log.Info("card composed", "card_id", c.ID, "topic", c.Topic, "state", c.State)  ✅
log.Info(fmt.Sprintf("card %s composed for %s", c.ID, c.Topic))                 ❌
```

**L-7 — Attribute keys are lowercase `snake_case`** and stable across the
codebase: `card_id`, `video_id`, `session_id`, `user_id`, `provider`,
`duration_ms`, `err`.

**L-8 — Levels mean something:**

| Level | Use |
|-------|-----|
| `Debug` | Per-video pipeline detail, prompt sizes, raw scores. Off in production. |
| `Info` | Batch lifecycle, feed composition, user actions, config summary (redacted). |
| `Warn` | Degradation: quota low, transcript missing, LLM slow, retry. |
| `Error` | A user-visible operation failed and the user will notice. |

**L-9 — No `Info` inside tight loops.** A 20-video batch logs one batch line
and per-video lines at `Debug`.

---

## What must never be logged

**L-10 — Never log:** OAuth tokens, API keys, refresh tokens, full request
bodies from Google, transcript text, LLM prompts containing transcript text,
downloaded file contents.

**L-11 — Log identifiers, not payloads.** `video_id` is fine. The transcript is
not.

**L-12 — Redaction is a function, not a habit.** A single `redact()` helper
handles config dumps and HTTP error bodies.

---

## Correlation

**L-13 — Every batch and every session has an ID. Every line in that scope
carries it.** Use `session_id` for interactive work and `batch_id` for analysis
runs.

**L-14 — The CLI prints the correlation ID on failure**, so a user report maps
to a log line.

---

## Metrics (v1 scope)

**L-15 — V1 has no metrics backend, but it emits local traces.**

- `ytfeed stats` reports from the local database: cards composed, analyzed vs
  pending, swipes by reason, cold periods active, quota units consumed today.
- The analysis pipeline writes OpenTelemetry spans to
  `./data/traces/YYYY-MM-DD.jsonl`. No collector, no exporter endpoint, no
  network dependency.
- Spans cover the four pipeline stages: `discovery.search`,
  `transcript.fetch`, `llm.analyze`, `feed.compose`.
- Trace files rotate daily and are pruned after 7 days (`TRACE_RETENTION`).
- Traces are diagnostic. They are never the source of truth. Losing them loses
  nothing durable.

Rationale: local model latency is the top technical risk. Per-stage span
durations are the only way to tell "the model is slow" from "yt-dlp is slow"
from "the network is slow" without guesswork.

**L-16 — Any future metric must be derivable from persisted events**, not
counted only in memory. If it is not in the database, it did not happen.

---

## Health and diagnostics

**L-17 — `ytfeed doctor` is the observability surface for v1.** It reports
config validity, DB path and migration state. It also reports OAuth token
presence and expiry, YouTube quota remaining, yt-dlp availability and version,
LLM reachability and model, and terminal image support.

**L-18 — Diagnostics never need network access to render.** Report an
unreachable service as unreachable, not as a crash.
