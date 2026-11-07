# ytfeed

`ytfeed` is a local-first YouTube feed composer. It runs on one laptop, discovers videos from YouTube, fetches transcripts, analyzes them with a local LLM against a versioned taste profile, and serves a finite card feed.

The feed is small on purpose. You accept, reject, save or open each card. Those gestures train the profile. Discovery uses keyword search, YouTube trending and subscription uploads. Every refresh targets unseen videos and never reuses a video that was shown before.

The project is also the working example for the AI-SDLC workflow in this repository. The rules, prompts and design documents used to build it are part of the repo.

## What it does

- Composes a feed of up to 20 cards.
- Supports five refresh sources: `mixed`, `trending`, `graph`, `fresh` and `keywords`.
- Keeps a 90-day shown-history and excludes every previously shown video.
- Caps cards per channel, default 2.
- Streams pending cards first, then updates each card as analysis finishes.
- Stores watched, saved, rejected and accepted signals.
- Runs entirely on loopback. One binary, one local SQLite database, no hosted service.

## Documentation map

| Document | Why read it |
|---|---|
| [`docs/00-rules/README.md`](docs/00-rules/README.md) | Engineering contract. Read this before changing code. |
| [`docs/10-product/prd-ytfeed.md`](docs/10-product/prd-ytfeed.md) | Product requirements and scope. |
| [`docs/20-system-design/README.md`](docs/20-system-design/README.md) | System design index: C4, DDD, flows, invariants. |
| [`docs/20-system-design/adr/INDEX.md`](docs/20-system-design/adr/INDEX.md) | Accepted architecture decisions. |
| [`AGENTS.md`](AGENTS.md) | Mandatory workflow for human and AI contributors. |
| [`docs/00-rules/11-ai-agent-workflow.md`](docs/00-rules/11-ai-agent-workflow.md) | Task selection, test-first rule, definition of done. |
| [`docs/30-implementation/tasks.md`](docs/30-implementation/tasks.md) | Task register and status. |
| [`docs/30-implementation/backlog.md`](docs/30-implementation/backlog.md) | Implementation sequence and progress history. |
| [`docs/30-implementation/test-plan.md`](docs/30-implementation/test-plan.md) | Test-first plan and test layers. |
| [`docs/30-implementation/oauth-setup.md`](docs/30-implementation/oauth-setup.md) | Google OAuth client setup. |
| [`docs/30-implementation/refresh-quota.md`](docs/30-implementation/refresh-quota.md) | Quota cost of each refresh mode. |
| [`docs/prompts/`](docs/prompts/) | Prompt library for specs, architecture, user stories and reviews. |

## Requirements

### Required for code and tests

| Tool | Version | Notes |
|---|---|---|
| Go | `1.26.0` or newer | The module declares `go 1.26.0`. |
| golangci-lint | `v2.6.2` | Pinned by the Makefile. |
| Git | current | Conventional Commits are enforced by review. |

The unit, contract and E2E suites use fakes. They do not need Google, an LLM or `yt-dlp`.

### Required for a live feed

| Tool | Why |
|---|---|
| Google OAuth desktop client JSON | Subscriptions and live YouTube calls. The scope is `youtube.readonly`. |
| OpenAI-compatible LLM endpoint | Transcript analysis. Default endpoint: `http://localhost:11434/v1`. |
| `yt-dlp` | Transcript download. Default binary: `yt-dlp`. |
| SQLite | Embedded through `modernc.org/sqlite`. No separate install. |

## Quick start

Run these commands from the repository root, the directory that contains `go.mod`.

```sh
git clone https://github.com/rktkov/ai-sdlc-workshop.git
cd ai-sdlc-workshop

cp .env.example .env
# Edit .env for your machine.

set -a
. ./.env
set +a

go build -o bin/ytfeed ./cmd/ytfeed
make check
```

`ytfeed` reads environment variables only. It does not load `.env` by itself. Use `set -a; . ./.env; set +a` in each shell, or a tool such as `direnv`.

Install the lint tool if it is missing:

```sh
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.6.2
```

## Local development setup

### 1. Start the LLM

Any endpoint that speaks OpenAI-compatible `/v1/chat/completions` works. The default model is `lfm2.5-1.2b-instruct`, chosen for the target laptop.

Ollama example:

```sh
ollama serve
ollama pull lfm2.5-1.2b-instruct
export LLM_BASE_URL=http://localhost:11434/v1
export LLM_MODEL=lfm2.5-1.2b-instruct
```

llama.cpp example:

```sh
llama-server -m /path/to/model.gguf --host 127.0.0.1 --port 8081
export LLM_BASE_URL=http://127.0.0.1:8081/v1
export LLM_MODEL=/path/to/model.gguf
```

