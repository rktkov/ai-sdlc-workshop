# 08 — CLI and UX rules

> **Scope:** `cmd/`, `internal/transport/cli/`, terminal rendering.
> **Derived from:** `sales-funnel-bot/internal/cli` (`Run*` functions, a typed
> HTTP client, `httptest`-based tests) and the product requirements for this
> project.

---

## Command shape

**CLI-1 — Commands are functions with a uniform signature:**

```go
func RunFeed(ctx context.Context, c *Client, args []string) error
```

The transport layer parses; the function returns a domain-ish error; `main`
translates it to an exit code.

**CLI-2 — Parsing is `flag` + explicit dispatch.** No CLI framework in v1. Add
one only if nested subcommands exceed what `flag` can express cleanly.

**CLI-3 — Every command supports `--help`, and `ytfeed help` lists commands
with one-line descriptions.**

**CLI-4 — Commands are verbs; nouns are arguments.**

```
ytfeed feed                      # compose and browse today's feed
ytfeed feed --size 20
ytfeed serve                     # run the local API server (auto-started if absent)
ytfeed auth login
ytfeed auth status
ytfeed mute "ai slop" --for 72h
ytfeed save <video-id>
ytfeed stats
ytfeed doctor
ytfeed migrate up
```

**CLI-5 — A command that changes state prints exactly what changed**, in one
line, before exiting.

```
✓ muted topic "ai-agents" until 2026-11-04 12:00 UTC (reason: ai-slop)
```

---

## Exit codes

**CLI-6 — Exit codes are stable and documented:**

| Code | Meaning |
|------|---------|
| 0 | success |
| 1 | unexpected error |
| 2 | invalid usage / invalid input |
| 3 | not found |
| 4 | conflict (already exists, already saved) |
| 5 | quota exhausted / rate limited |
| 6 | degraded: completed with reduced functionality |
| 130 | interrupted (Ctrl-C) |

The table above is a public contract. Changing a code is a breaking change and
requires a rules PR.

**CLI-7 — Ctrl-C exits 130 and leaves persisted state consistent.** No partial
writes, no half-written feed.

---

## Output

**CLI-8 — Human-readable by default, machine-readable on request.** Every
command supports `--json`, which prints one JSON object or array and nothing
else on stdout.

**CLI-9 — Diagnostics go to stderr; data goes to stdout.** Piping
`ytfeed feed --json | jq` must work.

**CLI-10 — No color when not a TTY**, and `--no-color` forces it off.
`NO_COLOR` is respected.

**CLI-11 — No meaning is carried by color alone.** A slop score prints as
`slop: 0.82 (high)`, not as a red bar. See `01-go-language.md` G-? and the
accessibility NFR.

**CLI-12 — Width is detected, not assumed.** Text wraps to the terminal width;
at 80 columns the card must still be readable.

---

## The card

**CLI-13 — A card always shows, in this order:** thumbnail (if supported),
title, channel + subscriber count + views + duration + publish date, topic,
TL;DR, similarity signal ("similar to: X" / "new territory"), why-this-card
reason, and analysis state.

**CLI-14 — Unanalyzed cards are visually and textually distinct**, with the
reason: `[unanalyzed: no transcript]` or `[unanalyzed: analysis slow]`.

**CLI-14a — Analyzed cards precede unanalyzed cards within a batch.** Analysis
streams in, and a card that finishes analysis moves ahead of cards still
pending. Ordering is stable for cards in the same state: by score descending,
then by discovery order. A card never moves *backwards* past the user's
current position.

**CLI-15 — Terminal image support is detected at startup** (Ghostty, iTerm2,
Kitty, WezTerm protocols) and cached in config. `--no-images` disables it
regardless of detection.

**CLI-16 — Degrade to text, never to a broken box.** If an image cannot be
rendered, the card renders without it and does not reserve space.

---

## Interaction

**CLI-17 — Keys are documented on screen**, in a footer line that can be hidden
with `--quiet`.

**CLI-18 — Every action is idempotent from the user's point of view.**
Left-swiping an already-muted target refreshes the expiry instead of erroring.

**CLI-19 — Destructive actions confirm once** and support `--yes` for
scripting. Removing from the watch list is not destructive; deleting the local
database is.

**CLI-20 — The reason picker after a left-swipe is keyboard-only.** Keys 1–4
select marketing / ai-slop / value-misaligned / other; the last used reason is
preselected; Enter accepts. Skipping with Esc records `other`. The card does
not advance until the reason is resolved or skipped.

**CLI-20a — The reason determines the mute scope:**

| Reason | Scope | Rationale |
|--------|-------|-----------|
| `value-misaligned` | channel | the author is the problem, not the subject |
| `marketing` | channel | promotional accounts repeat across topics |
| `ai-slop` | topic | the content type is the problem |
| `other` | topic | safest default, least destructive |

The picker displays the resulting scope (`mute channel: <name>, 24h`) before
confirming, so the consequence is never hidden.

**CLI-21 — A session opens on UI start and closes after 1 hour of inactivity**
(`SESSION_IDLE_TIMEOUT`, default `1h`). The CLI prints `session resumed` when an
existing session is still open, so "down" navigation is not silently reset.

---

## Failure UX

**CLI-22 — The app always starts if the database is readable.** Network
failures degrade to cached cards and saved videos; they never block the UI.

**CLI-23 — Saved videos open in the system player** via a configurable command
(`PLAYER_CMD`, default detected per OS).

**CLI-24 — Long operations show progress on stderr**, and never block
interaction: analysis streams in, cards appear as they become available.

---

## Onboarding

**CLI-25 — Onboarding is exactly two steps.**

1. Connect the Google profile and import subscriptions.
2. Choose ~5 current interests.

**CLI-26 — Step 2 offers three seed sources, selected by the user:**

| Source | What it uses | Notes |
|--------|--------------|-------|
| `youtube-trending` | `videos.list?chart=mostPopular` + topics inferred from subscriptions | default; no extra dependency |
| `external-trends` | Google Trends / Reddit / HN adapters | optional; if an adapter is absent the source is listed as unavailable, never silently substituted |
| `subscriptions-only` | the subscription graph alone | zero extra API cost, but cannot escape the current bubble |

The choice is stored as `ONBOARDING_SEED_SOURCE` and can be re-run with
`ytfeed onboard --reset`. Onboarding must work for the product owner with no
privileged setup path.

**CLI-27 — Onboarding never blocks the first feed.** If step 2 is skipped, the
feed composes from subscriptions with a visible banner: `interests not set —
feed may be narrow.`

**CLI-28 — The quality rating is asked weekly, on session open, never mid-feed.**
Scale: `-1 worse`, `0 same`, `+1 better`. The prompt is dismissible with Esc and
records `unrated`. Each rating stores the week number and the IDs of the feeds
shown that week, so a change in rating can be tied to a change in the feed.
