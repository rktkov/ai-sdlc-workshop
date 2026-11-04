# 03 — Error handling rules

> **Scope:** all Go code.
> **Derived from:** `sales-funnel-bot` — `internal/domain/*/repository.go`
> (sentinel errors), `internal/repository/postgres/*.go` (`fmt.Errorf` with
> `%w`), `internal/transport/middleware/error_handler.go`.

---

## Creating errors

**E-1 — Domain errors are sentinel values, declared next to the interface
that returns them.**

```go
// ErrNotFound is returned when no card matches the requested criteria.
var ErrNotFound = errors.New("card not found")
```

**E-2 — Error strings are lowercase, unpunctuated, and describe the state.**
They are not user-facing messages.

```go
errors.New("card not found")        ✅
errors.New("Card Not Found.")       ❌
```

**E-3 — A typed error is used only when the caller must inspect fields**, not
just the category. Prefer sentinels.

```go
type QuotaError struct{ ResetAt time.Time }
func (e *QuotaError) Error() string { return "youtube quota exhausted" }
```

---

## Wrapping and returning

**E-4 — Always wrap with `%w` and add context: what was being done, with which
identifiers.**

```go
return nil, fmt.Errorf("find card %s for session %s: %w", cardID, sessionID, err)
```

**E-5 — Never wrap with `%v`** when the caller may need `errors.Is`/`As`.

**E-6 — Do not add context that the caller already has.** Three layers of
`"failed to: failed to: failed to"` is a review failure.

**E-7 — Check with `errors.Is` / `errors.As`, never by string comparison.**

```go
if errors.Is(err, feed.ErrNotFound) { ... }
```

**E-8 — Never log an error and return it.** The caller logs it once, at the
boundary that decides the user-visible outcome.

```go
// ❌ double reporting
s.log.Error("failed", "err", err)
return err
```

**E-9 — Never swallow an error.** `_ =` is allowed only with an inline comment
explaining why the failure is irrelevant.

```go
_ = resp.Body.Close() // read-only, connection is discarded anyway
```

---

## Translation at the boundary

**E-10 — Domain errors are translated exactly once, at the transport edge.**

| Domain error | CLI exit code | HTTP status |
|--------------|---------------|-------------|
| `ErrNotFound` | 3 | 404 |
| `ErrInvalidInput` | 2 | 400 |
| `ErrConflict` | 4 | 409 |
| `ErrQuotaExhausted` | 5 | 429 |
| anything else | 1 | 500 |

Exit codes are a confirmed contract, documented in `08-cli-and-ux.md` (CLI-6).

**E-11 — Unexpected errors are logged with the stack-relevant context and
reported to the user as a generic message plus a correlation ID.** Never dump
an internal error string into a user-facing CLI line by default; put it behind
`--debug`.

**E-12 — Degradation is explicit, never accidental.** When an optional stage
fails (transcript missing, LLM slow, quota hit), the product rule decides the
outcome and the outcome is recorded on the entity:

```go
type Card struct {
	AnalysisState AnalysisState // analyzed | pending | failed
	AnalysisReason string       // "no transcript", "analysis slow", "quota"
}
```

A card with `AnalysisState = pending` is a valid card. A silently empty field
is not.

---

## Panics and recovery

**E-13 — `panic` is reserved for startup misconfiguration** (nil required
dependency, unreadable config). It is never used for runtime failure.

**E-14 — The top-level `Run` recovers from panics, logs the stack and exits
with code 1.** This is the only `recover()` in the codebase.

**E-15 — No `recover()` inside domain or repository code.**

---

## User-facing messages

**E-16 — Messages state what happened and what the user can do next.**

```
✗ YouTube quota exhausted for today (used 9,842 / 10,000 units).
  Cached feed is still available. Reset at 2026-11-02 00:00 UTC.
```

Not:

```
Error: quota
```

**E-17 — Never include secrets, tokens or transcript content in an error
message.**
