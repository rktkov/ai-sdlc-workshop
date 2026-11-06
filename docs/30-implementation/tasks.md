# Implementation tasks

> **Status:** ACTIVE from 2026-11-04. `T-001` is DONE.
> **Date:** 2026-11-04
> **Backlog:** [`backlog.md`](backlog.md)

## Progress

| ID | Status | Date | Note |
|----|--------|------|------|
| `T-001` | DONE | 2026-11-04 | Repo skeleton, Makefile and lint config. `make check` passes. |
| `T-001a` | DONE | 2026-11-04 | Test harness, contract fixtures and E2E fakes. All targets pass. |
| `T-002` | DONE | 2026-11-04 | Config validation, structured logger and redacted output. Happy and unhappy tests pass. |
| `T-003` | DONE | 2026-11-04 | SQLite open, pragmas, embedded migration, backup and newer-schema refusal. Happy and unhappy tests pass. |
| `T-004` | DONE | 2026-11-04 | HTTP health handler, bearer token, loopback guard and token file. Happy and unhappy tests pass. |
| `T-005` | DONE | 2026-11-04 | CLI dispatch, `doctor`, `serve`, version and help. Exit codes and remote warning tested. |
| `T-006` | DONE | 2026-11-04 | Card model, validation and text renderer in `CLI-13` order. Happy and unhappy tests pass. |
| `T-007` | DONE | 2026-11-04 | Gesture POST -> taste service -> SQLite. Signal and cold period commit together; retry is idempotent. |
| `S-02` | DONE | 2026-11-04 | Closed: provider B is out of v1. Local server API rejected; managed endpoint is a future preset only. |
| `S-01` | DONE | 2026-11-04 | Confirmed: `LFM2.5-1.2B-Instruct QAD Q4_0` is the default. 2.6B and 3B fail latency; 350M fails quality. |
| `T-020` | DONE | 2026-11-04 | Taste profile aggregate, validation, SQLite repository and migration v3. Happy and unhappy tests pass. |
| `T-021` | DONE | 2026-11-04 | Signal scores, bounded ±0.05 weight update, clamp and migration v4. Happy and unhappy tests pass. |
| `T-022` | DONE | 2026-11-04 | Cold-period validation, active query, find by target and refresh. Expired rows stay. Happy and unhappy tests pass. |
| `T-023` | DONE | 2026-11-04 | Reason picker keys 1–4, Enter default, Esc other, scope preview and invalid-key error. |
| `T-024` | DONE | 2026-11-04 | Profile snapshot returns axes, weights, examples, note and rubric version. No interests. |
| `S-03` | DONE | 2026-11-04 | Measured 510 units for a 5-interest batch (5 search, 5 videos, 5 channels). 20 cards GO within 10k/day. Decision A: five searches per batch. `OQ-21` resolved. |
| `T-010` | DONE | 2026-11-04 | OAuth authorization-code flow with PKCE S256, `youtube.readonly`, loopback on `127.0.0.1`, state check, 2-minute timeout and `auth login`. Unit and CLI tests pass. Real Google consent timed out twice; owner action remains. |
| `T-011` | DONE | 2026-11-04 | Atomic `0600` token file at `<data>/token`, `auth status`, and single-flight refresh. Revoked grants return `ErrUnauthorized` and exit `2`. Happy and unhappy tests pass. |
| `T-012` | DONE | 2026-11-04 | Paginated subscription import, channel metadata enrichment, upsert by `channel_id`, idempotent re-run, migration v5 and count. Happy and unhappy tests pass. |
| `T-013` | DONE | 2026-11-04 | Interest validation and storage for 1–10 slugs. Seed sources list availability; unavailable sources are shown and never substituted. |
| `T-014` | DONE | 2026-11-04 | Two-step onboarding: OAuth or stored token, subscription import, seed source choice, interest proposal/entry and the skip banner. Uses a provisional `spike-s01` profile until rubric v1 (`OQ-22`). |
| `T-030` | DONE | 2026-11-04 | Deterministic query planner: one query per interest, 50 results, 30-day window and stable order. |
| `T-031` | DONE | 2026-11-04 | One YouTube adapter port for search, channels, videos, subscriptions and mostPopular. Happy and unhappy tests pass. |
| `T-032` | DONE | 2026-11-04 | Quota ledger with reserve/charge/release, single 80% warning, 100% refusal, stale release and migration v6. |
| `T-033` | DONE | 2026-11-04 | Dedupe, watched/saved/shown exclusions, source priority, 20% micro-blogger target and explicit shortfall. |
| `T-034` | DONE | 2026-11-04 | Text-only yt-dlp fetcher, atomic `0600` transcript staging, hash verification, delete and TTL sweep. |
| `T-035` | DONE | 2026-11-04 | OpenAI-compatible LLM adapter, profile prompt, JSON schema and strict local validation with explicit failures. |
| `T-036` | DONE | 2026-11-04 | Analysis repository with terminal rows per video and rubric, axis scores, migration v7 and explicit failure states. |
| `T-037` | DONE | 2026-11-04 | Batch pipeline: plan, discover, enrich, select, fetch, stage, analyze and persist in order. Resume skips stored analyses. |
| `T-040` | DONE | 2026-11-05 | Batch composition: analyzed-first order, dense positions, deterministic ties, frozen seen prefix and late-analysis marking. |
| `T-041` | DONE | 2026-11-05 | Session and lease: one open session, live lease conflict, stale reclaim, closed-session resume and cursor preservation. |
| `T-042` | DONE | 2026-11-05 | Gestures: left/right/save call `taste` before advancing, down/up stay local, duplicate gesture IDs are idempotent. |
| `T-043` | DONE | 2026-11-05 | Feed stream: SSE `batch.started`/`card.upsert`/`batch.done`, one JSON object with `--json` and text card fallback. |
| `T-044` | DONE | 2026-11-05 | Library: saved/downloaded states, explicit download, file by `video_id`, remove and list. Migration v9. |
| `T-045` | DONE | 2026-11-05 | Playback: `PLAYER_CMD` or OS default, async launch, file checks and library open. |
| `T-050` | DONE | 2026-11-05 | `doctor` reports data dir, DB/schema, API and OAuth tokens, quota, yt-dlp, LLM, terminal images and server state. Redacts secrets and runs offline. |
| `T-051` | DONE | 2026-11-05 | Read-only stats: swipes, reasons, cold periods, quota trend and top topics. |
| `T-052` | DONE | 2026-11-05 | Local JSONL traces with daily files, 0600 permissions, 7-day sweep and a content/secret denylist. |
| `T-053` | DONE | 2026-11-05 | Weekly rating: one row per ISO week, -1/0/+1/unrated, batch IDs and idempotent duplicates. Migration v10. |
| `T-054` | DONE | 2026-11-05 | Cached feed, degradation banner, batch snapshots and zero-refetch resume. Migration v11 and E2E tests. |
| `T-055` | DONE | 2026-11-05 | Final gate: `make check`, `make coverage-check` and `make test-e2e` pass. Integration uses fakes; live data remains open. |
| `T-063` | DONE | 2026-11-06 | Migration v13 adds `shown_videos`. Batch save marks cards and prunes rows older than 90 days. Exclusions read the full window. |
| `T-060` | DONE | 2026-11-06 | Refresh source model with five modes. `FEED_REFRESH_SOURCE`, `X-Ytfeed-Source`, `--source` and the interactive picker. |
| `T-061` | DONE | 2026-11-06 | YouTube `SearchPage` and `PlaylistItems`, uploads playlist metadata on channel reads, and the `playlist_items` quota call. |
| `T-062` | DONE | 2026-11-06 | Source modes run search, trending and graph in the pipeline. Graph reads uploads from subscribed and related channels. |
| `T-064` | DONE | 2026-11-06 | Per-channel cap, search/graph expansion, source rotation and the unseen shortfall banner. |
| `T-065` | DONE | 2026-11-07 | Pending batch saved and streamed first. Concurrent analysis streams analyzed card updates before `batch.done`. |