Keep `LLM_JSON_MODE=true`. Analysis output is schema-constrained JSON. A model that cannot follow the schema produces explicit failed cards, not guessed values.

### 2. Install `yt-dlp`

```sh
brew install yt-dlp
# or
python3 -m pip install -U yt-dlp
```

Check it:

```sh
yt-dlp --version
```

### 3. Configure Google OAuth

1. Create a Google Cloud project.
2. Enable the YouTube Data API v3.
3. Configure an OAuth consent screen.
4. Create an OAuth client of type **Desktop app**.
5. Download the client JSON. Keep it outside the repository.
6. Export the path:

```sh
export GOOGLE_OAUTH_CLIENT_FILE=/absolute/path/to/client_secret.json
```

Then authorize and onboard:

```sh
bin/ytfeed auth login
bin/ytfeed auth status
bin/ytfeed onboard --source subscriptions-only
```

`auth login --no-browser` prints the URL instead of opening a browser. The callback uses a loopback address on `127.0.0.1`.

The OAuth client file and the token are secrets. Never commit them. See [`docs/30-implementation/oauth-setup.md`](docs/30-implementation/oauth-setup.md).

### 4. Configure the data directory

The default is `./data`. Override it with `YTFEED_DATA_DIR`.

The data directory contains:

| Path | Contents |
|---|---|
| `ytfeed.db` | SQLite database. Migrations run on first `doctor` or `serve`. |
| `ytfeed.db.bak-*` | Automatic backup before a migration. |
| `api-token` | Local API bearer token, mode `0600`. |
| `token` | Google OAuth token, mode `0600`. |
| `traces/` | Local JSONL traces, one file per day. |

`data/`, `.env`, `.secrets/` and database files are git-ignored.

## Run the application

### Check the environment first

```sh
bin/ytfeed doctor
```

`doctor` prints the data directory, database path, schema version, API token state, OAuth token state, quota, `yt-dlp`, LLM reachability, terminal image support and server state. It is the first debugging step.

### Start the local server

```sh
bin/ytfeed serve
```

The server binds to `127.0.0.1:7420`. It stores a bearer token in `<data>/api-token` and the CLI reads it automatically. Do not bind to a public interface. `--allow-remote` exists for exceptional cases and prints a warning.

### Compose and browse the feed

In another shell:

```sh
bin/ytfeed feed
```

Press `r` inside interactive mode to refresh. The source picker appears:

```text
1 mixed     trends + subscriptions + search (default)
2 trending  trends + search
3 graph     subscriptions + search
4 fresh     trends + subscriptions
5 keywords  search only
```

Refresh from the command line:

```sh
bin/ytfeed feed --refresh --source mixed
bin/ytfeed feed --refresh --source fresh
bin/ytfeed feed --json
```

`--source` accepts `mixed`, `trending`, `graph`, `fresh` and `keywords`. An unknown value exits with code `2`.

For progressive cards, use interactive mode and press `r`. `feed --refresh --interactive` composes the whole batch before it opens the cards.

### Other commands

```sh
bin/ytfeed stats
bin/ytfeed rate --value 1
bin/ytfeed rate --dismiss
bin/ytfeed onboard --reset
```

## Build, test and debug

### Build

```sh
make build                       # compile every package
go build -o bin/ytfeed ./cmd/ytfeed
go run ./cmd/ytfeed doctor
```

`make build` does not write `bin/ytfeed`. Use the explicit `go build` command for a runnable binary.

### Test targets

| Command | What it runs | Needs external services |
|---|---|---|
| `make test` | Unit tests across all packages | No |
| `make test-contract` | CLI and HTTP contract tests | No |
| `make test-e2e` | Integration-tagged end-to-end tests | No |
| `make coverage` | Per-package coverage summary | No |
| `make coverage-check` | Fails when a tested package is below 80% | No |
| `make lint` | `golangci-lint` with the pinned config | No |
| `make check` | Lint, unit tests, contract tests and build | No |

Run the full local gate before you report a change:

```sh
make check
make coverage-check
make test-e2e
```

`make check` does not include coverage or E2E. Run all three for the full status.

Useful focused commands:

```sh
go test ./internal/domain/feed -run TestPipelineHappy -v
go test ./internal/domain/discovery -run TestSelect -v
go test -tags=integration ./e2e -run TestRefreshReturnsTwentyNewVideos -v
go test -race ./...
```

Coverage report:

