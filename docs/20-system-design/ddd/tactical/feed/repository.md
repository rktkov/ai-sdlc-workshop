# `feed` — repositories and persistence contracts

## Metadata

| Field | Value |
|-------|-------|
| **Context** | `feed` |
| **Backlog** | `ARCH-302` |
| **Status** | Confirmed at gate `G5` (2026-11-04) |
| **Parent** | [`README.md`](README.md) |
| **Sources** | `D-1`, `D-8`, `D-9`, `D-13`, `D-16`, `D-17`, `D-18`, `D-19`, `D-20`, `D-22` |

## 1. Repository contracts

The interfaces live in the `feed` domain package. The SQLite adapter implements them.
Each adapter asserts the interface at compile time (`A-5`, `A-6`, `D-3`).

| Repository | Operation | Input | Output | Notes |
|------------|-----------|-------|--------|-------|
| `BatchRepository` | `Create` | `Batch` with cards | stored `BatchID` | One transaction for the batch and all cards. |
| `BatchRepository` | `Get` | `BatchID` | `Batch` with cards or `ErrBatchNotFound` | Loads positions in order. |
| `BatchRepository` | `UpdateCard` | `Card`, expected card version | new card version or `ErrStaleCard` | Conditional update on `version`. |
| `BatchRepository` | `Reorder` | `BatchID`, ordered card IDs, expected batch version | new batch version | Updates positions in one transaction. |
| `BatchRepository` | `LatestCompleted` | `UserID` | `Batch` or `ErrNoCachedFeed` | The cached feed for degradation. |
| `SessionRepository` | `GetOpen` | `UserID` | `Session` or `ErrNoOpenSession` | At most one open session. |
| `SessionRepository` | `Create` | `Session` | stored `SessionID` | Fails if an open session exists. |
| `SessionRepository` | `Save` | `Session`, expected version | new version or `ErrStaleSession` | Cursor, lease and last gesture. |
| `SessionRepository` | `Reopen` | `SessionID`, `LeaseOwner` | updated `Session` | Increments `resume_count`. |
| `SessionRepository` | `Close` | `SessionID`, reason | closed `Session` | Releases the lease. |
| `RatingRepository` | `FindByWeek` | `UserID`, `WeekStart` | `WeeklyRating` or `ErrRatingNotFound` | Uniqueness check. |
| `RatingRepository` | `LastForUser` | `UserID` | `WeeklyRating` or none | Decides whether to prompt. |
| `RatingRepository` | `Save` | `WeeklyRating` | stored `RatingID` or `ErrRatingExists` | Unique per user and week. |

A repository accepts a transaction handle when the caller composes writes (`D-17`). It
does not open a nested transaction.

## 2. Transaction boundaries

| Use case | Transaction scope | Why |
|----------|-------------------|-----|
| `ComposeBatch` | Batch row and all card rows | A partial batch has no valid order (`I-B1` to `I-B3`). |
| `ApplyAnalysisResult` | One card update and the reorder of the unseen suffix | The card state and its position must agree (`I-B4`, `D-313`). |
| `ApplyGesture` | Card decision, cursor, `last_gesture_id` | The advance and the idempotency key commit together (`D-314`). |
| `OpenSession` | Session row and lease | At most one open session per user (`I-S1`). |
| `CloseSession` | Session state and lease release | A closed session accepts no write (`I-S6`). |
| `RecordWeeklyRating` | Rating row and batch links | One rating per week (`I-R1`). |
| `GetCachedFeed` | Read-only snapshot | One consistent completed batch. |

`ApplyGesture` is two transactions in two contexts. `taste` commits first. `feed` commits
second. They are never one transaction (`context-map.md` §6.3).

## 3. Idempotency

| Mechanism | Rule |
|-----------|------|
| `sessions.last_gesture_id` | A retry with the same `gesture_id` returns the previous cursor and does not advance. |
| Unique `(batch_id, video_id)` | A candidate cannot appear twice in one batch. |
| Unique `(batch_id, position)` | Two cards cannot occupy one position. |
| Unique `(user_id, week_start)` | A second rating for the week is rejected. |
| `weekly_ratings` batch links | The same batch is linked once per rating. |

The gesture retry path is:

1. Read the session.
2. Compare `gesture_id` with `last_gesture_id`.
3. If equal, return the stored cursor.
4. Otherwise call `taste`, then commit the local change.

## 4. Logical schema

Table names are logical. Exact column types and indexes are fixed in the migration
(`D-9` to `D-12`). Every table carries `user_id` where the row belongs to a user
(`D-22`). Timestamps are UTC (`D-13`).

