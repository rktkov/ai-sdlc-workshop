# `feed` — value objects

## Metadata

| Field | Value |
|-------|-------|
| **Context** | `feed` |
| **Backlog** | `ARCH-302` |
| **Status** | Confirmed at gate `G5` (2026-11-04) |
| **Parent** | [`README.md`](README.md) |
| **Sources** | `F13`, `F14`, `F16`, `F21`, `CLI-14a`, `CLI-28`, `D-13`, `D-22` |

A value object has no identity. Two instances with the same fields are equal. Validation
happens at construction. An invalid value object cannot exist.

## 1. Identity and reference objects

| Object | Field | Validation | Equality |
|--------|-------|------------|----------|
| `BatchID` | `value` | Non-empty, at most 128 characters, no whitespace | By `value` |
| `CardID` | `value` | Non-empty, at most 128 characters, no whitespace | By `value` |
| `SessionID` | `value` | Non-empty, at most 128 characters, no whitespace | By `value` |
| `RatingID` | `value` | Non-empty, at most 128 characters | By `value` |
| `AnalysisID` | `value` | Non-empty, at most 128 characters | By `value` |
| `UserID` | `value` | Non-empty, at most 128 characters | By `value` |
| `VideoID` | `value` | Non-empty, at most 64 characters, `[A-Za-z0-9_-]` only | By `value` |
| `ChannelID` | `value` | Non-empty, at most 64 characters, `[A-Za-z0-9_-]` only | By `value` |
| `GestureID` | `value` | Non-empty, at most 128 characters | By `value` |
| `LeaseOwner` | `value` | Non-empty, at most 128 characters | By `value` |

`VideoID`, `ChannelID` and `AnalysisID` are references from other contexts. They are
values, not foreign keys (`context-map.md` §7).

## 2. Ordering objects

| Object | Fields | Validation | Notes |
|--------|--------|------------|-------|
| `Position` | `value` | Non-negative integer | Dense and unique inside a batch (`I-B3`). |
| `DiscoveryRank` | `value` | Non-negative integer | Stable order from `CandidateSelection`. |
| `Score` | `value` | Decimal in `[0,1]`, at most 4 fractional digits | Present when a card is analyzed. |
| `Cursor` | `batch_id`, `current_card_id`, `furthest_card_id` | All cards belong to the batch. `furthest` is never behind the last reached card. | The cursor uses card identity (`D-312`). |
| `WeekStart` | `date` | Monday, UTC | One rating per week (`I-R1`). |

## 3. State and decision objects

| Object | Allowed values | Validation | Notes |
|--------|----------------|------------|-------|
| `BatchState` | `composing`, `streaming`, `completed`, `degraded` | Terminal states are final | `degraded` carries one reason. |
| `CardState` | `pending`, `analyzed`, `failed` | `analyzed` requires a score and analysis ID. `failed` requires a reason. | `pending` is a valid state (`E-12`). |
| `CardDecision` | `none`, `rejected`, `accepted`, `saved` | A rejected card has a signal reason. | `none` covers an unseen or skipped card. |
| `SessionState` | `open`, `closed` | A closed session accepts no writes. | One open session per user. |
| `RatingValue` | `-1`, `0`, `+1`, `unrated` | Exactly one value | `unrated` is recorded, not omitted. |
| `AnalysisReason` | `no_transcript`, `analysis_slow`, `invalid_model_output` | Exactly one value when failed | Owned by `analysis`, copied by `feed`. |
| `DegradationReason` | `quota_exhausted`, `network_unavailable`, `ytdlp_missing`, `partial` | Exactly one value when degraded | Maps to `F24` and `OQ-12`. |

## 4. `CardPresentation`

`CardPresentation` is the display snapshot that `feed` copies from `AnalysisResult`. It is
not the analysis schema. It exists so a card can render without a cross-context join
(`D-319`).

| Field | Type | Validation |
|-------|------|------------|
| `topic` | `TopicSlug` | Normalized slug, 1–64 characters. |
| `tldr` | text | Trimmed, 1–500 characters. |
| `similarity` | text | "similar to: X" or "new territory", at most 200 characters. |
| `new_territory` | boolean | True only when the analysis says so. |
| `why_this_card` | text | Trimmed, 1–500 characters. |

`CardPresentation` is absent while a card is `pending`. It is absent for a `failed` card
unless the analysis produced a partial presentation. `feed` never invites a partial
presentation from a failed result (`A-12b`).

## 5. Shared rules

| Rule | Detail |
|------|--------|
| Trim text | Every text value is trimmed before validation. |
| UTC | Every timestamp and week boundary is UTC (`D-13`). |
| No empty IDs | An empty or whitespace-only identifier is invalid. |
| No path characters | An identifier never contains `/`, `\`, `..` or a NUL byte. |
| No secrets | No value object carries a token, a transcript or an API key (`C-2`, `L-10`). |
| No analysis schema | `CardPresentation` is a small copy, not the `AnalysisResult` type (`context-map.md` §6.2). |

## 6. Equality summary

| Object | Equal when |
|--------|------------|
| Identifier objects | All characters match exactly. |
| `Position`, `DiscoveryRank` | Integer values match. |
| `Score` | Numeric values match after the 4-digit scale. |
| `Cursor` | Batch and both card IDs match. |
| `WeekStart` | Calendar dates match. |
| `CardPresentation` | Every field matches. |
