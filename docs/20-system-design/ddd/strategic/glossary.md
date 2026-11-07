# DDD strategic — ubiquitous language glossary

## Metadata

| Field | Value |
|-------|-------|
| **Domain** | `ytfeed` — personal YouTube feed composition |
| **Document** | Ubiquitous language glossary, per bounded context |
| **Backlog** | `ARCH-204` |
| **Version** | 1.0 |
| **Date** | 2026-11-03 |
| **Owner** | Oleg K |
| **Status** | Confirmed at gate `G4` (2026-11-03) |
| **Sources** | [`prd-ytfeed.md`](../../../10-product/prd-ytfeed.md), [`02-architecture.md`](../../../00-rules/02-architecture.md), [`04-data-persistence.md`](../../../00-rules/04-data-persistence.md), [`08-cli-and-ux.md`](../../../00-rules/08-cli-and-ux.md) |
| **Siblings** | [`context-map.md`](context-map.md), [`subdomain-map.md`](subdomain-map.md) |
| **Next** | `ARCH-301`…`ARCH-307` (tactical design) |

## 0. Admission rule

A term enters this glossary only if it appears in one of three places.

1. the PRD (`F*`, `US-*`, `Q*`, or a named section), or
2. `docs/00-rules/` (a rule ID), or
3. a **resolved** owner decision recorded in an ADR.

A term whose meaning comes from an unresolved `OQ-*` is marked **provisional** and cites the
OQ. Architecture and implementation vocabulary is listed separately in §4 and is **not**
ubiquitous language. This rule exists because the strategic-design prompt forbids invented
domain language. If the owner does not say the word, the model must not either.

**Definition style.** A definition states what the term *is*, not how it is implemented. A
range or default is included only when the PRD or a rule fixes it. Otherwise it is labelled
an implementation detail and belongs to the tactical phase.

---

## 1. Product-level terms

Terms used across contexts. A context may specialise them (§2) or use them with a different
meaning (§3).

| Term | Definition | Source | Notes |
|------|------------|--------|-------|
| **Owner** | The human user of one local instance; the only actor in v1 | PRD §2, `SD-7` | Distinct from the `user` row (§3.15) |
| **User** | The persisted principal that owns rows; every row carries `user_id` | `D-22` | In v1, one owner = one user |
| **Instance** | One local installation with its own data directory and database | PRD §2, `SD-7` | Friends run independent instances; no shared state |
| **Feed** | The finite set of cards composed for the owner to browse | `F13`, `F14` | See the `feed` / `batch` distinction (§3.11) |
| **Batch** | One composition-and-analysis run that produces at most 20 cards | `F13`, `F14`, PRD §6.2 | A feed may span more than one batch in a session |
| **Card** | One video presented for judgment, with metadata, analysis and state | `F13`, `CLI-13` | Created from a candidate (§3.12) |
| **Candidate** | A video proposed by discovery before selection and analysis | `F8`, `A-14` | Not yet a card |
| **Session** | The interactive period opened when the UI starts and closed after inactivity | `F18`, `CLI-21` | Idle timeout default 1h (`R-18`) |
| **Gesture** | Any owner action on a card: left, right, up, down, save, remove | `F17`, PRD §6.2 | A swipe is one kind of gesture (§1) |
| **Swipe** | A left or right gesture | `US-4`, `US-6`, `F16` | Left rejects; right opens YouTube |
| **Analysis** | The act of judging a transcript against the rubric and profile | `F10` | Also names the result; see §3.16 |
| **Transcript** | The text of a video's speech, fetched as text only | `F9` | Never video or audio |
| **Rubric** | The versioned artifact that defines how a transcript is judged | `A-19`, `F10`, `F12` | Content, not code |
| **Topic** | The content label of a card, produced by analysis | `F10`, `CLI-13` | Distinct from interest, axis and topic tag (§3.13) |
| **Interest** | A content seed the owner chooses during onboarding, ~5 by default | `F3`, `US-2` | Count is provisional (`OQ-08`); vocabulary resolved at `G4` (`OQ-19`) |
| **Taste profile** | The structured preference: named axes, weights, examples and a note | `F4`, `A-20` | Never a prompt string (`A-20`) |
| **Cold period** | A persisted, time-bounded suppression of a topic or channel | `F19`, `D-21` | A record, not a flag |
| **Signal** | The durable record of a gesture's effect on preference | `F5`, `D-20` | Positive or negative; the product's only asset |
| **Quota** | The YouTube Data API daily budget and its consumption | `F7` | 10,000 units/day in the PRD |
| **Micro-blogger** | A channel below the subscriber threshold | `F6`, `Q4` | Default < 100k, configurable |
| **New territory** | A card flag meaning "outside the owner's current focus" | `F10`, `UG-3` | Produced by analysis, consumed by the feed |
| **Library** | The collection of saved and downloaded videos with local playback | `F20` | See `saved` / `downloaded` (§3.9) |
| **Weekly rating** | The −1 / 0 / +1 quality rating asked after 7 days | `F21`, `CLI-28` | Asked on session open, never mid-feed |
| **Degraded** | Completed with reduced functionality; the CLI exit code is 6 | `CLI-6`, `F24` | Not an error |
| **Resume** | Continue an interrupted session or batch | `F18`, `F25` | Two different meanings; qualify (§3.8) |
| **Seed source** | The source used to propose interests during onboarding | `F3`, `CLI-26` | `youtube-trending`, `external-trends`, `subscriptions-only` |
| **Data directory** | The single directory that holds all local state | `C-15`, `US-E5` | `$YTFEED_DATA_DIR` → `./data` |
| **Exit code** | The stable public CLI contract: 0–6 and 130 | `F28`, `CLI-6` | Changing one is a breaking change |
| **Doctor** | The offline diagnostics command that reports state with secrets redacted | `F27`, `C-14` | Returns 0 whenever it can report |
| **Stats** | The read-only statistics command over persisted facts | `F31`, `L-15` | P1 |

