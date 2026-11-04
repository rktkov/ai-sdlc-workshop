# `taste` — repositories and persistence contracts

## Metadata

| Field | Value |
|-------|-------|
| **Context** | `taste` |
| **Backlog** | `ARCH-301` |
| **Status** | Confirmed at gate `G5` (2026-11-04) |
| **Parent** | [`README.md`](README.md) |
| **Sources** | `D-1`, `D-8`, `D-9`, `D-13`, `D-16`, `D-17`, `D-18`, `D-19`, `D-20`, `D-21`, `D-22` |

## 1. Repository contracts

The interfaces live in the `taste` domain package. The SQLite adapter implements them.
Each adapter asserts the interface at compile time (`A-5`, `A-6`, `D-3`).

| Repository | Operation | Input | Output | Notes |
|------------|-----------|-------|--------|-------|
| `ProfileRepository` | `Get` | `UserID` | `TasteProfile` or `ErrProfileNotFound` | Loads axes, examples and interests in one read. |
| `ProfileRepository` | `Save` | `TasteProfile`, expected `ProfileVersion` | new `ProfileVersion` or `ErrStaleProfile` | Conditional update on `version`. |
| `SignalRepository` | `Append` | `Signal` | stored `SignalID` or `ErrDuplicateGesture` | Insert-only. Unique `gesture_id`. |
| `SignalRepository` | `FindByGesture` | `GestureID` | `Signal` or `ErrSignalNotFound` | Used for idempotent retries. |
| `ColdPeriodRepository` | `Upsert` | `ColdPeriod` | stored `ColdPeriodID`, created or refreshed | Unique `(user_id, scope, target)`. |
| `ColdPeriodRepository` | `ListActive` | `UserID`, instant | list of active periods | `expires_at` later than the instant. |
| `ColdPeriodRepository` | `FindByTarget` | `UserID`, scope, target | `ColdPeriod` or `ErrColdPeriodNotFound` | Used to decide start versus refresh. |

A repository accepts a transaction handle when the caller composes writes (`D-17`). It
does not open a nested transaction.

## 2. Transaction boundaries

| Use case | Transaction scope | Why |
|----------|-------------------|-----|
| `RecordGesture` | Signal insert, profile update, cold-period upsert | One gesture must commit all of its effects or none (`D-20`, `F5`). |
| `DefineProfile` | Profile insert with axes and examples | A partial profile violates `I-P2` to `I-P5`. |
| `SetInterests` | Profile update | The 1–10 set and the version change together. |
| `MuteTarget` | Cold-period upsert | A mute has no signal. The row and its expiry commit together. |
| `ProfileSnapshot` | Read-only snapshot | The caller sees one consistent profile version. |
| `ActiveColdPeriods` | Read-only snapshot | The caller sees one instant. |

`RecordGesture` is the only transaction that spans three aggregates. It never spans a
context boundary (`context-map.md` §6.3).

## 3. Idempotency

| Mechanism | Rule |
|-----------|------|
| Unique `signals.gesture_id` | The database rejects a second signal for the same gesture. |
| Retry path | Read the signal by `gesture_id` before the transaction. If it exists, return `applied` with the stored `signal_id`. |
| Race | Two retries can pass the read. The unique constraint lets one commit. The loser rolls back and returns the stored outcome. |
| Profile double-nudge | The retry path returns before the weight update. A duplicate signal cannot nudge twice. |
| Cold-period double-refresh | The upsert is monotone. A retry cannot shorten an expiry. |

## 4. Logical schema

Table names are logical. Exact column types and indexes are fixed in the migration
(`D-9` to `D-12`). Every table carries `user_id` (`D-22`). Timestamps are UTC (`D-13`).

