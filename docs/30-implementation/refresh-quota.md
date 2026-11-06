# Refresh quota

## Status

Active from 2026-11-07. This document replaces the quota estimate for the refresh overhaul.
The base measurement is in [`spikes/s-03-quota-measurement.md`](spikes/s-03-quota-measurement.md).

## Unit costs

| Call | Units | Note |
|------|------:|------|
| `search.list` | 100 | One page of up to 50 results |
| `videos.list` | 1 | One chunk of up to 50 video IDs |
| `channels.list` | 1 | One chunk of up to 50 channel IDs |
| `subscriptions.list` | 1 | One page of up to 50 subscriptions |
| `videos.list?chart=mostPopular` | 1 | One page of up to 50 videos |
| `playlistItems.list` | 1 | One page of up to 50 uploads |

The pipeline reserves and charges one unit for each 50-ID chunk. It never charges one unit
for a call that contains more than 50 IDs.

## Refresh modes

| Mode | Search pages | Trending | Graph channels | Upload pages |
|------|-------------:|---------:|---------------:|-------------:|
| `mixed` | 5 | 1 | 20 subscribed + 10 related | 30 |
| `trending` | 5 | 1 | 0 | 0 |
| `graph` | 5 | 0 | 20 subscribed + 10 related | 30 |
| `fresh` | 0 | 1 | 20 subscribed + 10 related | 30 |
| `keywords` | 5 | 0 | 0 | 0 |

## Cost estimate

The estimates below use the default limits: 20 subscribed channels, 10 related channels,
2 cards per channel and 50 results per page.

| Item | Keywords | Trending | Graph | Fresh | Mixed |
|------|---------:|---------:|------:|------:|------:|
| Search | 500 | 500 | 500 | 0 | 500 |
| Trending | 0 | 1 | 0 | 1 | 1 |
| Graph channel metadata | 0 | 0 | 1 | 1 | 1 |
| Graph uploads | 0 | 0 | 30 | 30 | 30 |
| Video metadata | 5 | 6 | 35 | 31 | 36 |
| Channel metadata | 5 | 6 | 35 | 31 | 36 |
| **Total** | **510** | **513** | **601** | **94** | **604** |

The mixed estimate assumes 1,800 unique candidates. The real count is often lower. Each
extra search page costs 100 units. A graph channel read costs 1 unit for the upload page.

At 80% of the 10,000-unit daily budget (8,000 units), the budget supports about:

| Mode | Refreshes per day |
|------|------------------:|
| `keywords` | 15 |
| `trending` | 15 |
| `mixed` | 13 |
| `graph` | 13 |
| `fresh` | 85 |

One daily user needs one or two refreshes. Quota is not the limit for normal use. The 80%
warning and 100% refusal still apply.

## Ledger behaviour

1. The pipeline reserves units before each call and charges after the call.
2. A call that fails still charges, because YouTube may bill it.
3. `playlist_items` and extra search pages use the same reserve and charge path.
4. At 80% the ledger warns once for the day.
5. At 100% the ledger refuses the call. The refresh returns a degraded quota batch or
   serves the cached feed.
6. Graph quota exhaustion stops graph expansion. Cards that are already selected remain
   valid, and the batch shows the shortfall banner if it is smaller than the target.
