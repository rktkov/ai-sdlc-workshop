# PRD — ytfeed: Personal YouTube Feed Composer

## Metadata

| Field | Value |
|-------|-------|
| **Title** | ytfeed — Personal YouTube Feed Composer |
| **Status** | Approved for implementation (v0.1; transcript-staging amendment 2026-11-02) |
| **Owner** | Oleg K (product owner, sole user in v1) |
| **Stakeholders** | Product owner; ~100 friends at 12 months (each running a local instance) |
| **Date** | 2026-11-02 |
| **Related rules** | [`docs/00-rules/`](../00-rules/README.md) — frozen, 27 decisions |
| **Implementation target** | 3 days, correctness over speed |

---

## 1. Problem and Context

YouTube's recommendation system optimizes for engagement, not for the viewer's
information diet. For a daily user with specific interests this produces three
distinct failures:

1. **Recommendation incest** — the same channels appear in search and home feed;
   last week's interest hijacks this week's feed.
2. **Title/thumbnail inflation** — the hook is strong, the substance is average,
   and this is unknowable until 10 minutes in.
3. **Filter-bubble push** — the user has patterns they like to follow, but the
   feed turns those patterns into a corridor instead of a starting point.

The second-order cost is the one that matters: **curiosity atrophy**. The user
marinates in the same trend topics and adjacent opinions, and is never pulled
sideways into unrelated territory.

**Who experiences it:** the product owner, daily.

**Cost of not solving it:** continued time loss on high-hook/low-substance
content; narrowing of topic range; FOMO driven by manufactured trends; and a
slow erosion of the exploratory behaviour that makes the platform valuable in
the first place.

**Evidence base:** direct personal experience (daily, three years of the same
pattern), plus a self-observed signal: the user can name the channels they
trust, and notices that YouTube's feed ignores the quiet, small, low-view
channels whose *topics* they value.

---

## 2. Users

### Primary user (v1)

**The product owner.** Technical, macOS laptop, Ghostty terminal, daily use,
low tolerance for onboarding friction, strong opinions about content quality,
already maintains a subscription graph that encodes taste better than any
survey.

### Secondary users (12 months)

**~100 friends**, each running **their own local instance**. Same shape as the
primary user: technically comfortable, willing to install a binary and connect
a Google account. Hard ceiling assumed at ~500 instances.

**Not a target:** a broad consumer audience expecting zero-config signup. That
would require a shared backend, hosted OAuth, and a web UI — all out of scope
(see §3).

### What secondary users need that the primary user does not

- A cold-start path that works with no subscription history.
- Onboarding that does not assume the owner's manual setup.
- A visible, honest statement of where data lives (their own machine).

---

## 3. Non-Goals

| Out of scope | Why |
|--------------|-----|
| **Web UI** | CLI first. The client/server split exists precisely so a UI can be added later without touching domain code. |
| **Shared/hosted backend** | Each user runs their own instance. Hosting implies accounts, billing, GDPR surface and uptime obligations — a different product. |
| **Live streams and Shorts** | Different content model, different value proposition. Live cannot be pre-analyzed; Shorts are the format the user is trying to escape. |
| **Direct paid LLM API calls** | Cost per analysis must stay near zero. Local models or a subscription-backed agent only. |
| **Curated third-party lists / newsletters** | The product's premise is the user's own recommendation logic, not someone else's taste. |
| **Monetization** | Undecided. v1 must not block it, but must not optimize for it. |
| **Multi-user concurrent sessions** | One UI connection at a time in v1. Per-user data exists in the schema from day one. |
| **Mobile app** | Later, if ever. |
| **Full WCAG compliance** | Minimal accessibility now (no color-only encoding), extensible later. |

---

## 4. Goals

### User goals (measurable)

| # | Goal | Measure |
|---|------|---------|
| UG-1 | Stop opening the YouTube home page | Self-reported home-page opens per week → 0 by week 4 |
| UG-2 | Reduce FOMO driven by manufactured trends | Weekly rating ≥ +1 in ≥ 60% of weeks by week 4 |
| UG-3 | Discover topic ranges outside the current focus | ≥ 20% of opened cards flagged "new territory" |
| UG-4 | Stop investing 40 minutes in a 4-minute idea | ≥ 60% of opened cards are finished or intentionally abandoned within 5 minutes |

