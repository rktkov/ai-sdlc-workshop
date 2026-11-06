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

## Open items

1. **Live consent — closed 2026-11-05.** `ytfeed auth status` reported `status=active`
   with the `youtube.readonly` scope and a stored `0600` token.
2. **Live composition — closed 2026-11-06.** `serve` now composes and saves a batch when
   no cached batch exists. `LiveFeedProvider` serves the cache first and calls
   `liveComposer` only when needed. The composer builds the E4 pipeline from config and
   saves the batch snapshot.
3. **Stats and rating UI.** `T-051` and `T-053` have domain and SQLite tests. The
   `ytfeed stats` and weekly prompt are not wired to the CLI or HTTP API.
4. **Trace wiring.** `T-052` writes spans, but the pipeline and transport do not call the
   trace logger yet.
5. **Real model.** `T-035` is tested with an OpenAI-compatible fake. The S-01 spike
   measured `llama-server`, but the product pipeline has not called it in a live run.
6. **`OQ-22`.** The onboarding profile is still the provisional `spike-s01` profile.
