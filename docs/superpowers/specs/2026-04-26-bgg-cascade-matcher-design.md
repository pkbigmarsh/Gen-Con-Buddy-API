# BGG Cascade Matcher — Design Spec
*2026-04-26*

## Background

We evaluated 18 BGG matching strategies and determined that a 3-stage exact cascade gives the best precision/coverage trade-off. See [`docs/bgg-matching-evaluation.md`](../../bgg-matching-evaluation.md) for the full findings.

This spec covers implementing that cascade as a production component: a shared `internal/bgg` package, a new `match-bgg` preprocessing command, and the wiring into `data update`.

## Scope

- **In scope:** `internal/bgg` package, `cmd/data/matchbgg` command, mapping file format, `data update` integration, `BggID` field on events, refactor of `cmd/evalbgg` to import shared code
- **Out of scope:** override system (future work — see TODOs), BGG API calls, UI changes

---

## Architecture

```
boardgames_ranks.csv + data.csv
        ↓
  cmd/data/matchbgg
        ↓
  bgg_mapping.json  ←── (future: bgg_overrides.json applied here)
        ↓
  cmd/data/update  (reads mapping, sets BggID on each event)
        ↓
     OpenSearch
```

The mapping file is committed to the repository. This makes every `data update` run reproducible — the same mapping is applied every time until someone explicitly reruns `match-bgg`.

---

## `internal/bgg/` package

The permanent logic. No eval-specific code here.

### Files

#### `normalize.go`

Moved from `cmd/evalbgg/normalize.go`. One addition: apply Unicode NFD decomposition before the character loop so that accented characters (`ō`, `é`, `á`, `ñ`) strip to their ASCII base before comparison. This recovers cases like SHŌBU, Orléans, Yucatán, Aerodrome where the Gen Con entry and the BGG entry differ only in diacritics.

Exports:
- `func Normalize(s string) string`
- `func IsInformativeEdition(edition string) bool`
- `func ExtractTitleDerived(gameSystem, title string) string`

`cmd/evalbgg/normalize.go` is deleted; evalbgg imports these instead.

#### `score.go`

Moved from `cmd/evalbgg/score.go`. No changes to logic.

Exports:
- `func SimilarityScore(a, b string) float64` (Levenshtein-based)
- `func JaccardScore(a, b string) float64` (token-set)

`cmd/evalbgg/score.go` is deleted; evalbgg imports these instead.

#### `types.go`

Exports:
```go
type BGGGame struct {
    ID            string
    Name          string
    YearPublished string
    IsExpansion   bool
    Rank          int
    UsersRated    int
    BayesAverage  float64
    Average       float64
    AbstractsRank string
    CGSRank       string
    ChildrensRank string
    FamilyRank    string
    PartyRank     string
    StrategyRank  string
    ThematicRank  string
    WarRank       string
}

type Corpus struct {
    BaseGames  []BGGGame
    Expansions []BGGGame
}

type GenConCombo struct {
    GameSystem   string
    RulesEdition string
    RepTitle     string // most common event title for this combo
    EventCount   int
}

type MatchResult struct {
    BGGID string // empty if no match found
    Name  string
}
```

`cmd/evalbgg/types.go` is updated to import `bgg.BGGGame`, `bgg.GenConCombo`, and `bgg.MatchResult` and alias or embed as needed. The evalbgg-specific `Matcher` interface stays in evalbgg.

#### `load.go`

Exports:
- `func LoadCorpus(path string) (Corpus, error)` — reads the BGG CSV and returns all games split into `BaseGames` and `Expansions`. Unlike the old `loadBGG`, expansions are not discarded — they go into the separate slice.
- `func LoadGenConCombos(path string) ([]GenConCombo, error)` — moved from `cmd/evalbgg/load.go`. Reads the Gen Con CSV and returns unique `(GameSystem, RulesEdition)` combos with representative title and event count. Both `cmd/evalbgg` and `cmd/data/matchbgg` import this.

`cmd/evalbgg/load.go` retains only event-loading logic used by the viewer (loading full event rows for the events tab). `loadBGG` is removed; `loadGenConCombos` is removed and replaced with the import.

#### `match.go`

Exports a single function:

```go
func Match(combo GenConCombo, corpus Corpus) MatchResult
```

The cascade:

1. **Exact match on smart query, base games** — query is `Normalize(combo.GameSystem)` if edition is not informative, or `Normalize(combo.GameSystem + " " + combo.RulesEdition)` if it is. Tiebreak: lowest BGG rank.
2. **Exact match on title-derived smart query, base games** — uses `ExtractTitleDerived` and `IsInformativeEdition` on the derived tokens. Falls back to system alone if derived tokens are empty or non-informative.
3. **Exact match on smart query, expansions only** — runs only when `IsInformativeEdition(combo.RulesEdition)` is true. Same query as stage 1.
4. **No result** — returns `MatchResult{}` with empty BGGID.

Override hook (to be implemented separately):

```go
// TODO(overrides): Accept an overrides map as a parameter once the override
// system is built:
//   func Match(combo GenConCombo, corpus Corpus, overrides map[string]string) MatchResult
// Before stage 1, check if (combo.GameSystem + "|" + combo.RulesEdition) has
// an entry in overrides. If so, look it up in the corpus and return it
// immediately, bypassing the cascade. This ensures manually verified results
// survive re-runs of match-bgg without needing to touch the cascade logic.
```

