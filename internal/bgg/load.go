package bgg

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// bggEventTypes is the set of Gen Con event type codes whose games appear on BGG.
var bggEventTypes = map[string]bool{
	"BGM": true, // Board Games
	"RPG": true, // Roleplaying Games
	"CGM": true, // Non-Collectible / Tradable Card Games
	"TCG": true, // Tradable Card Games
	"MHE": true, // Miniature Hobby Events
	"HMN": true, // Historical Miniatures
	"NMN": true, // Non-Historical Miniatures
}

// LoadCorpus reads the BGG CSV and returns all games split into BaseGames and Expansions.
func LoadCorpus(path string) (Corpus, error) {
	f, err := os.Open(path)
	if err != nil {
		return Corpus{}, fmt.Errorf("open bgg csv: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	headers, err := r.Read()
	if err != nil {
		return Corpus{}, fmt.Errorf("read bgg headers: %w", err)
	}
	idx := headerIndex(headers)

	var corpus Corpus
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			return Corpus{}, fmt.Errorf("read bgg row: %w", err)
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
	return corpus, nil
}

// LoadGenConCombos reads the Gen Con CSV and returns unique (GameSystem, RulesEdition)
// combos from BGG-eligible events (board games, RPGs, miniatures, card games),
// with representative title and event count.
func LoadGenConCombos(path string) ([]GenConCombo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open gencon csv: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(transform.NewReader(f, charmap.Windows1252.NewDecoder()))
	headers, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read gencon headers: %w", err)
	}

	idx := headerIndex(headers)

	type comboKey struct{ system, edition string }
	titleCounts := make(map[comboKey]map[string]int)

	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("read gencon row: %w", err)
		}

		code, _, _ := strings.Cut(csvField(row, idx, "Event Type"), " ")
		if !bggEventTypes[code] {
			continue
		}

		key := comboKey{
			system:  csvField(row, idx, "Game System"),
			edition: csvField(row, idx, "Rules Edition"),
		}

		if titleCounts[key] == nil {
			titleCounts[key] = make(map[string]int)
		}

		titleCounts[key][csvField(row, idx, "Title")]++
	}

	combos := make([]GenConCombo, 0, len(titleCounts))
	for key, titles := range titleCounts {
		total := 0
		for _, n := range titles {
			total += n
		}

		combos = append(combos, GenConCombo{
			GameSystem:   key.system,
			RulesEdition: key.edition,
			RepTitle:     mostCommon(titles),
			EventCount:   total,
		})
	}
	return combos, nil
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

func mostCommon(counts map[string]int) string {
	var best string
	var bestCount int
	for k, v := range counts {
		if v > bestCount || (v == bestCount && k < best) {
			best = k
			bestCount = v
		}
	}
	return best
}