A task is ready only when its dependencies are `DONE` and its happy and unhappy test
artifacts from [`test-plan.md`](test-plan.md) exist and fail. A task is done only when the `AGENTS.md`
Definition of Done passes and the backlog is updated. The implementation backlog is complete. Post-E6: live consent, live feed composition and the stats/rating UI are closed; trace call sites, the real model run and `OQ-22` are closed. All post-E6 verification items are complete. The interactive feed and step-by-step refresh progress are available.

## Sprint 1

| ID | Task | Epic | Deps | Context | Size | Acceptance |
|----|------|------|------|---------|------|------------|
| `T-001` | Repo skeleton, `go.mod`, Makefile, lint config | E1 | — | foundation | S | `make check` runs. No product code yet. |
| `T-001a` | Test harness, fakes and golden fixtures | E1 | T-001 | foundation | M | `make test-contract` and `make test-e2e` run. Fakes are deterministic. |
| `T-002` | Config, logger, `doctor` stub | E1 | T-001a | foundation | S | Bad config fails fast. Logs are structured. |
| `T-003` | SQLite, schema v1, migrations, backup | E1 | T-001 | storage | M | Fresh start creates the DB. A pending migration takes a backup. |
| `T-004` | HTTP server, token, loopback guard | E1 | T-002, T-003 | transport | M | `GET /v1/health` works with a bearer token. A non-loopback caller is rejected. |
| `T-005` | CLI client and dispatch | E1 | T-004 | transport | M | `ytfeed doctor` reaches the server. Exit codes follow `CLI-6`. `--allow-remote` warns. |
| `T-006` | Card model and hardcoded card | E5 | T-005 | `feed`, transport | S | One card renders in the `CLI-13` order. |
| `T-007` | One gesture persisted end to end | E3/E5 | T-003, T-006 | `taste`, `feed` | S | A left-swipe stores a signal and a cold period before the card advances. |
| `S-01` | Model comparison (`Q1`) | E4 | — | `analysis` | M | A report names the default model and the schema failure rate. |
| `S-02` | opencode API probe (`Q2`) | E4 | — | `analysis` | S | A report accepts or rejects opencode as a preset. |
| `S-03` | Quota measurement (`R3`) | E4 | — | `discovery` | S | Measured units for one real batch. Go or no-go on 20 cards. |

