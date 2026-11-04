# Implementation backlog

> **Status:** ACTIVE from 2026-11-04. The owner approved the proposal by starting `T-001`.
> **Date:** 2026-11-04
> **Owner:** Oleg K
> **Source:** the confirmed system-design suite in [`docs/20-system-design/`](../20-system-design/README.md).

## 1. Purpose

This file is the active implementation sequence for `ytfeed`. It turns the confirmed
design into epics, a walking skeleton and three risk-first spikes. It does not add scope.

After approval:

1. This file becomes the active implementation backlog (`DOC-11`).
2. The architecture backlog is folded into the history and deleted (`DOC-19`).
3. `tasks.md` becomes the task register.
4. Code work starts at `T-001`.

## 2. Test-first rule

Every task starts with the test artifacts that prove its acceptance criteria. The artifacts
must fail before the code exists. The task report must paste the failure. The full plan is
[`test-plan.md`](test-plan.md).

- **Unit tests** cover functional behavior.
- **Contract tests** fix the CLI/server payloads, SSE events and error mapping.
- **E2E tests** cover the batch data flow, the session control flow and regression.

Every behavior needs a **happy path** and an **unhappy path**. The unhappy path covers
invalid input, a denied dependency, a missing file, a timeout, a conflict or a malformed
payload. A task without both is not ready. A task with a skipped test is not done.

## 3. Inputs

| Input | Use |
|-------|-----|
| [`prd-ytfeed.md`](../10-product/prd-ytfeed.md) | Requirements `F1`…`F28`, epics E1–E6, Day 1–3 plan |
| [`20-system-design/README.md`](../20-system-design/README.md) | Design index and gaps |
| [`traceability.md`](../20-system-design/traceability.md) | Requirement to component and context |
| [`adr/INDEX.md`](../20-system-design/adr/INDEX.md) | Six accepted ADRs |
| [`open-questions.md`](../20-system-design/open-questions.md) | Provisional decisions |
| [`00-rules/`](../00-rules/README.md) | Engineering contract |

## 4. Epics

| Epic | Name | Scope | Exit criteria |
|------|------|-------|---------------|
| **E1** | Foundation | Repo, test harness, config, logging, SQLite, migrations, server, CLI, `doctor` stub | `make check` passes. Unit, contract and E2E targets run. One command reaches the server. |
| **E2** | Identity and onboarding | OAuth, token store, subscription import, interests, seed sources | A real account can onboard. Subscriptions and interests are stored. |
| **E3** | Taste and filtering | Profile, signals, weights, cold periods, reason picker | A left-swipe stores a signal and a cold period. A right-swipe stores a signal. |
| **E4** | Discovery and analysis | Search, graph expansion, quota, dedupe, transcripts, LLM, batch pipeline | A real batch produces valid analyses or explicit failures. |
| **E5** | Feed and library | Batch, order, session, stream, cards, gestures, downloads, playback | One card, one gesture, one download work end to end. |
| **E6** | Quality and ops | `doctor`, `stats`, traces, rating, degradation, resume | A crash resumes without a re-fetch. `doctor` explains every dependency. |

## 5. Sprint 1 — walking skeleton

**Goal:** one hardcoded card and one persisted gesture through the real layers.

| ID | Task | Size | Depends | Context |
|----|------|------|---------|---------|
| `T-001` | Repo skeleton, `go.mod`, Makefile, `golangci` config | S | — | foundation |
| `T-001a` | Test harness: unit, contract and E2E targets, fakes and golden fixtures | M | T-001 | foundation |
| `T-002` | Config, logger and `doctor` stub | S | T-001a | foundation |
| `T-003` | SQLite open, schema v1, migrations and backup | M | T-001 | storage |
| `T-004` | Local HTTP server, token and loopback guard | M | T-002, T-003 | transport |
| `T-005` | CLI client and command dispatch | M | T-004 | transport |
| `T-006` | Card model and one hardcoded card render | S | T-005 | `feed`, transport |
| `T-007` | One gesture persisted end to end | S | T-003, T-006 | `taste`, `feed` |
| `S-01` | Spike: model comparison (Q1) | M | — | `analysis` |
| `S-02` | Spike: opencode API probe (Q2) | S | — | `analysis` |
| `S-03` | Spike: quota measurement (R3) | S | — | `discovery` |

