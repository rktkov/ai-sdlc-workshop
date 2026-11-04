# Rules review — open decisions

> **Purpose:** collect every rule that is not simply inherited from the reference
> project in one place, so you can decide quickly. Delete this file once you make
> all decisions and freeze the rules.

**Status legend:** ☐ open · ☑ decided

---

## Blocking (needed before the first line of code)

| # | Decision | Options | Recommendation | Status |
|---|----------|---------|----------------|--------|
| R-1 | Module path | `github.com/rktkov/...` variants | **`github.com/rktkov/ytfeed`** | ☑ |
| R-2 | Product name / binary name | — | **`ytfeed`**. Binary `ytfeed`, data dir `~/.local/share/ytfeed`. | ☑ |
| R-3 | Documentation language | English / Russian / mixed | **English everywhere** | ☑ |
| R-4 | Package naming style | `feed.Service` vs `feed.FeedService` | **strict Go, no stutter** (`feed.Service`) | ☑ |
| R-5 | Linter | `go vet` only vs `golangci-lint` | **`golangci-lint` from day one**, pinned, minimal set | ☑ |
| R-6 | Task ID placement in commits | subject `(T-XXX)` vs body `Refs:` | **body `Refs:`** | ☑ |
| R-7 | Branch protection | `main` only vs `main` + `staging` | **`main` only** | ☑ |

## Architecture

| # | Decision | Options | Recommendation | Status |
|---|----------|---------|----------------|--------|
| R-8 | Bounded contexts | six vs four vs three | **six**: identity, taste, discovery, analysis, feed, library | ☑ |
| R-9 | CLI↔server transport in v1 | unix socket vs localhost TCP vs in-process | **HTTP over 127.0.0.1**, token-authenticated | ☑ |
| R-10 | Single binary layout | one vs two vs three binaries | **one binary**, subcommands incl. `serve` | ☑ |

## Data

| # | Decision | Options | Recommendation | Status |
|---|----------|---------|----------------|--------|
| R-11 | SQLite durability | `synchronous=NORMAL` vs `FULL` | **`FULL`**. Every commit is fsynced. | ☑ |
| R-12 | Migration trigger | auto on startup vs explicit `ytfeed migrate up` | **auto + backup before applying** | ☑ |
| R-13 | Data dir location | XDG vs `~/.ytfeed` vs project-local | **`./data`** (env override `YTFEED_DATA_DIR`) | ☑ |

## Behavioural contracts

| # | Decision | Options | Recommendation | Status |
|---|----------|---------|----------------|--------|
| R-14 | Exit-code table | proposed 0–6, 130 | **0–6 plus 130**, as documented | ☑ |
| R-15 | Analysis ordering | analyzed first vs discovery order vs two sections | **analyzed first, unanalyzed last** | ☑ |
| R-16 | Left-swipe reason | mandatory vs skippable | **skippable**, last used preselected, Esc → `other` | ☑ |
| R-17 | Cold period scope | topic vs channel vs both | **scope follows reason**: misaligned/marketing → channel, ai-slop/other → topic | ☑ |
| R-18 | Session timeout | 1h vs 30m vs explicit | **1h**, `SESSION_IDLE_TIMEOUT` | ☑ |

## Product

| # | Decision | Options | Recommendation | Status |
|---|----------|---------|----------------|--------|
| R-19 | Filter rubric ownership | agent-authored vs you author | **agent drafts from research, you edit**. Versioned artifact (A-19). | ☑ |
| R-20 | "Values alignment" definition | explicit axes vs free text | **named axes + weights + examples** (A-20) | ☑ |
| R-21 | Trend source for onboarding | youtube-trending vs external vs subscriptions-only | **user-selectable**, default `youtube-trending` (CLI-26) | ☑ |
| R-22 | NPS-like measure | −1 / 0 / +1 | **weekly**, on session open, tied to that week's feeds (CLI-28) | ☑ |
| R-23 | Function / file length | 50/80, 300/500 vs stricter vs none | **aim 50, hard 80; file aim 300, hard 500** (G-23/G-24) | ☑ |
| R-24 | Metrics backend | none vs local OTel traces vs Prometheus | **local OTel traces to JSONL**, no collector (L-15) | ☑ |
| R-25 | Model runtime abstraction | per-runtime adapters vs one OpenAI-compatible adapter | **one OpenAI-compatible chat adapter + presets** (A-12a) | ☑ |
| R-26 | Model output format | free text vs schema-constrained JSON | **schema-constrained JSON, validated** (A-12b) | ☑ |
| R-27 | Local model family | Qwen / Llama / Liquid AI LFM | **evaluate Liquid AI LFM2.5 + one dense baseline** in a spike | ☑ |

---

## Decision log summary

All 27 decisions are closed. The rules are frozen as of 2026-11-02. Further
changes go through the rules PR process in `README.md` §4.
