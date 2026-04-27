package matchbgg

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/gencon_buddy_api/internal/bgg"
	"github.com/spf13/cobra"
)

// MappingEntry is one resolved (game_system, rules_edition) → BGG game mapping.
type MappingEntry struct {
	GameSystem   string `json:"game_system"`
	RulesEdition string `json:"rules_edition"`
	BGGID        string `json:"bgg_id"`
	BGGName      string `json:"bgg_name"`
}

// MappingFile is the full output written to disk.
type MappingFile struct {
	GeneratedAt string         `json:"generated_at"`
	TotalCombos int            `json:"total_combos"`
	Matched     int            `json:"matched"`
	Mappings    []MappingEntry `json:"mappings"`
}

var MatchBGGCmd = &cobra.Command{
	Use:   "match-bgg",
	Short: "Match Gen Con game/edition combos to BGG game IDs and write a mapping file",
	Long: `Reads the Gen Con events CSV and BGG games CSV, runs the cascade matcher
against each unique (Game System, Rules Edition) combination, and writes
a JSON mapping file. Commit the mapping file to the repo so every
'data update' run uses the same mappings.`,
	RunE: run,
}

func init() {
	MatchBGGCmd.Flags().StringP("gencon", "g", "", "path to Gen Con events CSV (required)")
	MatchBGGCmd.Flags().StringP("bgg", "b", "", "path to BGG CSV (required)")
	MatchBGGCmd.Flags().StringP("output", "o", "bgg_mapping.json", "output path for the mapping file")
	_ = MatchBGGCmd.MarkFlagRequired("gencon")
	_ = MatchBGGCmd.MarkFlagRequired("bgg")
}

func run(cmd *cobra.Command, _ []string) error {
	genconPath, _ := cmd.Flags().GetString("gencon")
	bggPath, _ := cmd.Flags().GetString("bgg")
	outputPath, _ := cmd.Flags().GetString("output")

	corpus, err := bgg.LoadCorpus(bggPath)
	if err != nil {
		return fmt.Errorf("load bgg corpus: %w", err)
	}

	combos, err := bgg.LoadGenConCombos(genconPath)
	if err != nil {
		return fmt.Errorf("load gencon combos: %w", err)
	}

	// TODO(overrides): Before running the cascade, accept an --overrides flag
	// pointing to a JSON file with the same structure as this output. Load the
	// overrides, then after cascade results are computed, replace any entry for a
	// matching (game_system, rules_edition) key with the override value. Overrides
	// that name a combo not present in the Gen Con data are silently ignored.

	var mappings []MappingEntry
	for _, combo := range combos {
		result := bgg.Match(combo, corpus)
		if result.BGGID == "" {
			continue
		}
		mappings = append(mappings, MappingEntry{
			GameSystem:   combo.GameSystem,
			RulesEdition: combo.RulesEdition,
			BGGID:        result.BGGID,
			BGGName:      result.Name,
		})
	}

	// Sort for stable, diff-friendly output.
	sort.Slice(mappings, func(i, j int) bool {
		if mappings[i].GameSystem != mappings[j].GameSystem {
			return mappings[i].GameSystem < mappings[j].GameSystem
		}
		return mappings[i].RulesEdition < mappings[j].RulesEdition
	})

	out := MappingFile{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		TotalCombos: len(combos),
		Matched:     len(mappings),
		Mappings:    mappings,
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal mapping: %w", err)
	}

	if err := os.WriteFile(outputPath, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("write mapping file: %w", err)
	}

	cmd.Printf("Matched %d / %d combos → %s\n", out.Matched, out.TotalCombos, outputPath)
	return nil
}
