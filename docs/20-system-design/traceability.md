# Traceability matrix — PRD to components and contexts

## Status

In review for the final audit.

## Method

Each P0 requirement `F1`…`F28` maps to at least one component and one owning context. A
requirement that maps to transport or storage is marked as such. That is a valid mapping,
not a gap (`ARCH-503` acceptance). The matrix is derived from the PRD and the confirmed
C4 and DDD artifacts. It does not introduce new design.

## Matrix

| ID | Requirement | Component(s) | Context | Design source |
|----|-------------|--------------|---------|---------------|
| `F1` | Google OAuth, `youtube.readonly`, token `0600` | OAuth flow, Token store | `identity` | [`02-identity-taste`](c4/03-components/02-identity-taste.md) |
| `F2` | Import subscriptions | Subscription importer | `identity` | [`02-identity-taste`](c4/03-components/02-identity-taste.md) |
| `F3` | Choose ~5 interests from a seed source | Onboarding orchestrator, Profile service, Query planner | `identity`, `taste`, `discovery` | [`02-identity-taste`](c4/03-components/02-identity-taste.md), [`03-discovery-analysis`](c4/03-components/03-discovery-analysis.md) |
| `F4` | Taste profile with axes, weights, examples and note | Profile service | `taste` | [`taste/aggregates`](ddd/tactical/taste/aggregates.md) §1 |
| `F5` | Every gesture updates the profile | Signal recorder, Weight updater, Cold-period manager, Gesture service | `taste`, `feed` | [`taste/aggregates`](ddd/tactical/taste/aggregates.md) §2–§4, [`feed/aggregates`](ddd/tactical/feed/aggregates.md) §5 |
| `F6` | Search, graph expansion and micro-bloggers | Query planner, YouTube adapter, Candidate pool, Micro-blogger balancer | `discovery` | [`discovery/aggregates`](ddd/tactical/discovery/aggregates.md) §1 |
| `F7` | Quota accounting and rate limiting | Quota ledger | `discovery` | [`discovery/aggregates`](ddd/tactical/discovery/aggregates.md) §2 |
| `F8` | Dedupe watched, saved, shown 7d and duplicate IDs | Dedupe filter, `ExclusionSet` from `feed` | `discovery`, `feed` | [`discovery/aggregates`](ddd/tactical/discovery/aggregates.md) §1.3 |
| `F9` | Transcript-only ingestion and bounded staging | Transcript fetcher, `TranscriptStore` | `analysis` | [`analysis/aggregates`](ddd/tactical/analysis/aggregates.md) §2 |
| `F10` | Rubric-driven structured analysis | Prompt builder, LLM adapter, Validator | `analysis` | [`analysis/aggregates`](ddd/tactical/analysis/aggregates.md) §1 |
| `F11` | Explicit analysis failure states | Validator, Card state | `analysis`, `feed` | [`analysis/aggregates`](ddd/tactical/analysis/aggregates.md) §1.3, [`feed/aggregates`](ddd/tactical/feed/aggregates.md) §1 |
| `F12` | Rubric version per result | Rubric loader, Analysis repository | `analysis` | [`analysis/aggregates`](ddd/tactical/analysis/aggregates.md) §1 |
| `F13` | Compose 20 cards, configurable | Batch composer, Batch orchestrator | `discovery`, `feed` | [`discovery/aggregates`](ddd/tactical/discovery/aggregates.md) §1, [`feed/aggregates`](ddd/tactical/feed/aggregates.md) §1 |
| `F14` | Incremental streaming, analyzed first | Feed composer, Cursor, Stream writer | `feed`, transport | [`feed/aggregates`](ddd/tactical/feed/aggregates.md) §1.4, [`01-transport`](c4/03-components/01-transport.md) |
| `F15` | Card rendering in the CLI | Renderer | transport | [`01-transport`](c4/03-components/01-transport.md) |
| `F16` | Left-swipe with skippable reason picker | Gesture service, Gesture loop, Cold-period manager | `feed`, `taste`, transport | [`feed/aggregates`](ddd/tactical/feed/aggregates.md) §5, [`taste/aggregates`](ddd/tactical/taste/aggregates.md) §3 |
| `F17` | Right, down, up, tap and double-tap | Gesture service, Library service, Downloader | `feed`, `library` | [`feed/aggregates`](ddd/tactical/feed/aggregates.md) §5, [`library/model`](ddd/tactical/library/model.md) |
| `F18` | Session lifecycle and resume | Session manager, Session lease | `feed`, transport | [`feed/aggregates`](ddd/tactical/feed/aggregates.md) §2 |
| `F19` | Cold periods as records | Cold-period manager | `taste` | [`taste/aggregates`](ddd/tactical/taste/aggregates.md) §3 |
| `F20` | Library with local playback | Library service, Downloader, File manager, Player launcher | `library` | [`library/model`](ddd/tactical/library/model.md) §1 |
| `F21` | Weekly −1/0/+1 rating | Weekly rating | `feed` | [`feed/aggregates`](ddd/tactical/feed/aggregates.md) §3 |
| `F22` | Local HTTP server, token, loopback only | Router, Middleware, Handlers, Session lease | transport | [`01-transport`](c4/03-components/01-transport.md) |
| `F23` | SQLite, auto-migrations and backup | SQLite adapter, migration runner | storage, no domain context | [`02-container`](c4/02-container-diagram.md) |
| `F24` | Offline and quota degradation | Batch orchestrator, cached feed, banner | `feed`, transport | [`feed/use-cases`](ddd/tactical/feed/use-cases.md) §8 |
| `F25` | Batch resume | Batch orchestrator, `TranscriptStage` | `feed`, `analysis` | [`evented-patterns`](ddd/tactical/evented-patterns.md) §5, [`03a`](c4/03-components/03a-transcript-staging.md) |
| `F26` | Pluggable model endpoint | LLM adapter | `analysis` | [`analysis/repository`](ddd/tactical/analysis/repository.md) §5 |
| `F27` | `ytfeed doctor` | Diagnostics surface, all adapters | transport, storage | [`01-transport`](c4/03-components/01-transport.md), [`06-logging`](../00-rules/06-logging-and-observability.md) |
| `F28` | CLI contract: `--json`, exit codes, streams | Exit translator, API client, CLI dispatch | transport | [`01-transport`](c4/03-components/01-transport.md) |

## Coverage summary

| Check | Result |
|-------|--------|
| P0 requirements mapped | **28 of 28** |
| P0 requirements with no component | 0 |
| P0 requirements with no context | 3 (`F23`, `F27`, `F28`). They map to storage or transport, which is valid. |
| Contexts with no P0 requirement | 0 |
| P1 requirements | `F29`…`F34` are out of v1. They do not appear in the P0 matrix. |

## Notes

- `F3` spans three contexts because onboarding is a use case. `identity` coordinates,
  `taste` stores interests and `discovery` proposes them (`G4-6`).
- `F5` spans `taste` and `feed` because the gesture enters through `feed` and the signal
  belongs to `taste`.
- `F23`, `F27` and `F28` are technical capabilities. They are not bounded contexts
  (`A-7`, `context-map.md` §8).
- `F31` is P1. The `stats` projection exists as a P1 component in `feed`.
- A requirement may have an open `OQ`. The mapping does not resolve it. See
  [`open-questions.md`](open-questions.md).

## Gaps

None for P0. The only traceability gap is P1 work that is deliberately out of v1.
