# `taste` — value objects

## Metadata

| Field | Value |
|-------|-------|
| **Context** | `taste` |
| **Backlog** | `ARCH-301` |
| **Status** | Confirmed at gate `G5` (2026-11-04) |
| **Parent** | [`README.md`](README.md) |
| **Sources** | `F3`, `F4`, `F5`, `F17`, `F19`, `A-20`, `C-17`, `D-13`, `D-22`, `CLI-18`, `CLI-20a` |

A value object has no identity. Two instances with the same fields are equal. Validation
happens at construction. An invalid value object cannot exist.

## 1. Identity and reference objects

These objects carry a key from another context or from the local context. They are values,
not foreign keys (`context-map.md` §7).

| Object | Field | Validation | Equality |
|--------|-------|------------|----------|
| `UserID` | `value` | Non-empty, at most 128 characters, no whitespace, no path separator | By `value` |
| `SessionID` | `value` | Non-empty, at most 128 characters, no whitespace | By `value` |
| `CardID` | `value` | Non-empty, at most 128 characters, no whitespace | By `value` |
| `VideoID` | `value` | Non-empty, at most 64 characters, `[A-Za-z0-9_-]` only | By `value` |
| `ChannelID` | `value` | Non-empty, at most 64 characters, `[A-Za-z0-9_-]` only | By `value` |
| `SignalID` | `value` | Non-empty, at most 128 characters | By `value` |
| `GestureID` | `value` | Non-empty, at most 128 characters. The idempotency key. | By `value` |
| `ColdPeriodID` | `value` | Non-empty, at most 128 characters | By `value` |

`VideoID` and `ChannelID` accept the YouTube shape but do not encode YouTube-specific
rules. The adapter validates the external shape. The domain treats the value as opaque
(`A-12`).

## 2. Profile objects

| Object | Fields | Validation | Notes |
|--------|--------|------------|-------|
| `AxisName` | `value` | Normalized slug, 1–64 characters, `[a-z0-9]+(-[a-z0-9]+)*` | Unique inside one profile. Example: `depth-over-hype`. |
| `AxisWeight` | `value` | Decimal in `[0,1]`, at most 4 fractional digits | Default 0.5. The provisional update step is ±0.05 (`OQ-18`). |
| `Example` | `text` | Trimmed, 1–500 characters, no control characters | Used for positive and negative examples. |
| `ProfileNote` | `text` | Optional, at most 2000 characters, no control characters | Never replaces axes (`A-20`). |
| `ProfileVersion` | `value` | Positive integer | Optimistic concurrency token. |
| `InterestSlug` | `value` | Normalized slug, 1–64 characters, same grammar as `TopicSlug` | 1–10 per profile. Shares one namespace with `TopicSlug` (`G4-7`). |

### 2.1 Slug normalization

1. Trim leading and trailing whitespace.
2. Convert to lowercase ASCII.
3. Replace whitespace, underscores and repeated separators with one hyphen.
4. Remove characters outside `[a-z0-9-]`.
5. Reject an empty result or a leading or trailing hyphen.

Interests and topics use the same normalization. A topic from `analysis` and an interest
from onboarding can therefore match (`G4-7`). Topic tags are YouTube metadata and do not
use this namespace.

## 3. Gesture objects

| Object | Fields | Validation | Notes |
|--------|--------|------------|-------|
| `SignalGesture` | `value` | One of `left`, `right`, `save` | `up` and `down` are navigation and are rejected (`D-304`). |
| `RejectionReason` | `value` | One of `marketing`, `ai-slop`, `value-misaligned`, `other` | Required for `left`. Forbidden for `right` and `save`. |
| `AxisScore` | `axis`, `score` | `axis` is an `AxisName`. `score` is a decimal in `[0,1]` with at most 4 fractional digits. | The justification stays in `analysis`. |
| `ScoredAxes` | `list` | 0 or more `AxisScore` values. Axis names are unique. | An unknown axis is ignored for the weight update (`D-305`). |

## 4. Cold-period objects

| Object | Fields | Validation | Notes |
|--------|--------|------------|-------|
| `ColdPeriodScope` | `value` | One of `topic`, `channel` | Derived from the reason (`CLI-20a`). |
| `ColdPeriodTarget` | `scope`, `value` | `topic` requires an `InterestSlug` value. `channel` requires a `ChannelID`. | The scope and target must agree (`I-C2`). |
| `ColdPeriodDuration` | `value` | Positive duration, parsed from `24h`, `72h` or `168h` | Default 24h (`COLD_PERIOD_DEFAULT`). No upper bound in v1. |
| `Expiry` | `expires_at` | UTC timestamp, later than `created_at` | A refresh moves it forward (`I-C4`). |

## 5. Shared rules

| Rule | Detail |
|------|--------|
| Trim text | Every text value is trimmed before validation. |
| UTC | Every timestamp is stored and compared in UTC (`D-13`). |
| No empty IDs | An empty or whitespace-only identifier is invalid. |
| No path characters | An identifier never contains `/`, `\`, `..` or a NUL byte. |
| No secrets | No value object carries a token, a transcript or an API key (`C-2`, `L-10`, `E-17`). |
| No rubric content | No value object carries rubric text. Only `rubric_version` is a reference (`G4-8`). |

## 6. Equality summary

| Object | Equal when |
|--------|------------|
| Identifier objects | All characters match exactly. |
| `AxisName`, `InterestSlug`, `TopicSlug` | Normalized values match. |
| `AxisWeight`, `AxisScore` | Numeric values match after the 4-digit scale. |
| `Example`, `ProfileNote` | Text matches after trimming. |
| `ColdPeriodTarget` | Scope and value both match. |
| `Expiry` | Instants match in UTC. |
