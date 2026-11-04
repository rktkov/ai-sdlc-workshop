# 02 — Architecture rules

> **Scope:** package layout, dependency direction, layer responsibilities.
> **Derived from:** ADR-0001 (monolith), ADR-0005 (lightweight DDD),
> ADR-0006 (bounded contexts), ADR-0008 (layered architecture),
> ADR-0009 (repository pattern) in `sales-funnel-bot`.

---

## Style

**A-1 — Use a layered architecture with dependency inversion at the repository
boundary.** Three layers, no more:

```
Transport  →  Domain  →  Repository
(CLI/HTTP)    (logic)     (storage)
```

Adding a fourth layer requires an ADR.

**A-2 — The transport layer is thin.** It does routing, parsing,
(de)serialization and error translation. It contains zero business rules.

```go
// ✅ transport: parse, delegate, translate
func (h *Handler) Compose(w http.ResponseWriter, r *http.Request) {
	var req composeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	feed, err := h.service.Compose(r.Context(), req.toDomain())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, feed)
}
```

**A-3 — The domain layer owns all decisions.** If you can state a rule as a
sentence about the product, it belongs in `internal/domain/`, not in a
handler or a CLI command.

**A-4 — The repository layer owns all SQL.** Do not import `database/sql` outside
`internal/repository/`. Do not put an SQL string in a domain service. Do not use
SQL in tests except in repository tests.

**A-5 — Domain depends on interfaces, never on implementations.**

```go
// internal/domain/feed/repository.go
type Repository interface {
	Save(ctx context.Context, card *Card) error
	Next(ctx context.Context, session SessionID, limit int) ([]*Card, error)
}

// internal/repository/sqlite/feed_repository.go
type FeedRepository struct{ db *sql.DB }

var _ feed.Repository = (*FeedRepository)(nil) // compile-time check, required
```

**A-6 — The compile-time assertion `var _ Interface = (*Impl)(nil)` is
mandatory** for every adapter. It is cheap and catches drift at build time.

**A-7 — Bounded contexts are vertical slices.** Each context owns its domain
package, its repository interface, its tables and its tests. Contexts do not
import each other's internals. They communicate through published interfaces.

The six contexts are fixed:

| Context | Owns | Failure mode it isolates |
|---------|------|--------------------------|
| `identity` | OAuth, token refresh, imported subscriptions | expired/revoked credentials |
| `taste` | profile, signals, cold periods, rubric weights | corrupted personalization |
| `discovery` | YouTube search, candidate pool, quota accounting | API quota exhaustion |
| `analysis` | transcript fetch, LLM calls, rubric application | model latency or garbage output |
| `feed` | composition, sessions, card ordering, gestures | nothing. It is pure orchestration. |
| `library` | saved videos, downloads, local playback | disk and player failures |

**A-8 — No import cycles.** `go build` enforces this rule. Do not work around it
with a shared grab-bag package.

---

## This project's shape

**A-9 — Use a client/server split from day one, with a single binary and a single
process.**

```
execution/
├── cmd/
│   └── ytfeed/           # the only binary: CLI client + local server
├── internal/
│   ├── transport/
│   │   ├── cli/           # cobra-less command dispatch, Run* functions
│   │   └── http/          # local API the CLI talks to
│   ├── domain/
│   │   ├── identity/
│   │   ├── taste/
│   │   ├── discovery/
│   │   ├── analysis/
│   │   ├── feed/
│   │   └── library/
│   ├── repository/
│   │   └── sqlite/
│   ├── integration/       # outbound adapters
│   │   ├── youtube/
│   │   ├── ytdlp/
│   │   └── llm/           # ollama, opencode-agent, future providers
│   ├── config/
│   ├── storage/           # DB open, migration runner
│   └── pkg/logger/
├── shared/
├── migrations/
└── docs/
```

**A-10 — The CLI is a client, not the application.** It talks to the local
server over HTTP on `127.0.0.1` (port from config, default 7420). The server
starts with `ytfeed serve`, or the CLI starts it automatically when it is
absent. The interface is HTTP from commit one, so a future web UI or mobile
client can reuse it without touching domain code.

**A-10a — The server binds to loopback only.** Never use `0.0.0.0`. Never bind a
public interface. Binding is a config value, but a non-loopback address requires
an explicit `--allow-remote` flag and prints a warning.

**A-10b — Local API calls require a token.** The system generates a random token
on first run, stores it with `0600` permissions in the data directory, and sends
it as a `Bearer` header. This rule prevents any other local process from driving
the server. `ytfeed doctor` never prints the token. It prints only the
fingerprint.

**A-11 — Accept one UI connection at a time.** The server rejects a second
concurrent client session. V1 has no multi-tenancy. Per-user data still exists in
the schema.

**A-12 — Every external system is a port.** YouTube, yt-dlp, the LLM, the
terminal renderer and the storage engine are interfaces defined in
`internal/domain/*` or `internal/integration/*`, with adapters behind them. This
design makes "local model or opencode agent" and "SQLite or Postgres" config
changes, not rewrites.

```go
// internal/domain/analysis/analyzer.go
type Analyzer interface {
	Analyze(ctx context.Context, in AnalysisInput) (*Analysis, error)
	Name() string
}
```

**A-12a — Reach every local model runtime through one OpenAI-compatible
chat adapter.** Ollama, `llama-server`, the Llama desktop app, LM Studio and
Liquid AI GGUF builds all expose `/v1/chat/completions`. Provider differences
are config presets (`base_url`, `model`, `api_key`, `json_mode`), not code
branches. You add a new runtime with a preset, not a package.

**A-12b — Analysis output is schema-constrained JSON, never free text.** The
analyzer sends a JSON schema and validates the response against it before it
persists the result. If validation fails, the result is `AnalysisState = failed`
with reason `invalid_model_output`. The system never coerces it into a
plausible-looking card. Small models will otherwise produce confident prose where
numbers were required.

**A-13 — Construct adapters once, in `main`,** and inject them downward.
No adapter reaches for a global.

**A-14 — The feed pipeline is explicit.** Each stage is a named component
with a defined input and output, so it can be tested and swapped:

```
candidates → dedupe → transcript fetch → analysis → scoring → feed composition
```

No stage may call a later stage's adapter directly.

---

## Boundaries that are easy to violate

**A-15 — Transport must not know the storage engine.** Do not use a `sqlite3`
string, a `sql.DB`, or a table name in `internal/transport/`.

**A-16 — Domain must not know HTTP.** Do not use `http.StatusBadRequest` in
`internal/domain/`. Errors are domain errors. Translation happens at the edge
(see `03-errors.md`).

**A-17 — Integration adapters must not contain product rules.** "Skip videos
shorter than 4 minutes" is a domain rule, not a YouTube adapter rule.

**A-18 — Migrations are not domain logic.** A schema change is a migration
file plus a repository change. It never leaks into a service.

**A-19 — The filter rubric is a versioned data artifact, not code.** It lives
as a document in `docs/10-product/` and as a machine-readable copy that the
system loads at startup. A change to the rubric is a content change with a
version bump, not a code deploy. Every analysis result records the rubric
version that produced it. This design makes filter quality measurable: when
results change, you can tell whether the model changed or the rubric changed.

**A-20 — Values are a structured profile, not a prompt string.** The taste
profile stores named axes with weights and examples, plus a free-text note. The
analyzer receives the structured profile and must return a score per axis with
a one-line justification. This rule makes every rejection explainable. Do not
trust the output of a model that cannot justify a score.