---

## 2. Context glossaries

### 2.1 `identity` — credentials and the subscription graph

| Term | Definition | Source | Notes |
|------|------------|--------|-------|
| **Consent** | The owner's explicit grant of the read-only YouTube scope | `F1` | Given in a browser, exchanged for a token |
| **OAuth scope** | The single permission requested: `youtube.readonly` | `F1` | Not the cold-period scope (§3.4) |
| **OAuth token** | The local credential file written after consent | `F1`, `C-17` | `0600`; never logged (`C-2`, `L-10`) |
| **Refresh** | Obtaining a new access token without a new consent | `F1` | Single-flight on a 401 |
| **Revoked grant** | Google has invalidated the consent; the owner must re-authenticate | `F1` | `ErrUnauthorized`, exit 2 |
| **Subscription** | The owner's relationship to a channel | `F2` | Unique by channel ID |
| **Channel** | A YouTube channel with an ID, title, subscriber count and topic tags | `F2` | The natural key is `channel_id` |
| **Subscriber count** | The channel's audience size | `F2`, `F6` | Used to classify micro-bloggers |
| **Topic tag** | YouTube-provided channel metadata imported with a subscription | `F2` | External classification, **not** the product's `topic` (§3.13) |
| **Import** | The paginated, idempotent upsert of the owner's subscriptions | `F2`, `D-19` | A crash re-runs from page 1 |
| **Seed source** | The onboarding choice that determines how interests are proposed | `F3`, `CLI-26` | Product-level term (§1) |

### 2.2 `taste` — the accumulating preference