| Table | Key | Main columns | Constraints |
|-------|-----|--------------|-------------|
| `batches` | `batch_id` | `user_id`, `size`, `state`, `degradation_reason`, `version`, `created_at`, `completed_at` | `size` in 1–20. `state` in the allowed set. `degradation_reason` only when degraded. |
| `cards` | `card_id` | `batch_id`, `video_id`, `channel_id`, `discovery_rank`, `position`, `state`, `analysis_id`, `score`, `reason`, `decision`, `analyzed_late`, `topic`, `tldr`, `similarity`, `new_territory`, `why_this_card`, `version`, `created_at`, `updated_at` | Unique `(batch_id, video_id)`. Unique `(batch_id, position)`. Internal FK to `batches`. |
| `sessions` | `session_id` | `user_id`, `batch_id`, `state`, `current_card_id`, `furthest_card_id`, `last_gesture_id`, `lease_owner`, `last_activity_at`, `expires_at`, `opened_at`, `closed_at`, `resume_count`, `version` | One open session per user. Internal FK to `batches`. |
| `weekly_ratings` | `rating_id` | `user_id`, `week_start`, `value`, `created_at` | Unique `(user_id, week_start)`. `value` in the allowed set. |
| `rating_batches` | `(rating_id, batch_id)` | none | One row per batch shown that week. Internal FK to `weekly_ratings`. |

### 4.1 One open session

SQLite supports a partial unique index. The logical constraint is:

- At most one row in `sessions` with `state = 'open'` for a given `user_id`.
- A closed row can coexist with a new open row. The old row stays for history.

### 4.2 References

- `video_id`, `channel_id` and `analysis_id` are values from other contexts. They are not
  foreign keys (`context-map.md` §7).
- `batch_id` inside `cards`, `sessions` and `rating_batches` is an internal reference.
  Internal FKs are allowed (`D-18`).
- No table outside `feed` writes these rows (`A-7`, `D-1`).

### 4.3 Indexes

| Index | Purpose |
|-------|---------|
| `cards(batch_id, position)` | Ordered card reads. Unique. |
| `cards(batch_id, state)` | Find pending cards for streaming. |
| `sessions(user_id) WHERE state = 'open'` | One open session. Unique. |
| `batches(user_id, completed_at)` | Latest completed batch for the cached feed. |
| `weekly_ratings(user_id, week_start)` | One rating per week. Unique. |

## 5. Concurrency

| Concern | Control |
|---------|---------|
| One writer | SQLite is configured for one writer (`D-8`). |
| Card update race | Card `version` check. A stale analysis result is retried or dropped. |
| Reorder race | Batch `version` check. Positions commit as one set. |
| Session gesture race | Session `version` and `last_gesture_id`. The second writer sees the first. |
| Lease race | Partial unique index plus `version`. One open session wins. |
| Rating race | Unique `(user_id, week_start)`. The loser returns `ErrRatingExists`. |
| Cross-context lock | None. `taste` and `feed` commit separately. |

## 6. Failure modes

| Failure | Result | Recovery |
|---------|--------|----------|
| Batch not found | `ErrBatchNotFound` | The session cannot open. |
| No open session | `ErrNoOpenSession` | Open or resume a session first. |
| Stale card version | `ErrStaleCard` | Retry the analysis update once. |
| Stale session version | `ErrStaleSession` | Retry the gesture once. |
| Duplicate gesture | Stored cursor | Return the stored outcome. No second advance. |
| Duplicate rating | `ErrRatingExists` | Return the stored rating. |
| No cached feed | `ErrNoCachedFeed` | Show an empty-feed explanation. |
| Write failure | Transaction rolls back | No partial batch, card or cursor. |

## 7. Query contracts

### 7.1 `GetCachedFeed`

Returns the latest batch in `completed` state for the user, with its cards in position
order. It does not return a `degraded` batch as the cached feed. The degradation banner
carries the reason and the reset time (`F24`).

### 7.2 `GetActiveSession`

Returns the open session for the user, or none. It includes the cursor and the lease
state. The caller decides between conflict, resume and new session.

### 7.3 `GetLastRating`

Returns the most recent rating for the user. The session-open use case compares
`week_start` with the current week to decide whether to prompt (`F21`, `CLI-28`).

## 8. Retention

| Data | Retention | Rule |
|------|-----------|------|
| Batches and cards | Indefinite in v1 | `F13`, `F21` |
| Sessions | Indefinite, including closed rows | `F18`, `OQ-13` |
| Weekly ratings | Indefinite | `F21` |
| Cached feed | Derived from completed batches | `F24` |
| Transcript text | Never stored here | `F9`, `OQ-05` |
| Analysis documents | Never stored here, only the presentation snapshot | `D-319` |
