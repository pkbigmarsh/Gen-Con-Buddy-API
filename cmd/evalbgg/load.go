package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// loadBGG reads the BGG CSV and returns all non-expansion games.
func loadBGG(path string) ([]BGGGame, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open bgg csv: %w", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	headers, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read bgg headers: %w", err)
	}
	idx := headerIndex(headers)

	var games []BGGGame
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read bgg row: %w", err)
		}
		g := BGGGame{
			ID:            field(row, idx, "id"),
			Name:          field(row, idx, "name"),
			YearPublished: field(row, idx, "yearpublished"),
			IsExpansion:   field(row, idx, "is_expansion") == "1",
			Rank:          parseInt(field(row, idx, "rank")),
			BayesAverage:  parseFloat(field(row, idx, "bayesaverage")),
			Average:       parseFloat(field(row, idx, "average")),
			UsersRated:    parseInt(field(row, idx, "usersrated")),
			AbstractsRank: field(row, idx, "abstracts_rank"),
			CGSRank:       field(row, idx, "cgs_rank"),
			ChildrensRank: field(row, idx, "childrensgames_rank"),
			FamilyRank:    field(row, idx, "familygames_rank"),
			PartyRank:     field(row, idx, "partygames_rank"),
			StrategyRank:  field(row, idx, "strategygames_rank"),
			ThematicRank:  field(row, idx, "thematic_rank"),
			WarRank:       field(row, idx, "wargames_rank"),
		}
		if !g.IsExpansion {
			games = append(games, g)
		}
	}
	return games, nil
}

// loadGenConCombos reads the Gen Con CSV and returns unique (GameSystem, RulesEdition)
// combos from BGM events, with representative title and event count.
func loadGenConCombos(path string) ([]GenConCombo, error) {
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
	counts := make(map[comboKey]int)

	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read gencon row: %w", err)
		}
		evtType := field(row, idx, "Event Type")
		if !strings.HasPrefix(evtType, "BGM") {
			continue
		}
		key := comboKey{
			system:  field(row, idx, "Game System"),
			edition: field(row, idx, "Rules Edition"),
		}
		title := field(row, idx, "Title")
		counts[key]++
		if titleCounts[key] == nil {
			titleCounts[key] = make(map[string]int)
		}
		titleCounts[key][title]++
	}

	var combos []GenConCombo
	for key, count := range counts {
		combos = append(combos, GenConCombo{
			GameSystem:   key.system,
			RulesEdition: key.edition,
			RepTitle:     mostCommon(titleCounts[key]),
			EventCount:   count,
		})
	}
	return combos, nil
}

func headerIndex(headers []string) map[string]int {
	m := make(map[string]int, len(headers))
	for i, h := range headers {
		m[h] = i
	}
	return m
}

func field(row []string, idx map[string]int, name string) string {
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
