package bgg

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// ---- LoadMapping ----

func TestLoadMapping_EmptyPath(t *testing.T) {
	m, err := LoadMapping("")
	require.NoError(t, err)
	require.Empty(t, m)
}

func TestLoadMapping_FileNotFound(t *testing.T) {
	_, err := LoadMapping("/nonexistent/path/bgg_mapping.json")
	require.Error(t, err)
}

func TestLoadMapping_BadJSON(t *testing.T) {
	f := filepath.Join(t.TempDir(), "bad.json")
	require.NoError(t, os.WriteFile(f, []byte("not json"), 0600))
	_, err := LoadMapping(f)
	require.Error(t, err)
}

func TestLoadMapping_OldSliceFormat(t *testing.T) {
	// TotalCombos > 0 but mappings is null/absent → old format
	data := []byte(`{"total_combos": 5, "matched": 3}`)
	f := filepath.Join(t.TempDir(), "old.json")
	require.NoError(t, os.WriteFile(f, data, 0600))
	_, err := LoadMapping(f)
	require.ErrorContains(t, err, "old slice format")
}

func TestLoadMapping_Valid(t *testing.T) {
	file := MappingFile{
		GeneratedAt: "2025-01-01",
		TotalCombos: 2,
		Matched:     2,
		Mappings: map[string]MappingEntry{
			"Wingspan|1st": {BGGID: "266192", BGGName: "Wingspan", BGGRank: 10, BGGAvgRating: 8.1},
			"Ark Nova|1st": {BGGID: "342942", BGGName: "Ark Nova", BGGRank: 2, BGGAvgRating: 8.6},
		},
	}
	data, err := json.Marshal(file)
	require.NoError(t, err)

	f := filepath.Join(t.TempDir(), "mapping.json")
	require.NoError(t, os.WriteFile(f, data, 0600))

	m, err := LoadMapping(f)
	require.NoError(t, err)
	require.Len(t, m, 2)
	require.Equal(t, "266192", m["Wingspan|1st"].BGGID)
	require.Equal(t, 10, m["Wingspan|1st"].BGGRank)
	require.InDelta(t, 8.1, m["Wingspan|1st"].BGGAvgRating, 0.001)
	require.Equal(t, "342942", m["Ark Nova|1st"].BGGID)
}

func TestLoadMapping_NilMappings(t *testing.T) {
	// TotalCombos == 0, mappings absent → not old format, just empty
	data := []byte(`{"total_combos": 0, "matched": 0}`)
	f := filepath.Join(t.TempDir(), "empty.json")
	require.NoError(t, os.WriteFile(f, data, 0600))
	m, err := LoadMapping(f)
	require.NoError(t, err)
	require.Empty(t, m)
}

// ---- LoadCorpus ----

var corpusCSV = "id,name,yearpublished,is_expansion,rank,bayesaverage,average,usersrated,abstracts_rank,cgs_rank,childrensgames_rank,familygames_rank,partygames_rank,strategygames_rank,thematic_rank,wargames_rank\n" +
	"1,Wingspan,2019,0,10,8.0,8.2,50000,,,,,,,,\n" +
	"2,Wingspan: European Expansion,2019,1,0,7.5,7.8,10000,,,,,,,,\n" +
	"3,Ark Nova,2021,0,2,8.6,8.7,60000,,,,,,,,\n"

func TestLoadCorpus_SplitsBaseAndExpansion(t *testing.T) {
	f := filepath.Join(t.TempDir(), "bgg.csv")
	require.NoError(t, os.WriteFile(f, []byte(corpusCSV), 0600))

	corpus, err := LoadCorpus(f)
	require.NoError(t, err)
	require.NotNil(t, corpus)
	require.Len(t, corpus.BaseGames, 2)
	require.Len(t, corpus.Expansions, 1)
}

func TestLoadCorpus_ParsesRank(t *testing.T) {
	f := filepath.Join(t.TempDir(), "bgg.csv")
	require.NoError(t, os.WriteFile(f, []byte(corpusCSV), 0600))

	corpus, err := LoadCorpus(f)
	require.NoError(t, err)
	require.NotNil(t, corpus)

	var wingspan *BGGGame
	for i := range corpus.BaseGames {
		if corpus.BaseGames[i].ID == "1" {
			wingspan = &corpus.BaseGames[i]
			break
		}
	}
	require.NotNil(t, wingspan)
	require.Equal(t, 10, wingspan.Rank)
	require.InDelta(t, 8.0, wingspan.BayesAverage, 0.001)
}

func TestLoadCorpus_ExpansionIsUnranked(t *testing.T) {
	f := filepath.Join(t.TempDir(), "bgg.csv")
	require.NoError(t, os.WriteFile(f, []byte(corpusCSV), 0600))

	corpus, err := LoadCorpus(f)
	require.NoError(t, err)
	require.NotNil(t, corpus)
	require.Equal(t, 0, corpus.Expansions[0].Rank)
	require.True(t, corpus.Expansions[0].IsExpansion)
}

func TestLoadCorpus_FileNotFound(t *testing.T) {
	corpus, err := LoadCorpus("/nonexistent/bgg.csv")
	require.Error(t, err)
	require.Nil(t, corpus)
}

// ---- LoadCorpusFromReader ----

func TestLoadCorpusFromReader(t *testing.T) {
	const csvData = `id,name,yearpublished,rank,bayesaverage,average,usersrated,is_expansion,abstracts_rank,cgs_rank,childrensgames_rank,familygames_rank,partygames_rank,strategygames_rank,thematic_rank,wargames_rank
1,Wingspan,2019,10,8.0,8.2,50000,0,,,,,,,,
2,Wingspan: European Expansion,2019,0,7.5,7.8,10000,1,,,,,,,,
`

	corpus, err := LoadCorpusFromReader(strings.NewReader(csvData))
	require.NoError(t, err)
	require.Len(t, corpus.BaseGames, 1)
	require.Len(t, corpus.Expansions, 1)
	require.Equal(t, "Wingspan", corpus.BaseGames[0].Name)
	require.Equal(t, 10, corpus.BaseGames[0].Rank)
	require.InDelta(t, 8.2, corpus.BaseGames[0].Average, 0.001)
	require.Equal(t, Normalize("Wingspan"), corpus.BaseGames[0].NormalizedName)
	require.True(t, corpus.Expansions[0].IsExpansion)
}
