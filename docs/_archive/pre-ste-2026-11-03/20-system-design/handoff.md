# Handoff — architecture documentation session

> **Snapshot:** 2026-11-03 · **Live status:** [`backlog.md`](backlog.md) is authoritative.
> This file is written to be handed to a fresh agent session with no conversation history.

---

## Kickoff prompt (paste this)

```
Read execution/docs/20-system-design/handoff.md in full and follow it as your
operating brief. You are continuing the architecture documentation for ytfeed.
Work from execution/docs/20-system-design/backlog.md. Start with task ARCH-201.
Do not write code. Stop at gate G4 and report.
```

---

# Operating brief

## 1. What this work is

`ytfeed` is a local-first, single-binary CLI that composes a personal YouTube feed:
discovery → dedupe → transcript → LLM analysis → scored card stack → gestures that train
a taste profile. The PRD is approved; the engineering rules are frozen; **no code exists
yet**.

Your job is to finish the system-design documentation suite under
`execution/docs/20-system-design/`. This is a documentation effort, not an implementation
task.

## 2. Read in this order (do not skip)

| # | Path | Why |
|---|------|-----|
| 1 | `execution/AGENTS.md` | Hard rules H-1…H-10, DoD, report format |
| 2 | `execution/docs/00-rules/README.md` | Rule index and precedence |
| 3 | `execution/docs/00-rules/02-architecture.md` | A-1…A-20: layers, six contexts, package layout |
| 4 | `execution/docs/00-rules/04-data-persistence.md` | D-1…D-24: SQLite, migrations, durability |
| 5 | `execution/docs/00-rules/08-cli-and-ux.md` | CLI-1…CLI-28: exit codes, card, gestures |
| 6 | `execution/docs/00-rules/10-docs-and-decisions.md` | DOC-1…DOC-19: docs tree, ADR format |
| 7 | `execution/docs/10-product/prd-ytfeed.md` | The product, F1…F28, NFRs, Q1…Q8 |
| 8 | `execution/docs/20-system-design/backlog.md` | **Live task list and status** |
| 9 | `execution/docs/20-system-design/open-questions.md` | OQ-01…OQ-18 and their provisional defaults |
| 10 | `execution/docs/20-system-design/c4/01-system-context.md` | Level 1, confirmed |
| 11 | `execution/docs/20-system-design/c4/02-container-diagram.md` | Level 2, confirmed + consistency check |
| 12 | `execution/docs/20-system-design/c4/03-components/*.md` | Level 3, confirmed |
| 13 | `execution/docs/20-system-design/flows/*.md` | Data flow and control flow |
| 14 | `preparation/docs/prompts/02-architecture/02-ddd-strategic-en.md` | The prompt for your next stage |
| 15 | `preparation/docs/prompts/templates/05-ddd-context-map-en.md` | Output template |

Paths are relative to the workspace root
`/Users/olegk/Documents/_docs/_dev/_horizont/ai-sdlc-workshop/`.

## 3. Current state (snapshot — verify against the backlog)

```
Done 12 | In review 2 | In progress 0 | Blocked 0 | Remaining 25
```

- **Passed gates:** G0 (mode), G0a–G0d (scope decisions), G1 (Level 1), G2 (Level 2),
  G3 (Level 3).
- **Pending gates:** G4 (bounded context boundaries), G5 (tactical model), G6 (ADR set).
- **In review:** `ARCH-107` (data flow), `ARCH-108` (control flow). Do not rewrite them
  unless the owner asks.
- **C4 stage is complete.** Do not reopen Level 1–3 unless a contradiction is found; if
  one is found, report it as an open question, do not silently edit confirmed artifacts.

