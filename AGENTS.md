# AGENTS.md — mandatory operating instructions for AI agents

This repository is built and maintained by AI agents under human review.
Read this file completely before touching any file. It is not advisory.

If a rule here conflicts with a task instruction, **stop and ask**. Do not
silently choose.

---

## 1. Read order (do not skip)

1. `AGENTS.md` — this file.
2. `docs/00-rules/README.md` — rule index, precedence, how to change a rule.
3. Every file in `docs/00-rules/` that is relevant to the task.
4. `docs/10-product/` — what we are building and why.
5. `docs/20-system-design/` — ADRs and architecture.
6. `docs/30-implementation/backlog.md` — the only source of truth for what to
   do next.

Skipping steps 2–3 is the most common cause of rejected work.

---

## 2. Hard rules (a violation blocks the merge)

| ID | Rule |
|----|------|
| H-1 | **Never run `git push`.** The owner pushes. Agents may commit locally. |
| H-2 | **Never commit secrets.** Tokens, API keys, OAuth client secrets, `.env` files stay out of git. |
| H-3 | **Never overwrite an existing file silently.** Read it first; if it is user work, ask. |
| H-4 | **Never widen the task scope.** If the task needs more, stop and report it as a proposed follow-up. |
| H-5 | **Never leave the repo un-buildable.** `make check` must pass before you report done. |
| H-6 | **Never write SQL outside `internal/repository/`.** See `02-architecture.md`. |
| H-7 | **Never put business logic in transport handlers or CLI commands.** See `02-architecture.md`. |
| H-8 | **Never mark a task done without updating `docs/30-implementation/backlog.md`.** See `11-ai-agent-workflow.md`. |
| H-9 | **Never invent a convention that contradicts `docs/00-rules/`.** If a rule is wrong, propose a change to the rule, do not fork it. |
| H-10 | **Never use a library that is not already in `go.mod` without asking.** |

---

## 3. Definition of Done

A task is done only when **all** of the following are true:

- [ ] Code follows every applicable rule in `docs/00-rules/`.
- [ ] Unit tests cover new logic; `make test` passes.
- [ ] `make coverage-check` passes (≥ 80% per tested package).
- [ ] `make lint` (golangci-lint) passes.
- [ ] `make build` produces a working binary.
- [ ] `make check` passes end to end.
- [ ] Behaviour changes are reflected in the relevant doc.
- [ ] `docs/30-implementation/backlog.md` is updated: task moved to
      "Done", counters and dates recalculated.
- [ ] `docs/30-implementation/tasks.md` status is set to `DONE`.
- [ ] Commits follow `09-git-workflow.md` and reference the task ID.
- [ ] The final report lists: what changed, what was verified, what is still
      open, and the commits ready to push.

---

## 4. Report format (every task)

```
Task:       T-XXX — <title>
Status:     DONE / BLOCKED / PARTIAL
Changed:    <files or areas>
Verified:   <commands run and their result>
Decisions:  <anything you chose that a reviewer must confirm>
Open:       <follow-ups, risks, questions>
Commits:    <sha> <subject>   (ready to push, not pushed)
```

---

## 5. When to stop and ask

Stop immediately if:

- the task contradicts a rule in `docs/00-rules/`;
- the task requires a new dependency;
- the task touches authentication, secrets, or data deletion;
- acceptance criteria are missing or untestable;
- you would have to guess a product decision.

A blocked task is a valid outcome. A silent guess is not.

---

## 6. Where things live

| Path | Contents |
|------|----------|
| `docs/00-rules/` | Engineering rules — the contract for all code |
| `docs/10-product/` | PRD, product decisions |
| `docs/20-system-design/` | ADRs, C4, DDD, RFCs |
| `docs/30-implementation/` | Backlog, tasks, epics, user stories |
| `docs/40-business-flows/` | End-to-end flows |
| `docs/50-runbooks/` | Operational guides |
| `cmd/` | Entry points (one folder per binary) |
| `internal/` | Private application code |
| `shared/` | Cross-cutting types and utilities |
| `migrations/` | Schema migrations |
| `scripts/` | Dev/build/ops scripts |
