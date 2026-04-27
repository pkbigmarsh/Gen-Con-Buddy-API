package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
)

// writeCSV writes the full comparison CSV to w.
func writeCSV(w io.Writer, combos []GenConCombo, matchers []Matcher, results [][]MatchResult) error {
	cw := csv.NewWriter(w)

	// Build header
	header := []string{
		"game_system",
		"rules_edition",
		"edition_informative",
		"event_count",
		"representative_title",
	}
	for _, m := range matchers {
		n := m.Name()
		header = append(header, n+"_id", n+"_name", n+"_score")
	}
	header = append(header, "agreement_count", "consensus_id", "consensus_name", "correct_bgg_id")

	if err := cw.Write(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	for i, combo := range combos {
		row := []string{
			combo.GameSystem,
			combo.RulesEdition,
			strconv.FormatBool(isInformativeEdition(combo.RulesEdition)),
			strconv.Itoa(combo.EventCount),
			combo.RepTitle,
		}

		// Count votes for consensus
		votes := make(map[string]int)
		for j := range matchers {
			r := results[i][j]
			if r.BGGGame != nil {
				votes[r.BGGGame.ID]++
			}
		}

		for j := range matchers {
			r := results[i][j]
			if r.BGGGame == nil {
				row = append(row, "", "", "")
			} else {
				row = append(row,
					r.BGGGame.ID,
					r.BGGGame.Name,
					fmt.Sprintf("%.4f", r.Score),
				)
			}
		}

		consensusID, consensusName := consensus(votes, results[i])
		agreementCount := 0
		if consensusID != "" {
			agreementCount = votes[consensusID]
		}

		row = append(row,
			strconv.Itoa(agreementCount),
			consensusID,
			consensusName,
			"", // correct_bgg_id — left blank for manual scoring
		)

		if err := cw.Write(row); err != nil {
			return fmt.Errorf("write row: %w", err)
		}
	}

	cw.Flush()
	return cw.Error()
}

// consensus returns the BGG id and name that the most matchers agreed on.
// Returns empty strings if no matcher returned a result.
func consensus(votes map[string]int, rowResults []MatchResult) (id, name string) {
	var bestID string
	var bestCount int
	for vid, count := range votes {
		if count > bestCount || (count == bestCount && vid < bestID) {
			bestID = vid
			bestCount = count
		}
	}
	if bestID == "" {
		return "", ""
	}
	// Find name for bestID from any result
	for _, r := range rowResults {
		if r.BGGGame != nil && r.BGGGame.ID == bestID {
			return bestID, r.BGGGame.Name
		}
	}
	return bestID, ""
}
