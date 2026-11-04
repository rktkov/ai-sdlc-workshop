# `discovery` — repositories, persistence and use cases

## Metadata

| Field | Value |
|-------|-------|
| **Context** | `discovery` |
| **Backlog** | `ARCH-303` |
| **Status** | Confirmed at gate `G5` (2026-11-04) |
| **Parent** | [`README.md`](README.md) |
| **Sources** | `F6`, `F7`, `F8`, `F13`, `D-1`, `D-8`, `D-16`, `D-17`, `D-18`, `D-19`, `D-22` |

## 1. Repository contracts

| Repository | Operation | Input | Output | Notes |
|------------|-----------|-------|--------|-------|
| `CandidatePoolRepository` | `Create` | `BatchID`, `UserID`, `size` | `CandidatePool` | One pool per batch. |
| `CandidatePoolRepository` | `AddCandidates` | `BatchID`, candidates | stored count | Insert raw candidates. |
| `CandidatePoolRepository` | `CommitSelection` | `BatchID`, selected, positions, shortfall | updated pool | One transaction for selected flags and positions. |
| `CandidatePoolRepository` | `Get` | `BatchID` | pool with candidates | Ordered by position. |
| `QuotaLedgerRepository` | `GetOrCreate` | `UserID`, `AccountingDay` | `QuotaLedger` | Creates the day row on first use. |
| `QuotaLedgerRepository` | `Reserve` | `ReservationID`, units, call | updated ledger | Fails when the budget would be exceeded. |
| `QuotaLedgerRepository` | `Charge` | `ReservationID`, units | updated ledger | Moves units from reserved to charged. |
| `QuotaLedgerRepository` | `Release` | `ReservationID`, reason | updated ledger | Returns units to the budget. |
| `QuotaLedgerRepository` | `ReleaseStale` | `UserID`, instant | released count | Called on read and at startup. |

Repositories accept a transaction handle when the caller composes writes (`D-17`).

## 2. Transaction boundaries

| Use case | Transaction scope | Why |
|----------|-------------------|-----|
| `BuildCandidatePool` | Pool row and raw candidates | A pool without candidates is not useful. |
| `SelectCandidates` | Selected flags, positions and shortfall | The selection and its order must agree. |
| `ReserveQuota` | Reservation row and ledger counters | The budget check and the reservation must commit together. |
| `ChargeQuota` | Reservation state and ledger counters | A reservation cannot be both reserved and charged. |
| `ReleaseQuota` | Reservation state and ledger counters | Released units must return exactly once. |
| `ReleaseStale` | Stale reservations and counters | Startup recovery is one transaction. |

Quota transactions are separate from pool transactions. A pool build can fail after a
quota charge. That is correct. The call consumed quota even if the pool write failed.

## 3. Idempotency

| Mechanism | Rule |
|-----------|------|
| Unique `candidate_pools(batch_id)` | One pool per batch. |
| Unique `candidates(batch_id, video_id)` | A video appears once in a pool. |
| Unique `quota_reservations(reservation_id)` | A retry cannot create a second reservation. |
| Reservation state | `charged` and `released` are terminal. A second transition is rejected. |
| `GetOrCreate` ledger | One row per user and day. |

## 4. Logical schema

| Table | Key | Main columns | Constraints |
|-------|-----|--------------|-------------|
| `candidate_pools` | `batch_id` | `user_id`, `size`, `state`, `shortfall_reason`, `version`, `created_at`, `selected_at` | `size` 1–20. |
| `candidates` | `(batch_id, video_id)` | `channel_id`, `title`, `subscriber_count`, `topic_tags`, `source`, `discovery_rank`, `excluded`, `exclusion_reason`, `selected`, `position` | Unique `(batch_id, video_id)`. Internal FK to `candidate_pools`. |
| `quota_ledgers` | `(user_id, accounting_day)` | `budget`, `reserved`, `charged`, `warned_at_80`, `exhausted`, `version`, `updated_at` | `reserved + charged <= budget`. |
| `quota_reservations` | `reservation_id` | `user_id`, `accounting_day`, `units`, `call`, `state`, `created_at`, `expires_at` | `units > 0`. Internal reference to the ledger. |

