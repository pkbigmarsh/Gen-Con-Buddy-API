package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	genconPath := flag.String("gencon", "", "path to Gen Con events CSV (required)")
	bggPath := flag.String("bgg", "", "path to BGG CSV (required)")
	outputPath := flag.String("output", "bgg_eval.csv", "path for output CSV")
	flag.Parse()

	if *genconPath == "" || *bggPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	log.Println("Loading BGG data...")
	bggGames, err := loadBGG(*bggPath)
	if err != nil {
		log.Fatalf("failed to load BGG CSV: %v", err)
	}
	log.Printf("Loaded %d non-expansion BGG games", len(bggGames))

	log.Println("Loading Gen Con combos...")
	combos, err := loadGenConCombos(*genconPath)
	if err != nil {
		log.Fatalf("failed to load Gen Con CSV: %v", err)
	}
	log.Printf("Loaded %d unique (Game System, Rules Edition) combos", len(combos))

	matchers := []Matcher{
		exactSystemRank{},
		fuzzySystemRank{},
		fuzzySystemRated{},
		tokenSystemRank{},
		exactAlwaysEditionRank{},
		fuzzyAlwaysEditionRank{},
		tokenAlwaysEditionRank{},
		exactSmartEditionRank{},
		fuzzySmartEditionRank{},
		fuzzySmartEditionRated{},
		tokenSmartEditionRank{},
		fuzzyTitleRank{},
		exactTitleDerivedAlwaysRank{},
		fuzzyTitleDerivedAlwaysRank{},
		exactTitleDerivedSmartRank{},
		fuzzyTitleDerivedSmartRank{},
		fuzzyTitleDerivedSmartRated{},
		tokenTitleDerivedSmartRank{},
	}
	log.Printf("Running %d matchers across %d combos...", len(matchers), len(combos))

	results := make([][]MatchResult, len(combos))
	for i, combo := range combos {
		results[i] = make([]MatchResult, len(matchers))
		for j, m := range matchers {
			results[i][j] = m.Match(combo, bggGames)
		}
		if (i+1)%100 == 0 {
			log.Printf("  processed %d/%d combos", i+1, len(combos))
		}
	}

	log.Printf("Writing output to %s...", *outputPath)
	f, err := os.Create(*outputPath)
	if err != nil {
		log.Fatalf("failed to create output file: %v", err)
	}
	defer f.Close()

	if err := writeCSV(f, combos, matchers, results); err != nil {
		log.Fatalf("failed to write CSV: %v", err)
	}

	fmt.Printf("Done. Output written to %s\n", *outputPath)
}