```sh
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Debug the running app

1. Run `bin/ytfeed doctor`.
2. Check the server line. If it says `server: unavailable`, start `bin/ytfeed serve`.
3. Check the token line. If it says `expired`, run `bin/ytfeed auth status` with `GOOGLE_OAUTH_CLIENT_FILE` set. If it says `missing`, run `bin/ytfeed auth login`.
4. Check the LLM line. If it says `unreachable`, start the LLM or fix `LLM_BASE_URL`.
5. Check `yt-dlp`. If it is missing, install it or set `YTDLP_BIN`.
6. Check the quota line. A refresh refuses at 100% of the daily budget.

Live traces are written to `data/traces/YYYY-MM-DD.jsonl`. They contain spans, not transcript content or secrets.

### Common failures

| Symptom | Likely cause | Fix |
|---|---|---|
| `server unavailable` | `serve` is not running | Start `bin/ytfeed serve`. |
| `token: missing` | No Google token | Run `bin/ytfeed auth login`. |
| `token: expired` | Access token expired | Run `bin/ytfeed auth status` with the client file set. |
| `llm: unreachable` | No local model server | Start Ollama or `llama-server`, then check `LLM_BASE_URL`. |
| `yt-dlp` missing | Binary not on `PATH` | Install `yt-dlp` or set `YTDLP_BIN`. |
| `quota exhausted` | Daily YouTube quota reached | Wait for the next UTC day or use a cheaper refresh mode. |
| `no transcript` cards | Video has no usable transcript | Expected failure state. The card is not padded or replaced. |
| `interests_not_set` | Onboarding step 2 was skipped | Run `bin/ytfeed onboard`. |

## Contributing

### Read order

1. [`AGENTS.md`](AGENTS.md).
2. [`docs/00-rules/README.md`](docs/00-rules/README.md) and every rule file that applies to the task.
3. [`docs/10-product/prd-ytfeed.md`](docs/10-product/prd-ytfeed.md).
4. [`docs/20-system-design/README.md`](docs/20-system-design/README.md) and the relevant ADRs.
5. [`docs/30-implementation/tasks.md`](docs/30-implementation/tasks.md) and [`docs/30-implementation/backlog.md`](docs/30-implementation/backlog.md).

### Human contributor workflow

1. Pick one task. Do not mix unrelated changes.
2. Write the failing test first. The failure must be for the expected reason.
3. Implement the smallest change that makes the test pass.
4. Run `make check`, `make coverage-check` and `make test-e2e`.
5. Update `docs/30-implementation/backlog.md` and `docs/30-implementation/tasks.md`.
6. Commit with Conventional Commits and put the task ID in the body:

```text
feat(feed): compose cards with cold-period filtering

Refs: T-012
```

7. Never commit `.env`, `.secrets/`, `data/`, tokens or database files.

### Working with a code agent

Start the agent in the repository root and give it one task ID at a time. The agent must read `AGENTS.md` first and follow the same rules as a human contributor.

- The agent writes tests before code.
- The agent runs the full gate and pastes the output.
- The agent updates the backlog and task register.
- The agent reports in the fixed format from `AGENTS.md` §4.
- The agent never runs `git push`. The owner pushes.
- The agent stops and asks when a rule, product decision or dependency is unclear.

The prompt library in [`docs/prompts/`](docs/prompts/) is for producing specs, architecture notes, user stories and reviews. It does not replace the rules or the definition of done.

## Repository layout

| Path | Contents |
|---|---|
| `cmd/ytfeed/` | The single binary entry point. |
| `internal/transport/cli/` | CLI commands, interactive feed, composition wiring. |
| `internal/transport/http/` | Local API, SSE feed stream, auth middleware. |
| `internal/domain/identity/` | OAuth, tokens, subscriptions. |
| `internal/domain/taste/` | Profile, signals, cold periods, rubric weights. |
| `internal/domain/discovery/` | Search plan, sources, selection, quota. |
| `internal/domain/analysis/` | Transcripts and LLM analysis contract. |
| `internal/domain/feed/` | Batch pipeline, cards, sessions, gestures. |
| `internal/domain/library/` | Saved and downloaded videos. |
| `internal/integration/` | YouTube, LLM, `yt-dlp`, transcripts, player, files. |
| `internal/repository/sqlite/` | All SQL and persistence. |
| `internal/storage/` | Database open, migrations, backups. |
| `migrations/` | Embedded goose migrations. |
| `e2e/` | Integration-tagged end-to-end tests. |
| `docs/` | Rules, product, design, implementation and prompts. |
| `scripts/spikes/` | One-off measurement tools kept for reference. |

## Security and privacy

- The app binds to loopback only.
- Every local API call needs the bearer token from `<data>/api-token`.
- Google access is read-only: `youtube.readonly`.
- Transcript text is staged briefly and deleted after analysis.
- The database, tokens, traces and downloaded media stay local.
- Secrets are never logged, traced or committed.

## Status

The status of the code is the gate output, not a line in this file.

Run:

```sh
make check
make coverage-check
make test-e2e
```

All three pass on a healthy checkout. Live feed verification additionally needs OAuth, `yt-dlp` and a reachable LLM.
