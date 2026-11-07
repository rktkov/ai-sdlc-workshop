# ytfeed — executive summary

## What it is

`ytfeed` is a local-first command-line tool that builds a small, personal YouTube feed.
It does not host content and does not need an account on a server. One person runs it on
one laptop. The product is for a viewer who wants fewer, better videos and wants to know
why each one is there.

## The problem

YouTube recommends for engagement. The home feed repeats the same channels, rewards strong
titles and thumbnails, and narrows the topics the viewer already likes. The cost is time
lost on weak videos and a slow loss of curiosity. The viewer cannot explain why a video
appeared and cannot correct the system except by watching or ignoring more.

## The solution

`ytfeed` composes a finite feed from three inputs:

1. The viewer's own subscription graph.
2. A small set of stated interests.
3. The viewer's gestures on each card.

For each candidate, it fetches the transcript and judges it against a versioned rubric and
a structured taste profile. The result is a ranked card with a topic, a short summary, a
score per profile axis, a one-line justification and a "why this card" reason. A left or
right swipe becomes a durable signal. That signal changes the profile and the next batch.

## The differentiator

The product is not search and not storage. It is **judgment plus preference**. The closed
loop between the rubric-driven judgment in `analysis` and the gesture-driven profile in
`taste` is the part a competitor cannot copy from an API. Discovery, feed and library
deliver that loop. They are not the product on their own.

## How it works

1. **Onboard.** Connect Google with the read-only YouTube scope. Import subscriptions.
   Choose interests from a seed source.
2. **Discover.** Search YouTube and expand through the subscription graph. Exclude watched,
   saved and recently shown videos. Respect the daily quota. Aim for at least 20%
   micro-bloggers.
3. **Analyze.** Fetch transcript text only. Stage it briefly for crash-safe resume. Send
   it with the profile and rubric to one OpenAI-compatible model. Validate the JSON.
4. **Compose.** Order analyzed cards first. A card never moves backwards past the cursor.
   Stream cards to the terminal.
5. **Learn.** Every gesture commits a signal before the card advances. A left-swipe also
   creates a cold period. The next batch uses the updated profile.
6. **Rate.** Ask for a weekly `-1 / 0 / +1` rating. Store it with the week and the feed IDs.

## Architecture in plain terms

One binary runs as two process roles: a short-lived CLI client and a long-lived local API
server over loopback HTTP (`ADR-0001`). SQLite stores all durable state behind repository
interfaces (`ADR-0002`). The domain has six bounded contexts (`ADR-0003`):

| Context | Role |
|---------|------|
| `identity` | Credential and subscription graph |
| `taste` | Profile, signals, cold periods |
| `discovery` | Candidate pool and quota |
| `analysis` | Transcript judgment |
| `feed` | Batches, sessions, cards and ratings |
| `library` | Saved and downloaded videos |

The LLM is external. Any OpenAI-compatible endpoint is allowed, including a local runtime.
`yt-dlp` is required for the feed and optional for the library. There is no queue, broker,
worker binary or hosted backend. The pipeline is in-process and resumes from database
checkpoints (`ADR-0005`).

## Privacy and trust

The product is local. The database, tokens, library and traces stay under `./data`. The
Google scope is read-only. Transcript text is never stored permanently. It is staged only
between fetch and analysis commit, then deleted (`OQ-05`). The LLM is external. If the
owner points `LLM_BASE_URL` at a remote host, transcript text leaves the machine. The
privacy claim is "not retained by ytfeed", not "never leaves the host" (`SD-5`).

## Scope

In v1: one owner, one local instance, CLI, Google OAuth, discovery, analysis, card feed,
gestures, cold periods, library, weekly rating, `doctor`, `stats` and degradation.

Out of v1: web or mobile client, hosted backend, multi-user server, live streams, Shorts,
curated third-party lists, paid LLM APIs by default and full WCAG compliance.

## Status

The documentation suite is complete for the design gates `G0` to `G6`. There is no code
yet. The next step is the final audit, then the implementation backlog proposal
(`ARCH-601`).

The main risks are the local model quality (`R1`, `R2`), `yt-dlp` fragility (`R4`), quota
limits (`F7`) and the privacy trade-off of transcript staging. Each has a documented
mitigation in the ADRs and the tactical model.

## Where to read more

- Product: [`prd-ytfeed.md`](../10-product/prd-ytfeed.md)
- Design index: [`README.md`](README.md)
- Decisions: [`adr/INDEX.md`](adr/INDEX.md)