---

## `cmd/data/matchbgg/`

New file: `cmd/data/matchbgg/matchbgg.go`

Registered in `cmd/data/data.go` alongside `UpdateCmd` and `initialize.InitCmd`.

### Flags

| Flag | Default | Description |
|---|---|---|
| `--gencon` / `-g` | (required) | Path to Gen Con events CSV |
| `--bgg` / `-b` | (required) | Path to BGG CSV (`boardgames_ranks.csv`) |
| `--output` / `-o` | `bgg_mapping.json` | Output path for the mapping file |

### Behavior

1. Load `bgg.Corpus` from the BGG CSV.
2. Call `bgg.LoadGenConCombos(genconPath)` to extract unique `(GameSystem, RulesEdition)` combos.
3. For each combo, call `bgg.Match(combo, corpus)`.
4. Write the mapping file.
5. Log a summary: total combos, matched, unmatched.

### Mapping file format

```json
{
  "generated_at": "2026-04-26T12:00:00Z",
  "total_combos": 865,
  "matched": 643,
  "mappings": [
    {
      "game_system": "Axis & Allies",
      "rules_edition": "1st",
      "bgg_id": "98",
      "bgg_name": "Axis & Allies"
    }
  ]
}
```

Only matched combos appear in `mappings`. Unmatched combos are simply absent — `data update` treats any missing combo as no BGG ID.

The file is human-readable and diff-friendly: sorted alphabetically by `game_system` then `rules_edition` so changes between runs are easy to review in git.

Override hook:

```go
// TODO(overrides): Before writing mappings, accept an --overrides flag pointing
// to a JSON file with the same structure as this output. Merge the overrides
// into the results map, replacing any cascade-produced entry for the same
// (game_system, rules_edition) key. Overrides that name a combo not in the
// Gen Con data are silently ignored.
```

---

## `cmd/data/update` changes

New persistent flag on `cmd/data/data.go` (shared across all data subcommands):

```
--bgg-mapping  path to bgg_mapping.json (default: "bgg_mapping.json")
```

In `update.go`, before processing events:

1. Load and parse `bgg_mapping.json` into `map[string]string` keyed by `"game_system|rules_edition"`.
2. If the flag is set but the file doesn't exist, log a warning and continue with an empty map (don't fail the update).
3. After loading each event from the CSV, look up `event.GameSystem + "|" + event.RulesEdition` in the map. If found, set `event.BggID = bggID`. If not found, `BggID` remains empty.

---

## Event struct changes

**`internal/event/types.go`**

Add to `Event` struct:
```go
BggID string `json:"bggId"`
```

No changes to the CSV reader — `BggID` is never in the Gen Con CSV, so it will always be set programmatically from the mapping.

**`gcbapi/event.go`**

Add to `EventAttributes`:
```go
BggID string `json:"bggId"`
```

Update the `ToAPI()` method (or equivalent conversion) to map `event.BggID` to `gcbapi.EventAttributes.BggID`.

**OpenSearch mapping**

Add `bggId` as a `keyword` field in the OpenSearch index mapping (in `cmd/data/initialize`). This allows exact-match filtering by BGG game ID.

---

## `cmd/evalbgg` refactor

Changes are purely mechanical — no behavioral changes to the eval tool.

| Current file | Change |
|---|---|
| `cmd/evalbgg/normalize.go` | Deleted. References updated to `bgg.Normalize`, `bgg.IsInformativeEdition`, `bgg.ExtractTitleDerived`. |
| `cmd/evalbgg/score.go` | Deleted. References updated to `bgg.SimilarityScore`, `bgg.JaccardScore`. |
| `cmd/evalbgg/types.go` | `BGGGame`, `GenConCombo`, `MatchResult` replaced with types from `internal/bgg`. `Matcher` interface stays. |
| `cmd/evalbgg/load.go` | `loadBGG` removed, replaced with `bgg.LoadCorpus`. `loadGenConCombos` removed, replaced with `bgg.LoadGenConCombos`. Retains event-row loading logic for the viewer's events tab. |
| `cmd/evalbgg/matchers.go` | Internal calls to `normalize()`, `similarityScore()`, `jaccardScore()` updated to package-qualified names. No logic changes. |
| `cmd/evalbgg/normalize_test.go` | Updated to test `bgg.Normalize` etc. — moved to `internal/bgg/normalize_test.go`. |
| `cmd/evalbgg/score_test.go` | Moved to `internal/bgg/score_test.go`. |

---

## Testing

- `internal/bgg/normalize_test.go` — covers diacritic stripping, punctuation handling, `IsInformativeEdition`, `ExtractTitleDerived`
- `internal/bgg/match_test.go` — covers all 4 cascade stages using a small in-memory corpus fixture; verifies the expansion stage only fires on informative editions; verifies empty result when nothing matches
- `cmd/evalbgg/matchers_test.go` — existing tests should pass unchanged after the refactor (behavior is identical)

---

## Future work (override system)

The two TODO comments above are the integration points. When the override system is built:

1. `match-bgg` gets `--overrides bgg_overrides.json` flag
2. `bgg.Match` signature gains an `overrides map[string]string` parameter
3. The overrides file has the same structure as the mapping file (or a subset of it)
4. `bgg_eval_scored.csv` is the natural source for populating overrides — the 100 labeled rows already represent the ground truth

No other changes are needed. The architecture is open.