## E1 — Foundation

| ID | Task | Deps | Context | Size | Acceptance |
|----|------|------|---------|------|------------|
| `T-001` | Repo skeleton | — | foundation | S | See Sprint 1. |
| `T-001a` | Test harness | T-001 | foundation | M | See Sprint 1. |
| `T-002` | Config and logger | T-001a | foundation | S | See Sprint 1. |
| `T-003` | SQLite and migrations | T-001 | storage | M | See Sprint 1. |
| `T-004` | HTTP server and token | T-002, T-003 | transport | M | See Sprint 1. |
| `T-005` | CLI client | T-004 | transport | M | See Sprint 1. |

## E2 — Identity and onboarding

| ID | Task | Deps | Context | Size | Acceptance |
|----|------|------|---------|------|------------|
| `T-010` | OAuth flow with PKCE and `youtube.readonly` | T-004 | `identity` | M | Consent returns a token. `state` is checked. Exit `2` on refusal. |
| `T-011` | Token store with `0600` and refresh | T-010 | `identity` | M | Token file is `0600`. Refresh is single-flight. A revoked grant exits `2`. |
| `T-012` | Subscription import | T-011 | `identity` | M | Paginated upsert by `channel_id`. Re-run does not duplicate. Count is printed. |
| `T-013` | Interests and seed sources | T-012 | `identity`, `taste` | M | 1–10 interests stored. An unavailable source is listed, never substituted. |
| `T-014` | Onboarding orchestration | T-011, T-012, T-013 | `identity` | M | Two steps. Step 2 can be skipped with the banner. |

## E3 — Taste and filtering

| ID | Task | Deps | Context | Size | Acceptance |
|----|------|------|---------|------|------------|
| `T-020` | Taste profile aggregate | T-003 | `taste` | M | ≥ 3 axes, weights, examples and note. `I-P1`…`I-P9` hold. |
| `T-021` | Signal and weight update | T-007, T-020 | `taste` | M | Signal is append-only. `gesture_id` is unique. Weight nudge is bounded. |
| `T-022` | Cold periods | T-021 | `taste` | M | Scope from reason. Repeat mute refreshes expiry. Expired rows stay. |
| `T-023` | Reason picker | T-006, T-022 | transport, `taste` | S | Keys 1–4. Esc records `other`. Scope is shown before confirm. |
| `T-024` | Profile snapshot query | T-020 | `taste` | S | Axes, weights, examples, note and `rubric_version` returned. No interests. |

## E4 — Discovery and analysis

