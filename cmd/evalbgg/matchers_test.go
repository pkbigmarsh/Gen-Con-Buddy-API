package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// fixture is a small BGG dataset for matcher tests.
var fixture = []BGGGame{
	{ID: "1", Name: "Wingspan", Rank: 10, UsersRated: 50000, IsExpansion: false},
	{ID: "2", Name: "Wingspan: European Expansion", Rank: 0, UsersRated: 10000, IsExpansion: false},
	{ID: "3", Name: "Ark Nova", Rank: 2, UsersRated: 60000, IsExpansion: false},
	{ID: "4", Name: "Axis & Allies: 1941", Rank: 100, UsersRated: 5000, IsExpansion: false},
	{ID: "5", Name: "Axis & Allies: 1942", Rank: 80, UsersRated: 8000, IsExpansion: false},
	{ID: "6", Name: "Axis & Allies", Rank: 200, UsersRated: 20000, IsExpansion: false},
}

func TestExactSystemRank(t *testing.T) {
	m := exactSystemRank{}
	require.Equal(t, "exact-system-rank", m.Name())

	// exact match on "Wingspan" → picks ID 1 (only exact match)
	result := m.Match(GenConCombo{GameSystem: "Wingspan", RulesEdition: "1st", RepTitle: "Wingspan"}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "1", result.BGGGame.ID)
	require.Equal(t, 1.0, result.Score)

	// no match for unknown game
	result = m.Match(GenConCombo{GameSystem: "Unknown Game XYZ", RulesEdition: "1st", RepTitle: "Unknown"}, fixture)
	require.Nil(t, result.BGGGame)
}

func TestFuzzySystemRank(t *testing.T) {
	m := fuzzySystemRank{}
	require.Equal(t, "fuzzy-system-rank", m.Name())

	// fuzzy match on "Wingspaan" (typo) → should still find Wingspan
	result := m.Match(GenConCombo{GameSystem: "Wingspaan", RulesEdition: "1st", RepTitle: "Wingspaan"}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "1", result.BGGGame.ID)
	require.Greater(t, result.Score, 0.7)

	// "Axis & Allies" fuzzy → exact match on ID 6 wins with score 1.0;
	// tiebreak-by-rank only applies when scores are equal, but here ID 6
	// ("Axis & Allies") scores 1.0 vs lower scores for the variant titles.
	result = m.Match(GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1st", RepTitle: "Axis & Allies"}, fixture)
	require.NotNil(t, result.BGGGame)
	// ID 6 wins because it is an exact normalized match (score 1.0)
	require.Equal(t, "6", result.BGGGame.ID)
}

func TestFuzzySystemRated(t *testing.T) {
	m := fuzzySystemRated{}
	require.Equal(t, "fuzzy-system-rated", m.Name())

	// "Axis & Allies" → ID 6 is an exact normalized match (score 1.0), so the
	// rated tiebreak never fires here. The tiebreaker is correct; this test
	// validates that the right result is returned regardless.
	result := m.Match(GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1st", RepTitle: "Axis & Allies"}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "6", result.BGGGame.ID)
}

func TestTokenSystemRank(t *testing.T) {
	m := tokenSystemRank{}
	require.Equal(t, "token-system-rank", m.Name())

	// token match on "Axis Allies" (missing &) → finds axis & allies entries
	result := m.Match(GenConCombo{GameSystem: "Axis Allies", RulesEdition: "1st", RepTitle: "Axis Allies"}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Greater(t, result.Score, 0.5)
}

func TestExactAlwaysEditionRank(t *testing.T) {
	m := exactAlwaysEditionRank{}
	require.Equal(t, "exact-always-edition-rank", m.Name())

	// "Axis & Allies" + "1941" → matches "Axis & Allies: 1941" (ID 4)
	result := m.Match(GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1941", RepTitle: "Axis & Allies 1941"}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "4", result.BGGGame.ID)

	// "Wingspan" + "1st" → no exact match (BGG name is just "Wingspan", not "Wingspan 1st")
	result = m.Match(GenConCombo{GameSystem: "Wingspan", RulesEdition: "1st", RepTitle: "Wingspan"}, fixture)
	require.Nil(t, result.BGGGame)
}

func TestFuzzyAlwaysEditionRank(t *testing.T) {
	m := fuzzyAlwaysEditionRank{}
	require.Equal(t, "fuzzy-always-edition-rank", m.Name())

	// "Axis & Allies" + "1941" → fuzzy matches "Axis & Allies: 1941" (ID 4) better than others
	result := m.Match(GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1941", RepTitle: "Axis & Allies 1941"}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "4", result.BGGGame.ID)
}

func TestTokenAlwaysEditionRank(t *testing.T) {
	m := tokenAlwaysEditionRank{}
	require.Equal(t, "token-always-edition-rank", m.Name())

	result := m.Match(GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1941", RepTitle: "Axis & Allies 1941"}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "4", result.BGGGame.ID)
}

func TestExactSmartEditionRank(t *testing.T) {
	m := exactSmartEditionRank{}
	require.Equal(t, "exact-smart-edition-rank", m.Name())

	// informative edition → uses System + Edition
	result := m.Match(GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1941", RepTitle: "Axis & Allies 1941"}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "4", result.BGGGame.ID)

	// generic edition ("1st") → falls back to System only → matches "Axis & Allies" (ID 6)
	result = m.Match(GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1st", RepTitle: "Axis & Allies"}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "6", result.BGGGame.ID)
}

func TestFuzzySmartEditionRank(t *testing.T) {
	m := fuzzySmartEditionRank{}
	require.Equal(t, "fuzzy-smart-edition-rank", m.Name())

	// informative edition → fuzzy on System + Edition
	result := m.Match(GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1941", RepTitle: "Axis & Allies 1941"}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "4", result.BGGGame.ID)
}

func TestFuzzySmartEditionRated(t *testing.T) {
	m := fuzzySmartEditionRated{}
	require.Equal(t, "fuzzy-smart-edition-rated", m.Name())

	// generic edition → fuzzy on System alone → most rated axis & allies (ID 6, 20000)
	result := m.Match(GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1st", RepTitle: "Axis & Allies"}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "6", result.BGGGame.ID)
}

func TestTokenSmartEditionRank(t *testing.T) {
	m := tokenSmartEditionRank{}
	require.Equal(t, "token-smart-edition-rank", m.Name())

	result := m.Match(GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1941", RepTitle: "Axis & Allies 1941"}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "4", result.BGGGame.ID)
}

func TestFuzzyTitleRank(t *testing.T) {
	m := fuzzyTitleRank{}
	require.Equal(t, "fuzzy-title-rank", m.Name())

	// RepTitle "Wingspan" → fuzzy matches "Wingspan" (ID 1)
	result := m.Match(GenConCombo{GameSystem: "Wingspan", RulesEdition: "1st", RepTitle: "Wingspan"}, fixture)
	require.NotNil(t, result.BGGGame)
	require.Equal(t, "1", result.BGGGame.ID)

	// RepTitle "Axis & Allies 1941 for Beginners" → should find something about axis & allies
	result = m.Match(GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1941", RepTitle: "Axis & Allies 1941 for Beginners"}, fixture)
	require.NotNil(t, result.BGGGame)
}
