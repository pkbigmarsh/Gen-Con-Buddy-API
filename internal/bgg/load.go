package bgg

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// LoadCorpus opens path and parses the BGG ranks CSV into a Corpus.
func LoadCorpus(path string) (*Corpus, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open bgg csv: %w", err)
	}

	defer f.Close()

	return LoadCorpusFromReader(f)
}

// LoadCorpusFromReader parses BGG ranks CSV from r into a Corpus, splitting
// rows into BaseGames and Expansions. Column order is irrelevant; fields are
// resolved by header name.
func LoadCorpusFromReader(r io.Reader) (*Corpus, error) {
	cr := csv.NewReader(r)
	headers, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("read bgg headers: %w", err)
	}

	idx := headerIndex(headers)

	var corpus Corpus
	for {
		row, err := cr.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("read bgg row: %w", err)
		}

		name := csvField(row, idx, "name")
		g := BGGGame{
			ID:             csvField(row, idx, "id"),
			Name:           name,
			NormalizedName: Normalize(name),
			YearPublished:  csvField(row, idx, "yearpublished"),
			IsExpansion:    csvField(row, idx, "is_expansion") == "1",
			Rank:           parseInt(csvField(row, idx, "rank")),
			BayesAverage:   parseFloat(csvField(row, idx, "bayesaverage")),
			Average:        parseFloat(csvField(row, idx, "average")),
			UsersRated:     parseInt(csvField(row, idx, "usersrated")),
			AbstractsRank:  csvField(row, idx, "abstracts_rank"),
			CGSRank:        csvField(row, idx, "cgs_rank"),
			ChildrensRank:  csvField(row, idx, "childrensgames_rank"),
			FamilyRank:     csvField(row, idx, "familygames_rank"),
			PartyRank:      csvField(row, idx, "partygames_rank"),
			StrategyRank:   csvField(row, idx, "strategygames_rank"),
			ThematicRank:   csvField(row, idx, "thematic_rank"),
			WarRank:        csvField(row, idx, "wargames_rank"),
		}

		if g.IsExpansion {
			corpus.Expansions = append(corpus.Expansions, g)
		} else {
			corpus.BaseGames = append(corpus.BaseGames, g)
		}
	}

	return &corpus, nil
}

// LoadMapping reads a bgg_mapping.json file and returns its Mappings field.
// Returns an empty map (no error) when path is empty.
// Returns an error if the file cannot be read, parsed, or appears to be the
// old slice format (total_combos > 0 but mappings is empty after unmarshal).
func LoadMapping(path string) (map[string]MappingEntry, error) {
	if path == "" {
		return map[string]MappingEntry{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read bgg mapping: %w", err)
	}

	var file MappingFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse bgg mapping: %w", err)
	}

	if file.TotalCombos > 0 && len(file.Mappings) == 0 {
		return nil, fmt.Errorf("bgg mapping file appears to be in the old slice format; regenerate it")
	}

	if file.Mappings == nil {
		return map[string]MappingEntry{}, nil
	}

	return file.Mappings, nil
}

func headerIndex(headers []string) map[string]int {
	m := make(map[string]int, len(headers))
	for i, h := range headers {
		m[h] = i
	}

	return m
}

func csvField(row []string, idx map[string]int, name string) string {
	i, ok := idx[name]
	if !ok || i >= len(row) {
		return ""
	}

	return strings.TrimSpace(row[i])
}

func parseInt(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}
