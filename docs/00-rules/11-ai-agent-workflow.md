# 11 — AI agent workflow rules

> **Scope:** every agent session in this repository.
> **Derived from:** `sales-funnel-bot/README.md` — "START HERE", work rules,
> mandatory Definition of Done, backlog discipline, agent checklist.

---

## Session start

**AI-1 — Read before you write.** Read in this order: `AGENTS.md`, this folder's
`README.md`, the rules relevant to the task, the PRD, the ADRs, and the backlog.

**AI-2 — State the task before you start.** Quote the task ID and its acceptance criteria.
If either is missing, stop.

**AI-3 — Check dependencies.** Do not start a task with unmet dependencies.

**AI-4 — Restate the plan in 5 lines or fewer.** Wait for confirmation when the task is
larger than one obvious change.

---

## During work

**AI-5 — Work on one task at a time.** No opportunistic refactors, no drive-by
renames, and no "while I was here" changes.

**AI-6 — Test first.** Write the failing test, run it, paste the failure, then
implement it (rule T-21).

**AI-7 — Ask before you add a dependency, change a schema, or touch auth.**

**AI-8 — Keep the build green between commits.** A commit that does not build is a broken
checkpoint.

**AI-9 — When a rule blocks you, name the rule.** "Blocked by A-4: the task
requires SQL in the handler" is a valid report. A silent violation is not.

**AI-10 — Do not invent product behaviour.** If the PRD is silent, record an
open question and pick the smallest reversible behaviour. Mark it as provisional.

---

## Session end

**AI-11 — Run the full gate:** `make lint && make test && make coverage-check
&& make build`. Paste the output.

**AI-12 — Update the backlog** (rule DOC-12). A stale backlog invalidates the
task.

**AI-13 — Report in the fixed format** (see `AGENTS.md` §4).

**AI-14 — List the commits. Never push.**

**AI-15 — Leave a "next step" note** in the backlog if the task produced
follow-ups.

---

## Self-audit before reporting

Run this checklist against your own diff. Answer each line in the report:

- [ ] Does every new exported symbol have a doc comment?
- [ ] Is there any SQL outside `internal/repository/`?
- [ ] Is there any business logic in a handler or CLI command?
- [ ] Is every error wrapped with `%w` and checked with `errors.Is`?
- [ ] Is any secret, token or transcript content logged?
- [ ] Does every new branch have a test that would fail without the change?
- [ ] Is any interface defined in the implementing package instead of the
      consuming one?
- [ ] Did I add a dependency without justification?
- [ ] Did I leave commented-out code, debug prints, or a skipped test?
- [ ] Does the diff contain anything unrelated to the task?

A "yes" to any of the last four questions is a defect. Fix it before you report.

---

## Escalation triggers

Stop and ask when:

| Trigger | Example |
|---------|---------|
| Contradiction | The task says "store in Postgres", rule D-6 says SQLite behind an interface |
| Missing decision | The PRD does not say what a left-swipe on an unanalyzed card does |
| New dependency | A library is needed that is not in `go.mod` |
| Data loss risk | A migration drops a column or rewrites a table |
| Security surface | OAuth scopes, token storage, anything that touches credentials |
| Unclear acceptance | "Make it good" is not testable |

---

## Anti-patterns that will be rejected

| Anti-pattern | Why it is rejected |
|--------------|--------------------|
| Implementing before the test exists | Cannot prove the behaviour |
| Broad refactor inside a feature task | Unreviewable diff |
| `// TODO` left without a backlog item | Hidden debt |
| Silent degradation | The user cannot tell what is missing |
| Guessing a product rule | Wrong product. Cheap to avoid. |
| "It works on my machine" | No gate output pasted |
| Updating code but not the rule it violates | The rule decays |
| Pushing to remote | Owner-only action |
