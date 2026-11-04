# 01 — Go language rules

> **Scope:** every `.go` file in this repository.
> **Derived from:** `sales-funnel-bot` (`go.mod`, `internal/**`, `Makefile`).

---

## Version and module

**G-1 — Go 1.25 or newer.** The `go` directive in `go.mod` must match the
toolchain used in CI and in the `Dockerfile`.

```go
module github.com/rktkov/ytfeed

go 1.25.4
```

> The module path is fixed: `github.com/rktkov/ytfeed`. The physical folder in
> this workspace is `execution/`; that is a workshop detail, not part of the
> module identity.

**G-2 — The module is a monorepo with a single binary.** `cmd/` holds entry
points; everything else is a library package. One `go.mod` at the root.

---

## Formatting and static checks

**G-3 — `gofmt` is non-negotiable.** No manual formatting decisions.

**G-4 — `golangci-lint run` must pass, with a pinned version.** Enabled
linters are fixed in `.golangci.yml`:

```yaml
version: "2"
linters:
  enable:
    - errcheck
    - govet
    - staticcheck
    - ineffassign
    - unused
    - funlen
  settings:
    funlen:
      lines: 80
      statements: 50
```

```make
GOLANGCI_VERSION := v2.6.2
lint:
	golangci-lint run
```

`errcheck` is the reason this rule exists: the reference project's `_ =`
error-discard habit is invisible to `go vet` and would let a failed write to
the taste profile pass silently.

> Adding a linter to the set is a rules change (see `README.md` §4). Do not
> add linters mid-task to make a diff pass.

**G-5 — No build tags except `integration`** (see `07-testing.md`).

---

## Naming

**G-6 — Packages:** lowercase, one word, no underscores, no `util`/`common`/
`helpers` grab-bags.

```
internal/domain/feed        ✅
internal/pkg/utils          ❌
```

> The reference project has `shared/util`; it is a known smell, kept for
> history. Do not repeat it.

**G-7 — Files:** `snake_case.go`, one primary type or concern per file.

```
service.go            ✅
feed_repository.go    ✅
FeedServiceStuff.go   ❌
```

**G-8 — Acronyms are uppercase inside identifiers**, matching the reference
project's style: `CTAText`, `CTAURL`, `UTM`, `URL`, `ID`, `API`, `HTTP`.

```go
type Product struct {
	ID      int
	CTAURL  string
	UTM     UTM
}
```

**G-9 — Constructors are `NewX` and return the concrete type** (or an
interface only when the caller genuinely needs one).

```go
func NewFeedService(repo FeedRepository, analyzer Analyzer) *FeedService
```

**G-10 — Methods are named for the domain verb**, not the CRUD verb, when the
domain has a verb for it: `EnterFunnel`, `Compose`, `Mute` — not `DoEnter`.

**G-11 — Do not stutter.** In package `feed`, the type is `Service`, not
`FeedService`; in package `taste`, it is `Profile`, not `TasteProfile`.

```go
package feed

type Service struct{ ... }        // ✅ call site: feed.NewService(...)
type FeedService struct{ ... }    // ❌ stutter
```

This diverges from the reference project on purpose: `product.ProductService`
was readable but repeated itself at every call site, and the repetition grew
with the package count.

---

## Comments

**G-12 — Every exported identifier has a doc comment.** English, full
sentences, starting with the identifier's name. This is the strongest and most
consistently applied convention in the reference project.

```go
// ProductRepository abstracts persistence for Product entities (ADR-0009).
// The interface lives in the domain layer and carries no PostgreSQL
// dependency, so domain services can be unit-tested with mock repositories.
type ProductRepository interface { ... }
```

**G-13 — Every package has a package comment**, on the file that defines the
package's purpose.

```go
// Package logger configures the application's structured logger. The project
// standard is log/slog; this package centralizes handler and level
// configuration so production logs are JSON and development logs are readable.
package logger
```

**G-14 — Comments explain *why*, not *what*.** A comment that restates the
code is noise.

