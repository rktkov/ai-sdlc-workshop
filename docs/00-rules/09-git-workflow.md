# 09 — Git workflow rules

> **Scope:** every commit, branch and pull request.
> **Derived from:** `sales-funnel-bot/README.md` (work rules, Definition of
> Done, commit format `feat(sfb): ... (T-XXX)`), `.gitignore`.

---

## Branching

**GH-1 — Use trunk-based development with short-lived branches.** `main` is always
buildable.

**GH-2 — Name a branch as follows:**

```
feature/{task-id}-{short-description}    feature/T-012-feed-composer
fix/{task-id}-{short-description}        fix/T-031-cold-period-expiry
chore/{short-description}                chore/add-goose
docs/{short-description}                 docs/rules-review
```

**GH-3 — Keep a branch for hours, not weeks.** If a branch is older than two days, split it
or rebase it.

**GH-4 — `main` is the only integration branch.** Do not add `develop` or `staging`.
Releases are tags on `main`. This diverges from the reference project, which protects
`main` and `staging`. A second long-lived branch adds no value while the product runs
locally. You can add one later without rewriting history.

---

## Commits

**GH-5 — Conventional Commits are mandatory.**

```
{type}({scope}): {description}
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `perf`,
`ci`.

**GH-6 — The scope is the bounded context or area**, not the project prefix:
`feed`, `taste`, `analysis`, `discovery`, `cli`, `repo`, `docs`, `deps`.

```
feat(feed): compose cards with cold-period filtering
fix(analysis): treat missing transcript as pending, not failed
test(taste): cover cold-period expiry boundaries
chore(deps): add modernc.org/sqlite
docs(rules): add go language rules
```

**GH-7 — Put the task ID in the commit body, not the subject.** The reference project puts
`(T-XXX)` in the subject. That habit rots and bloats the log. Use:

```
feat(feed): compose cards with cold-period filtering

Refs: T-012
```

This is a deliberate divergence from the reference project, which puts `(T-XXX)` in the
subject. Confirmed: the subject stays clean and the task ID lives in the body.

**GH-8 — Keep the subject at 72 characters or fewer. Use imperative mood. No trailing
period.**

**GH-9 — A body is required for `feat` and `fix`.** State what changed, why, and how you
verified it.

**GH-10 — One logical change per commit.** A reviewer cannot review or revert a commit that
adds a dependency and refactors three packages.

**GH-11 — Never commit generated artefacts**: binaries, `coverage.out`,
`node_modules`, local databases, downloaded videos, `.env`.

---

## Pushing and review

**GH-12 — Only the owner pushes.** Agents commit locally and report the commit list. This
is a hard rule (`AGENTS.md` H-1).

**GH-13 — A pull request states:** the task ID, what changed, how you verified it, and any
decision a reviewer must confirm.

**GH-14 — A PR is mergeable only when `make check` and `make coverage-check` pass.**

**GH-15 — Squash merge into `main`**, so the history is one commit per task.

**GH-16 — Delete the branch after merge.**

---

## Versioning

**GH-17 — Use semantic versioning with tags `v{major}.{minor}.{patch}`.** A single binary
needs a `--version` flag. The flag prints the tag and the commit SHA, which the build
injects:

```make
build:
	go build -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT)" ./cmd/ytfeed
```

**GH-18 — Stay at `v0.x` until the first daily-use milestone.** Make no stability promises
before that milestone.

---

## Repository hygiene

**GH-19 — `.gitignore` covers:** `/bin/`, `*.test`, `coverage.out`, `.env`,
`.env.*` (except `.env.example`), local data dir, `*.db`, `*.db-wal`,
`*.db-shm`, downloaded media.

**GH-20 — `.gitattributes` normalises line endings** (`* text=auto eol=lf`).

**GH-21 — `.editorconfig` enforces spaces, LF, UTF-8 and a final newline.** Markdown is
exempt from trailing-whitespace trimming.
