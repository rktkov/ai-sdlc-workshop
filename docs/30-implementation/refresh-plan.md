# Refresh plan — trending, graph and 20 unseen cards

## Status

Approved on 2026-11-06. Implementation is in progress. `T-060` and `T-063` are done.
The owner confirmed every decision in §10.

## 1. Request

The owner wants:

1. **Trending discovery.** Use YouTube `mostPopular` as a candidate source.
2. **Graph discovery.** Use the subscription graph to discover more videos, not only search
   the names of subscribed channels.
3. **A refresh choice.** Let the user choose how the next refresh discovers videos.
4. **20 new videos per refresh.** Every refresh must produce at least 20 cards whose video
   IDs were never shown before.
5. **No repeats.** A video that appeared in any earlier feed must not appear again.

## 2. Current behaviour

The current refresh is deterministic and subscription-bound:

1. Onboarding with `subscriptions-only` takes the five largest subscription channels and
   stores their title slugs as interests.
2. Every refresh reads the same five interests.
3. The planner makes one `search.list` call per interest. The query is the interest slug
   with hyphens turned into spaces.
4. The search is `order=relevance`, `maxResults=50`, `publishedAfter=now-30d`.
5. The selector excludes watched, saved and shown-in-the-last-7-days videos. It has no
   per-channel cap.
6. The pipeline analyzes every selected card before the batch is returned.

That is why a refresh returns videos from one subscription channel again. There is no
randomness, no trending source, no graph source and no full shown history.

## 3. Definitions

| Term | Meaning |
|------|---------|
| New video | A video ID that has no row in `shown_videos` and is not watched or saved. |
| Refresh | Compose a new batch of `FEED_SIZE` cards. The default is 20. |
| Trending | `videos.list?chart=mostPopular` for the configured region. |
| Graph | Recent uploads from subscribed channels and from channels found in search or trending results. |
| Related channel | A channel that owns a search or trending result, not a separate "related channels" API call. |
| Source mode | The set of discovery sources used by one refresh. |

YouTube has no reliable "related channels" endpoint. The graph source must use uploads
playlists (`playlistItems.list`), not a guessed relationship endpoint.

## 4. Design

### 4.1 Source modes

| Mode | Keyword search | Trending | Graph uploads | Use |
|------|:--------------:|:--------:|:-------------:|-----|
| `mixed` (default) | 5 | yes | yes | Best mix of new and relevant |
| `trending` | 5 | yes | no | Escape the subscription bubble |
| `graph` | 5 | no | yes | More from the subscription graph |
| `fresh` | no | yes | yes | Cheap refresh from current sources |
| `keywords` | 5 | no | no | Current behaviour |

The mode is a per-refresh choice. The default comes from `FEED_REFRESH_SOURCE=mixed`. A
successful choice can be remembered for the next interactive refresh.

### 4.2 Source mechanics

**Keywords.** The planner keeps the five interest queries. Interests should be topics, not
channel names. See §4.6.

**Trending.** Call `videos.list?chart=mostPopular&regionCode=<region>&maxResults=50`. Tag
the candidates `trending`.

**Graph.**

1. Take the subscribed channels. Choose up to `GRAPH_CHANNEL_LIMIT` (default 20) by
   subscriber count, recent activity or a stable rotation.
2. Read the uploads playlist ID from `channels.list?part=contentDetails`. The field is
   `contentDetails.relatedPlaylists.uploads`.
3. Call `playlistItems.list` for each selected channel. One page is 50 videos and costs
   1 unit.
4. Also take channels that own keyword or trending results. Choose up to
   `GRAPH_RELATED_LIMIT` (default 10) new channels and read their uploads.
5. Tag all candidates `graph`.

`playlistItems.list` is cheap compared with search. It is the right graph primitive. A
missing or private uploads playlist is skipped, not fatal.

### 4.3 Selection and the 20-new guarantee

Add `shown_videos`:

```sql
CREATE TABLE shown_videos (
    user_id  TEXT NOT NULL,
    video_id TEXT NOT NULL,
    batch_id TEXT NOT NULL,
    shown_at DATETIME NOT NULL,
    PRIMARY KEY (user_id, video_id)
);
```

Rules:

