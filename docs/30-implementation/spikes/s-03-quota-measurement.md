# Spike S-03 — YouTube quota measurement

## Status

Confirmed on 2026-11-04. One planned batch costs **510 quota units**. A 20-card batch is
feasible within the 10,000-unit day. The owner chose **five searches per batch** (option A).
The strict ~100-unit target is withdrawn. `OQ-21` is resolved.

## Question

Does a 20-card batch cost at most ~100 YouTube Data API quota units? (`R3`)

## Method

| Item | Value |
|------|-------|
| API | YouTube Data API v3 |
| Project | `giigii-app` |
| Credential | Existing `youtube-data-api-key`, restricted to `youtube.googleapis.com` |
| Credential location | `.secrets/youtube/api_key` (gitignored, `0600`) |
| Data | Public data only. No OAuth, no subscriptions, no mostPopular |
| Batch shape | 5 interests, 50 results per search, 30-day window, region `US` |
| Interests | AI agents, local LLM, distributed systems, developer productivity, machine learning |
| Harness | [`scripts/spikes/quota`](../../../scripts/spikes/quota) |
| Report | [`s-03-quota-result.json`](s-03-quota-result.json) |

The interests are representative. The real profile does not exist yet. The search count is
the variable that matters, not the words.

The harness counts calls and applies the documented unit cost. Google does not expose the
remaining quota through an endpoint. A `200` status proves that the call succeeded and was
billable.

### Unit cost

| Call | Units | Note |
|------|------:|------|
| `search.list` | 100 | One page. The dominant cost |
| `videos.list` | 1 | Up to 50 IDs |
| `channels.list` | 1 | Up to 50 IDs |
| `subscriptions.list` | 1 | Up to 50 subscriptions per page |
| `videos.list?chart=mostPopular` | 1 | One page |

## Result

| Call | Count | Units each | Units |
|------|------:|-----------:|------:|
| `search.list` | 5 | 100 | 500 |
| `videos.list` | 5 | 1 | 5 |
| `channels.list` | 5 | 1 | 5 |
| **Total** | **15** | | **510** |

| Metric | Value |
|--------|------:|
| Unique video IDs | 249 |
| Unique channel IDs | 207 |
| Batches per day at 80% (8,000 units) | 15 |
| Batches per day at 100% (10,000 units) | 19 |
| Strict ≤100-unit target | No |

All calls returned `200`. No quota error occurred. The project had enough quota for the
measurement.

Optional calls do not change the result. `mostPopular` adds 1 unit. `subscriptions.list`
adds 1 unit per 50 subscriptions. A user with 200 subscriptions adds 4 units. The worst
case is about 515 units.

## Verdict

**20 cards: GO.** One batch costs 510 units. The 80% budget supports 15 batches per day.
The full budget supports 19. A daily user needs one or two batches. Quota is not the limit
for the default batch size.

**Strict ~100 units: NO-GO.** Five searches alone cost 500 units. One search plus the
minimum metadata calls costs about 102–105 units. The ~100-unit target therefore permits
one search per batch and almost no metadata calls.

## Finding — the target conflicts with the design

The system design estimated **2–5 searches per batch**, or **200–500 units**
([`batch-pipeline.md`](../../20-system-design/flows/batch-pipeline.md)). The measurement
confirms the upper end of that estimate and sets it as the contract.

The PRD had two statements that did not agree with that estimate:

1. `R3` said "~100 searches/day at 10k units". Five searches per batch gives 20 batches
   per day, not a shortage.
2. §15.4 said that a batch above ~100 units leaves "fewer than 3 batches" per day. This is
   false for 510 units. `10,000 / 510` is about 19 batches, not fewer than 3.

The ~100-unit target permits one search per batch and almost no metadata calls. The design
estimated 2–5 searches. The two could not both be true.

## Decision

The owner chose **option A** on 2026-11-04: five searches per batch, 510 units.

| Option | Planner | Units per batch | Batches/day at 80% | Result |
|--------|---------|----------------:|-------------------:|--------|
| **A (chosen)** | 5 interests, 5 searches | 510 | 15 | Keep the current design. Correct `R3` and §15.4. 20 cards GO |
| B (rejected) | 1 search per batch, rotate interests; fill with mostPopular and subscriptions | ~105 | ~76 | Meet the ~100-unit intent. Each batch covers one interest, not five |

Option A keeps candidate breadth and micro-blogger balance simple. The daily budget is
ample for one owner. Option B would narrow each batch to one interest and add work to
`T-030` and `T-032`.

## Consequence for T-030 and T-032

- `T-030` plans five searches per batch, one per interest.
- `T-032` reserves 510 units before a batch.
- `T-032` must not use 100 units as a hard gate for a 20-card batch. The minimum for one
  search plus metadata is already about 102 units.
- `F7` behavior does not change: warn at 80%, refuse at 100%, serve the cached feed.

## Limits

- The harness did not measure `subscriptions.list`. It needs OAuth, which is `T-010`.
  The cost is 1 unit per page, so the estimate is safe.
- The harness did not measure `mostPopular`. It costs 1 unit.
- The quota is per Google Cloud project, not per product user. Several users would share
  the 10,000 units.
- The quota resets at midnight Pacific, not UTC. This is `OQ-02`.
- A different number of results per search changes the metadata calls by a few units. The
  five searches stay at 500 units.

## Reproduce

```sh
# Plan only. No API call.
go run ./scripts/spikes/quota -dry-run

# Real measurement. Writes the JSON report.
go run ./scripts/spikes/quota -out docs/30-implementation/spikes/s-03-quota-result.json
```

The key comes from `YOUTUBE_API_KEY` or `.secrets/youtube/api_key`. The harness never
prints it.

## Security

- The key is stored outside git at `.secrets/youtube/api_key` with mode `0600`.
- `.secrets/` is in `.gitignore`.
- The JSON report contains counts, not the key or video titles.
- The harness redacts the key from error messages.

## Addendum — refresh overhaul (2026-11-07)

The refresh overhaul adds trending, graph uploads and shown history. The base measurement
above still applies to a keyword-only refresh. The active estimate for all refresh modes is
[`refresh-quota.md`](../refresh-quota.md). It includes `playlistItems.list`, per-50-ID
metadata chunks and extra search pages.
