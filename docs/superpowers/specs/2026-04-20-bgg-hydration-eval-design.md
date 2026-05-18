# BGG Hydration Evaluation — Design Spec
*2026-04-20*

## Background

Gen Con events reference board games via two free-text fields: `Game System` (the board game name, e.g. "Axis & Allies") and `Rules Edition` (e.g. "1941", "Global 1942 2nd", or generic ordinals like "1st"). BoardGameGeek (BGG) maintains a ranked dataset of ~176k games with ratings, rank, user counts, and category sub-ranks.

The goal is to enrich Gen Con events with BGG data during init. Before integrating anything, we need to understand the quality of the name matching — how reliably can we map a Gen Con game system to the correct BGG entry?

This spec covers the evaluation tool only. Integration with OpenSearch init is a follow-on decision.

## Problem

The join between Gen Con and BGG is fuzzy:
- `Game System` is free text; capitalization, punctuation, and edition info vary
- One Game System (e.g. "Axis & Allies") can match 20+ BGG entries (different editions, expansions, variants)
- `Rules Edition` is sometimes informative ("1941", "Prison Outbreak") and sometimes not ("1st", "2nd")
- We don't yet know which matching strategy is most accurate

## Scope

- **In scope**: standalone evaluation tool, 18 matching strategies, comparison CSV output, scoring scaffold
- **Out of scope**: OpenSearch integration, changes to existing CLI, production hydration

## Data

**Gen Con CSV** (`data.csv`, ~23,500 events):
- BGM events only: 7,457 events across 751 unique `Game System` values
- 867 unique `(Game System, Rules Edition)` combos
- 172 combos have an informative edition (adds signal beyond bare ordinals)
- 695 combos have a generic edition ("1st", "2nd", etc.) — edition adds no signal

**BGG CSV** (`boardgames_ranks.csv`, ~176k games):
- Fields: `id, name, yearpublished, rank, bayesaverage, average, usersrated, is_expansion, abstracts_rank, cgs_rank, childrensgames_rank, familygames_rank, partygames_rank, strategygames_rank, thematic_rank, wargames_rank`
- All matchers pre-filter `is_expansion = 1` before candidate selection

## Tool

### Binary

Standalone Go binary at `cmd/evalbgg/main.go`. Not wired into the existing command tree. Takes flags:

```
--gencon   path to Gen Con CSV
--bgg      path to BGG CSV
--output   path for output CSV (default: bgg_eval.csv)
```

### Matcher Interface

```go
type Matcher interface {
    Name() string
    Match(gameSystem, rulesEdition, title string, bgg []BGGGame) *BGGGame
}
```

### The 18 Matchers

All matchers exclude expansions (`is_expansion = 1`) before considering candidates.

**Normalization** applied to all inputs before matching: lowercase, strip punctuation, keep `&`.

**"Informative edition"** means the string contains meaningful tokens beyond bare ordinals: 1st, 2nd, 3rd, 4th, 5th, first, second, third, revised, standard, deluxe, basic, classic.

**Title-derived edition** is computed by stripping Game System tokens and the following stopwords from the Title field, leaving only edition-like tokens:
`tournament, finals, final, qualifier, round, semi-final, beginner, beginners, experienced, advanced, mini, open, championship, preliminary, non-qualifier, event, demo, intro, introduction, teach, teaching, with, for, to, the, a, an, of, in, and, by, at, upgraded, components, expansion`

**"Smart" title-derived edition** applies the same informative check — if the derived tokens are all stopwords or empty, fall back to Game System alone.

**Algorithms:**
- `Exact` — case-insensitive normalized string equality
- `Fuzzy` — string similarity ratio (normalized edit distance); outputs raw score alongside match
- `Token` — Jaccard similarity on word token sets; handles word reordering and punctuation differences

**Tiebreakers** (when multiple BGG candidates pass matching):
- `rank` — lowest BGG overall rank wins
- `rated` — highest `usersrated` wins

**Stage-1 filter** (matchers 13–18 only): Game System fuzzy-filtered at threshold 0.5 to cast a wide net before title-derived edition disambiguates.

Rather than fixing a fuzzy threshold upfront, each fuzzy and token matcher outputs the top match AND its raw similarity score. Threshold sensitivity can be analyzed from the output data without re-running.