**Documentation language.** `DOC-20` makes ASD-STE100 the standard. Each source has an
STE edition beside it at `<name>.ste.md`. The migration table and per-file links are in
[`backlog.md`](backlog.md#documentation-language--ste-100-migration). Pre-STE originals
are archived at [`_archive/pre-ste-2026-11-03/`](../_archive/pre-ste-2026-11-03/). Write
new documents in STE from the first draft. Do not edit an archived original.

## 4. Frozen decisions — do not revisit

| ID | Decision |
|----|----------|
| SD-1 | Mode: from scratch |
| SD-2 | **Exactly 6 ADRs** (0001–0006); `DECISIONS.md` R-1…R-27 adopted by reference, never duplicated |
| SD-3 | Tactical DDD: full for `taste`, `feed`, `discovery`, `analysis`; light for `identity`, `library` |
| SD-4 | Contradictions are recorded as open questions with provisional defaults; **no rule edits** |
| SD-5 | The LLM is **external**; any `LLM_BASE_URL` is allowed; privacy claim is "not retained", not "never leaves the host" |
| SD-6 | `yt-dlp` is required for `feed`, optional for `library` |
| SD-7 | One person, one system at Level 1; friends' instances are not drawn |
| SD-8 | Artifacts live in `docs/20-system-design/`, **not** `docs/architecture/` (DOC-1 wins over the guide) |
| SD-9 | Transcripts are staged in `./data/tmp/transcripts/` between fetch and analysis commit, deleted on commit, TTL-swept |

## 5. Facts already established — do not rediscover or contradict

1. **One binary, two process roles.** A short-lived CLI client and a long-lived API
   server, both from `cmd/ytfeed`, over loopback HTTP (`OQ-16` covers lifecycle).
2. **Four containers:** CLI client, local API server, SQLite database, data directory.
3. **Six bounded contexts are fixed by `A-7`:** `identity`, `taste`, `discovery`,
   `analysis`, `feed`, `library`. DDD strategic must **validate** this split, not invent
   a new one. A different split requires an ADR plus a rules-PR proposal — stop and ask.
4. **Storage:** SQLite, WAL, `foreign_keys=ON`, `busy_timeout=5000`, `synchronous=FULL`,
   one writer, embedded auto-migrations with a backup before applying.
5. **Transport:** `127.0.0.1:7420`, bearer token `0600`, one UI lease, SSE for card
   streaming, `/v1` prefix.
6. **Transcripts:** never in SQLite, logs, traces, or backups. Staged only between fetch
   and analysis commit. The PRD was amended for this on 2026-11-02.
7. **Analysis:** one OpenAI-compatible adapter, schema-constrained JSON, explicit failure
   states (`no_transcript`, `analysis_slow`, `invalid_model_output`), rubric version
   recorded per result.
8. **No queue, no broker, no worker binary, no cache server.** The pipeline is in-process
   and DB-checkpointed.
9. **Batch ordering:** analyzed first; a card never moves backwards past the cursor; late
   analysis stays in place (`OQ-15`).
10. **Gesture protocol:** commit signal → acknowledge → advance the card; `gesture_id` is
    the idempotency key.

## 6. Hard constraints

| Constraint | Source |
|------------|--------|
| Never `git push` | `H-1` |
| Never commit secrets | `H-2` |
| Read before overwriting any file; ask if it is user work | `H-3` |
| Never widen the task scope | `H-4` |
| Never edit `docs/00-rules/` | `H-9`, SD-4 |
| Never edit the PRD without the owner approving the exact diff | `H-9` |
| No code, no `go.mod`, no dependencies | This effort is documentation only |
| No C4 Level 4 (code) diagrams | Prompt 01 `no_code_level` |
| No fourth architecture layer | `A-1` |
| Every diagram is Mermaid, in-file | Guide requirement |
| English everywhere | `R-3`, `DOC-3a` |

## 7. Operating protocol

1. **Pick one task** from `backlog.md` whose dependencies are `DONE` or `REVIEW`.
2. **Set it to `IN PROGRESS`** and recalculate the counters.
3. **Produce the artifact** at the exact output path in the backlog.
4. **Self-check** against the backlog's Definition of Done and the task's acceptance
   criteria.
5. **Stop at the stage gate.** Set the artifact to `REVIEW`, write the report, wait.
   After owner confirmation, set it to `DONE`.
6. **Update the backlog:** statuses, counters, gate log, notes log (append-only).
7. **Never mark a gated artifact `DONE` without the owner's confirmation.**

### Gates and what they mean

| Gate | Trigger |
|------|---------|
| G4 | After the bounded context boundaries are confirmed (end of `ARCH-205`) |
| G5 | After the tactical model is confirmed (end of `ARCH-307`) |
| G6 | After the ADR set is confirmed (end of `ARCH-407`) |
| Final | After assembly and audit (`ARCH-501`…`ARCH-505`) |

## 8. Your next five tasks

| ID | Task | Output | Acceptance |
|----|------|--------|------------|
| `ARCH-201` | DDD viability check: ask the four questions, score, record the decision | Section in `ddd/strategic/context-map.md` | Score and recommendation recorded; if 0–1, say DDD is overkill and ask before continuing |
| `ARCH-202` | Subdomain map: core / supporting / generic | `ddd/strategic/subdomain-map.md` | Every subdomain traces to a PRD capability; the differentiator is named |
| `ARCH-203` | Bounded context map + relationship patterns, validating `A-7` | `ddd/strategic/context-map.md` | Six contexts mapped to subdomains; every relationship has a named DDD pattern; any change to `A-7` is escalated, not applied |
| `ARCH-204` | Ubiquitous language glossary | `ddd/strategic/glossary.md` | Every term comes from the PRD or rules — no invented domain language; terms that differ per context are called out |
| `ARCH-205` | Boundary confirmation gate | Gate log + backlog | Owner confirms; then and only then proceed to tactical |

**Do not pre-commit the answers.** The viability score, the subdomain classification and
the relationship patterns must be derived from the PRD and confirmed with the owner. A
plausible guess presented as a finding is a defect.

### A hypothesis you may test, not assume

The differentiator is likely the **rubric-driven judgment plus the gesture-driven taste
profile**, not discovery or storage. If the analysis supports that, `taste` and `analysis`
are core; `discovery`, `feed` and `library` are supporting; `identity` is supporting or
generic. Confirm it, do not assert it.

## 9. Open questions — how to handle them

- The register is `open-questions.md`. `OQ-05` is resolved; do not renumber the rest.
- Every provisional default is a **documentation assumption**, not a decision.
- If a task depends on an unresolved OQ, use the provisional default and cite the OQ ID
  in the artifact.
- When the owner answers an OQ: move the row to **Resolved**, record the date and the
  document that carries the decision, and update any artifact that cited it.
- If the answer changes a PRD or rules statement: produce the exact diff, **wait for
  approval**, then apply. The transcript-staging amendment is the model for this.
- New contradictions found during DDD go into the register first, never straight into a
  document as fact.

## 10. Report format

```
Task:       ARCH-XXX — <title>
Status:     DONE / REVIEW / BLOCKED
Changed:    <files>
Verified:   <self-check performed>
Decisions:  <anything the owner must confirm>
Open:       <OQs touched, follow-ups>
Next:       <the next task ID>
```

Keep it short. The backlog carries the detail.

## 11. What "done" means for this documentation effort

The suite is complete when:

- Every P0 requirement `F1`…`F28` maps to a component and a bounded context
  (`ARCH-503` traceability matrix).
- Every architectural rule `A-*`, `D-*`, `CLI-*` is either applied or explicitly marked
  non-architectural (`ARCH-505`).
- Every ADR has real negatives and risks; the index lists the rules that have no ADR.
- Every open question is either resolved or carries an owner and a provisional default.
- No document contradicts `docs/00-rules/` without a linked ADR and a proposed rules PR.
- The master index (`docs/20-system-design/README.md`) and the executive summary exist.
- `ARCH-601` proposes the implementation backlog and folds this temporary backlog into it.

---

## Operator notes (not part of the paste)

- The brief is deliberately self-contained: a fresh session needs only the kickoff prompt.
- If the new session drifts, point it at `backlog.md` §"Operating protocol" and §"Hard
  constraints".
- `ARCH-107` and `ARCH-108` are in `REVIEW`; confirm or correct them before the final
  assembly, since `ARCH-504` depends on them.
- The DDD guide is at `preparation/docs/prompts/02-architecture/02-ddd-strategic-en.md`.
  Its critical rules are: viability first, language from the domain, confirm boundaries.
