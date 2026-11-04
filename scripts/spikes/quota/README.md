# Quota spike harness

Measures the YouTube Data API cost of one planned batch. This is spike code. It is not
part of the product binary.

## Usage

```sh
# Plan only. No API call and no key needed.
go run ./scripts/spikes/quota -dry-run

# Real measurement.
go run ./scripts/spikes/quota \
  -out docs/30-implementation/spikes/s-03-quota-result.json
```

The key comes from `YOUTUBE_API_KEY` or `.secrets/youtube/api_key`. The harness never
prints the key.

## Flags

| Flag | Default | Use |
|------|---------|-----|
| `-interests` | five representative topics | Comma-separated interests |
| `-max-results` | `50` | Results per search, 1–50 |
| `-days` | `30` | `publishedAfter` window |
| `-region` | `US` | Search and mostPopular region |
| `-key-file` | `.secrets/youtube/api_key` | API key file |
| `-out` | empty | Write the JSON report |
| `-dry-run` | false | Print the planned cost only |
| `-most-popular` | false | Add one `videos.list?chart=mostPopular` call |
| `-access-token` | empty | OAuth token, needed for subscriptions |
| `-subscriptions` | false | Page `subscriptions.list` |

## Cost model

| Call | Units |
|------|------:|
| `search.list` | 100 |
| `videos.list` | 1 per 50 IDs |
| `channels.list` | 1 per 50 IDs |
| `subscriptions.list` | 1 per 50 subscriptions |
| `mostPopular` | 1 |

The harness counts billable calls. Google does not expose the remaining quota. A `200`
status proves that the call succeeded.
