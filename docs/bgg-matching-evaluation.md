# BGG Matching Evaluation — Findings & Decision

**Date:** April 2026
**Author:** Michail Yasonik

---

## Background

Gen Con event organizers fill in free-text "Game System" and "Rules Edition" fields when submitting events. Mapping those strings to a BGG game ID would let us enrich events with rank, rating, and metadata from BoardGameGeek's dataset of ~176k games. Before integrating anything into the API, we needed to understand how reliably we can actually make that match.

This document records what we tested, what we found, and what we're building as a result.

The full design context lives in the companion specs and plans under `docs/superpowers/`:
- [`specs/2026-04-20-bgg-hydration-eval-design.md`](superpowers/specs/2026-04-20-bgg-hydration-eval-design.md) — original problem framing and matcher design
- [`specs/2026-04-20-bgg-eval-viewer-design.md`](superpowers/specs/2026-04-20-bgg-eval-viewer-design.md) — evaluation viewer design
- [`plans/2026-04-20-bgg-hydration-eval.md`](superpowers/plans/2026-04-20-bgg-hydration-eval.md) — implementation plan for the eval tool
- [`plans/2026-04-20-bgg-eval-viewer.md`](superpowers/plans/2026-04-20-bgg-eval-viewer.md) — implementation plan for the viewer

---

## What we evaluated

We ran 18 candidate matching algorithms across 865 unique (Game System, Rules Edition) combinations extracted from the 2024 Gen Con event catalog. The 18 matchers varied along three axes:

- **String comparison:** exact, fuzzy (Levenshtein), token-set (Jaccard)
- **Query construction:** game system alone; system always concatenated with edition; or system+edition only when the edition is "informative" (not a bare ordinal like "1st")
- **Tiebreak strategy:** BGG overall rank vs. user-rating count

