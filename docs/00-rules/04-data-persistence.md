# 04 — Data and persistence rules

> **Scope:** `internal/repository/`, `internal/storage/`, `migrations/`.
> **Derived from:** ADR-0009 (repository pattern), `internal/storage/migrations.go`,
> `migrations/` (goose, embedded), `internal/repository/postgres/product_repository.go`.

---

## Access

**D-1 — All SQL lives in `internal/repository/<driver>/`.** No exceptions.
Do not put it in services, handlers, scripts, or tests outside the
repository package.

**D-2 — Repository interfaces live in the domain package** and are named for
the domain, not the storage.

```go
// internal/domain/feed/repository.go
type Repository interface {
	Save(ctx context.Context, card *Card) error
	ListForSession(ctx context.Context, id SessionID, limit int) ([]*Card, error)
}
```

**D-3 — Every adapter asserts its interface at compile time.**

```go
var _ feed.Repository = (*FeedRepository)(nil)
```

**D-4 — No ORM and no query builder.** Use `database/sql`, hand-written SQL and
prepared statements. This is the reference project's choice. It keeps the SQL
visible in review.

**D-5 — Queries are named constants or `const` blocks**, so column order and
`Scan` order cannot drift apart silently.

```go
const cardColumns = "id, video_id, title, channel_id, state, reason, created_at"
```

---

## Storage engine

**D-6 — V1 stores data in SQLite. The engine sits behind the repository
interface.** A switch to Postgres must be "add a package and change config", not
a refactor.

**D-7 — SQLite settings are explicit and identical everywhere:**

```sql
PRAGMA journal_mode = WAL;      -- concurrent readers with one writer
PRAGMA foreign_keys = ON;       -- off by default, must be enabled per connection
PRAGMA busy_timeout = 5000;     -- milliseconds, avoids instant SQLITE_BUSY
PRAGMA synchronous = FULL;      -- every commit is fsynced
```

`synchronous = FULL` is a deliberate choice. User signals (swipes, mutes,
bookmarks) must survive power loss. The write rate is a few per minute, so the
fsync cost is invisible. Do not "optimise" it to `NORMAL` without an ADR that
explicitly accepts the loss of the last transactions.

**D-8 — Use one writer.** The SQLite connection pool uses `SetMaxOpenConns(1)`
for writes, or all writes go through a single goroutine. Concurrent writes to
SQLite are a bug, not a tuning problem.

---

## Schema and migrations

**D-9 — Number migrations, order them, and never edit one after you apply it.**
`0001_init.sql`, `0002_add_cold_period.sql`.

**D-10 — Each migration contains both directions**, goose-style:

```sql
-- +goose Up
CREATE TABLE cards (...);

-- +goose Down
DROP TABLE cards;
```

**D-11 — Embed migrations in the binary** with `go:embed`. The server applies
them automatically when it starts. A single-binary product must not need a
separate migration tool installed.

**D-11a — Take a backup before you apply any migration.** If pending migrations
exist at startup:

1. copy the database file to `ytfeed.db.bak-<UTC timestamp>`.
2. apply migrations in a transaction.
3. verify `schema_version` matches the binary's expectation.
4. keep the last 5 backups, delete older ones.

If any step fails, the server refuses to start and prints the backup path. It
never runs against a half-migrated database.

**D-11b — A database newer than the binary is a hard error.** If
`schema_version` exceeds the version that the binary knows, the server exits
with: "database schema 7 is newer than this binary (6). Upgrade ytfeed." Never
attempt a downgrade.

**D-12 — No destructive migration without a backup step.** Dropping or
rewriting a table requires an ADR and a documented recovery path.

**D-13 — Store timestamps in UTC**, as `TIMESTAMP` or `DATETIME`, never as
local strings. Format at the edge, not in the database.

**D-14 — A nullable column is explicit.** Every nullable column has a stated
meaning in the schema comment. In Go it maps to `sql.Null*` or a pointer. The
mapping is visible in one `scanX` function.

**D-15 — Enums are `TEXT` with a `CHECK` constraint**, not magic integers.

```sql
state TEXT NOT NULL CHECK (state IN ('pending','analyzed','failed'))
```

---

## Transactions and integrity

**D-16 — A multi-table write is one transaction.** If two rows must agree,
they commit together.

**D-17 — A repository accepts a transaction handle when the caller composes
writes.** It does not open nested transactions.

**D-18 — Foreign keys are enforced** (`PRAGMA foreign_keys = ON`), and every
FK has an explicit `ON DELETE` policy.

**D-19 — Idempotency is a schema property, not a code hope.** Every externally
sourced row carries a natural key with a `UNIQUE` constraint, for example
`videos.video_id`. A retried batch cannot duplicate data.

---

## Data that must be durable

**D-20 — Write user signals before the UI confirms them.** A left-swipe,
a cold period, a bookmark and a watch-list removal each commit before the
card advances. Losing a signal loses the product's only asset.

**D-21 — Cold periods are records, not flags.** Each row names a scope and a
target. Expiry is a query, and history is auditable.

```sql
CREATE TABLE cold_periods (
  id         INTEGER PRIMARY KEY,
  user_id    TEXT NOT NULL,
  scope      TEXT NOT NULL CHECK (scope IN ('topic','channel')),
  target     TEXT NOT NULL,          -- topic slug or channel_id
  reason     TEXT NOT NULL CHECK (reason IN ('marketing','ai-slop','value-misaligned','other')),
  created_at DATETIME NOT NULL,
  expires_at DATETIME NOT NULL
);
CREATE INDEX idx_cold_active ON cold_periods(user_id, scope, target, expires_at);
```

**D-22 — Key per-user data by `user_id` from day one**, even though v1
has exactly one user. Do not use `WHERE 1=1` shortcuts that would need a
rewrite for the second user.

---

## Test data

**D-23 — Unit tests never touch a real database.** Repository tests use a
temporary SQLite file or an in-memory database, created and destroyed per test.

**D-24 — Integration tests run migrations from scratch**, never against a
pre-existing schema.
