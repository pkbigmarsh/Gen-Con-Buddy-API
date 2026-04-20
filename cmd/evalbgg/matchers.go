package main

// tiebreakByRank returns true if a is a better pick than b by BGG rank.
// Rank 0 means unranked — always loses to a ranked game.
func tiebreakByRank(a, b BGGGame) bool {
	if a.Rank == 0 {
		return false
	}
	if b.Rank == 0 {
		return true
	}
	return a.Rank < b.Rank
}

// tiebreakByRated returns true if a has more users rated than b.
func tiebreakByRated(a, b BGGGame) bool {
	return a.UsersRated > b.UsersRated
}

// exactMatch finds all BGG games whose normalized name exactly matches query,
// then picks the best using tiebreak.
func exactMatch(query string, candidates []BGGGame, tiebreak func(a, b BGGGame) bool) MatchResult {
	var best *BGGGame
	for i := range candidates {
		c := &candidates[i]
		if normalize(c.Name) == query {
			if best == nil || tiebreak(*c, *best) {
				best = c
			}
		}
	}
	if best == nil {
		return MatchResult{}
	}
	return MatchResult{BGGGame: best, Score: 1.0}
}

// bestScoredMatch finds the BGG game with the highest score > 0 using scoreFn,
// using tiebreak when scores are equal.
func bestScoredMatch(query string, candidates []BGGGame, scoreFn func(a, b string) float64, tiebreak func(a, b BGGGame) bool) MatchResult {
	var best *BGGGame
	var bestScore float64
	for i := range candidates {
		c := &candidates[i]
		score := scoreFn(query, normalize(c.Name))
		if score <= 0 {
			continue
		}
		if best == nil || score > bestScore || (score == bestScore && tiebreak(*c, *best)) {
			best = c
			bestScore = score
		}
	}
	if best == nil {
		return MatchResult{}
	}
	return MatchResult{BGGGame: best, Score: bestScore}
}