The evaluation artifacts included in this PR are described in [Included Files](#included-files) below.

---

## How we evaluated it

We built an interactive viewer (`bgg_eval_viewer.html`) that ran all 18 matchers side-by-side and surfaced an agreement count — how many of the 18 matchers voted for the same BGG game.

Manual review was focused on the hard cases: rows where matchers disagreed heavily (agreement ≤ 7). For each of those rows, every result was verified against BGG directly and labeled as one of:

- A **correct BGG ID**
- **`BAD_GROUPING`** — the system/edition combo doesn't represent a single coherent game (e.g. events for different games accidentally merged under the same key)
- **`NO_GOOD_RESULT`** — the game exists but isn't in BGG (self-published, prototype, or very new)

This produced **100 ground-truth labels**, **6 bad-grouping flags**, and **13 no-good-result flags** across the 865-row dataset. The labeled export is `bgg_eval_scored.csv`.

---

## Findings

### The consensus vote fails badly where it matters most

On the 100 labeled rows, the consensus approach — take the most-agreed BGG ID across all 18 matchers — achieves only **36% precision**: 64 false positives out of 100. The labeled set is intentionally the hard cases, but these are also the cases that most need a reliable answer.

Representative failures:

| Gen Con input | Consensus picked | Correct answer |
|---|---|---|
| Ticket to Ride / Africa | Ticket to Ride (base) | Ticket to Ride: Heart of Africa |
| Five Tribes / 1st | Four Tribes | Five Tribes: The Djinns of Naqala |
| Deep Rock Galactic / 1st | Terra Galactix | Deep Rock Galactic: The Board Game |
| Steam Up / 1st | Team UP! | Steam Up: A Feast of Dim Sum |
| Incan Gold / 2nd | Mayan Gold | Diamant |

### There is a sharp quality cliff at agreement = 8

| Agreement count | Precision on labeled rows |
|---|---|
| ≥ 8 | **95.5%** |
| ≤ 7 | **45%** — worse than a coin flip |

At low agreement, matchers are all picking different structurally similar-sounding games that happen to have overlapping tokens. The vote produces a winner, but the winner is wrong most of the time.

### Exact matching is the only reliable individual signal

| Matcher | Precision | Coverage |
|---|---|---|
| `exact-smart-edition-rank` | 94.4% | 71.6% of all rows |
| `exact-title-derived-smart-rank` | 100% | 53.4% (subset of above) |
| `exact-always-edition-rank` | 100% | 3.9% |

When exact matching fires, it is right nearly every time. Its only confirmed failure ("The Resistance / 1st" matching the base game instead of Avalon) is a genuine ambiguity where two BGG entries share nearly identical names.

### Fuzzy matching cannot rescue the hard cases

For the 78 labeled rows that exact matching misses, fuzzy matching points to the correct game in only **7** — and introduces **71 wrong answers** at every threshold. There is no score cutoff that makes fuzzy a net-positive fallback. Correct and incorrect results are score-indistinguishable in this range.

### 21 of the 78 missed cases are BGG expansions

The BGG dataset filters out expansion-type entries. Games like Ticket to Ride: Africa, Wingspan: European Expansion, and Twilight Imperium: Prophecy of Kings are listed as expansions on BGG — they can never match against a base-game-only corpus. Only ~5 of those 21 would be recoverable by exact expansion search; the rest use BGG's verbose expansion naming conventions ("Map Collection 3: The Heart of Africa") that don't match how Gen Con organizers write edition names.

### The remaining 57 misses are genuinely hard

Root causes include: the BGG canonical name differs substantially from how organizers write it ("Incan Gold" is "Diamant" on BGG); the BGG entry has a subtitle absent from the Gen Con field ("Awkward Guests" vs. "Awkward Guests: The Walton Case"); or there's a minor typo that exact matching can't bridge.

---

## Decision: Replace consensus vote with a 3-stage exact cascade

### Algorithm

**Stage 1 — Exact match on smart query, base games only.**
Query is game system alone if the edition is generic (e.g. "1st"), or system + edition if the edition is informative (e.g. "20th Anniversary"). This is the primary workhorse: ~72% coverage at ~99% precision.

**Stage 2 — Exact match on title-derived smart query, base games only.**
Strips game system tokens and stopwords from the most common event title to infer an edition hint the organizer may have omitted from the edition field. Adds a small number of additional correct matches at 100% precision.

**Stage 3 — Exact match on smart query, expansions only.**
Runs only when the edition is informative. Recovers ~5 expansion cases (Wingspan: European Expansion, CATAN: Cities & Knights, Space Base: Command Station, etc.) that are currently unreachable.

**Stage 4 — No result.**
Everything that doesn't clear the above stages returns nothing. There is no fuzzy fallback.

### Also: add diacritic normalization

Unicode NFD decomposition in the string normalizer recovers cases where Gen Con data omits accents that BGG preserves: SHŌBU, Orléans, Yucatán, Aerodrome.

### Expected outcome

| | Current (consensus) | New (cascade) |
|---|---|---|
| Coverage | ~800 / 865 hydrated | ~645 / 865 hydrated |
| Precision on hard cases | 36% | 95.5% |
| False positives (labeled set) | 64 / 100 | 1 / 100 |

We trade coverage for correctness. ~215 events return no BGG match and remain un-hydrated, rather than being hydrated with the wrong game. Any consumer of the API gets reliable data rather than plausible-looking noise.

---

## Included files

Files committed alongside this document:

| File | Status | Notes |
|---|---|---|
| `docs/bgg-matching-evaluation.md` | **Keep** | This document |
| `bgg_eval_scored.csv` | **For explanation only** | 100 hand-labeled ground-truth rows. Supports the precision numbers above. Keep until the production matcher is built and validated against it, then delete. |
| `bgg_eval.csv` | **Deletable** | Generated output of the eval tool. Can be reproduced by running `cmd/evalbgg` against the source CSVs. |
| `bgg_eval_viewer.html` | **Deletable** | Generated HTML viewer. Can be reproduced from `cmd/evalbgg/viewer_template.html` + the source CSVs. |
| `cmd/evalbgg/viewer_template.html` | **Keep** | Template source for the viewer. Contains the bad-grouping and no-good-result flag features added during review. |

Source data files (`data.csv`, `boardgames_ranks.csv`) are not committed — they are large and available from their original sources.