### Business goals (measurable)

| # | Goal | Measure |
|---|------|---------|
| BG-1 | Validate whether others want it | ≥ 10 friends running it weekly by month 3 |
| BG-2 | Keep marginal cost near zero | ≥ 95% of analyses served by a local model |
| BG-3 | Produce a reusable, reviewable filter rubric | Rubric versioned, edited at least twice, with measurable score agreement |

> Goals are outcomes, not outputs. "Build a card stack" is not a goal; "stop
> opening the home page" is.

---

## 5. User Stories

### P0 — must-have for launch

| # | Story | Requirements |
|---|-------|--------------|
| US-1 | As the owner, I want to connect my Google account and import my subscriptions, so that the feed starts from taste I already have rather than from a survey. | F1, F2 |
| US-2 | As the owner, I want to choose ~5 current interests from a seed source I pick, so that the feed reflects today's focus and can be re-seeded later. | F3 |
| US-3 | As the owner, I want each card to show a TL;DR, the topic, and a why-this-card reason, so that I can judge in seconds instead of minutes. | F10, F15 |
| US-4 | As the owner, I want to left-swipe a card and pick a reason, so that bad content is muted immediately and my profile learns from it. | F5, F16 |
| US-5 | As the owner, I want the mute scope to follow the reason, so that one misaligned author does not erase an entire topic I care about. | F16, F19 |
| US-6 | As the owner, I want a right-swipe to open the video in YouTube, so that watching stays in the tool built for watching. | F17 |
| US-7 | As the owner, I want down/up navigation that never removes cards, so that "not now" is not a destructive decision. | F17, F18 |
| US-8 | As the owner, I want to bookmark and download a video, so that I can watch it later without the feed. | F17, F20 |
| US-9 | As the owner, I want the feed to keep working when quota is exhausted or the network is down, so that a bad day does not block my routine. | F7, F24 |
| US-10 | As the owner, I want to rate my information life weekly on a −1/0/+1 scale, so that I can tell whether the product is actually working. | F21 |
| US-11 | As the owner, I want a single binary I can run with one command, so that the tool costs nothing to start. | F22, F28 |
| US-12 | As a friend, I want to install and onboard in two steps, so that I can try it without asking the owner for help. | F1, F3 |
| US-13 | As the owner, I want `ytfeed doctor` to tell me what is broken, so that I do not debug by guesswork. | F27 |

### P1 — high-value fast follows

| # | Story |
|---|-------|
| US-14 | As the owner, I want external trend sources, so that onboarding can surface topics YouTube itself would never show me. |
| US-15 | As the owner, I want statistics on swipes, cold periods and topic diversity, so that I can see whether the filter is improving. |
| US-16 | As the owner, I want to edit the rubric and dry-run it against known videos, so that I can tune judgment without guessing. |

### Edge-case stories

| # | Story |
|---|-------|
| US-E1 | As the owner, when a video has no transcript, I want the card shown as unanalyzed with that exact reason, so that I know why it is thin. |
| US-E2 | As the owner, when the model returns invalid JSON, I want the card marked `failed: invalid_model_output`, so that garbage is never presented as judgment. |
| US-E3 | As the owner, when a batch is interrupted, I want it resumed on restart, so that I do not pay twice for the same analysis. |
| US-E4 | As the owner, when the database schema is newer than my binary, I want a clear refusal to start, so that I cannot corrupt my data by downgrading. |
| US-E5 | As the owner, when I launch from a different directory, I want the resolved data path printed, so that I do not silently create a second profile. |
| US-E6 | As the owner, when the feed is empty, I want an explanation and the next action, not a blank screen. |
| US-E7 | As the owner, when a second UI connects, I want it rejected with a clear message, so that two clients cannot corrupt session state. |

---

## 6. User Journeys

### 6.1 First run (onboarding)

```
ytfeed onboard
  → opens browser, Google OAuth (youtube.readonly)
  → imports subscriptions
  → shows 3 seed sources, user picks one
      [1] YouTube trending + subscription topics   (default)
      [2] External trends (Google Trends / Reddit / HN)
      [3] Subscriptions only
  → proposes ~5 interests, user edits/accepts
  → "ready. run `ytfeed feed`"
```