1. Mark every selected card in `shown_videos` when the batch is saved.
2. Exclude **all** rows in `shown_videos` from the next refresh. Do not use a 7-day window.
3. Also exclude watched and saved videos.
4. Add a per-channel cap. Default 2 selected cards from one channel per batch. The cap is
   configurable.
5. Deduplicate by video ID after all sources merge.
6. If fewer than `FEED_SIZE` unseen cards remain, expand the pool:
   - next search page on the queries with the most unseen results;
   - more graph channels;
   - more trending pages if available.
7. Stop at `FEED_SIZE`, the quota budget, or an exhausted source.
8. If fewer than 20 unseen cards are available, return a shortfall banner. Never pad with
   seen videos.

Candidate target is at least 3× `FEED_SIZE` before selection, so exclusions and the
per-channel cap still leave enough cards.

### 4.4 Refresh choice

**CLI flag:**

```sh
ytfeed feed --refresh --source mixed
ytfeed feed --refresh --source trending
ytfeed feed --refresh --source graph
ytfeed feed --refresh --source fresh
ytfeed feed --refresh --source keywords
```

**Interactive:**

Press `r` to open a small source picker:

```text
Refresh source
  1 mixed     trends + subscriptions + search   (default)
  2 trending  trends + search
  3 graph     subscriptions + search
  4 fresh     trends + subscriptions
  5 keywords  search only
Enter uses the last choice. Esc cancels.
```

The picker calls `GET /v1/feed` with `X-Ytfeed-Refresh: true` and
`X-Ytfeed-Source: <mode>`.

**HTTP:**

- `X-Ytfeed-Refresh: true` bypasses the cache.
- `X-Ytfeed-Source: mixed|trending|graph|fresh|keywords` selects the mode.
- An unknown mode returns `400`.

### 4.5 Progressive analysis

Twenty cards on the 1.2B model take about 7–8 minutes at one analysis per 22 seconds.
Waiting for all 20 before showing the feed is poor UX.

Proposed flow:

1. Discovery and selection finish first. Save the batch with 20 pending cards.
2. Mark the cards in `shown_videos`.
3. Stream `batch.started` and 20 `card.upsert` events immediately.
4. Analyze with `ANALYSIS_CONCURRENCY=2`.
5. Stream `card.analyzed` or `card.failed` for each result.
6. Stream `batch.done` when every card is terminal.

The interactive CLI can start navigation on pending cards and update a card when its
analysis event arrives. This also satisfies the PRD streaming requirement.

### 4.6 Interests should be topics, not channel names

The stored interests are channel names. That keeps keyword search inside the current
subscription bubble. Change onboarding:

1. Propose topic phrases from channel topic tags, mostPopular titles and the subscription
   graph.
2. Let the user edit the five interests.
3. Keep `subscriptions-only` as a source, but store topics, not channel titles.
4. `youtube-trending` becomes an available seed source and uses `mostPopular`.

This is a product decision. It changes existing profiles. A migration can keep old
interests, and the user can re-run onboarding to replace them.

## 5. Code areas

| Area | Change |
|------|--------|
| `internal/domain/discovery/query.go` | Add refresh source and query plan. |
| `internal/domain/discovery/selection.go` | Add per-channel cap and source rotation. |
| `internal/integration/youtube` | Add `SearchPage`, `PlaylistItems`, uploads playlist metadata. |
| `internal/domain/discovery/youtube.go` | Extend the adapter port. |
| `internal/repository/sqlite` | Add `shown_videos`, backfill, exclusion query. |
| `internal/domain/feed/pipeline.go` | Merge sources, expand pool, expose progress. |
| `internal/transport/http/feed.go` | Read the source header, stream pending/analysis events. |
| `internal/transport/cli/interactive.go` | Source picker and card update events. |
| `internal/transport/cli/compose.go` | Pass the source mode to the composer. |
| `migrations/0013_shown_videos.sql` | New table. |

## 6. Quota

| Source | Units |
|--------|------:|
| 5 keyword searches | 500 |
| mostPopular | 1 |
| channels metadata (250 IDs, 50 per call) | 5 |
| videos metadata (about 800 IDs, 50 per call) | 16 |
| graph uploads (20 channels + 10 related channels) | 30 |
| **Mixed batch** | **about 552** |