// filterBySystem returns candidates whose normalized name fuzzy-matches
// gameSystem at or above threshold.
func filterBySystem(gameSystem string, candidates []BGGGame, threshold float64) []BGGGame {
	query := normalize(gameSystem)
	var filtered []BGGGame
	for _, c := range candidates {
		if similarityScore(query, normalize(c.Name)) >= threshold {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// --- Matchers 1–4: System signal ---

type exactSystemRank struct{}

func (exactSystemRank) Name() string { return "exact-system-rank" }
func (exactSystemRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	return exactMatch(normalize(combo.GameSystem), candidates, tiebreakByRank)
}

type fuzzySystemRank struct{}

func (fuzzySystemRank) Name() string { return "fuzzy-system-rank" }
func (fuzzySystemRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	return bestScoredMatch(normalize(combo.GameSystem), candidates, similarityScore, tiebreakByRank)
}

type fuzzySystemRated struct{}

func (fuzzySystemRated) Name() string { return "fuzzy-system-rated" }
func (fuzzySystemRated) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	return bestScoredMatch(normalize(combo.GameSystem), candidates, similarityScore, tiebreakByRated)
}

type tokenSystemRank struct{}

func (tokenSystemRank) Name() string { return "token-system-rank" }
func (tokenSystemRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	return bestScoredMatch(normalize(combo.GameSystem), candidates, jaccardScore, tiebreakByRank)
}

// --- Matchers 5–7: Always edition signal (System + Edition always concatenated) ---

type exactAlwaysEditionRank struct{}

func (exactAlwaysEditionRank) Name() string { return "exact-always-edition-rank" }
func (exactAlwaysEditionRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	query := normalize(combo.GameSystem + " " + combo.RulesEdition)
	return exactMatch(query, candidates, tiebreakByRank)
}

type fuzzyAlwaysEditionRank struct{}

func (fuzzyAlwaysEditionRank) Name() string { return "fuzzy-always-edition-rank" }
func (fuzzyAlwaysEditionRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	query := normalize(combo.GameSystem + " " + combo.RulesEdition)
	return bestScoredMatch(query, candidates, similarityScore, tiebreakByRank)
}

type tokenAlwaysEditionRank struct{}

func (tokenAlwaysEditionRank) Name() string { return "token-always-edition-rank" }
func (tokenAlwaysEditionRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	query := normalize(combo.GameSystem + " " + combo.RulesEdition)
	return bestScoredMatch(query, candidates, jaccardScore, tiebreakByRank)
}

// smartQuery returns System+Edition if edition is informative, else System alone.
func smartQuery(combo GenConCombo) string {
	if isInformativeEdition(combo.RulesEdition) {
		return normalize(combo.GameSystem + " " + combo.RulesEdition)
	}
	return normalize(combo.GameSystem)
}

// --- Matchers 8–11: Smart edition signal ---

type exactSmartEditionRank struct{}

func (exactSmartEditionRank) Name() string { return "exact-smart-edition-rank" }
func (exactSmartEditionRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	return exactMatch(smartQuery(combo), candidates, tiebreakByRank)
}

type fuzzySmartEditionRank struct{}

func (fuzzySmartEditionRank) Name() string { return "fuzzy-smart-edition-rank" }
func (fuzzySmartEditionRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	return bestScoredMatch(smartQuery(combo), candidates, similarityScore, tiebreakByRank)
}

type fuzzySmartEditionRated struct{}

func (fuzzySmartEditionRated) Name() string { return "fuzzy-smart-edition-rated" }
func (fuzzySmartEditionRated) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	return bestScoredMatch(smartQuery(combo), candidates, similarityScore, tiebreakByRated)
}

type tokenSmartEditionRank struct{}

func (tokenSmartEditionRank) Name() string { return "token-smart-edition-rank" }
func (tokenSmartEditionRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	return bestScoredMatch(smartQuery(combo), candidates, jaccardScore, tiebreakByRank)
}

// --- Matcher 12: Pure title signal ---

type fuzzyTitleRank struct{}

func (fuzzyTitleRank) Name() string { return "fuzzy-title-rank" }
func (fuzzyTitleRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	return bestScoredMatch(normalize(combo.RepTitle), candidates, similarityScore, tiebreakByRank)
}

// --- Matchers 13–14: Two-stage, title-derived edition always ---

const systemFilterThreshold = 0.5

type exactTitleDerivedAlwaysRank struct{}

func (exactTitleDerivedAlwaysRank) Name() string { return "exact-title-derived-always-rank" }
func (exactTitleDerivedAlwaysRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	filtered := filterBySystem(combo.GameSystem, candidates, systemFilterThreshold)
	derived := extractTitleDerived(combo.GameSystem, combo.RepTitle)
	if derived == "" || len(filtered) == 0 {
		return MatchResult{}
	}
	query := normalize(combo.GameSystem + " " + derived)
	return exactMatch(query, filtered, tiebreakByRank)
}

type fuzzyTitleDerivedAlwaysRank struct{}

func (fuzzyTitleDerivedAlwaysRank) Name() string { return "fuzzy-title-derived-always-rank" }
func (fuzzyTitleDerivedAlwaysRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	filtered := filterBySystem(combo.GameSystem, candidates, systemFilterThreshold)
	derived := extractTitleDerived(combo.GameSystem, combo.RepTitle)
	if derived == "" || len(filtered) == 0 {
		return MatchResult{}
	}
	query := normalize(combo.GameSystem + " " + derived)
	return bestScoredMatch(query, filtered, similarityScore, tiebreakByRank)
}

// smartTitleDerivedQuery returns system+derived if derived is informative,
// else falls back to the game system alone. Used within the already-filtered candidate set.
func smartTitleDerivedQuery(combo GenConCombo) string {
	derived := extractTitleDerived(combo.GameSystem, combo.RepTitle)
	if derived != "" && isInformativeEdition(derived) {
		return normalize(combo.GameSystem + " " + derived)
	}
	return normalize(combo.GameSystem)
}

// --- Matchers 15–18: Two-stage, title-derived edition smart ---

type exactTitleDerivedSmartRank struct{}

func (exactTitleDerivedSmartRank) Name() string { return "exact-title-derived-smart-rank" }
func (exactTitleDerivedSmartRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	filtered := filterBySystem(combo.GameSystem, candidates, systemFilterThreshold)
	if len(filtered) == 0 {
		return MatchResult{}
	}
	return exactMatch(smartTitleDerivedQuery(combo), filtered, tiebreakByRank)
}

type fuzzyTitleDerivedSmartRank struct{}

func (fuzzyTitleDerivedSmartRank) Name() string { return "fuzzy-title-derived-smart-rank" }
func (fuzzyTitleDerivedSmartRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	filtered := filterBySystem(combo.GameSystem, candidates, systemFilterThreshold)
	if len(filtered) == 0 {
		return MatchResult{}
	}
	return bestScoredMatch(smartTitleDerivedQuery(combo), filtered, similarityScore, tiebreakByRank)
}

type fuzzyTitleDerivedSmartRated struct{}

func (fuzzyTitleDerivedSmartRated) Name() string { return "fuzzy-title-derived-smart-rated" }
func (fuzzyTitleDerivedSmartRated) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	filtered := filterBySystem(combo.GameSystem, candidates, systemFilterThreshold)
	if len(filtered) == 0 {
		return MatchResult{}
	}
	return bestScoredMatch(smartTitleDerivedQuery(combo), filtered, similarityScore, tiebreakByRated)
}

type tokenTitleDerivedSmartRank struct{}

func (tokenTitleDerivedSmartRank) Name() string { return "token-title-derived-smart-rank" }
func (tokenTitleDerivedSmartRank) Match(combo GenConCombo, candidates []BGGGame) MatchResult {
	filtered := filterBySystem(combo.GameSystem, candidates, systemFilterThreshold)
	if len(filtered) == 0 {
		return MatchResult{}
	}
	return bestScoredMatch(smartTitleDerivedQuery(combo), filtered, jaccardScore, tiebreakByRank)
}