Skipping step 2 is allowed; the feed then composes from subscriptions with a
visible banner: `interests not set — feed may be narrow.`

### 6.2 Daily loop

```
ytfeed feed
  → session opens (or resumes, if < 1h since last activity)
  → feed composes: discovery → dedupe → transcript → analysis → scoring
  → cards stream in, analyzed first
  → user gestures:
      left   → reason picker → cold period (scope by reason)
      right  → open in YouTube, record positive signal
      down   → next card (nothing removed)
      up     → previous card not left-swiped
      tap    → yt-dlp download to library
      dbltap → remove from library
  → feed exhausted → next batch requested on demand
  → idle 1h → session closes
```

### 6.3 Rejection with reason

```
[left swipe]
  What was wrong?
  [1] marketing   [2] ai-slop   [3] value-misaligned   [4] other
  → mute channel "X" for 24h        (1, 3)
  → mute topic "ai-agents" for 24h  (2, 4)
  → signal stored; picker remembers the last choice
```

### 6.4 Weekly check-in

```
[first session after 7 days]
  How was your information life this week?
  [-1] worse   [0] same   [+1] better
  → stored with week number and the IDs of feeds shown that week
```

### 6.5 Degraded paths

| Condition | Behaviour |
|-----------|-----------|
| YouTube quota exhausted | Cached feed + library; banner with reset time; exit code 6 if a fresh batch was requested |
| YouTube unreachable | Same as above |
| No transcript | Card shown as `[unanalyzed: no transcript]` |
| LLM slow or failing | Card shown as `[unanalyzed: analysis slow]`; batch continues |
| Model returns invalid JSON | Card `failed: invalid_model_output`; never coerced |
| Crash mid-batch | Batch resumes on next start from the last persisted stage |
| Database schema newer than binary | Refuse to start; explain how to upgrade |

---

## 7. Functional Requirements

### P0 — Must have (the product does not work without these)