| Term | Definition | Source | Notes |
|------|------------|--------|-------|
| **Axis** | A named evaluative dimension of the profile | `F4`, `A-20` | Example from the component doc: `depth-over-hype` |
| **Axis weight** | How much an axis matters when judging a card | `F4`, `A-20` | Range and default are tactical; the component doc proposes `[0,1]`, 0.5 |
| **Positive example** | Content that exemplifies the high end of an axis | `F4` | At least one per axis |
| **Negative example** | Content that exemplifies the low end of an axis | `F4` | At least one per axis |
| **Profile note** | Optional free text on the profile | `F4` | Not a replacement for structured axes |
| **Interest** | A user-selected content seed stored with the profile | `F3`, `OQ-19` (resolved at `G4`) | Default 5, range 1–10 (`OQ-08`) |
| **Signal** | One durable record per gesture that affects preference | `F5`, `D-20` | Written before the card advances |
| **Positive signal** | A signal that raises preference; at minimum a right-swipe | `F5` | A save is also a gesture (`F17`); its weight is tactical |
| **Negative signal** | A signal that lowers preference; at minimum a left-swipe | `F5` | Accompanied by a rejection reason |
| **Cold period** | A record that suppresses a topic or channel until `expires_at` | `F19`, `D-21` | Expired records are retained for audit |
| **Mute** | The owner-facing action and outcome that creates or refreshes a cold period | PRD §6.3, `CLI-18` | Not a synonym for the record (§3.3) |
| **Cold-period scope** | What the cold period targets: `topic` or `channel` | `F19`, `CLI-20a` | Derived from the rejection reason |
| **Rejection reason** | Why the owner rejected a card: `marketing`, `ai-slop`, `value-misaligned`, `other` | `CLI-20a`, `D-21` | Not the analysis reason (§3.5) |
| **Expiry** (`expires_at`) | The instant after which a cold period stops filtering | `F19` | The row itself remains |
| **Refresh** | Re-muting an already-muted target updates its expiry instead of erroring | `CLI-18` | Not token refresh (§3.17) |
| **Weight update** | The bounded nudge applied to the axes a card scored | `F5`, `OQ-18` (provisional) | ±0.05, clamped, versioned with the rubric — provisional default |

### 2.3 `discovery` — the bounded candidate set

| Term | Definition | Source | Notes |
|------|------------|--------|-------|
| **Query** | A keyword search derived from an interest | `F6` | Each interest becomes one or more queries |
| **Subscription-graph expansion** | Finding channels related to the owner's subscriptions | `F6` | The second candidate source |
| **Candidate** | A video proposed before selection and analysis | `F8`, `A-14` | Product-level term (§1) |
| **Candidate pool** | The persisted set of unique, source-tagged candidates | `A-7` | Owned by `discovery` |
| **Dedupe** | Excluding watched, saved, shown in the last 7 days, or duplicate IDs | `F8` | The rule is `discovery`'s; the data is supplied by `feed` (`context-map.md` §6.4) |
| **Quota** | The YouTube Data API daily budget and its consumption | `F7` | Product-level term (§1) |
| **Quota ledger** | The persisted, per-day record of quota units consumed | `F7`, PRD §9 | Warns at 80%, refuses at 100% |
| **Accounting day** | The day boundary the quota ledger uses | `F7`, `OQ-02` (provisional) | UTC by the PRD; YouTube resets at midnight Pacific |
| **Micro-blogger** | A channel below the subscriber threshold | `F6`, `Q4` | Product-level term (§1) |
| **Micro-blogger share** | The fraction of candidates from channels below the threshold | `F6` | Target ≥ 20% |
| **Micro-blogger floor** | The 20% target treated as a goal | `F6`, `OQ-09` (provisional) | A target, not a hard gate; a shortfall is displayed and recorded |

### 2.4 `analysis` — the explicable judgment

| Term | Definition | Source | Notes |
|------|------------|--------|-------|
| **Transcript** | The text of a video's speech, fetched as text only | `F9` | Product-level term (§1) |
| **Staging** | Keeping a transcript in a restricted temporary file between fetch and analysis commit | `F9` (amended), `OQ-05` (resolved) | Deleted on commit; TTL-swept |
| **Rubric** | The versioned artifact that defines how a transcript is judged | `A-19`, `F10` | Product-level term (§1) |
| **Filter rubric** | The same artifact, named for its purpose: filtering the feed | `A-19`, `BG-3` | Not a second artifact |
| **Rubric version** | The identifier of the rubric that produced an analysis | `F12`, `A-19` | Stored with every result; old results keep theirs |
| **Analysis result** | The structured output of judging one transcript | `F10` | The persisted document, not the process |
| **Axis score** | The score for one profile axis, with a one-line justification | `F10`, `A-20` | One per axis |
| **Justification** | The one-line reason for an axis score | `A-20` | A score without one is not trusted |
| **Marketing score** | A `[0,1]` judgment of promotional intent | `F10` | Not a rejection reason |
| **AI-slop score** | A `[0,1]` judgment of low-substance generated content | `F10` | Not the `ai-slop` rejection reason (§3.7) |
| **Similarity signal** | "similar to X" or "new territory" | `F10`, `CLI-13` | |
| **New territory** | The flag that the card is outside the current focus | `F10`, `UG-3` | Product-level term (§1) |
| **Why-this-card** | The one-sentence reason the card is in the feed | `F10`, `CLI-13` | |
| **Analysis state** | The card's analysis status: `analyzed`, `pending` or `failed` | `F11`, `E-12` | `pending` is a valid state (`E-12`) |
| **Analysis reason** | The cause of a failed analysis: `no_transcript`, `analysis_slow` or `invalid_model_output` | `F11` | Not the rejection reason (§3.5) |
| **`no_transcript`** | No transcript was available for the video | `F11`, PRD §6.5 | The card is still shown |
| **`analysis_slow`** | The model exceeded the per-request deadline | `F11`, PRD §6.5 | The batch continues |
| **`invalid_model_output`** | The model output failed schema validation | `F11`, `A-12b` | Never coerced into a plausible card |
| **Model** | The LLM runtime and model selected for analysis | `F26` | Any OpenAI-compatible endpoint |
| **TL;DR** | The one-line summary of the video | `F10`, `CLI-13` | |
| **Topic** | The content label assigned to the video | `F10`, `CLI-13` | Product-level term (§1) |