| # | Name | Signal | Algorithm | Tiebreaker | What it tests |
|---|------|--------|-----------|------------|---------------|
| 1 | `exact-system-rank` | System | Exact | Rank | Baseline |
| 2 | `fuzzy-system-rank` | System | Fuzzy | Rank | Does fuzzy beat exact? |
| 3 | `fuzzy-system-rated` | System | Fuzzy | Rated | Does rated tiebreaker beat rank? |
| 4 | `token-system-rank` | System | Token | Rank | Does token overlap beat fuzzy? |
| 5 | `exact-always-edition-rank` | System + Edition (always) | Exact | Rank | Does always-edition help? |
| 6 | `fuzzy-always-edition-rank` | System + Edition (always) | Fuzzy | Rank | Fuzzy tolerance on noisy always-edition |
| 7 | `token-always-edition-rank` | System + Edition (always) | Token | Rank | Token overlap on noisy always-edition |
| 8 | `exact-smart-edition-rank` | System + Edition (smart) | Exact | Rank | Smart vs. always edition |
| 9 | `fuzzy-smart-edition-rank` | System + Edition (smart) | Fuzzy | Rank | |
| 10 | `fuzzy-smart-edition-rated` | System + Edition (smart) | Fuzzy | Rated | Tiebreaker on smart edition |
| 11 | `token-smart-edition-rank` | System + Edition (smart) | Token | Rank | |
| 12 | `fuzzy-title-rank` | Title (most common per combo) | Fuzzy | Rank | Pure title baseline |
| 13 | `exact-title-derived-always-rank` | System (filter) + Title-derived (always) | Exact | Rank | Title-derived vs. Rules Edition |
| 14 | `fuzzy-title-derived-always-rank` | System (filter) + Title-derived (always) | Fuzzy | Rank | |
| 15 | `exact-title-derived-smart-rank` | System (filter) + Title-derived (smart) | Exact | Rank | Smart title-derived edition |
| 16 | `fuzzy-title-derived-smart-rank` | System (filter) + Title-derived (smart) | Fuzzy | Rank | |
| 17 | `fuzzy-title-derived-smart-rated` | System (filter) + Title-derived (smart) | Fuzzy | Rated | Tiebreaker on smart title-derived |
| 18 | `token-title-derived-smart-rank` | System (filter) + Title-derived (smart) | Token | Rank | Token on cleanest combined signal |

## Output

One row per unique `(Game System, Rules Edition)` combo. For matcher #8 (`fuzzy-title-rank`), which uses the Title field, the most frequently occurring title for that combo is the representative input. This is defensible: for the 79% of combos with only one title it's identical to per-event, and for the rest it reflects the majority of GMs' phrasing. A known limitation: for high-variance combos (e.g. "Ticket to Ride" + "1st" has 13 different titles spanning distinct map variants), matcher #8 likely degenerates to behaving like matcher #2 — the output data will make this visible.

Columns:

| Column | Description |
|--------|-------------|
| `game_system` | Raw Gen Con value |
| `rules_edition` | Raw Gen Con value |
| `edition_informative` | `true`/`false` |
| `event_count` | BGM events sharing this combo |
| `representative_title` | Most common title for this combo (input to matcher #8) |
| `{matcher_name}_id` | BGG id chosen, or empty if no match |
| `{matcher_name}_name` | BGG name chosen, or empty |
| `{matcher_name}_score` | Similarity score (fuzzy matchers only; empty for exact matchers) |
| *(×18 matchers = 54 data columns)* | |
| `agreement_count` | Number of matchers that returned the same top result |
| `consensus_id` | Most common BGG id across matchers (empty if split) |
| `consensus_name` | BGG name for the consensus id |
| `correct_bgg_id` | **Empty — filled in manually during scoring** |

The `correct_bgg_id` column makes the output CSV double as a scoring sheet. Focus manual review on rows with low `agreement_count`.

## Scoring (post-eval)

Metric per matcher: `score = (correct_matches − incorrect_matches) / scored_combos`

Where `scored_combos` is all rows where `correct_bgg_id` has been manually filled in. No-match results (matcher returned empty) count as 0 — neither rewarded nor penalized. This optimizes for net correct matches: a matcher that returns more correct answers scores higher, a matcher that returns wrong answers is penalized, and a matcher that conservatively returns nothing is neutral. Maximizing this score reflects the goal of maximizing recall while subtracting precision errors.

After scoring, the winning strategy informs how BGG data is integrated into OpenSearch init.

## BGG Fields to Include (when integration happens)

All non-expansion fields from the BGG CSV:
- `id, name, yearpublished`
- `rank, bayesaverage, average, usersrated`
- `abstracts_rank, cgs_rank, childrensgames_rank, familygames_rank, partygames_rank, strategygames_rank, thematic_rank, wargames_rank`