| # | Requirement | Acceptance criteria | Depends on |
|---|-------------|---------------------|------------|
| **F1** | Google OAuth with `youtube.readonly`; token stored locally with `0600` | Given no token, when `ytfeed onboard` runs, then a browser opens and a token file with `0600` exists after consent. Given a revoked token, when any API call runs, then the user is told to re-authenticate and exit code is 2. | — |
| **F2** | Import subscriptions after auth | Given a connected account, when onboarding completes, then every subscription is persisted with channel ID, title, subscriber count and topic tags; count is printed. | F1 |
| **F3** | Choose ~5 interests from a user-selected seed source | Given onboarding step 2, when the user picks a source, then exactly 5 interests are proposed and editable. Given `external-trends` with no adapter configured, then the source is listed as unavailable — never silently substituted. | F2 |
| **F4** | Per-user taste profile as named value axes with weights, examples and a free-text note | Given a new user, when the profile is created, then it contains ≥ 3 named axes, each with weight, ≥ 1 positive and ≥ 1 negative example, and the rubric version is recorded. | F3 |
| **F5** | Every gesture updates the taste profile | Given a left-swipe with reason `marketing`, when committed, then a channel-scoped cold period exists and a negative signal is stored. Given a right-swipe, then a positive signal is stored. | F4 |
| **F6** | Discovery via YouTube search + subscription-graph expansion, including micro-bloggers | Given 5 interests, when a batch is composed, then candidates come from both keyword search and channels related to subscriptions; at least 20% of candidates have < 100k subscribers (configurable). | F2 |
| **F7** | Quota accounting and rate limiting | Given a 10,000-unit daily budget, when usage exceeds 80%, then a warning is printed. Given 100%, then no further search calls are made and the cached feed is served. Usage is persisted per UTC day. | F6 |
| **F8** | Dedupe | Given a candidate already watched, saved, shown in the last 7 days, or duplicated by video ID, then it is excluded from the batch. | F6 |
| **F9** | Transcript-only ingestion | Given a candidate, when ingestion runs, then only the transcript is fetched (no video/audio download). The transcript is staged in a permission-restricted temporary directory inside the data directory between fetch and analysis commit, and deleted as soon as the analysis result is stored. It is never written to the database, logs, traces, or backups. | F8 |
| **F10** | Rubric-driven analysis producing structured attributes | Given a transcript, when analysis runs, then the result contains: topic, TL;DR, per-axis scores with one-line justification, marketing score, ai-slop score, similarity signal, "new territory" flag, and why-this-card reason. The response must validate against the JSON schema. | F4, F9 |
| **F11** | Analysis failure states are explicit | Given a missing transcript, an LLM timeout, or schema-invalid output, then the card carries `state` and `reason` (`no_transcript`, `analysis_slow`, `invalid_model_output`) and is still shown. | F10 |
| **F12** | Rubric is versioned and recorded per result | Given a rubric change, when a new analysis runs, then the new version is stored with the result; old results keep their original version. | F10 |
| **F13** | Feed composition, default 20 cards, configurable | Given `FEED_SIZE=20`, when a batch completes, then exactly 20 unique candidates are queued (or fewer with an explicit explanation). `--size` overrides per run. | F8 |
| **F14** | Incremental streaming, analyzed first | Given a batch in progress, when the UI opens, then cards appear as they finish; analyzed cards precede unanalyzed; a card never moves backwards past the cursor. | F13 |
| **F15** | Card rendering in the CLI | Given Ghostty or another supported terminal, then the thumbnail is rendered inline; otherwise the card renders text-only with identical attributes. `--no-images` forces text. | F14 |
| **F16** | Left-swipe with skippable reason picker | Given a left-swipe, then a picker appears with 1–4; last choice preselected; Enter accepts; Esc records `other`. The resulting scope is displayed before confirming. | F5 |
| **F17** | Remaining gestures | Given a card: right opens YouTube; down advances; up returns to the previous non-left-swiped card; tap downloads via yt-dlp to the library; double-tap removes from the library. All are idempotent. | F15 |
| **F18** | Session lifecycle | Given UI start, then a session opens or resumes; given 1h of inactivity, then it closes and navigation history is discarded. `session resumed` is printed on resume. | F17 |
| **F19** | Cold periods as records | Given a mute, then a row exists with scope (`topic`/`channel`), target, reason, `created_at`, `expires_at`; expired rows are excluded from composition and retained for audit. | F16 |
| **F20** | Library with local playback | Given a downloaded video, then it appears in the library and opens in the system player via `PLAYER_CMD`; files live under `./data/library/`. | F17 |
| **F21** | Weekly −1 / 0 / +1 rating | Given 7 days since the last rating, when a session opens, then the prompt appears once, is dismissible (records `unrated`), and stores week number + feed IDs. | F18 |
| **F22** | Local HTTP server, token-authenticated, loopback-only | Given `ytfeed serve`, then the API listens on `127.0.0.1:7420`; every request requires a bearer token stored with `0600`; a second concurrent UI connection is rejected; binding elsewhere requires `--allow-remote` plus a warning. | — |
| **F23** | SQLite with auto-migrations and backup | Given pending migrations, when the server starts, then the DB is copied to `ytfeed.db.bak-<UTC>`, migrations apply in a transaction, `schema_version` is verified, and the last 5 backups are kept. Given a newer schema, then startup is refused. | F22 |
| **F24** | Offline/quota degradation | Given no network or exhausted quota, when the UI opens, then cached cards and the library are available and the reason is displayed. | F7, F23 |
| **F25** | Batch resume | Given a crash mid-batch, when the app restarts, then the batch resumes from the last persisted stage and does not re-fetch completed transcripts. | F23 |
| **F26** | Pluggable model endpoint | Given `LLM_BASE_URL`, `LLM_MODEL`, `LLM_JSON_MODE`, then any OpenAI-compatible runtime works (Ollama, llama-server, Llama app, LM Studio, opencode) with no code change. | F10 |
| **F27** | `ytfeed doctor` | Given any state, then it prints: resolved data dir, DB path and schema version, token presence and expiry, quota used today, yt-dlp version, LLM reachability and model, terminal image support — with secrets redacted, working offline. | F22 |
| **F28** | CLI contract: `--json`, exit codes, stderr/stdout split | Given any command, then `--json` prints only JSON on stdout; diagnostics go to stderr; exit codes follow 0–6/130; no colour when not a TTY. | F22 |

### P1 — Should have (fast follow)

