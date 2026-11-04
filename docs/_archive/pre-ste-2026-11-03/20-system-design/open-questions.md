# Open questions — architecture documentation

> **Status:** living document · **Last updated:** 2026-11-03
> **Policy (`SD-4`):** contradictions are recorded here, not resolved by the
> documentation. No entry proposes an edit to `docs/00-rules/`; a rule change needs a
> rules PR (`H-9`, `README.md` §4).
>
> **How to use this file:** architecture documents cite `OQ-NN` instead of asserting an
> answer. Each entry names the **provisional default** that diagrams and contracts
> assume, so the docs stay internally consistent. A provisional default is not a
> decision. When the owner answers an entry, the row gets a `Resolved` line and the
> default is promoted or removed.

## Register

| ID | Question | Conflicting sources | Provisional default used in docs | Blocks | Owner |
|----|----------|---------------------|----------------------------------|--------|-------|
| OQ-01 | **Quota exit code.** Is an exhausted quota that still serves a cached feed `5` (quota exhausted) or `6` (degraded)? Is it different when a fresh batch was explicitly requested? | PRD §6.5 says `6`; `CLI-6` and `E-10` map `ErrQuotaExhausted` → `5` | `5` when the command's primary purpose is a fresh batch; `6` when a cached feed is served and the command completes | ADR-0006, `ARCH-108` | Owner |
| OQ-02 | **Quota accounting day.** `F7` persists usage per UTC day, but YouTube quota resets at midnight Pacific Time. Which boundary does the ledger use? | PRD `F7`; YouTube platform behaviour | UTC day, as written; the reset time reported to the user is taken from the API error when available | ADR-0006, `ARCH-303` | Owner |
| OQ-03 | **`tap` and `dbltap` gestures.** `F17` and §6.2 define tap = download, double-tap = remove-from-library, but no key binding or mouse-reporting mode is specified for a terminal UI. | PRD §6.2, `F17`, `US-8`; `CLI-17`, `CLI-20` define keys only for the reason picker | Two explicit keys (`s` save/download, `x` remove) shown in the footer; no mouse reporting in v1 | `ARCH-103`, `ARCH-302` | Owner |
| OQ-04 | **Bookmark vs download.** `US-8` asks for "bookmark and download"; `F17` makes tap a download; `F20` defines the library as downloaded files. Is there a save-without-download state? | PRD `US-8`, `F17`, `F20`; `CLI-19` ("removing from the watch list is not destructive") | One library entity with two states: `saved` (no file) and `downloaded` (file present) | `ARCH-306` | Owner |
| OQ-06 | **opencode as a provider.** `A-12a` says every runtime goes through one OpenAI-compatible adapter, but `A-9` lists `opencode-agent` and `Q2` is unresolved. Is opencode a preset or a second adapter? | `A-12a`, `A-9`, PRD `F26`, `Q2` | Preset only, inside `integration/llm`; provider B is out of v1 until the `Q2` spike answers | ADR-0004, `ARCH-304` | Owner |
| OQ-07 | **Rubric v1 content and location.** `A-19` places the rubric doc in `docs/10-product/` plus a machine-readable copy, but no rubric exists and `Q3` is blocking. Where does the machine-readable copy live? | `A-19`, PRD `Q3`, `F10`, `F12` | `docs/10-product/rubric/` for the human doc; the machine-readable file is loaded from the data directory so it can be versioned without a deploy. Content is an input dependency, not a design output | ADR-0004, `ARCH-304` | Owner |
| OQ-08 | **Interest count.** `F3` says "exactly 5 interests, editable"; `US-2` says "~5". Is 5 a hard validation rule or a default? | PRD `F3`, `US-2` | Default 5, minimum 1, maximum 10; validated at onboarding | `ARCH-104` | Owner |
| OQ-09 | **Micro-blogger floor.** `F6` requires ≥ 20% of candidates from channels < 100k subscribers; `F13` allows fewer cards "with an explicit explanation". What happens when the floor cannot be met? | PRD `F6`, `F13`, `Q4` | The floor is a target, not a hard gate. Falling short prints an explicit banner and records the shortfall; the batch is never padded with non-matching channels | `ARCH-303` | Owner |
| OQ-10 | **Implementation backlog.** `AGENTS.md` §1 makes `docs/30-implementation/backlog.md` the only source of truth, but it does not exist and epics E1–E6 are unplanned. | `AGENTS.md` §1, `DOC-11`, PRD §15.1 | Out of scope here. Tracked as `ARCH-601`; this documentation backlog is explicitly *not* that file | `ARCH-601` | Owner |
| OQ-11 | **Off-host LLM egress.** `SD-5` allows any `LLM_BASE_URL`, so transcripts may leave the machine. The PRD's premise is local models and near-zero marginal cost. Is a non-loopback endpoint a supported configuration, and does it need consent? | `SD-5`, PRD §3 non-goals, Cost NFR, Privacy NFR | Supported. Default is loopback. `doctor` prints the resolved endpoint host. No consent prompt in v1; the privacy statement says "not persisted by ytfeed", not "never leaves the host" | ADR-0004, `ARCH-101` | Owner |
| OQ-12 | **yt-dlp absent during onboarding.** `SD-6` makes yt-dlp required for `feed`, optional for `library`. Does `onboard` fail, warn, or succeed with a degraded feed? | `SD-6`, `C-11`, PRD `F27` | `onboard` succeeds with a warning; `feed` fails with exit `6` and an explicit message naming the missing binary; `doctor` reports the version or its absence | `ARCH-103`, `ARCH-105` | Owner |
| OQ-13 | **Stale session lease.** `A-11` rejects a second UI connection, but no lease expiry or heartbeat is specified. After `kill -9`, can the user be locked out of their own instance? | `A-11`, PRD `US-E7` | Lease row with a heartbeat; a lease older than `SESSION_IDLE_TIMEOUT` is reclaimable without manual cleanup | `ARCH-103` | Owner |
| OQ-14 | **Navigation history after resume.** `F18` discards navigation history when a session closes, `CLI-21` prints `session resumed`; does "up" work across a resume boundary? | PRD `F18`, `CLI-21` | History is in-memory only; after a resume, "up" moves within the current run and does not cross the resume boundary. No card is left-swiped by a stale cursor | `ARCH-302` | Owner |
| OQ-15 | **Degraded card ordering.** `CLI-14a` requires analyzed cards first and "never backwards past the cursor". A card that finishes analysis after the cursor has passed it cannot move forward without reordering under the user. Confirm the resolution. | `CLI-14a`, `F14` | A card that completes late is inserted *ahead of the cursor* only if the user has not yet passed its slot; otherwise it stays in place and is marked `analyzed (late)`. Ordering is stable within a view | `ARCH-302` | Owner |
| OQ-16 | **Server lifecycle.** `A-9` says "single binary, single process"; `A-10` says the CLI auto-starts the server if absent. At runtime that is two processes. Who owns the server, when does it exit, what happens on port collision or after `kill -9` of the CLI? | `A-9`, `A-10`, `R-10` | One binary, two process roles. The CLI spawns a detached server if `GET /v1/health` fails, waits for readiness, and never kills a server it did not start. The server exits after `SESSION_IDLE_TIMEOUT` with no lease if it was auto-started; an explicitly started `ytfeed serve` stays up until signalled | ADR-0001, `ARCH-103` | Owner |
| OQ-17 | **`--json` vs streaming.** `F28`/`CLI-8` require `--json` to print exactly one JSON object or array on stdout; `F14` requires cards to stream as they finish. A single array cannot be emitted incrementally without buffering or NDJSON (which violates the contract). | `F28`, `CLI-8`, `F14`, `CLI-14a` | Interactive mode streams cards (SSE + terminal rendering). `--json` buffers the batch and prints one JSON array when composition completes; it performs no rendering and never mixes stream events into stdout | `ARCH-103`, ADR-0006 | Owner |
| OQ-18 | **How a gesture changes profile weights.** `F5` requires every gesture to update the taste profile, but the update rule (magnitude, decay, clamping, per-axis attribution) is unspecified. | `F5`, `F4`, `A-20` | Store the raw signal; apply a bounded nudge (±0.05, clamped to [0,1]) to the axes the card scored, and never let one gesture flip an axis. The rule is versioned with the rubric so it can be tuned | `ARCH-304` | Owner |

