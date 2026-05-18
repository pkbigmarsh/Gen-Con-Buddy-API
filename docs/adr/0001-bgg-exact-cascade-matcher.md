# BGG matching uses a 3-stage exact cascade with no fuzzy fallback

Gen Con organizers fill in free-text GameSystem and RulesEdition fields with no controlled vocabulary. We evaluated 18 matching strategies — varying string comparison method (exact, Levenshtein, Jaccard token-set), query construction (system alone vs. system+edition vs. "smart" edition detection), and tiebreak heuristic — against 865 unique combos and 100+ hand-labeled ground-truth rows. The result was clear: a precision/coverage tradeoff where returning no result is strictly better than returning a wrong one.

We chose a 3-stage exact cascade (`internal/bgg/match.go`):

1. Exact match on a "smart" query against base games — system alone when the edition is a bare ordinal ("1st", "2nd"), system+edition when it's informative ("Prophecy of Kings", "20th Anniversary"). ~72% coverage at ~99% precision.
2. Exact match on a title-derived query against base games — strips system tokens from the most common event title for that combo to infer an edition hint organizers may have omitted.
3. Exact match on the smart query against BGG expansions — runs only when the edition is informative; recovers cases like Wingspan: European Expansion and CATAN: Cities & Knights.
4. No result. Nothing else fires.

## Considered Options

**Fuzzy fallback (Levenshtein similarity):** Tested at every threshold. For the 78 labeled combos that exact matching misses, fuzzy points to the correct game in only 7 cases and introduces 71 wrong answers. Correct and incorrect fuzzy scores are indistinguishable in this range — there is no cutoff that makes it a net positive.

**Consensus vote across all 18 matchers:** Achieves only 36% precision on the labeled (hard) cases — 64 false positives out of 100. Matchers vote among a field of wrong answers and produce a confident-looking winner.

## Consequences

~645 of 865 combos are hydrated (~220 fewer than the consensus approach). Those ~220 stay un-hydrated rather than carrying a wrong BGG ID. All string comparisons run through Unicode NFD diacritic normalization, which recovers cases where Gen Con data omits accents BGG preserves (SHŌBU, Orléans, Yucatán).

The `bgg_mapping.json` file is committed to the repo so every `data update` run is reproducible against the same mapping until someone explicitly reruns `data match-bgg`. A manual override system is deferred but TODOs are in place at the two integration points.