Sprint 1 is complete when `T-001`…`T-007` pass and the three spikes have a written result.

## 6. Spikes

| ID | Question | Exit criteria | Blocks |
|----|----------|---------------|--------|
| `S-01` | Which LFM2.5 size or quant clears schema validity, rubric agreement, tok/s and RAM on the target laptop? (`Q1`) | A short report names the default `LLM_MODEL` and the failure rate. | `T-035`, `ADR-0004` default |
| `S-02` | What is the opencode server API surface, and is it stable enough to be a preset? (`Q2`) | A written probe result. Preset or reject. | `T-035`, `OQ-06` |
| `S-03` | Does a 20-card batch cost at most ~100 quota units? (`R3`) | Measured units for one real batch. A go or no-go for the default size. | `T-030`, `T-032`, `F13` |

`S-03` result: one 5-interest batch costs 510 units (5 `search.list`, 5 `videos.list`,
5 `channels.list`). The 80% budget supports 15 batches per day. A 20-card batch is GO.
The owner chose five searches per batch. `OQ-21` is resolved. The full report is
[`spikes/s-03-quota-measurement.md`](spikes/s-03-quota-measurement.md).

The spikes run first because they can change the estimate and the default model. They do
not block the foundation work.

## 7. Day 2 — pipeline

| Epic | Tasks |
|------|-------|
| E2 | OAuth, token store, subscription import, interests, seed source |
| E4 | Query planner, YouTube adapter, quota ledger, dedupe, transcript fetch and stage |
| E3 | Profile, rubric v1 load, signal recorder, weight update |
| E4 | LLM adapter, prompt builder, JSON schema, failure states |
| E5 | Batch composition, order, streaming, card render |

**E2 status:** `T-010`…`T-014` are `DONE` (2026-11-04). The live Google consent is the
only manual check left. The first profile is provisional (`OQ-22`).

**E4 status:** `T-030`…`T-037` are `DONE` (2026-11-04). Discovery, quota, transcript
staging, analysis and batch orchestration pass their unit tests. Live YouTube, yt-dlp and
model calls still need the owner's consent and local binaries.

## 8. Day 3 — loop and quality

| Epic | Tasks |
|------|-------|
| E5 | All gestures, session lease, library, download, playback |
| E3 | Cold periods, reason picker, profile snapshot |
| E6 | Degradation, resume, `doctor`, `stats`, traces, weekly rating |
| E6 | End-to-end run with real data and the coverage gate |

## 9. Open decisions before code

| Decision | Default | Effect |
|----------|---------|--------|
| `OQ-01` quota exit code | `5` for a refused batch, `6` for a cached feed | `CLI-6`, `ADR-0006` |
| `OQ-07` rubric location | `docs/10-product/rubric/` plus a data-dir copy | `T-034`, `T-035` |
| `Q1` default model | Chosen by `S-01` | `LLM_MODEL` |
| `Q3` rubric v1 content | Owner and agent draft | `T-035` quality |

These are not blockers for Sprint 1. They block the analysis and quota tasks.

## 10. Approval

The owner approved this backlog on 2026-11-04 by starting `T-001`. The remaining fold step
needs explicit confirmation:

1. Copy the architecture backlog notes into the history section.
2. Delete `docs/20-system-design/backlog.md` after the fold (`DOC-19`).
3. Keep working from `tasks.md`.

The proposal was approved on 2026-11-04 when the owner started `T-001`. The architecture
backlog is kept as history until the owner confirms the fold and deletion (`DOC-19`).

## Progress