### 2.5 `feed` — the daily loop

| Term | Definition | Source | Notes |
|------|------------|--------|-------|
| **Card** | One video presented for judgment | `F13`, `CLI-13` | Product-level term (§1) |
| **Batch** | One composition-and-analysis run, at most 20 cards | `F13`, `F14` | Product-level term (§1) |
| **Feed** | The finite set of cards the owner browses | `F13` | Product-level term (§1) |
| **Position** | A card's slot within a batch | `CLI-14a` ("current position") | Assigned at composition |
| **Cursor** | The owner's current position in the batch | `F14`, `CLI-14a` | Never moves backwards over a card that was not left-swiped |
| **Analyzed first** | The ordering rule: analyzed cards precede unanalyzed; ties by score descending, then discovery order | `F14`, `CLI-14a`, `R-15` | Deterministic within a state |
| **Analyzed (late)** | A card that finished analysis after the cursor passed its slot; it stays in place | `OQ-15` (provisional) | Visible as analyzed, not moved |
| **Advance** | Moving the cursor to the next card after a committed gesture | `D-20`, `F17` | Only after the server acknowledges |
| **Session** | The interactive period, resumed if younger than the idle timeout | `F18`, `CLI-21` | Product-level term (§1) |
| **Session resumed** | The state and message when an open session is reused | `F18`, `CLI-21` | Navigation history is not restored across a resume (`OQ-14`) |
| **Idle timeout** | 1 hour of inactivity closes the session | `F18`, `R-18` | `SESSION_IDLE_TIMEOUT` |
| **Gesture** | An owner action on a card | `F17` | Product-level term (§1) |
| **Reason picker** | The keyboard picker shown after a left-swipe | `F16`, `CLI-20` | Keys 1–4; Esc records `other` |
| **Down** | Advance without removing the card | `F17` | "Not now" is not destructive |
| **Up** | Return to the previous card that was not left-swiped | `F17` | Does not cross a resume boundary (`OQ-14`) |
| **Tap** | Download the card's video to the library | `F17` | Key binding is provisional (`OQ-03`) |
| **Double-tap** | Remove the card's video from the library | `F17` | Key binding is provisional (`OQ-03`) |
| **Weekly rating** | The −1 / 0 / +1 quality rating | `F21`, `CLI-28` | Product-level term (§1) |
| **Unrated** | The recorded outcome when the weekly prompt is dismissed | `F21` | The prompt is not shown again for 7 days |
| **Week number** | The week the rating covers | `F21` | Stored with the rating |
| **Feed IDs** | The identifiers of the feeds shown in the rated week | `F21` | Read as the batch IDs; see §3.11 |
| **Cached feed** | Previously composed cards served when discovery is unavailable | `F24`, PRD §6.5 | The library remains available too |
| **Degraded** | Completed with reduced functionality | `CLI-6`, `F24` | Product-level term (§1) |

### 2.6 `library` — saved and downloaded videos

