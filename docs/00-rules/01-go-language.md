# 01 — Go language rules

> **Scope:** every `.go` file in this repository.
> **Derived from:** `sales-funnel-bot` (`go.mod`, `internal/**`, `Makefile`).

---

## Version and module

**G-1 — Use Go 1.25 or newer.** The `go` directive in `go.mod` must match the
toolchain used in CI and in the `Dockerfile`.

```go
module github.com/rktkov/ytfeed

go 1.25.4
```

> The module path is fixed: `github.com/rktkov/ytfeed`. The physical folder in
> this workspace is `execution/`. That folder is a workshop detail, not part of the
> module identity.

**G-2 — The module is a monorepo with a single binary.** `cmd/` holds entry
points. Every other folder is a library package. One `go.mod` sits at the root.

---

## Formatting and static checks

**G-3 — `gofmt` is non-negotiable.** Do not make manual formatting decisions.

**G-4 — `golangci-lint run` must pass with a pinned version.** The enabled
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

`errcheck` is the reason for this rule. The reference project discards errors
with `_ =`. `go vet` cannot see that habit, and it would let a failed write to
the taste profile pass silently.

> Adding a linter to the set is a rules change (see `README.md` §4). Do not
> add linters mid-task to make a diff pass.

**G-5 — Do not use build tags except `integration`** (see `07-testing.md`).

---

## Naming

**G-6 — A package name is lowercase and one word.** Do not use underscores or
`util`, `common`, `helpers` packages.

```
internal/domain/feed        ✅
internal/pkg/utils          ❌
```

> The reference project has `shared/util`. That package is a known smell, kept for
> history. Do not repeat it.

**G-7 — A file name is `snake_case.go`.** One primary type or concern per file.

```
service.go            ✅
feed_repository.go    ✅
FeedServiceStuff.go   ❌
```

**G-8 — An acronym is uppercase inside an identifier.** Match the reference
project's style: `CTAText`, `CTAURL`, `UTM`, `URL`, `ID`, `API`, `HTTP`.

```go
type Product struct {
	ID      int
	CTAURL  string
	UTM     UTM
}
```

**G-9 — A constructor is `NewX` and returns the concrete type.** Return an
interface only when the caller genuinely needs one.

```go
func NewFeedService(repo FeedRepository, analyzer Analyzer) *FeedService
```

**G-10 — Name a method for the domain verb**, not the CRUD verb, when the
domain has a verb for it: `EnterFunnel`, `Compose`, `Mute` — not `DoEnter`.

**G-11 — Do not stutter.** In package `feed`, the type is `Service`, not
`FeedService`. In package `taste`, it is `Profile`, not `TasteProfile`.

```go
package feed

type Service struct{ ... }        // ✅ call site: feed.NewService(...)
type FeedService struct{ ... }    // ❌ stutter
```

This diverges from the reference project on purpose. `product.ProductService`
was readable, but it repeated itself at every call site. The repetition grew
with the package count.

---

## Comments

**G-12 — Every exported identifier has a doc comment.** Write it in English with
full sentences. Start with the identifier's name. This is the strongest and most
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

**G-14 — A comment explains *why*, not *what*.** A comment that restates the
code is noise.

**G-15 — Reference the decision, not the ticket, in a durable comment.**
`(ADR-0009)` is durable. `(T-047)` rots once the ticket closes. The reference
project does this everywhere, and the comments have become archaeology. Use
ticket IDs in commit messages and the backlog, not in code comments.

**G-16 — No commented-out code.** Delete it. Git remembers.

---

## Functions and types

**G-17 — `context.Context` is the first parameter** of any function that does
I/O. Never store it in a struct.

```go
func (s *Service) Compose(ctx context.Context, req Request) (*Feed, error)
```

**G-18 — Accept interfaces, return structs.** The package that *consumes* an
interface declares it, not the package that implements it.

**G-19 — Interfaces stay cohesive.** Do not split a repository into six
one-method interfaces without a reason. Do split it when a consumer needs only
a subset, for example a read-only view for the CLI.

**G-20 — No `init()` with side effects.** Wiring happens in `main` or in an
explicit `NewX` or `Run` function.

**G-21 — No global mutable state.** Do not use a package-level `var` that is
written after startup. Inject config and clients.

**G-22 — No `panic` in library code.** Use `panic` only for programmer errors
detected at startup, for example a nil required dependency in `main`. Everything
else returns an error.

**G-23 — Aim for a function of 50 lines or fewer. The hard limit is 80.** The
`funlen` linter enforces the hard limit (G-4). The reference project's 1059-line
`service.go` is the failure mode that this rule prevents.

**G-24 — Aim for a file of 300 lines or fewer. The hard limit is 500.** A
`make lint` check enforces the hard limit, because no Go linter measures file
length:

```make
FILE_LIMIT := 500
lint: lint-files
lint-files:
	@find . -name '*.go' -not -name '*_test.go' -exec wc -l {} + \
	  | awk -v l=$(FILE_LIMIT) '$$1 > l { print "file too long:", $$2, $$1; bad=1 } END { exit bad }'
```

Test files are exempt. A 1600-line table-driven test file is legitimate.

**G-25 — Prefer explicit code over clever code.** Do not use reflection,
`unsafe`, or code generation unless an ADR justifies it.

---

## Dependencies

**G-26 — Standard library first.** A new dependency needs a one-line
justification in the PR. State what it replaces, why the standard library is
insufficient, and its maintenance status.

**G-27 — The approved dependency baseline for this project** (subject to your
confirmation):

| Concern | Choice | Rationale |
|---------|--------|-----------|
| CLI parsing | `flag` + explicit subcommand dispatch | matches reference `internal/cli` style, zero deps |
| Logging | `log/slog` | reference standard |
| SQLite | `modernc.org/sqlite` (pure Go, no CGO) | single static binary |
| Migrations | `pressly/goose` | reference standard |
| YouTube API | `google.golang.org/api/youtube/v3` | official |
| HTTP client | `net/http` | stdlib |
| LLM calls | `net/http` against an OpenAI-compatible `/v1/chat/completions` endpoint | covers Ollama, llama-server, Llama app, LM Studio, Liquid AI LFM GGUFs, opencode. No SDK lock-in. |
| Tracing | `go.opentelemetry.io/otel` + `sdk/trace`, JSONL file exporter | local spans only, no collector (L-15) |
| Terminal images | Ghostty/iTerm2 inline protocol via raw escape sequences | no dependency. Detection at startup (CLI-15). |

**G-28 — No ORM.** See `04-data-persistence.md`.

**G-29 — Do not use `testify`.** The reference project uses the standard
library `testing` package only. Keep it that way. Hand-written mocks are the
standard.

---

## Concurrency

**G-30 — Every goroutine has an owner and a termination path.** Do not start
fire-and-forget goroutines. A goroutine takes a `context`, and the code waits for
it on shutdown.

**G-31 — Shared state is either channel-owned or mutex-protected.** Document
which one in the struct comment.

**G-32 — Analysis runs are bounded.** Use a worker pool with an explicit
concurrency limit from config. Never start one goroutine per video.
