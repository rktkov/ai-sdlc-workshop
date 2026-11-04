# 09 — Git workflow rules

> **Scope:** every commit, branch and pull request.
> **Derived from:** `sales-funnel-bot/README.md` (work rules, Definition of
> Done, commit format `feat(sfb): ... (T-XXX)`), `.gitignore`.

---

## Branching

**GH-1 — Trunk-based with short-lived branches.** `main` is always buildable.

**GH-2 — Branch naming:**

```
feature/{task-id}-{short-description}    feature/T-012-feed-composer
fix/{task-id}-{short-description}        fix/T-031-cold-period-expiry
chore/{short-description}                chore/add-goose
docs/{short-description}                 docs/rules-review
```

**GH-3 — Branches live for hours, not weeks.** If a branch is older than two
days, split it or rebase it.

**GH-4 — `main` is the only integration branch.** No `develop`, no `staging`.
Releases are tags on `main`. This diverges from the reference project, which
protects `main` and `staging`; a second long-lived branch buys nothing while
the product runs locally, and can be added later without rewriting history.

---

## Commits

**GH-5 — Conventional Commits, mandatory.**

```
{type}({scope}): {description}
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `perf`,
`ci`.

**GH-6 — Scope is the bounded context or area**, not the project prefix:
`feed`, `taste`, `analysis`, `discovery`, `cli`, `repo`, `docs`, `deps`.

```
feat(feed): compose cards with cold-period filtering
fix(analysis): treat missing transcript as pending, not failed
test(taste): cover cold-period expiry boundaries
chore(deps): add modernc.org/sqlite
docs(rules): add go language rules
```

**GH-7 — Task ID goes in the commit body, not the subject.** The reference
project puts `(T-XXX)` in the subject; that rots and bloats the log. Use:

```
feat(feed): compose cards with cold-period filtering

Refs: T-012
```

This is a deliberate divergence from the reference project, which puts
`(T-XXX)` in the subject. Confirmed: the subject stays clean, the task ID
lives in the body.

**GH-8 — Subject ≤ 72 characters, imperative mood, no trailing period.**

**GH-9 — Body is required for `feat` and `fix`**: what changed, why, and how it
was verified.

**GH-10 — One logical change per commit.** A commit that adds a dependency and
refactors three packages cannot be reviewed or reverted.

**GH-11 — Never commit generated artefacts**: binaries, `coverage.out`,
`node_modules`, local databases, downloaded videos, `.env`.

---

## Pushing and review

**GH-12 — Only the owner pushes.** Agents commit locally and report the commit
list. This is a hard rule (`AGENTS.md` H-1).

**GH-13 — Pull requests state:** task ID, what changed, how it was verified,
and any decision a reviewer must confirm.

**GH-14 — A PR is mergeable only when `make check` and `make coverage-check`
pass.**

**GH-15 — Squash merge into `main`**, so the history is one commit per task.

**GH-16 — Delete the branch after merge.**

---

## Versioning

**GH-17 — Semantic versioning, tags `v{major}.{minor}.{patch}`.** A single
binary needs a `--version` flag that prints the tag and commit SHA, injected at
build time:

```make
build:
	go build -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT)" ./cmd/ytfeed
```

**GH-18 — `v0.x` until the first daily-use milestone.** No stability promises
before then.

---

## Repository hygiene

**GH-19 — `.gitignore` covers:** `/bin/`, `*.test`, `coverage.out`, `.env`,
`.env.*` (except `.env.example`), local data dir, `*.db`, `*.db-wal`,
`*.db-shm`, downloaded media.

**GH-20 — `.gitattributes` normalises line endings** (`* text=auto eol=lf`).

**GH-21 — `.editorconfig` enforces spaces, LF, UTF-8, final newline**, with
Markdown exempt from trailing-whitespace trimming.