## Resolved

| ID | Resolution | Decided | Record |
|----|------------|---------|--------|
| OQ-05 | Transcripts are staged in `./data/tmp/transcripts/` between fetch and analysis commit, deleted on commit, TTL-swept, and never written to the database, logs, traces, or backups. The PRD `F9` and Privacy-NFR wording must be amended to match — proposed text is in the component document, not yet applied | 2026-11-02 | [`c4/03-components/03a-transcript-staging.md`](c4/03-components/03a-transcript-staging.md) §16 |
| OQ-19 | Four concepts: `axis` (evaluative dimension, `taste`), `interest` (user-selected content seed, `taste`), `topic` (content label from `analysis`, cold-period target), `topic tag` (YouTube channel metadata, `identity`). Interests and topics share one normalised slug namespace; topic tags do not; axes are independent | 2026-11-03 | [`ddd/strategic/glossary.md`](ddd/strategic/glossary.md) §3.13; [`ddd/strategic/context-map.md`](ddd/strategic/context-map.md) §10 (`G4-7`) |
| OQ-20 | `taste` owns **profile axis weights** only. The rubric is a content artifact (`A-19`) applied by `analysis`; its version is bound per result (`F12`). "Rubric weights" in `A-7` is shorthand for axis weights, not rubric ownership | 2026-11-03 | [`ddd/strategic/glossary.md`](ddd/strategic/glossary.md) §3.6; [`ddd/strategic/context-map.md`](ddd/strategic/context-map.md) §10 (`G4-8`) |

## Change log

- 2026-11-02 — Register created (`ARCH-004`) from the C1–C10 analysis, extended with
  `OQ-11`…`OQ-15` surfaced by `SD-5`…`SD-7` and by the rule cross-check.
- 2026-11-02 — `OQ-16` (server lifecycle) and `OQ-17` (`--json` vs streaming) added during
  Level 2.
- 2026-11-02 — `OQ-05` resolved by owner decision: bounded transcript staging. Upstream
  PRD amendment proposed, awaiting approval.
- 2026-11-03 — `OQ-19` and `OQ-20` added during DDD strategic design (`ARCH-204`):
  interests/topics/axes/topic tags, and the meaning of "rubric weights" in `A-7`. Both are
  cited by the glossary and block `ARCH-301`, `ARCH-303` and `ARCH-304` until confirmed.
- 2026-11-03 — `OQ-19` and `OQ-20` resolved by the owner at gate `G4`: four vocabulary
  concepts, and `taste` owns axis weights while `analysis` owns the rubric artifact. Rows
  moved to Resolved; `ARCH-301`, `ARCH-303` and `ARCH-304` are unblocked.
