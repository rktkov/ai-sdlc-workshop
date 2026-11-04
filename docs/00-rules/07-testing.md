# 07 — Testing rules

> **Scope:** all `*_test.go` files and the `Makefile` test targets.
> **Derived from:** `sales-funnel-bot` — stdlib `testing` only, hand-written
> mocks, `//go:build integration`, testcontainers, `make coverage-check`
> (≥ 80% per tested package), `make test-integration`.

---

## Framework

**T-1 — Use the standard library `testing` package only.** No `testify`, no
`ginkgo`, no mock generator. This is the reference project's standard. It keeps
tests readable without framework knowledge.

**T-2 — Make assertions explicit.**

```go
if got != want {
	t.Fatalf("state = %q, want %q", got, want)
}
```

**T-3 — A failure message states the actual value, the expected value, and the
identifier that produced them.**

---

## Structure

**T-4 — Put tests next to the code** (`service.go` → `service_test.go`), in
the same package, so unexported behaviour is testable.

**T-5 — Test names describe behaviour, not methods:**

```go
func TestCompose_SkipsColdTopics(t *testing.T)        ✅
func TestCompose1(t *testing.T)                       ❌
```

**T-6 — Use table-driven tests with `t.Run` for multiple cases.** The reference
project uses `t.Run` 39 times but rarely uses tables. For pure functions,
tables are mandatory.

```go
tests := []struct{ name string; in string; want string }{...}
for _, tc := range tests {
	t.Run(tc.name, func(t *testing.T) { ... })
}
```

**T-7 — One logical assertion per test.** A test that checks five things hides
which one broke.

---

## Mocks and fakes

**T-8 — Hand-write mocks in `_test.go`. Keep them small and in-memory.** A mock
implements the domain interface and nothing else.

```go
// mockFeedRepo is an in-memory feed.Repository for service tests.
type mockFeedRepo struct{ cards map[string]*Card }
```

**T-9 — A mock records calls when the interaction is the behaviour under test**,
for example "the cold period was saved before the card advanced".

**T-10 — No mocking library and no generated mocks.** If an interface is painful
to fake, the interface is too large. Fix the interface.

---

## Layers

**T-11 — Test domain services with fake repositories.** No database, no
network, no filesystem.

**T-12 — Repository tests use a real SQLite database** in a temp directory,
created and destroyed per test, with migrations applied from scratch.

**T-13 — Integration tests carry `//go:build integration`** and are excluded
from `go test ./...`.

```go
//go:build integration

package sqlite
```

Run them with `make test-integration`.

**T-14 — Never call an external service from a test.** YouTube, yt-dlp and
the LLM are faked behind their ports. An adapter test that needs the network
uses a recorded fixture or `httptest.Server`, never the real endpoint.

**T-15 — Time is injectable.** Anything with expiry (cold periods, sessions,
token refresh) takes a `Clock` interface or a `now func() time.Time`. Tests do
not sleep.

**T-16 — No `time.Sleep` in tests.** Use channels, fakes, or an injected clock.

---

## Coverage and gates

**T-17 — `make test` runs unit tests only and must pass in under 60 seconds.**

**T-18 — `make coverage-check` fails if any tested package is below 80%.**
This is the reference project's gate, and it is mandatory.

```make
coverage-check:
	@go test -cover $(TEST_PKGS) | awk -v t=$(COVERAGE_THRESHOLD) '...'
```

**T-19 — Coverage is a floor, not a goal.** A 90%-covered function with no
assertions on error paths is not tested.

**T-20 — Every bug fix ships with a regression test that fails before the fix.**

---

## Test-driven generation (workshop rule)

**T-21 — Write the failing test first.** Write the implementation only after the
test fails for the right reason. Paste the failure message into the task report.

**T-22 — A test that cannot fail is not a test.** If you cannot describe the
change that would break it, delete it.

**T-23 — No skipped tests on `main`.** Use `t.Skip` only with a linked issue and
a reason.

---

## Makefile contract

**T-24 — These targets exist and behave exactly like this:**

| Target | Behaviour |
|--------|-----------|
| `make test` | unit tests only |
| `make test-integration` | `-tags=integration`, needs Docker |
| `make coverage` | coverage summary per package |
| `make coverage-check` | fails below threshold |
| `make lint` | `golangci-lint run` (see G-4) |
| `make build` | compiles all packages |
| `make check` | lint + test + build |
| `make help` | self-documenting target list |