| Table | Key | Main columns | Constraints |
|-------|-----|--------------|-------------|
| `taste_profiles` | `user_id` | `note`, `rubric_version`, `version`, `created_at`, `updated_at` | One row per user. `version > 0`. |
| `taste_axes` | `(user_id, name)` | `weight` | `weight` in `[0,1]`. Internal FK to `taste_profiles`. |
| `taste_examples` | `example_id` | `user_id`, `axis_name`, `polarity`, `text` | `polarity` is `positive` or `negative`. Internal FK to `taste_axes`. |
| `taste_interests` | `(user_id, slug)` | `position` | 1–10 rows per user. Internal FK to `taste_profiles`. |
| `signals` | `signal_id` | `gesture_id`, `user_id`, `session_id`, `card_id`, `video_id`, `channel_id`, `topic`, `gesture`, `reason`, `rubric_version`, `occurred_at` | `gesture_id` unique. `gesture` in the allowed set. `reason` follows the gesture. |
| `signal_axis_scores` | `(signal_id, axis_name)` | `score` | `score` in `[0,1]`. Internal FK to `signals`. |
| `cold_periods` | `cold_period_id` | `user_id`, `scope`, `target`, `reason`, `created_at`, `expires_at`, `last_refreshed_at` | Unique `(user_id, scope, target)`. `expires_at > created_at`. |

### 4.1 Keys and references

- `video_id`, `channel_id`, `card_id` and `session_id` are values from other contexts. They
  are not foreign keys (`context-map.md` §7).
- `user_id` is a value, but the tables inside `taste` may use it as an internal key.
- `signal_axis_scores` is inside `taste`. Its FK to `signals` is allowed (`D-18`).
- No table outside `taste` writes these rows (`A-7`, `D-1`).

### 4.2 Indexes

| Index | Purpose |
|-------|---------|
| `cold_periods(user_id, expires_at)` | `ActiveColdPeriods` and the expiry check. |
| `signals(user_id, occurred_at)` | Audit and `stats` reads. |
| `signals(gesture_id)` | Idempotent retry. Unique. |
| `taste_interests(user_id, position)` | Stable order of the interest set. |

## 5. Concurrency

| Concern | Control |
|---------|---------|
| One writer | SQLite is configured for one writer (`D-8`). |
| Lost profile update | `version` check on `Save`. A stale write returns `ErrStaleProfile`. |
| Gesture retry | Unique `gesture_id`. The loser reads the winner. |
| Cold-period race | Upsert on the unique target. The later expiry wins. |
| Read consistency | `ProfileSnapshot` reads the profile and its children in one transaction. |
| Cross-context lock | None. `feed` never shares a transaction with `taste`. |

## 6. Failure modes

| Failure | Result | Recovery |
|---------|--------|----------|
| Profile not found | `ErrProfileNotFound` | `feed` reports a setup error. The gesture does not advance. |
| Invalid gesture or reason | `ErrInvalidGesture` | The command is rejected. The card stays current. |
| Stale profile version | `ErrStaleProfile` | The use case retries once. A second failure returns a conflict. |
| Duplicate `gesture_id` | Stored outcome | Return `applied`. No second write. |
| Cold-period upsert conflict | The row exists | Refresh the expiry. Do not create a second row. |
| Database write failure | Transaction rolls back | No signal, no profile change, no cold period. |
| Read failure | The query fails | `feed` keeps the current card. |

## 7. Query contracts

### 7.1 `ProfileSnapshot`

Returns the axes (name, weight, examples), the note and `rubric_version`. It does not
return interests, signals or cold periods. `analysis` uses it to build the prompt
(`F4`, `A-20`).

### 7.2 `InterestSet`

Returns 1–10 normalized slugs in stable position order. `discovery` uses it to plan
queries (`F3`, `F6`).

### 7.3 `ActiveColdPeriods`

Returns the scope and target of every row whose `expires_at` is later than the supplied
instant. It does not return expired rows. It does not delete them (`D-21`).

## 8. Retention

| Data | Retention | Rule |
|------|-----------|------|
| Profile, axes, examples, interests | Until the owner changes them | `F4`, `F3` |
| Signals | Indefinite in v1 | `D-20` |
| Axis scores | Indefinite with their signal | `F5` |
| Cold periods | Indefinite, including expired rows | `D-21`, `F19` |
| Transcript text | Never stored here | `F9`, `OQ-05` |
| Rubric content | Never stored here | `A-19`, `G4-8` |

Deleting a profile is out of scope for v1. A future delete must define the effect on
signals and cold periods, because those records are the product's audit trail.
