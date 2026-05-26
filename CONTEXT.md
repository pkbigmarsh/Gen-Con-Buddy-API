# Gen Con Buddy API

A search API that helps Gen Con attendees discover and plan around convention events. It ingests Gen Con's official event catalog, enriches it with BoardGameGeek data, and exposes it through a filterable, faceted search interface.

## Language

### Events

**Event**:
A single scheduled activity at Gen Con — the core entity this system stores, searches, and tracks.
_Avoid_: Game (legacy term from Gen Con's source data; may appear in field names like `gameId`)

**EventType**:
A fixed category classifying what kind of activity an Event is (e.g. RPG, board game, LARP, seminar). See `internal/event/enums.go` for the full list.

**GameSystem**:
The rules framework an Event runs on (e.g. "Dungeons & Dragons", "Pathfinder"). Not the event title — other games can run on the same system. The event `Title` is the closest field to a game name, but it's freeform.
_Avoid_: Game title, game name

**RulesEdition**:
The version or edition of the GameSystem (e.g. "5th Edition"). Paired with GameSystem for BGG matching.

**Group**:
The organization or publisher hosting the Event at Gen Con.

**SpecialCategory**:
A Gen Con-assigned classification (`Gen Con Presents` or `Premier Event`). Meaning is defined by Gen Con; treat as an opaque label from the source data.

### Ticketing

**Generic Ticket**:
A convention ticket that can be used for any Event in place of a specific event ticket, provided spots are available. Events marked `Registration: Generic` require this ticket type.

**VIG**:
Very Important Gamer — a special badge tier at Gen Con granting access to VIG-only Events.

**TicketsAvailable**:
The live remaining ticket count for an Event, as reported in Gen Con's source data on each pull.

**TotalTickets**:
The original ticket count snapshotted from `TicketsAvailable` at the time of the first data init. Frozen thereafter — does not update on subsequent pulls. Useful for displaying capacity vs. remaining seats.

### Data pipeline

**ChangeLog**:
A batch record of Events created, updated, or deleted in a single data pull from Gen Con. Pulls are scheduled externally in Railway every 6 hours (`0 */6 * * *`). Each ChangeLog entry is the authoritative record of what changed and when.

**Soft-deleted Event**:
An Event that no longer appears in Gen Con's source data. The `deleted` flag is set to `true` in OpenSearch but the document is not removed — it stays in the index so ChangeLog entries can reference it. The search API excludes soft-deleted Events from all results by default.

**BGG (BoardGameGeek)**:
An external data source used to enrich Events with community rank and average rating. Matched to Events via the `GameSystem` + `RulesEdition` pair.

## Relationships

- An **Event** has exactly one **EventType**
- An **Event** is hosted by one **Group**
- A **ChangeLog** entry references disjoint sets of created, updated, and deleted **Events**
- **BGG** data is optionally attached to an **Event** via `GameSystem` + `RulesEdition` matching

## Example dialogue

> **Dev:** "Should I call this field `gameName`?"
> **Domain expert:** "No — use `title`. `GameSystem` is the rules framework, not the game name. An event running D&D 5e homebrew would have `GameSystem: Dungeons & Dragons`, `RulesEdition: 5th Edition`, and its own distinct `Title`."

> **Dev:** "Why does this Event have no tickets available but it's still showing as open registration?"
> **Domain expert:** "Check if it's a Generic Ticket event — those don't reserve specific tickets; attendees use a generic pass at the door if seats remain."

## Convention facts

- Gen Con runs **Wednesday–Sunday** in late July / early August in Indianapolis.
- All event times are in the **America/Indianapolis** timezone.

## Frontend consumer

The primary consumer is **Gen Con Buddy** (`github.com/myasonik/Gen-Con-Buddy`), a React/TypeScript SPA. Key behaviors that affect the API:

- Pagination is 0-indexed in the API, 1-indexed in the UI.
- Ranges are encoded as `[min,max]` (e.g. `cost=[10,50]`).
- Day + time picker inputs are converted client-side into ISO datetime ranges on the `startDateTime` param.
- Multi-field sort is encoded as `field.direction,field.direction`.
- All times are displayed in America/Indianapolis timezone.
- **Staff Picks** — a small hardcoded list of event IDs surfaced on empty search results. Maintained in the frontend, not the API.

## Flagged ambiguities

- **"Game" vs "Event"**: resolved — **Event** is canonical. "Game" appears in source-data field names (`gameId`, `gameSystem`) and Gen Con's own exports; don't use it in new code or domain language.
- **GameSystem vs title**: `GameSystem` is the rules framework, not the event's name. There is no single canonical title field for the game being played — `Title` is the closest but is freeform.

## Open questions

These came up during domain modelling and were not resolved. Revisit when someone has access to Gen Con's official documentation or a knowledgeable attendee.

- **SpecialCategory**: What do "Gen Con Presents" and "Premier Event" mean to attendees in practice? What distinguishes the two?

## Deprecated fields

The following fields were removed from Gen Con's source data in 2024 but remain in the codebase pending cleanup: `AlsoRuns`, `Year`, `MaterialsProvided`, `Prize`, `RulesComplexity`, `OriginalOrder`. Do not build features against them.

The `ANI` (Anime Activities) EventType was removed from Gen Con's catalog in 2026.
