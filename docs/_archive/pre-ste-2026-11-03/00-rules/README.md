# Engineering rules — index

> **Status:** draft for review. Every file here is expected to be challenged,
> corrected and only then frozen.
> **Derived from:** the `sales-funnel-bot` project
> (`/horizont-tech/telegram-bots/sales-funnel-bot`), its `README.md`, ADRs
> 0001–0014, `Makefile`, `go.mod`, `Dockerfile`, `.env.example` and sampled
> source files.
> **Applies to:** all code, docs and commits in this repository.

---

## 1. How to read these rules

Each file is a small, reviewable unit. Rules have stable IDs (`G-3`, `A-1`,
`T-5`) so a review comment can point at exactly one rule.

Rules are marked:

| Marker | Meaning |
|--------|---------|
| *(no marker)* | Observed and enforced in the reference project. |
| `[proposed]` | Not present in the reference project — my recommendation for this one. Needs your decision. |
| `[decision needed]` | A real fork in the road; the rule cannot be finalised without you. |

**A rule that is not testable is not a rule.** If you cannot check it in review
or in CI, it should not be here.

---

## 2. Files

| # | File | Covers |
|---|------|--------|
| 01 | [`01-go-language.md`](01-go-language.md) | Go version, formatting, naming, comments, dependency policy |
| 02 | [`02-architecture.md`](02-architecture.md) | Layers, package boundaries, ports and adapters, directory layout |
| 03 | [`03-errors.md`](03-errors.md) | Sentinel errors, wrapping, translation at boundaries |
| 04 | [`04-data-persistence.md`](04-data-persistence.md) | Repository pattern, SQLite, migrations, transactions |
| 05 | [`05-config-and-secrets.md`](05-config-and-secrets.md) | Env-only config, grouped structs, fail-fast validation |
| 06 | [`06-logging-and-observability.md`](06-logging-and-observability.md) | `log/slog`, levels, what must never be logged |
| 07 | [`07-testing.md`](07-testing.md) | stdlib testing, mocks, integration tag, coverage gate |
| 08 | [`08-cli-and-ux.md`](08-cli-and-ux.md) | Command shape, exit codes, output modes, terminal capability |
| 09 | [`09-git-workflow.md`](09-git-workflow.md) | Branching, Conventional Commits, PR rules |
| 10 | [`10-docs-and-decisions.md`](10-docs-and-decisions.md) | Docs tree, ADR format, backlog discipline |
| 11 | [`11-ai-agent-workflow.md`](11-ai-agent-workflow.md) | Task selection, DoD, reporting, escalation |
| — | [`DECISIONS.md`](DECISIONS.md) | Log of the 27 frozen decisions that shaped these rules |

---

## 3. Precedence

When rules disagree, resolve in this order:

1. `AGENTS.md` hard rules (H-1 … H-10)
2. An accepted ADR in `docs/20-system-design/adr/`
3. These rule files
4. The task description
5. Existing code that predates the rules

Existing code is **never** an argument for keeping a bad pattern. If code
contradicts a rule, the code is wrong.

---

## 4. Changing a rule

1. Open a PR that edits the rule file **and** states the reason.
2. If the change alters architecture, write an ADR first and link it.
3. Update `AGENTS.md` if the change affects agent behaviour.
4. Do not edit a rule to justify code you already wrote. Change the code.

---

## 5. Review checklist for a rules PR

- [ ] Each rule is imperative and testable.
- [ ] Each `[proposed]` rule has an explicit decision or is removed.
- [ ] Examples compile (Go) or are clearly marked pseudo-code.
- [ ] No rule duplicates another; cross-reference instead.
- [ ] Provenance is honest: nothing claims to be "observed" that is invented.