| # | Requirement | Acceptance criteria | Notes |
|---|-------------|---------------------|-------|
| F29 | External trend adapters (Google Trends / Reddit / HN) | Given a configured adapter, then onboarding source 2 becomes available and returns ≥ 10 candidate topics | Optional; each adapter is a port |
| F30 | Friends onboarding | Given a second machine, then install → onboard → feed works with no owner intervention | Requires packaged release |
| F31 | `ytfeed stats` depth | Swipe distribution by reason, cold-period effectiveness, topic diversity over time, quota trend | Read-only over persisted events |
| F32 | Rubric editing flow | Given `ytfeed rubric edit`, then the user edits a versioned YAML and a dry-run scores 5 known videos | Depends on F12 |
| F33 | Similarity model | "Similar to what you watched" computed from embeddings, not LLM judgment | Needs an embedding model |
| F34 | Bulk actions | Multi-select mute/save in the library | UX only |

### P2 — Future considerations (design to support, do not build)

| # | Requirement | Notes |
|---|-------------|-------|
| F35 | Web UI | Reuses the same local HTTP API |
| F36 | Shared backend | Would require accounts, hosting, billing — different product |
| F37 | Postgres storage provider | Repository interface already isolates it |
| F38 | Mobile client | Same API |
| F39 | Accessibility extensions | No colour-only semantics are baked in |
| F40 | Scheduled background composition | v1 is on-demand by design |

---

## 8. Non-Functional Requirements

| Category | Requirement | Acceptance criteria | Priority |
|----------|-------------|---------------------|----------|
| **Performance** | No hard time-to-first-card; streaming is the design | Given a cached feed, then the first card renders < 2 s after start. Given a cold batch on the target laptop, then the first analyzed card appears within 60 s (to be validated by the model spike). Full batch has no SLA but must show progress and be resumable. | P0 |
| **Performance** | Analysis throughput is bounded by an explicit worker limit | Given `ANALYSIS_CONCURRENCY=2`, then never more than 2 LLM requests are in flight. | P0 |
| **Security** | OAuth scope is minimal; tokens never logged | `youtube.readonly` only; `redact()` applied to all config and error dumps; no token in traces. | P0 |
| **Security** | Local API is not reachable off-host | Binds `127.0.0.1` by default; bearer token required; non-loopback bind requires an explicit flag and prints a warning. | P0 |
| **Privacy** | Transcripts are not persisted beyond the analysis window | Given an analysis, then only derived attributes are stored permanently. Transcript text is held in memory and, between fetch and analysis commit, in `./data/tmp/transcripts/` with `0600` permissions; it is deleted when the analysis result commits and swept after `TRANSCRIPT_STAGING_TTL` (default 24h). It never appears in the database, logs, traces, or backups. | P0 |
| **Scalability** | Per-instance only, no shared state | Given 100 independent instances, then no instance depends on another. The 5,000 analyses/day figure is a stated ceiling, not a validated target. | P0 |
| **Accessibility** | Minimal now, extensible later | No meaning conveyed by colour alone; text-only mode is complete; keyboard-only operation. | P1 |
| **Reliability** | User signals are durable | `synchronous=FULL`; a swipe is committed before the card advances; a power loss loses no committed signal. | P0 |
| **Reliability** | Crash recovery | Batch resumes; no partial rows; no half-migrated schema. | P0 |
| **Reliability** | No uptime SLA | Local app; if it does not start, the reason must be explicit. | — |
| **Cost** | Zero marginal cost target | ≥ 95% of analyses on a local model; no direct paid API by default. | P0 |
| **Maintainability** | Rules are enforced by gates | `make check` and `make coverage-check` pass; no rule in `docs/00-rules/` is violated. | P0 |

---

## 9. Success Metrics

### Leading indicators

| Metric | Target | Measurement | Window |
|--------|--------|-------------|--------|
| Cards opened (right-swipe) / cards shown | ≥ 30% | Local DB, per session | 14 days |
| Left-swipes with an explicit reason | ≥ 80% | Local DB | 14 days |
| Schema-valid analysis rate | ≥ 98% | `ytfeed stats` | 7 days |
| Median time-to-first-card | < 10 s (cached) | OTel span | 7 days |
| Quota usage | ≤ 80% of 10k/day | Quota ledger | 7 days |
| New-territory share of opened cards | ≥ 20% | Card flags | 30 days |

### Lagging indicators

