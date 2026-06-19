package bgg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// testCorpus is a small in-memory BGG dataset for cascade tests.
var (
	testCorpus = Corpus{
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

	testGameMap = make(map[string]*BGGGame)
)

func init() {
	for i, g := range testCorpus.BaseGames {
		testCorpus.BaseGames[i].NormalizedName = Normalize(testCorpus.BaseGames[i].Name)
		testGameMap[g.ID] = &testCorpus.BaseGames[i]
	}

	for i, g := range testCorpus.Expansions {
		testCorpus.Expansions[i].NormalizedName = Normalize(testCorpus.Expansions[i].Name)
		testGameMap[g.ID] = &testCorpus.Expansions[i]
	}
}

func TestMatch(t *testing.T) {
	tests := []struct {
		name        string
		combo       GenConCombo
		corpus      Corpus
		wantBGGID   string
		wantNoMatch bool
		wantGame    *BGGGame
	}{
		{
			name:      "stage1 informative edition uses system+edition query",
			combo:     GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1941", RepTitle: "Axis & Allies 1941"},
			corpus:    testCorpus,
			wantBGGID: "4",
			wantGame:  testGameMap["4"],
		},
		{
			name:      "stage1 tiebreak picks lower rank",
			combo:     GenConCombo{GameSystem: "Axis & Allies", RulesEdition: "1st", RepTitle: "Axis & Allies"},
			corpus:    testCorpus,
			wantBGGID: "6a",
			wantGame:  testGameMap["6a"],
		},
		{
			name:      "stage1 diacritic normalization Orleans",
			combo:     GenConCombo{GameSystem: "Orleans", RulesEdition: "1st", RepTitle: "Orleans"},
			corpus:    testCorpus,
			wantBGGID: "8",
			wantGame:  testGameMap["8"],
		},
		{
			name:      "stage1 diacritic normalization SHOBU",
			combo:     GenConCombo{GameSystem: "SHOBU", RulesEdition: "1st", RepTitle: "SHOBU"},
			corpus:    testCorpus,
			wantBGGID: "7",
			wantGame:  testGameMap["7"],
		},
		{
			name: "stage2 title-derived edition hint",
			combo: GenConCombo{
				GameSystem:   "Terraforming Mars",
				RulesEdition: "1st",
				RepTitle:     "Terraforming Mars: Ares Expedition Demo",
			},
			corpus: Corpus{
				BaseGames: []BGGGame{
					{ID: "30", Name: "Terraforming Mars: Ares Expedition", NormalizedName: Normalize("Terraforming Mars: Ares Expedition"), Rank: 50},
				},
			},
			wantBGGID: "30",
			wantGame:  &BGGGame{ID: "30", Name: "Terraforming Mars: Ares Expedition", NormalizedName: Normalize("Terraforming Mars: Ares Expedition"), Rank: 50},
		},
		{
			name: "stage3 expansion found when edition is informative",
			combo: GenConCombo{
				GameSystem:   "Wingspan",
				RulesEdition: "European Expansion",
				RepTitle:     "Wingspan European Expansion",
			},
			corpus:    testCorpus,
			wantBGGID: "2",
			wantGame:  testGameMap["2"],
		},
		{
			name: "stage3 expansion not searched when edition is generic",
			combo: GenConCombo{
				GameSystem:   "Wingspan",
				RulesEdition: "1st",
				RepTitle:     "Wingspan",
			},
			corpus:    testCorpus,
			wantBGGID: "1",
			wantGame:  testGameMap["1"],
		},
		{
			name: "no match returns empty result",
			combo: GenConCombo{
				GameSystem:   "Completely Unknown Board Game",
				RulesEdition: "1st",
				RepTitle:     "Unknown",
			},
			corpus:      testCorpus,
			wantNoMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Match(tt.combo, &tt.corpus)
			if tt.wantNoMatch {
				require.Empty(t, result.BGGID)
				require.Empty(t, result.Name)
			} else {
				require.Equal(t, tt.wantBGGID, result.BGGID)
				if tt.wantGame != nil {
					require.NotNil(t, result.Game)
					require.Equal(t, *tt.wantGame, *result.Game)
				}
			}
		})
	}
}