| Term | Definition | Source | Notes |
|------|------------|--------|-------|
| **Library** | The collection of saved and downloaded videos | `F20` | Product-level term (§1) |
| **Saved** | A library entry that exists without a local file | `OQ-04` (provisional) | The entry is durable; the file is not |
| **Downloaded** | A library entry whose local file is present | `F20` | File lives under `./data/library/` |
| **Bookmark** | `US-8`'s word for keeping a video for later | `US-8` | Read as `saved`; see §3.9 |
| **Download** | Fetching the media file via `yt-dlp` on an explicit request | `F17`, `F20` | Never automatic during analysis |
| **Remove** | Deleting a library entry and its file | `F17`, `CLI-19` | Removing from the library is not destructive to the database |
| **Playback** | Opening the local file in the system player | `F20`, `CLI-23` | `PLAYER_CMD` or the OS default |

---

## 3. Terms that differ by context

This is the part of the glossary that a tactical model must read first. Each entry names a
word that appears in more than one place with more than one meaning. It states the resolution
that the documentation will use. A provisional resolution cites the OQ.

### 3.1 `token`

- **OAuth token** — the Google credential in `./data/token`, `0600` (`F1`, `C-17`).
- **API bearer token** — the local HTTP credential in `./data/api-token`, `0600` (`A-10b`).

**Resolution:** always qualify. The bearer token is a transport control, not a domain
concept. It does not appear in the domain glossary (§4).

### 3.2 `gesture` vs `signal`

- **Gesture** — the owner's action (left, right, up, down, save, remove) (`F17`).
- **Signal** — the durable preference record a gesture may produce (`F5`, `D-20`).

Not every gesture produces a signal. `down` and `up` are navigation (`F17`). A left-swipe
produces a signal and a cold period. A right-swipe produces a signal. A save is a gesture
whose signal weight is tactical.

**Resolution:** `gesture` belongs to `feed` and transport. `signal` belongs to `taste`. Never
write "store a gesture" when "store a signal" is meant.

### 3.3 `mute` vs `cold period`

- **Mute** — what the owner does and sees: "mute channel X for 24h" (PRD §6.3, `CLI-18`).
- **Cold period** — the persisted row with scope, target, reason, `created_at`, `expires_at`
  (`F19`, `D-21`).

A mute creates or refreshes exactly one cold period (`CLI-18`, `F19`).

**Resolution:** use `cold period` in the model and the schema. Use `mute` in user-facing
copy. They are related by an action, not by identity.

### 3.4 `scope`

- **OAuth scope** — the single Google permission `youtube.readonly` (`F1`).
- **Cold-period scope** — what a suppression targets: `topic` or `channel` (`F19`,
  `CLI-20a`).

**Resolution:** always qualify. A bare "scope" is a defect in a design document.

### 3.5 `reason`

- **Rejection reason** — why the owner rejected a card: `marketing`, `ai-slop`,
  `value-misaligned`, `other` (`CLI-20a`, `D-21`).
- **Analysis reason** — why analysis failed: `no_transcript`, `analysis_slow`,
  `invalid_model_output` (`F11`).

**Resolution:** `rejection reason` and `analysis reason`. They are different columns in
different contexts, with different enums and different lifetimes.

### 3.6 `rubric` vs `axis weight` (`OQ-20`)

- **Rubric** — the versioned judgment artifact, owned as content (`A-19`), applied by
  `analysis` (`F10`, `F12`).
- **Axis weight** — how much a profile axis matters (`F4`, `A-20`).

`A-7` gives `taste` "rubric weights", which could be read as rubric ownership. That would
collide with `A-19` (the rubric is a content artifact) and with `analysis` (rubric
application).

**Resolution (confirmed at `G4`, 2026-11-03, `OQ-20`):** `taste` owns **axis weights**.
`analysis` owns rubric application. The rubric artifact is owned by the product
documentation (`A-19`). "Rubric weights" in `A-7` is shorthand for axis weights.

### 3.7 `ai-slop`

- **`ai-slop`** — a rejection reason (`CLI-20a`).
- **AI-slop score** — a `[0,1]` attribute in the analysis result (`F10`).

**Resolution:** the hyphenated enum is the reason. "AI-slop score" is the numeric attribute.
A card can score high on AI-slop and still be accepted. A card can be rejected as `ai-slop`
with any score. The reason is the owner's judgment. The score is the model's.

### 3.8 `resume`

- **Session resume** — reusing a session younger than the idle timeout (`F18`, `CLI-21`).
- **Batch resume** — continuing an interrupted batch from its last checkpoint (`F25`).

**Resolution:** always qualify. They have different durability rules. Session history is
in-memory and dies (`OQ-14`). Batch state is persisted (`F25`).

