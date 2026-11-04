# 03 — Error handling rules

> **Scope:** all Go code.
> **Derived from:** `sales-funnel-bot` — `internal/domain/*/repository.go`
> (sentinel errors), `internal/repository/postgres/*.go` (`fmt.Errorf` with
> `%w`), `internal/transport/middleware/error_handler.go`.

---

## Creating errors

**E-1 — A domain error is a sentinel value. Declare it next to the interface that returns
it.**

```go
// ErrNotFound is returned when no card matches the requested criteria.
var ErrNotFound = errors.New("card not found")
```

**E-2 — An error string is lowercase and has no punctuation. It describes the state.**
An error string is not a user-facing message.

```go
errors.New("card not found")        ✅
errors.New("Card Not Found.")       ❌
```

**E-3 — Use a typed error only when the caller must inspect fields.** Do not use one only
for the category. Prefer sentinels.

```go
type QuotaError struct{ ResetAt time.Time }
func (e *QuotaError) Error() string { return "youtube quota exhausted" }
```

---

## Wrapping and returning

**E-4 — Always wrap with `%w` and add context.** State what the code was doing and which
identifiers it used.

```go
return nil, fmt.Errorf("find card %s for session %s: %w", cardID, sessionID, err)
```

**E-5 — Never wrap with `%v`** when the caller may need `errors.Is` or `errors.As`.

**E-6 — Do not add context that the caller already has.** Three layers of
`"failed to: failed to: failed to"` is a review failure.

**E-7 — Check with `errors.Is` or `errors.As`. Never compare error strings.**

```go
if errors.Is(err, feed.ErrNotFound) { ... }
```

**E-8 — Never log an error and return it.** The caller logs the error once, at the boundary
that decides the user-visible outcome.

```go
// ❌ double reporting
s.log.Error("failed", "err", err)
return err
```

**E-9 — Never swallow an error.** You may use `_ =` only with an inline comment that
explains why the failure is irrelevant.

```go
_ = resp.Body.Close() // read-only, connection is discarded anyway
```

---

## Translation at the boundary

**E-10 — Translate a domain error exactly once, at the transport edge.**

| Domain error | CLI exit code | HTTP status |
|--------------|---------------|-------------|
| `ErrNotFound` | 3 | 404 |
| `ErrInvalidInput` | 2 | 400 |
| `ErrConflict` | 4 | 409 |
| `ErrQuotaExhausted` | 5 | 429 |
| anything else | 1 | 500 |

Exit codes are a confirmed contract. `08-cli-and-ux.md` documents them (CLI-6).

**E-11 — Log an unexpected error with the stack-relevant context.** Report a generic
message and a correlation ID to the user. Do not put an internal error string in a
user-facing CLI line by default. Put it behind `--debug`.

**E-12 — Degradation is explicit, never accidental.** When an optional stage fails
(transcript missing, LLM slow, quota hit), the product rule decides the outcome. The system
records the outcome on the entity:

```go
type Card struct {
	AnalysisState AnalysisState // analyzed | pending | failed
	AnalysisReason string       // "no transcript", "analysis slow", "quota"
}
```

A card with `AnalysisState = pending` is a valid card. A silently empty field is not.

---

## Panics and recovery

**E-13 — Use `panic` only for startup misconfiguration** (nil required dependency,
unreadable config). Never use it for a runtime failure.

**E-14 — The top-level `Run` recovers from panics, logs the stack and exits with code 1.**
This is the only `recover()` in the codebase.

**E-15 — No `recover()` inside domain or repository code.**

---

## User-facing messages

**E-16 — A message states what happened and what the user can do next.**

```
✗ YouTube quota exhausted for today (used 9,842 / 10,000 units).
  Cached feed is still available. Reset at 2026-11-02 00:00 UTC.
```

Not:

```
Error: quota
```

**E-17 — Never include secrets, tokens or transcript content in an error message.**