`topic_tags` is a list. It may be stored as a child table `candidate_topic_tags` if the
adapter does not support arrays. That choice belongs to the migration.

### 4.1 References

- `batch_id` is a value from `feed`. It is not a foreign key.
- `video_id` and `channel_id` are YouTube values. They are not foreign keys.
- No table outside `discovery` writes these rows (`A-7`, `D-1`).

### 4.2 Indexes

| Index | Purpose |
|-------|---------|
| `candidates(batch_id, position)` | Selected order. |
| `candidates(batch_id, selected)` | Selection reads. |
| `quota_ledgers(user_id, accounting_day)` | Daily budget. Unique. |
| `quota_reservations(state, expires_at)` | Stale reservation sweep. |

## 5. Use cases

### 5.1 `BuildCandidatePool`

**Preconditions:** `size` is 1–20. The interest set is valid. The subscription graph is
available.

**Steps:**

1. Create the pool row.
2. Plan queries from the interests.
3. Reserve quota for each planned call.
4. Call YouTube.
5. Charge or release each reservation.
6. Persist raw candidates with source and rank.
7. Emit `PoolBuildStarted` and `CandidatePoolBuilt`.

**Failure:** a quota refusal stops further calls. The pool keeps the candidates already
found. The shortfall reason is `quota`.

### 5.2 `SelectCandidates`

**Preconditions:** the pool has raw candidates. The `ExclusionSet` is available.

**Steps:**

1. Apply the exclusions and dedupe.
2. Mark micro-blogger candidates.
3. Sort by source priority and discovery rank.
4. Select up to `size`, aiming for 20% micro-bloggers.
5. Assign dense positions.
6. Commit selection and any shortfall.
7. Emit `CandidateSelectionCommitted` and `ShortfallRecorded`.

**Failure:** no usable candidates produces a `shortfall` pool. `feed` shows the reason.

### 5.3 `ReserveQuota`

**Preconditions:** the ledger exists for the UTC day.

**Steps:**

1. Release stale reservations.
2. Refuse if `reserved + charged + units > budget`.
3. Insert the reservation and increase `reserved`.
4. Emit `QuotaReserved`.

**Failure:** `ErrQuotaExhausted`. No reservation is written. `feed` uses the cached feed.

### 5.4 `ChargeQuota`

**Preconditions:** the reservation is `reserved`.

**Steps:**

1. Mark it `charged`.
2. Decrease `reserved` and increase `charged`.
3. Emit `QuotaCharged`.
4. If usage crosses 80% and `warned_at_80` is false, emit `QuotaWarningIssued` and set
   the flag.
5. If `reserved + charged == budget`, set `exhausted` and emit `QuotaExhausted`.

### 5.5 `ReleaseQuota`

**Preconditions:** the reservation is `reserved`.

**Steps:**

1. Mark it `released`.
2. Decrease `reserved`.
3. Emit `QuotaReservationReleased`.

A reservation older than its TTL is released by `ReleaseStale` with reason `expired`.

## 6. Failure modes

| Failure | Result | Recovery |
|---------|--------|----------|
| Budget would be exceeded | `ErrQuotaExhausted` | Use the cached feed. |
| Reservation not found | `ErrReservationNotFound` | Do not charge. Log and stop. |
| Reservation already charged | `ErrReservationFinal` | Return the stored charge. |
| YouTube 5xx | Charge the reservation | Retry once with backoff. |
| Network failure before send | Release the reservation | Retry later. |
| Duplicate candidate | Unique constraint | Keep the first occurrence. |
| No candidates | `shortfall` pool | `feed` shows the explanation. |