| ID | Task | Deps | Context | Size | Acceptance |
|----|------|------|---------|------|------------|
| `T-030` | Query planner | T-013 | `discovery` | M | Five interests become deterministic queries. |
| `T-031` | YouTube adapter | T-010 | `discovery` | M | Search, channels, videos, subscriptions and mostPopular work through one port. |
| `T-032` | Quota ledger | T-003, T-031 | `discovery` | M | Reserve before call. Charge after. Warn at 80%. Refuse at 100%. |
| `T-033` | Dedupe and micro-blogger balance | T-030, T-032 | `discovery` | M | Exclusions apply. Target is 20%, not a gate. Shortfall is recorded. |
| `T-034` | Transcript fetch and staging | T-033 | `analysis`, `discovery` | L | Text only. Stage `0600`. Delete on commit. TTL sweep. |
| `T-035` | LLM adapter, prompt and schema | T-024, T-034, S-01, S-02 | `analysis` | L | One adapter. Schema validation. Explicit failures. Rubric version bound. |
| `T-036` | Analysis repository and failures | T-035 | `analysis` | M | Terminal row per video and rubric. No coercion. |
| `T-037` | Batch pipeline orchestration | T-032, T-033, T-036 | `feed` | L | Stages run in order. Crash resumes from checkpoint. |

## E5 — Feed and library

| ID | Task | Deps | Context | Size | Acceptance |
|----|------|------|---------|------|------------|
| `T-040` | Batch and card order | T-037 | `feed` | M | ≤ 20 cards. Dense positions. Analyzed first. Seen prefix frozen. |
| `T-041` | Session and lease | T-040 | `feed` | M | One open session. Stale lease reclaim. Resume keeps cursor. |
| `T-042` | Gestures | T-021, T-040, T-041 | `feed` | M | Left/right/save call `taste`. Down/up are local. Idempotent. |
| `T-043` | Stream and card render | T-040, T-006 | `feed`, transport | M | SSE events. `--json` buffers one array. Text fallback. |
| `T-044` | Library and download | T-042 | `library` | M | Save state. Explicit download. File by `video_id`. |
| `T-045` | Playback | T-044 | `library` | S | `PLAYER_CMD` or OS default. Never blocks the UI. |

## E6 — Quality and ops

| ID | Task | Deps | Context | Size | Acceptance |
|----|------|------|---------|------|------------|
| `T-050` | `doctor` | T-004, T-011, T-031, T-035 | transport | M | Reports data dir, DB, token, quota, yt-dlp, LLM and terminal. Redacts secrets. |
| `T-051` | `stats` | T-040, T-042 | `feed` | S | Swipes, cold periods, quota trend. Read-only. P1. |
| `T-052` | Local traces | T-037 | transport | S | JSONL spans. Daily rotation. 7-day retention. No content. |
| `T-053` | Weekly rating | T-041 | `feed` | S | Once per week. `unrated` recorded. Batch IDs stored. |
| `T-054` | Degradation and resume | T-037, T-040 | `feed`, `analysis` | M | Cached feed. Banner. Crash test with zero re-fetch. |
| `T-055` | End-to-end run and coverage | T-001…T-054 | all | M | Real data. `make check`. Coverage ≥ 80%. |

## E7 — Refresh overhaul

Plan: [`refresh-plan.md`](refresh-plan.md).

| ID | Task | Deps | Context | Size | Acceptance |
|----|------|------|---------|------|------------|
| `T-060` | Refresh source model | — | `discovery`, transport | M | Five modes. Config default `mixed`. HTTP header, CLI flag and interactive picker. |
| `T-061` | YouTube graph primitives | T-060 | `discovery`, integration | M | `SearchPage`, `playlistItems.list`, uploads metadata and `playlist_items` quota. |
| `T-062` | Trending and graph sources | T-061 | `discovery`, `feed` | M | Both sources merge and deduplicate with keyword results. |
| `T-063` | Shown history | — | `feed`, repository | M | `shown_videos`, 90-day pruning, full exclusion and mark on save. |
| `T-064` | 20-unseen selection | T-062, T-063 | `discovery`, `feed` | M | Per-channel cap and expansion. Shortfall banner. Never pad with seen cards. |
| `T-065` | Progressive analysis | T-064 | `feed`, transport | L | Pending cards stream first. Analysis updates follow at concurrency 2. |
| `T-066` | Interactive updates | T-065 | transport | M | Cards update in place. The source picker drives refresh. |
| `T-067` | Docs and end-to-end proof | T-060…T-066 | all | M | Quota docs, contract tests and E2E no-repeat test. |

## Open dependencies

- `OQ-01` decides the quota exit code before `T-032`.
- `OQ-07` decides the rubric location before `T-035`.
- `Q1` decides the default model before `T-035`.
- `Q3` decides rubric v1 content before `T-035`.