**G-15 — Reference the decision, not the ticket, in durable comments.**
`(ADR-0009)` is durable. `(T-047)` rots once the ticket closes — the reference
project does this everywhere and it has become archaeology. Use ticket IDs in
commit messages and backlog, not in code comments.

**G-16 — No commented-out code.** Delete it; git remembers.

---

## Functions and types

**G-17 — `context.Context` is the first parameter** of anything that does I/O,
and is never stored in a struct.

```go
func (s *Service) Compose(ctx context.Context, req Request) (*Feed, error)
```

**G-18 — Accept interfaces, return structs.** Interfaces are declared by the
package that *consumes* them, not the one that implements them.

**G-19 — Interfaces stay cohesive.** Do not split a repository into six
one-method interfaces for the sake of it; do split when a consumer needs only
a subset (e.g. a read-only view for the CLI).

**G-20 — No `init()` with side effects.** Wiring happens in `main` or in an
explicit `NewX`/`Run` function.

**G-21 — No global mutable state.** No package-level `var` that is written
after startup. Config and clients are injected.

**G-22 — No `panic` in library code.** `panic` is allowed only for
programmer errors detected at startup (e.g. a nil required dependency in
`main`). Everything else returns an error.

**G-23 — Function length: aim ≤ 50 lines, hard limit 80.** The hard limit is
enforced by the `funlen` linter (G-4). The reference project's 1059-line
`service.go` is the failure mode this rule exists to prevent.

**G-24 — File length: aim ≤ 300 lines, hard limit 500.** Enforced by a
`make lint` check, because no Go linter measures file length:

```make
FILE_LIMIT := 500
lint: lint-files
lint-files:
	@find . -name '*.go' -not -name '*_test.go' -exec wc -l {} + \
	  | awk -v l=$(FILE_LIMIT) '$$1 > l { print "file too long:", $$2, $$1; bad=1 } END { exit bad }'
```

Test files are exempt: a 1600-line table-driven test file is legitimate.

**G-25 — Prefer explicit over clever.** No reflection, no `unsafe`, no
code generation unless an ADR justifies it.

---

## Dependencies

**G-26 — Standard library first.** A new dependency needs a one-line
justification in the PR: what it replaces, why stdlib is insufficient, and its
maintenance status.

**G-27 — Approved dependency baseline for this project** (subject to your
confirmation):

| Concern | Choice | Rationale |
|---------|--------|-----------|
| CLI parsing | `flag` + explicit subcommand dispatch | matches reference `internal/cli` style, zero deps |
| Logging | `log/slog` | reference standard |
| SQLite | `modernc.org/sqlite` (pure Go, no CGO) | single static binary |
| Migrations | `pressly/goose` | reference standard |
| YouTube API | `google.golang.org/api/youtube/v3` | official |
| HTTP client | `net/http` | stdlib |
| LLM calls | `net/http` against an OpenAI-compatible `/v1/chat/completions` endpoint | covers Ollama, llama-server, Llama app, LM Studio, Liquid AI LFM GGUFs, opencode; no SDK lock-in |
| Tracing | `go.opentelemetry.io/otel` + `sdk/trace`, JSONL file exporter | local spans only, no collector (L-15) |
| Terminal images | Ghostty/iTerm2 inline protocol via raw escape sequences | no dependency; detection at startup (CLI-15) |

**G-28 — No ORM.** See `04-data-persistence.md`.

**G-29 — No `testify`.** The reference project uses stdlib `testing` only.
Keep it that way; hand-written mocks are the standard.

---

## Concurrency

**G-30 — Every goroutine has an owner and a termination path.** No
fire-and-forget goroutines; they take a `context` and are waited for on
shutdown.

**G-31 — Shared state is either channel-owned or mutex-protected.** Document
which one in the struct comment.

**G-32 — Analysis runs are bounded.** Use a worker pool with an explicit
concurrency limit from config; never one goroutine per video.