### 3.9 `saved`, `bookmark`, `downloaded`

- **Bookmark** — `US-8`'s word for keeping a video for later.
- **Saved** — the library entry without a file (`OQ-04`, provisional).
- **Downloaded** — the library entry with a local file (`F20`).

**Resolution:** a `library` entry with two states, `saved` and `downloaded`. `bookmark` is a
deprecated synonym for `saved`. This is a vocabulary decision, not a rule edit.

### 3.10 `unanalyzed` vs `pending` / `failed`

- **Unanalyzed** — the presentation label in a card: `[unanalyzed: no transcript]`
  (`CLI-14`).
- **Pending** — a card whose analysis has not finished. A valid state (`E-12`).
- **Failed** — a card whose analysis terminated with a reason (`F11`).

**Resolution:** `unanalyzed` is a **presentation** word, not a stored state. The stored state
is `pending`, `analyzed` or `failed`. The reason is separate. A UI that maps `failed` to
"unanalyzed" is correct. A schema that stores `unanalyzed` is not.

### 3.11 `feed` vs `batch`

- **Batch** — one composition-and-analysis run, at most 20 cards (`F13`, `F14`).
- **Feed** — the set of cards the owner browses (`F13`).

`F21` stores "the IDs of feeds shown that week". There is no separate `feed` entity in the
data model.

**Resolution:** read `feed IDs` in `F21` as the **batch IDs** whose cards were shown that
week. If a later design introduces a feed entity, this glossary entry must be revised.

### 3.12 `candidate` vs `card`

- **Candidate** — a video proposed by discovery (`F8`, `A-14`).
- **Card** — a selected video presented for judgment (`F13`).

**Resolution:** a card is created from a candidate. The words are not interchangeable. A
candidate that is never selected is still a candidate.

### 3.13 `interest`, `topic`, `topic tag`, `axis` (`OQ-19`)

- **Interest** — a user-selected content seed, ~5 by default (`F3`, `US-2`).
- **Topic** — the content label produced by analysis (`F10`, `CLI-13`), and the target of an
  `ai-slop` or `other` cold period (`CLI-20a`, `F19`).
- **Topic tag** — YouTube-provided channel metadata imported with a subscription (`F2`).
- **Axis** — an evaluative dimension of the taste profile (`F4`, `A-20`).

These are four different concepts. The PRD does not state their relationship.

**Resolution (confirmed at `G4`, 2026-11-03, `OQ-19`):**

| Concept | Owner | Meaning | Shares a namespace with |
|---------|-------|---------|--------------------------|
| **Axis** | `taste` | An evaluative dimension, e.g. depth over hype | Nothing. It is not content. |
| **Interest** | `taste` | A content seed the owner chose | **Topic**: one slug namespace |
| **Topic** | `analysis` | The content label of a card | **Interest**: same slugs |
| **Topic tag** | `identity` | YouTube's channel classification | Nothing. External metadata only. |

An interest and a topic share one normalized slug namespace, so muting a topic can suppress
queries for the matching interest. A topic tag is YouTube metadata and is not assumed to use
the same taxonomy. Mapping it is a `discovery` concern. Axes are independent of all three.

### 3.14 `watch list` vs `library`

- **Watch list** — the word in `CLI-19` ("removing from the watch list is not destructive").
- **Library** — the word in `F20`, `F17` and `A-7`.

**Resolution:** `library` is canonical. `watch list` in `CLI-19` is read as the library. No
third concept is implied, and no rule edit is proposed.

### 3.15 `owner` vs `user`

- **Owner** — the human actor (`PRD §2`).
- **User** — the persisted principal key (`D-22`).

**Resolution:** use `owner` in prose about the person. Use `user` only for the key and the
data model. In v1 they are one-to-one, but the words are not interchangeable.

### 3.16 `analysis` — process and result

- **Analysis** — the act of judging a transcript (`F10`).
- **Analysis result** — the persisted document produced by the act (`F10`, `F12`).

**Resolution:** use `analysis result` when referring to the stored object, especially in
contracts. A sentence like "the analysis is stored" should read "the analysis result is
stored".

### 3.17 `refresh`

- **Token refresh** — obtaining a new Google access token (`F1`).
- **Cold-period refresh** — re-muting a target updates `expires_at` (`CLI-18`).

**Resolution:** always qualify.