| Metric | Target | Measurement | Window |
|--------|--------|-------------|--------|
| Weekly quality rating | ≥ +1 in ≥ 60% of weeks | Weekly prompt | 90 days |
| Micro-blogger share of opened cards | ≥ 30% from channels < 100k subs | Local DB | 90 days |
| Repeat left-swipe rate (same channel, same reason) | Declining month over month | Local DB | 90 days |
| Self-reported home-page opens | → 0 per week | Weekly prompt (free-text) | 90 days |

> All numeric targets are **initial calibration values**, not commitments. They
> are revised after the first 30 days of real data.

---

## 10. Scope and Phasing

### v1 (3 days) — P0 only

Day 1 — walking skeleton:
- repo, `go.mod`, Makefile, lint, CI-less local gates;
- config, logger, SQLite, migrations, backup;
- local HTTP server + token + CLI client;
- one hardcoded card rendered and one gesture persisted end-to-end;
- **spike:** model comparison (Q-model), opencode API probe (Q2), quota measurement.

Day 2 — the pipeline:
- OAuth, subscription import, interests;
- discovery, dedupe, transcript fetch;
- analyzer with JSON schema, failure states, rubric v1;
- feed composition, streaming, card rendering.

Day 3 — the loop:
- all gestures, cold periods, sessions, library;
- degradation, resume, `doctor`, `stats`, weekly rating;
- end-to-end run with real data, coverage gate, backlog closure.

### v1.1 (fast follow)

F29, F31, F32, F33, F34; friends packaging (F30).

### v2 (next quarter)

F35–F40, provider abstraction for Postgres, accessibility work.

---

## 11. Risks and Assumptions

| # | Risk | Likelihood | Impact | Mitigation |
|---|------|-----------|--------|------------|
| R1 | **opencode server API shape/stability is unknown** — never called programmatically | High | High | Spike on day 1; keep the OpenAI-compatible adapter as the fallback; provider B is optional for v1 |
| R2 | **Small local models fail the rubric** — prose instead of schema-valid JSON, or weak judgment | High | High | Schema-constrained output + validation; ladder of LFM2.5 sizes; rubric v1 kept deliberately simple; `invalid_model_output` is a visible state |
| R3 | **YouTube quota** — ~100 searches/day at 10k units | Medium | High | Quota ledger, 80% warning, cached-feed degradation, batch size configurable |
| R4 | **yt-dlp fragility** — YouTube changes break it | Medium | Medium | Transcript-only for analysis; yt-dlp used only for explicit downloads; version surfaced in `doctor` |
| R5 | **5,000 analyses/day is implausible on a laptop** | High | Medium | Accept per-instance limits; document the ceiling honestly; a hosted provider is a v2 decision |
| R6 | **Ollama stable cannot load `lfm2moe`** (`missing tensor 'output_norm.weight'`) | High | Medium | Use `llama-server` for MoE models, or Ollama ≥ v0.17.1-rc0; dense LFM2 unaffected |
| R7 | **3-day scope is optimistic** | High | High | Walking skeleton first; P1/P2 explicitly deferred; risk-first ordering in Sprint 1 |
| A1 | The user's subscription graph is a good enough taste seed | — | — | Validated during onboarding |
| A2 | Google OAuth consent for a personal project does not require verification | — | — | Personal use only; friends use their own Cloud project if needed |
| A3 | Ghostty inline images work via the iTerm2/Kitty protocol | — | — | Text fallback is complete, not degraded |

---

## 12. Open Questions

| # | Question | Owner | Blocking? |
|---|----------|-------|-----------|
| Q1 | Which LFM2.5 size/quant clears the bar on the target laptop (schema validity, rubric agreement, tok/s, RAM)? | Owner + agent | **Yes** — determines default `LLM_MODEL` |
| Q2 | What is the opencode server API surface, and is it stable enough to be provider B? | Agent | **Yes** for provider B; no for v1 |
| Q3 | What are the named value axes, weights and examples for rubric v1? | Owner + agent draft | **Yes** for F10 quality |
| Q4 | What subscriber threshold defines a "micro-blogger"? | Owner | No — default < 100k, configurable |
| Q5 | What daily quota budget should the composer target? | Owner | No — default 80% of 10k |
| Q6 | Are external trend adapters worth building, or is YouTube trending enough? | Owner | No — P1 |
| Q7 | Is 1 hour the right session idle timeout in practice? | Owner | No — configurable |
| Q8 | Should transcripts ever be persisted for re-analysis without re-fetch? | Owner | Transient staging only, for crash-safe resume; deleted when the analysis commits. |

