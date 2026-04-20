# BGG Eval Viewer — Design Spec
*2026-04-20*

## Goal

A standalone HTML file that lets you evaluate the output of the BGG hydration evaluation tool (`bgg_eval.csv`). Two purposes in one tool:

1. **Explore matcher behavior** — see which matchers agree, which diverge, and where confidence is low.
2. **Score rows** — fill in `correct_bgg_id` for manual ground-truth scoring, then export an updated CSV.

## Data Sources (embedded in the HTML)

Both datasets are JSON-encoded and embedded as `const` variables in a `<script>` block at the top of the file. No server required.

| Variable | Source | Content |
|----------|--------|---------|
| `EVAL_DATA` | `bgg_eval.csv` | 865 rows — one per (Game System, Rules Edition) combo; all 61 columns |
| `EVENTS_DATA` | `data.csv` (BGM rows only) | ~7,457 events; 20 key fields per event |

`EVENTS_DATA` is indexed by `game_system + "||" + rules_edition` at startup so event lookup per combo is O(1).

**Fields kept from `data.csv`:**
`Game ID`, `Title`, `Short Description`, `Event Type`, `Game System`, `Rules Edition`, `Minimum Players`, `Maximum Players`, `Age Required`, `Experience Required`, `Start Date & Time`, `Duration`, `Location`, `Room Name`, `Table Number`, `GM Names`, `Website`, `Cost $`, `Tickets Available`, `Tournament?`

## Layout

Two-column layout, full viewport height, no scroll on the outer shell.

```
┌─────────────────────┬──────────────────────────────────────────────┐
│   Sidebar (260px)   │  Toolbar                                     │
│                     ├──────────────────────────────────────────────┤
│  • Title            │  Table (scrollable)                          │
│  • Scored progress  │                                              │
│  • Agreement chart  │  ┌─ expanded row ──────────────────────────┐ │
│  • Match rate bars  │  │  Tabs: Matchers & Scoring | Gen Con     │ │
│                     │  │        Events (N)                        │ │
│  [Export CSV]       │  └─────────────────────────────────────────┘ │
└─────────────────────┴──────────────────────────────────────────────┘
```

## Sidebar

**Scored progress** — `X / 865 scored` with a progress bar. Updates live as rows are scored.

**Agreement distribution** — inline SVG bar chart, one bar per agreement count (3–15). Color-coded red→blue left to right.

**Match rate by matcher** — horizontal bar per matcher, color-coded by rate (red <30%, orange 30–79%, green ≥80%). Shows all 18 matchers (scrollable within the block if needed).

**Export CSV button** — downloads the full eval CSV with `correct_bgg_id` column populated from current scores. Uses `localStorage` state merged over the original `EVAL_DATA`.

## Toolbar

- **Search input** — filters table by game system (case-insensitive substring match)
- **Filter dropdown** — presets:
  - Agreement ≤ 7 (uncertain) — default
  - Agreement ≤ 10
  - All rows
  - Unscored only
  - Scored only
- **Row count tag** — "Showing N rows"

Table is sorted by `agreement_count` ascending by default (most uncertain first). Clicking a column header re-sorts.

## Table

Columns: Game System · Edition · Events · Agree · Consensus match · Scored (✓/○)

**Agreement pill** color-coding:
- Red (≤5) — high uncertainty
- Orange (6–9) — moderate
- Green (10–12) — good
- Blue (13–15) — near-unanimous

**Consensus match cell** — shows BGG name and a `↗ #ID` link to `https://boardgamegeek.com/boardgame/{id}` when consensus exists; "split — no consensus" otherwise.

**Scored column** — ✓ (green) when `correct_bgg_id` is set, ○ otherwise.

Clicking a row expands it inline. Only one row is expanded at a time; clicking an already-expanded row collapses it.

## Expanded Row Panel

Two tabs:

### Tab 1: Matchers & Scoring

**Matcher grid** — one card per matcher (18 cards), `auto-fill` grid at `minmax(185px, 1fr)`.

Each card shows:
- Matcher name
- BGG game name (or "no match" in muted italic)
- BGG ID as a link: `↗ BGG #ID` → `https://boardgamegeek.com/boardgame/{id}` (opens in new tab)
- Similarity score (fuzzy/token matchers only)
- **"✓ Use this ID"** button — sets `correct_bgg_id` to this card's ID

Cards where multiple matchers returned the same ID are visually highlighted (purple border + darker background) and show a "N matchers agree" badge.

**Score row** (below the grid):
- Label: `correct_bgg_id`
- Text input — pre-populated when a row was previously scored (from localStorage)
- "clear" button
- "✓ saved" indicator that appears after any change (auto-saves on input)

### Tab 2: Gen Con Events (N)

Scrollable table (max-height 280px) of all BGM events for this combo from `EVENTS_DATA`.

Columns: Game ID · Title · Short Description · Players (min–max) · Date & Time · Duration · Location (Room + Table) · GM Names · Tags

**Tags cell** renders `Tournament?` and `Experience Required` as colored badges.

**Game ID** is displayed in monospace.

No BGG links on this tab (it's raw Gen Con data).

## Persistence

All scoring state lives in `localStorage` under key `bgg-eval-scores`. Format:

```json
{
  "Axis & Allies||Global 1942": "98778",
  "Wingspan||2nd": "266192"
}
```

Key is `game_system + "||" + rules_edition`. State is read on page load and merged into the rendered rows. Auto-saves on every input change (no save button needed).

## Export

Clicking "Export scored CSV" triggers a client-side download of the full CSV. The output is identical to `bgg_eval.csv` except the `correct_bgg_id` column is populated from `localStorage` where available.

Uses the `Blob` + `URL.createObjectURL` pattern. Filename: `bgg_eval_scored.csv`.

## Implementation Notes

- Single `.html` file, no external dependencies (no CDN, no npm). Vanilla JS + inline CSS.
- A Python helper script (`cmd/evalbgg/embed_data.py`) generates the final HTML by reading `bgg_eval.csv` and `data.csv` and injecting the JSON blobs into a template. This keeps the template readable and lets the data be re-embedded after a new eval run.
- The template lives at `cmd/evalbgg/viewer_template.html`. The embed script outputs `bgg_eval_viewer.html` at the repo root (gitignored — regenerated from data).
- Dark theme throughout, matching the mockup.

## Out of Scope

- Matcher accuracy scoring (requires `correct_bgg_id` to be filled in; a follow-on analysis)
- Editing Gen Con event data
- Multi-file or server-based deployment
