# E6 final gate — 2026-11-05

## Result

The local gate passes. The product is not yet wired for one live end-to-end run with
Google, `yt-dlp` and the local model.

| Gate | Result |
|------|--------|
| `make check` | Pass. Lint, unit tests, contract tests and build. |
| `make coverage-check` | Pass. Every tested product package is at or above 80%. |
| `make test-e2e` | Pass. The integration suite runs with fake external services. |

## Coverage

| Package | Coverage |
|---------|---------:|
| `internal/domain/feed` | 86.0% |
| `internal/domain/identity` | 84.1% |
| `internal/domain/library` | 82.2% |
| `internal/domain/taste` | 94.1% |
| `internal/integration/libraryfiles` | 80.8% |
| `internal/integration/llm` | 84.8% |
| `internal/integration/player` | 85.7% |
| `internal/integration/transcripts` | 81.1% |
| `internal/integration/youtube` | 89.2% |
| `internal/integration/ytdlp` | 84.5% |
| `internal/pkg/trace` | 83.6% |
| `internal/repository/sqlite` | 81.5% |
| `internal/repository/tokenfile` | 82.7% |
| `internal/storage` | 81.9% |
| `internal/transport/cli` | 81.6% |
| `internal/transport/contract` | 86.2% |
| `internal/transport/http` | 84.4% |

## E2E coverage

The integration suite covers the local flows with real SQLite and fake external
processes:

- Gesture, cold period and session resume.
- Cached feed from a saved batch.
- Weekly rating and duplicate week.
- Library save, download, file naming and playback.
- Crash resume with zero transcript re-fetch.

The suite does not cover a live Google consent, a live YouTube call, a live `yt-dlp`
download or a live model call. Those need the owner at the browser and local binaries.

## Post-E6 additions

- **Interactive feed.** TTY card mode with accept, reject, save, navigation, open, refresh
  and quit. `--interactive` forces line mode; `--json` stays one-shot.
- **Step-by-step refresh.** The server streams SSE `progress` events during composition.
  The CLI prints token, interests, profile, planning, search, metadata, selection,
  transcript, analysis and save steps before the new cards appear.

## Open items

1. **Live consent — closed 2026-11-05.** `ytfeed auth status` reported `status=active`
   with the `youtube.readonly` scope and a stored `0600` token.
2. **Live composition — closed 2026-11-06.** `serve` now composes and saves a batch when
   no cached batch exists. `LiveFeedProvider` serves the cache first and calls
   `liveComposer` only when needed. The composer builds the E4 pipeline from config and
   saves the batch snapshot. Verified live with `FEED_SIZE=1`: batch
   `batch-1793961392159263000` was composed from one real YouTube search result, the card
   was stored as `analysis_slow` because the local LLM was unreachable, and a second
   `feed --json` returned the same cached batch.
3. **Stats and rating UI — closed 2026-11-06.** `GET /v1/stats` and `GET/POST
   /v1/rating` are wired. `ytfeed stats [--json]` prints swipes, reasons, cold periods,
   quota trend and topics. `ytfeed rate --value -1|0|1` records the week;
   `ytfeed rate --dismiss` records `unrated`. `ytfeed feed` prints a one-line weekly
   prompt when a rating is due. Verified live: stats reported 502 units for the current
   day and the rating endpoint reported the week as due.
4. **Trace wiring — closed 2026-11-06.** The pipeline records `feed.compose` and
   `feed.analyze`; the HTTP server records `http.request` with method, path and status.
   Verified live: `data/traces/2026-11-06.jsonl` contains a `GET /v1/stats` span.
5. **Real model — closed 2026-11-06.** `llama-server` b12110 served
   `LFM2.5-1.2B-Instruct QAD Q4_0`. The live pipeline composed a card, called the model
   and stored an `analyzed` result: topic `blueberry-consumption-myths`, score `0.6875`.
   The previous cached card was cleared first so the model was actually called.
6. **`OQ-22` — closed 2026-11-06.** The four S-01 axes are rubric v1:
   `depth`, `novelty`, `signal-to-noise`, `practical-value`, each weight `0.5`.
   `rubric_version=v1`. Migration `0012` updates existing profiles; old analyses keep
   `spike-s01`.
