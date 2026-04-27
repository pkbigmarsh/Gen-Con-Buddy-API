package bgg

// Match runs the 3-stage exact cascade against the given corpus and returns the
// best confident result, or an empty MatchResult if nothing clears any stage.
//
// TODO(overrides): check overrides map before the cascade; see design spec at
// docs/superpowers/specs/2026-04-26-bgg-cascade-matcher-design.md
func Match(combo GenConCombo, corpus Corpus) MatchResult {
	query := smartQuery(combo)
	if r := exactBest(query, corpus.BaseGames); r.BGGID != "" {
		return r
	}

	if r := exactBest(smartTitleDerivedQuery(combo), corpus.BaseGames); r.BGGID != "" {
		return r
	}

	// Only runs when edition is informative enough to distinguish an expansion from its base game.
	if IsInformativeEdition(combo.RulesEdition) {
		if r := exactBest(query, corpus.Expansions); r.BGGID != "" {
			return r
		}
	}

	return MatchResult{}
}

// smartQuery returns the normalized search query for a combo:
// system+edition if the edition is informative, system alone otherwise.
func smartQuery(combo GenConCombo) string {
	if IsInformativeEdition(combo.RulesEdition) {
		return Normalize(combo.GameSystem + " " + combo.RulesEdition)
	}
	return Normalize(combo.GameSystem)
}

// smartTitleDerivedQuery derives an edition hint from the representative title
// and returns a query using that hint if informative, or system alone otherwise.
func smartTitleDerivedQuery(combo GenConCombo) string {
	derived := ExtractTitleDerived(combo.GameSystem, combo.RepTitle)
	if derived != "" && IsInformativeEdition(derived) {
		return Normalize(combo.GameSystem + " " + derived)
	}
	return Normalize(combo.GameSystem)
}

// exactBest finds all games in candidates whose normalized name exactly matches
// query and returns the one with the lowest BGG rank (rank 0 = unranked, loses
// to any ranked game). Returns empty MatchResult if nothing matches.
func exactBest(query string, candidates []BGGGame) MatchResult {
	var best *BGGGame
	for i := range candidates {
		c := &candidates[i]
		if c.NormalizedName != query {
			continue
		}
		if best == nil || betterRank(*c, *best) {
			best = c
		}
	}
	if best == nil {
		return MatchResult{}
	}
	return MatchResult{BGGID: best.ID, Name: best.Name}
}

// betterRank returns true if a is a better pick than b by BGG rank.
// Rank 0 means unranked — always loses to a ranked game.
func betterRank(a, b BGGGame) bool {
	if a.Rank == 0 {
		return false
	}
	if b.Rank == 0 {
		return true
	}
	return a.Rank < b.Rank
}