| Date | Task | Result |
|------|------|--------|
| 2026-11-04 | `T-001` | DONE. `go.mod`, Makefile, lint config, editor and git files. `make check` and `make coverage-check` pass. |
| 2026-11-04 | `T-001a` | DONE. Contract harness, E2E harness, fakes and golden fixture. `make test-contract`, `make test-e2e` and `make check` pass. |
| 2026-11-04 | `T-002` | DONE. Config load and validation, structured logger, redacted config. Happy and unhappy tests pass. |
| 2026-11-04 | `T-003` | DONE. SQLite, schema v1, embedded migrations, backup and newer-schema refusal. Happy and unhappy tests pass. |
| 2026-11-04 | `T-004` | DONE. HTTP health handler, bearer token, loopback guard and `0600` token file. Happy and unhappy tests pass. |
| 2026-11-04 | `T-005` | DONE. CLI dispatch, `doctor`, `serve`, `--allow-remote` warning, version and help. Happy and unhappy tests pass. |
| 2026-11-04 | `T-006` | DONE. Card model, validation and text renderer in `CLI-13` order. Happy and unhappy tests pass. |
| 2026-11-04 | `T-007` | DONE. Gesture endpoint, taste service and SQLite repository. Signal and cold period commit together; retry is idempotent. |
| 2026-11-04 | `S-02` | DONE. Provider B is out of v1. Local server API rejected; managed endpoint is a future preset only. |
| 2026-11-04 | `S-01` | DONE. Confirmed `LFM2.5-1.2B-Instruct QAD Q4_0` as the v1 default. 2.6B and 3B fail latency; 350M fails quality. |
| 2026-11-04 | `T-020` | DONE. Taste profile aggregate, validation, SQLite repository and migration v3. Happy and unhappy tests pass. |
| 2026-11-04 | `T-021` | DONE. Signal scores, ±0.05 weight update, clamp and migration v4. Happy and unhappy tests pass. |
| 2026-11-04 | `T-022` | DONE. Cold-period validation, active query, find by target and refresh. Expired rows stay. |
| 2026-11-04 | `T-023` | DONE. Reason picker keys 1–4, Enter default, Esc other and scope preview. |
| 2026-11-04 | `T-024` | DONE. Profile snapshot returns axes, weights, examples, note and rubric version. No interests. |
| 2026-11-04 | `S-03` | DONE. Measured 510 units for a 5-interest batch. 20 cards GO within 10k/day. Decision A: five searches per batch. `OQ-21` resolved. |
| 2026-11-04 | `T-010` | DONE. OAuth authorization-code flow with PKCE S256, `youtube.readonly`, loopback on `127.0.0.1`, state check, 2-minute timeout and `auth login`. Unit and CLI tests pass. Real Google consent timed out twice; owner action remains. |
| 2026-11-04 | `T-011` | DONE. Atomic `0600` token file, `auth status`, single-flight refresh and revoked-grant handling. Happy and unhappy tests pass. |
| 2026-11-04 | `T-012` | DONE. Paginated subscription import, channel metadata, upsert by `channel_id`, idempotent re-run, migration v5 and count. Happy and unhappy tests pass. |
| 2026-11-04 | `T-013` | DONE. Interest validation and storage for 1–10 slugs; seed source availability is explicit and unavailable sources are never substituted. |
| 2026-11-04 | `T-014` | DONE. Two-step onboarding with token reuse, subscription import, seed source choice, interest proposal/entry and skip banner. Provisional `spike-s01` profile (`OQ-22`). |
| 2026-11-04 | `T-030` | DONE. Deterministic query planner; 5 interests become 5 ordered queries. |
| 2026-11-04 | `T-031` | DONE. One YouTube adapter port for search, channels, videos, subscriptions and mostPopular. |
| 2026-11-04 | `T-032` | DONE. Quota ledger with reserve/charge/release, 80% warning, 100% refusal and migration v6. |
| 2026-11-04 | `T-033` | DONE. Dedupe, exclusions, source priority and 20% micro-blogger target with shortfall. |
| 2026-11-04 | `T-034` | DONE. Text-only transcript fetch and atomic `0600` staging with hash, delete and TTL sweep. |
| 2026-11-04 | `T-035` | DONE. One LLM adapter, profile prompt, JSON schema and strict validation. |
| 2026-11-04 | `T-036` | DONE. Analysis repository with terminal rows and failure states; migration v7. |
| 2026-11-04 | `T-037` | DONE. Batch pipeline in stage order with resume from stored analyses and explicit failures. |