### 3.18 `position` vs `cursor`

- **Position** — a card's slot in a batch (`CLI-14a`).
- **Cursor** — the owner's current slot (`F14`, `CLI-14a`).

**Resolution:** `position` is a property of a card. `cursor` is a property of the session.
A card can have a position ahead of the cursor. That is exactly the late-analysis case
(`OQ-15`).

---

## 4. Terms deliberately **not** admitted

These words appear in architecture documents and will appear in code. They are not
ubiquitous language. The owner does not use them. Admitting them would let the
implementation dictate the model.

| Term | Where it belongs | Why it is excluded |
|------|------------------|--------------------|
| **Lease**, **heartbeat** | Transport / `feed` concurrency | Mechanism for `A-11`; not a domain concept |
| **SSE**, **event**, **`card.upsert`**, **`batch.done`** | Transport streaming | Wire protocol (`OQ-17`), not domain language |
| **API bearer token** | Transport security | `A-10b` control; not a domain object |
| **Source tag**, **`search` / `graph` / `trending`** | `discovery` implementation | Component-level detail; no PRD term |
| **Exclusion set** | `discovery` contract input | A design device for `F8`; not in the PRD |
| **Snapshot** (`ProfileSnapshot`, `CandidateSelection`) | Contract shape | A transport/document shape, not a domain noun |
| **Read model**, **projection** | Architecture | Implementation pattern; the domain fact is the signal or the card |
| **Port**, **adapter**, **ACL**, **OHS**, **C/S**, **PL** | This document's analysis | Pattern names for architects, not domain experts |
| **Worker pool**, **prompt builder**, **validator** | `analysis` implementation | Components, not concepts |
| **`model_id`** | `analysis` contract field | Operational provenance; `F26` gives "model", which is admitted |
| **Staged hash**, **`.part`**, **atomic rename** | Transcript staging implementation | Mechanism behind `F9`/`OQ-05`; the domain term is `staging` |
| **Schema-constrained JSON** | `A-12b` | A contract format, not a domain term |

**Not excluded by association.** `Cursor` is admitted because `F14` uses the word, even
though it sounds like an implementation detail. `Quota ledger` and `rubric version` are
admitted for the same reason: the PRD or a rule names them.

**Rule:** if a term appears only in a component document, it stays out of §1–§3. If a term
appears in the PRD or a rule, it is admitted even when it sounds technical.

---

## 5. Naming rules that follow from the glossary

1. **Prefer the PRD's word.** Do not rename `cold period` to `mute record`, or `card` to
   `item`.
2. **Qualify the six ambiguous words:** `token`, `scope`, `reason`, `resume`, `refresh`,
   `position`. A bare use is a review defect.
3. **Do not normalise enum spellings across contexts.** Rejection reasons are hyphenated
   (`ai-slop`, `value-misaligned`). Analysis reasons are snake_case (`no_transcript`). They
   are different enums in different contexts. The schema will keep them distinct.
4. **`unanalyzed` never enters the schema.** It is a UI label (§3.10).
5. **`stop` and `start` are not domain verbs here.** The domain has `compose`, `analyze`,
   `reject`, `save`, `download`, `rate` and `resume`. All come from the PRD or rules.
6. **A new term needs a source.** If the PRD and the rules do not contain it, it goes to §4
   or to the open-questions register, not to §1–§3.

---

## 6. Open questions

`OQ-19` and `OQ-20` were resolved at gate `G4` on 2026-11-03 and are no longer open. §3.13
and §3.6 carry the answers. The remaining entries still affect the glossary.

| ID | Effect on the glossary |
|----|------------------------|
| `OQ-08` | Interest count: default 5, range 1–10 |
| `OQ-09` | Micro-blogger floor: target, not gate |
| `OQ-04` | `saved` vs `downloaded` |
| `OQ-14` | Whether navigation history crosses a session resume |
| `OQ-15` | `analyzed (late)` placement |
| `OQ-18` | The weight-update rule and its magnitude |
| `OQ-03` | Tap / double-tap key bindings (the gesture names are stable; the binding is not) |

## 7. Next step

Gate `G4` confirms the vocabulary together with the boundaries (`context-map.md` §10).
Phase 3 then uses these terms as the only allowed nouns in aggregates, value objects and
events (`ARCH-301`…`ARCH-307`).