At 80% of 10,000 units, that is about **14 mixed refreshes per day**. A `fresh` refresh is
about 20–30 units. Extra search pages cost 100 each. The quota ledger must reserve
`playlist_items` and extra pages.

## 7. Task plan

| Task | Work | Deps | Size |
|------|------|------|------|
| `T-060` | Refresh source model, HTTP header, CLI flag, interactive picker | — | M |
| `T-061` | YouTube `SearchPage`, `PlaylistItems`, uploads playlist metadata | T-060 | M |
| `T-062` | Trending and graph candidate sources in the pipeline | T-061 | M |
| `T-063` | `shown_videos` table, backfill, full exclusion query, mark on save | — | M |
| `T-064` | Per-channel cap, candidate expansion, 20-unseen shortfall rule | T-062, T-063 | M |
| `T-065` | Progressive batch: save pending cards, analyze with concurrency 2, stream updates | T-064 | L |
| `T-066` | Interactive card updates and source picker integration | T-065 | M |
| `T-067` | Docs, quota update, contract/E2E tests | T-060…T-066 | M |

Suggested delivery order:

1. `T-063` + `T-060` — stop repeats and let the user choose a mode.
2. `T-061` + `T-062` — trending and graph sources.
3. `T-064` — guarantee 20 unseen or show a shortfall.
4. `T-065` + `T-066` — progressive cards and analysis updates.
5. `T-067` — documentation and end-to-end proof.

## 8. Tests

- Unit: source plan for each mode; graph channel selection; `playlistItems` parsing;
  per-channel cap; all-shown exclusion; expansion loop; shortfall.
- Contract: `X-Ytfeed-Source`; unknown mode; pending card and `card.analyzed` events.
- E2E: fake YouTube returns overlapping videos. First refresh returns 20 unseen. Second
  refresh excludes all 20 and returns 20 different videos. A source mode with no search
  still returns cards or an explicit shortfall.
- Quota: mixed batch reserves and charges `search`, `videos`, `channels`,
  `most_popular` and `playlist_items`.

## 9. Risks

| Risk | Effect | Mitigation |
|------|--------|------------|
| Source pool cannot yield 20 unseen | Shortfall | Expand pages/graph; explicit banner |
| 5 searches + graph exceeds budget | Fewer refreshes | 80% warning, source modes, `fresh` |
| 20 analyses are slow | Long refresh | Progressive card stream, concurrency 2 |
| One channel still dominates | Feels repetitive | Per-channel cap, source rotation |
| Interests remain channel names | Keyword search stays narrow | Topic interests, user edit |
| `playlistItems` missing/private | Graph gaps | Skip channel, continue |
| Shown table grows forever | Slow query/storage | Index; optional retention config |
| "Random" is not deterministic | Hard tests | Seeded rotation, recorded seed |

## 10. Decisions confirmed

The owner confirmed all six decisions on 2026-11-06:

1. **20 new** means 20 cards with unseen video IDs, not 20 candidates.
2. **Shown history** is 90 days. `shown_videos` keeps rows for 90 days and prunes older
   rows on every batch save.
3. **Graph definition** is subscribed channels plus channels found in search or trending.
   It never uses separate channel searches.
4. **Progressive analysis** is in scope for this change.
5. **Default source** is `mixed` through `FEED_REFRESH_SOURCE=mixed`.
6. **Interests** become editable topics, not channel names.

## 11. Delivery status

| Task | Status | Notes |
|------|--------|-------|
| `T-060` | DONE | Source model, `FEED_REFRESH_SOURCE`, HTTP header, CLI flag, interactive picker. |
| `T-061` | DONE | `SearchPage`, `PlaylistItems`, uploads playlist metadata and the `playlist_items` quota call. |
| `T-062` | DONE | Trending and graph sources in the pipeline. Graph reads subscription uploads and related channels. |
| `T-063` | DONE | `shown_videos`, 90-day pruning, full exclusion, mark on save. |
| `T-064` | DONE | Per-channel cap, search/graph expansion, source rotation and the 20-unseen shortfall banner. |
| `T-065` | DONE | Pending cards save and stream first. Analysis runs at concurrency 2 and streams analyzed updates. |
| `T-066` | TODO | Interactive card updates and source picker integration. |
| `T-067` | TODO | Docs, quota update, contract and E2E tests. |
