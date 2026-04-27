package bgg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// testCorpus is a small in-memory BGG dataset for cascade tests.
var testCorpus = Corpus{
	BaseGames: []BGGGame{
		{ID: "1", Name: "Wingspan", Rank: 10, UsersRated: 50000},
		{ID: "3", Name: "Ark Nova", Rank: 2, UsersRated: 60000},
		{ID: "4", Name: "Axis & Allies: 1941", Rank: 100, UsersRated: 5000},
		{ID: "5", Name: "Axis & Allies: 1942", Rank: 80, UsersRated: 8000},
		// IDs 6a and 6b both normalize to "axis & allies" — used to test tiebreak.
		{ID: "6a", Name: "Axis & Allies", Rank: 50, UsersRated: 20000},
		{ID: "6b", Name: "Axis & Allies!", Rank: 200, UsersRated: 20000},
		{ID: "7", Name: "SHŌBU", Rank: 50, UsersRated: 3000},
		{ID: "8", Name: "Orléans", Rank: 30, UsersRated: 15000},
	},
	Expansions: []BGGGame{
		{ID: "2", Name: "Wingspan: European Expansion", Rank: 0, UsersRated: 10000},
		{ID: "9", Name: "Catan: Cities & Knights", Rank: 0, UsersRated: 25000},
	},
}

func TestMatchStage1_GenericEdition_SystemOnly(t *testing.T) {
	// generic edition → query is system alone; 6a and 6b both match but 6a has lower rank
	result := Match(GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1st", RepTitle: "Axis & Allies"}, testCorpus)
	require.Equal(t, "6a", result.BGGID)
	require.Equal(t, "Axis & Allies", result.Name)
}

func TestMatchStage1_InformativeEdition_SystemPlusEdition(t *testing.T) {
	// informative edition → query is "axis & allies 1941"
	result := Match(GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1941", RepTitle: "Axis & Allies 1941"}, testCorpus)
	require.Equal(t, "4", result.BGGID)
}

func TestMatchStage1_TiebreakByRank(t *testing.T) {
	// IDs 6a (rank 50) and 6b (rank 200) both normalize to "axis & allies".
	// The lower-ranked game (6a) must win the tiebreak.
	result := Match(GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1st", RepTitle: "Axis & Allies"}, testCorpus)
	require.Equal(t, "6a", result.BGGID)
}

func TestMatchStage1_DiacriticNormalization(t *testing.T) {
	// Gen Con entry omits accent; BGG has "Orléans"
	result := Match(GenConCombo{GameSystem: "Orleans", RulesEdition: "1st", RepTitle: "Orleans"}, testCorpus)
	require.Equal(t, "8", result.BGGID)
}

func TestMatchStage1_DiacriticNormalization_Shobu(t *testing.T) {
	// BGG has "SHŌBU"; Gen Con has "SHOBU"
	result := Match(GenConCombo{GameSystem: "SHOBU", RulesEdition: "1st", RepTitle: "SHOBU"}, testCorpus)
	require.Equal(t, "7", result.BGGID)
}

func TestMatchStage2_TitleDerived(t *testing.T) {
	localCorpus3 := Corpus{
		BaseGames: []BGGGame{
			{ID: "30", Name: "Terraforming Mars: Ares Expedition", Rank: 50},
		},
	}
	result := Match(GenConCombo{
		GameSystem:   "Terraforming Mars",
		RulesEdition: "1st",
		RepTitle:     "Terraforming Mars: Ares Expedition Demo",
	}, localCorpus3)
	// Stage 1: system alone "terraforming mars" → no exact match
	// Stage 2: derived from title = "ares expedition" (informative) → query "terraforming mars ares expedition" → exact match ID 30
	require.Equal(t, "30", result.BGGID)
}

func TestMatchStage3_ExpansionSearch_InformativeEdition(t *testing.T) {
	// Wingspan + "European Expansion" (informative) → stage 1 finds nothing in BaseGames,
	// stage 2 finds nothing, stage 3 searches Expansions and finds ID 2.
	result := Match(GenConCombo{
		GameSystem:   "Wingspan",
		RulesEdition: "European Expansion",
		RepTitle:     "Wingspan European Expansion",
	}, testCorpus)
	require.Equal(t, "2", result.BGGID)
}

func TestMatchStage3_ExpansionNotSearched_GenericEdition(t *testing.T) {
	// Wingspan + "1st" (generic) → stage 3 does NOT run → finds base game ID 1, not expansion
	result := Match(GenConCombo{
		GameSystem:   "Wingspan",
		RulesEdition: "1st",
		RepTitle:     "Wingspan",
	}, testCorpus)
	require.Equal(t, "1", result.BGGID)
}

func TestMatchStage4_NoMatch(t *testing.T) {
	// Game not in corpus → empty result
	result := Match(GenConCombo{
		GameSystem:   "Completely Unknown Board Game",
		RulesEdition: "1st",
		RepTitle:     "Unknown",
	}, testCorpus)
	require.Empty(t, result.BGGID)
	require.Empty(t, result.Name)
}
