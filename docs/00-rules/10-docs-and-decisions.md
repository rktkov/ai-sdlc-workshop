# 10 — Documentation and decision rules

> **Scope:** everything under `docs/`.
> **Derived from:** the `sales-funnel-bot` docs tree and its ADR format.

---

## Tree

**DOC-1 — Number documentation by audience and lifecycle:**

```
docs/
├── 00-rules/            # engineering contract (this folder)
├── 10-product/          # PRD, product decisions
├── 20-system-design/    # ADRs, C4, DDD, RFCs
│   ├── adr/
│   └── rfc/
├── 30-implementation/   # backlog, tasks, epics, user stories
├── 40-business-flows/   # end-to-end flows
└── 50-runbooks/         # operations
```

**DOC-2 — The number is the reading order.** Put a new document in the folder
that matches its audience. Never put it at the root.

**DOC-3 — File names are `kebab-case.md`.** ADRs are
`NNNN-short-title.md`, numbered in sequence, never reused.

**DOC-3a — Write all documentation in English.** Code comments, docs, ADRs,
backlog items and commit messages are English. Russian appears only in quoted
user-facing content or source material, and you translate it when you cite it.

---

## ADRs

**DOC-4 — Every architectural decision gets an ADR.** A change to a boundary, a
dependency, a storage engine, a protocol or a deployment shape needs an ADR.

**DOC-5 — ADR structure is fixed:**

```markdown
# ADR-NNNN: Title

## Status
Proposed | Accepted | Superseded by ADR-NNNN | Rejected

## Date
YYYY-MM-DD

## Deciders
- ...

## Context
What forces are at play. What problem must be solved.

## Driving factors
* **Must:** ...
* **Should:** ...

## Considered options
### Option 1: ...
Pros / cons.

## Decision
What we will do.

## Rationale
Why this option, in terms of the driving factors.

## Consequences
### Positive
### Negative
### Risks and mitigations

## Related decisions
## References
```

**DOC-6 — ADRs are immutable.** To change a decision, write a new ADR that
supersedes the old one. Update only the old one's status line.

**DOC-7 — An ADR that contradicts a rule file wins, but only until the rule
file is updated.** After the update, the rule wins. Until then, the
inconsistency is a bug.

**DOC-8 — `docs/20-system-design/adr/INDEX.md` lists every ADR** with number,
title, status and one-line decision.

---

## RFCs

**DOC-9 — Use an RFC for a proposal that is not yet a decision.** An accepted
RFC becomes an ADR. The RFC stays as history.

**DOC-10 — An RFC states the problem, the options, the recommendation, the
open questions and a decision deadline.**

---

## Backlog discipline

**DOC-11 — `docs/30-implementation/backlog.md` is the single source of truth
for progress.** Do not use the issue tracker, the chat, or memory.

**DOC-12 — After every completed task, before starting the next one, the agent
updates `backlog.md`:**

- move the task from "Remaining" to "Done".
- recalculate the header counters and the section counts.
- update the "last updated" date.
- set the task status in `tasks.md` to `DONE`.
- update the epic status if the epic is complete.

**DOC-13 — A task is not done while the backlog is stale.** This is the
reference project's hardest rule, and it is correct.

**DOC-14 — Backlog items carry:** ID, title, epic, size (S/M/L/XL), status,
dependencies, and acceptance criteria.

---

## Writing rules

**DOC-15 — Documents state decisions, not options.** An option list belongs in
the ADR's "Considered options" section, not in the product doc.

**DOC-16 — Every requirement is testable.** "Fast" is not a requirement.
"p95 first card < 2 s" is.

**DOC-17 — Numbers beat adjectives.** Make targets, thresholds and limits
explicit.

**DOC-18 — Link, do not copy.** A rule referenced in three documents is a rule
that will diverge.

**DOC-19 — Delete a document that is no longer true, or mark it superseded.**
Stale documentation is worse than none.

**DOC-20 — Write all documentation in ASD-STE100 Simplified Technical English.**
Apply the STE-100 writing rules. Use sentences of 25 words or fewer, active voice and
simple tenses. Write one idea per sentence. Do not use semicolons. Use one term for one
meaning. Keep code, identifiers, paths, links and Mermaid blocks unchanged. New documents
are STE from the first draft. An existing document migrates to an STE edition. When the
owner accepts that edition, it becomes the normative text and the pre-STE original moves to
`docs/_archive/`. After migration, the STE edition takes the original file name. This rule
applies to every file under `docs/`.