---

## 13. Release Acceptance Criteria (v1)

- [ ] `ytfeed onboard` completes on a clean machine and produces a usable feed.
- [ ] `ytfeed feed` shows 20 cards, streams them, and every gesture persists.
- [ ] Left-swipe reason changes the mute scope exactly as specified.
- [ ] A killed process resumes its batch on restart without duplicate work.
- [ ] Quota exhaustion degrades to cache with a clear message and exit code 6.
- [ ] `ytfeed doctor` reports every integration accurately with secrets redacted.
- [ ] `make check` and `make coverage-check` pass.
- [ ] No rule in `docs/00-rules/` is violated; any deviation has an ADR.
- [ ] Backlog is closed and `tasks.md` statuses are `DONE`.

---

## 14. Related Documents

| Document | Purpose |
|----------|---------|
| [`docs/00-rules/`](../00-rules/README.md) | Engineering contract (frozen) |
| [`docs/00-rules/DECISIONS.md`](../00-rules/DECISIONS.md) | 27 recorded decisions |
| `docs/20-system-design/adr/` | Architecture decisions (to be written) |
| `docs/30-implementation/backlog.md` | Source of truth for progress |
| `templates/01-prd-en.md` | PRD template this document follows |

---

## 15. Backlog Planning Guidance

### 15.1 Epics

| Epic | Name | Covers | Rationale |
|------|------|--------|-----------|
| **E1** | Foundation | Repo, config, logging, SQLite, migrations, server, CLI skeleton | Nothing else can be tested without it |
| **E2** | Identity & Onboarding | OAuth, subscription import, interests, seed sources | The taste seed |
| **E3** | Taste & Filtering | Profile, rubric, cold periods, reason picker, signals | The differentiator |
| **E4** | Discovery & Analysis | YouTube search, dedupe, transcripts, LLM adapter, batch pipeline | The engine |
| **E5** | Feed & Library | Composition, streaming, card UI, gestures, downloads, playback | The daily loop |
| **E6** | Quality & Ops | Doctor, stats, traces, rating, degradation, resume | Trust and diagnosability |

### 15.2 Story sizing

| Size | Meaning |
|------|---------|
| **S** | < 1 day, single package, UI-only or pure function |
| **M** | 1–3 days, full stack, one context |
| **L** | 3–7 days, multi-system or external integration |
| **XL** | > 1 week, architectural change or spike first |

### 15.3 Sprint 1 (Day 1) — walking skeleton

**Goal:** one card, one gesture, end to end, through the real layers.

| Task | Size | Why now |
|------|------|---------|
| Repo skeleton, `go.mod`, Makefile, golangci config | S | Gate for everything |
| Config + logger + `doctor` stub | S | Fail-fast foundation |
| SQLite open, migrations, backup, schema v1 | M | Durability rules |
| Local HTTP server + token + CLI client | M | Proves the client/server split |
| Card model + one hardcoded card rendered | S | Visual target |
| One gesture persisted end to end | S | Proves the write path |
| **Spike: model comparison (Q1)** | M | Highest technical risk |
| **Spike: opencode API probe (Q2)** | S | Highest integration risk |
| **Spike: quota measurement (Q3/R3)** | S | Determines batch feasibility |

### 15.4 Risk-first ordering

Tackle these **before** committing to estimates:

1. **Q1 — model quality and schema compliance.** If no local model can return
   valid JSON with usable judgments, the product's core promise fails. Nothing
   downstream matters until this is answered.
2. **Q2 — opencode API.** If provider B is not viable, the cost model changes
   from "free" to "local only", which affects the 12-month plan.
3. **R3 — quota.** If a 20-card batch costs more than ~100 units, the daily
   budget supports fewer than 3 batches and the feed size must shrink.

### 15.5 Definition of Ready for any v1 task

- [ ] Acceptance criteria are testable.
- [ ] The owning bounded context is named.
- [ ] Dependencies are satisfied.
- [ ] The rule file it must satisfy is identified.
