package data

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/gencon_buddy_api/cmd/app"
	"github.com/gencon_buddy_api/internal/bgg"
	"github.com/gencon_buddy_api/internal/event"
)

const (
	outputFlag string = "output"
)

var bggCmd = &cobra.Command{
	Use:   "bgg",
	Short: "Generate the BGG mappings file",
	Long:  "Builds bgg_mapping.json by matching Gen Con event combos against the BGG corpus. The corpus is read from --filepath, or fetched from BGG when --filepath is omitted.",
	RunE:  updateBgg,
}

func init() {
	bggCmd.Flags().StringP(outputFlag, "o", "", "the filepath for the output of the bgg mappings")
	bggCmd.Flags().String(flagBGGUsername, "", "BGG username for auto-fetch (overrides BGG_USERNAME)")
	bggCmd.Flags().String(flagBGGPassword, "", "BGG password for auto-fetch (overrides BGG_PASSWORD)")
}

func updateBgg(cmd *cobra.Command, _ []string) error {
	gcb := app.GetAppFromContext(cmd.Context())
	if gcb == nil {
		return fmt.Errorf("couldn't initialize gcb app context")
	}

	corpus, err := loadCorpus(cmd, gcb)
	if err != nil {
		return err
	}

	gcb.Logger.Info().
		Int("game_count", len(corpus.BaseGames)).
		Int("expansion_count", len(corpus.Expansions)).
		Msg("Corpus loaded")

	outputPath, err := cmd.Flags().GetString(outputFlag)
	if err != nil {
		return fmt.Errorf("failed to read %s flag: %w", outputFlag, err)
	}

	if outputPath == "" {
		return fmt.Errorf("--%s is required", outputFlag)
	}

	gcb.Logger.Info().Msg("Starting mapping generation")
	mappingFile, err := generateBggMapping(gcb, corpus)
	if err != nil {
		return fmt.Errorf("failed to generate bgg mapping: %w", err)
	}

	gcb.Logger.Info().
		Int("total_combos", mappingFile.TotalCombos).
		Int("matched", mappingFile.Matched).
		Msg("Mapping generated")

	if err := writeMappingFile(gcb, outputPath, mappingFile); err != nil {
		return err
	}

	return nil
}

// loadCorpus reads the BGG corpus from --filepath, or auto-fetches it from BGG
// when --filepath is empty.
func loadCorpus(cmd *cobra.Command, gcb *app.App) (*bgg.Corpus, error) {
	corpusPath, err := cmd.Flags().GetString(filepathFlag)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s flag: %w", filepathFlag, err)
	}

	if corpusPath != "" {
		gcb.Logger.Info().Str("corpus_path", corpusPath).Msg("Loading BGG corpus from file")
		corpus, err := bgg.LoadCorpus(corpusPath)
		if err != nil {
			return nil, fmt.Errorf("invalid file [%s], failed to load corpus: %w", corpusPath, err)
		}

		return corpus, nil
	}

	gcb.Logger.Info().Msg("No --filepath provided; fetching BGG corpus from boardgamegeek.com")
	creds, err := resolveBGGCredentials(cmd)
	if err != nil {
		return nil, err
	}

	fetcher, err := bgg.NewFetcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create bgg fetcher: %w", err)
	}

	rc, err := fetcher.FetchRanksCSV(cmd.Context(), creds)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch bgg ranks dump: %w", err)
	}

	defer func() {
		if err := rc.Close(); err != nil {
			gcb.Logger.Err(err).Msg("failed to close bgg csv stream")
		}
	}()

	corpus, err := bgg.LoadCorpusFromReader(rc)
	if err != nil {
		return nil, fmt.Errorf("failed to load fetched bgg corpus: %w", err)
	}

	return corpus, nil
}

func writeMappingFile(gcb *app.App, outputPath string, mappingFile bgg.MappingFile) error {
	data, err := json.Marshal(mappingFile)
	if err != nil {
		return fmt.Errorf("failed to marshal bgg mappings: %w", err)
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create a mapping file at [%s]: %w", outputPath, err)
	}

	defer func() {
		if err := out.Close(); err != nil {
			gcb.Logger.Err(err).Str("filepath", outputPath).Msg("Failed to close mapping output file")
		}
	}()

	if _, err := out.Write(data); err != nil {
		return fmt.Errorf("failed to write bgg mappings to [%s]: %w", outputPath, err)
	}

	return nil
}

func generateBggMapping(gcb *app.App, corpus *bgg.Corpus) (bgg.MappingFile, error) {
	mappingFile := bgg.MappingFile{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		TotalCombos: 0,
		Matched:     0,
		Mappings:    make(map[string]bgg.MappingEntry),
	}

	scanSearchRequest := event.SearchRequest{
		Page:  0,
		Limit: gcb.BatchSize,
		Sorts: []event.SortEntry{
			{
				Field: event.GameID,
				Dir:   "desc",
			},
			{
				Field: event.Title,
				Dir:   "desc",
			},
		},
		SearchAfter: nil,
	}

	type comboKey struct{ system, edition string }

	var (
		results     event.SearchResponse
		evts        []*event.Event
		err         error
		titleCounts = make(map[comboKey]map[string]int)
	)

	results, err = gcb.EventRepo.Search(Cmd.Context(), scanSearchRequest)

	for err == nil && len(results.Events) == gcb.BatchSize {
		gcb.Logger.Debug().Int64("total_events", results.TotalEvents).Msgf("found %d events for page", len(results.Events))
		for _, e := range results.Events {
			if e == nil {
				continue
			}

			evts = append(evts, e)
			key := comboKey{
				system:  e.GameSystem,
				edition: e.RulesEdition,
			}

			if titleCounts[key] == nil {
				titleCounts[key] = make(map[string]int)
			}

			titleCounts[key][e.Title]++
		}

		scanSearchRequest.SearchAfter = results.SearchAfter
		results, err = gcb.EventRepo.Search(Cmd.Context(), scanSearchRequest)
	}

	if err != nil {
		gcb.Logger.Warn().Err(err).
			Msg("failed to scan events to build bgg mapping file")
		if len(mappingFile.Mappings) == 0 && len(evts) == 0 {
			return mappingFile, fmt.Errorf("failed to scan gencon events")
		}
	}

	gcb.Logger.Debug().Msgf("final page of events %d", len(results.Events))
	for _, e := range results.Events {
		if e == nil {
			continue
		}

		key := comboKey{
			system:  e.GameSystem,
			edition: e.RulesEdition,
		}

		if titleCounts[key] == nil {
			titleCounts[key] = make(map[string]int)
		}

		titleCounts[key][e.Title]++
	}

	gcb.Logger.Debug().
		Int("total_entries", len(titleCounts)).
		Msg("titles counted")
	// Mappings maps "GameSystem|RulesEdition" → MappingEntry.
	mappingFile.Mappings = make(map[string]bgg.MappingEntry, len(titleCounts))
	mappingFile.TotalCombos = len(titleCounts)
	missedMatchCount := 0
	for key, titles := range titleCounts {
		total := 0
		for _, n := range titles {
			total += n
		}

		combo := bgg.GenConCombo{
			GameSystem:   key.system,
			RulesEdition: key.edition,
			RepTitle:     mostCommon(titles),
			EventCount:   total,
		}

		matchResult := bgg.Match(combo, corpus)
		if matchResult.BGGID == "" && matchResult.Name == "" {
			missedMatchCount++
			gcb.Logger.Warn().
				Str("game_system", combo.GameSystem).
				Str("rules_edition", combo.RulesEdition).
				Str("rep_title", combo.RepTitle).
				Msg("No match found for gencon event combo")
			continue
		}

		if matchResult.Game == nil {
			missedMatchCount++
			gcb.Logger.Warn().
				Str("game_system", combo.GameSystem).
				Str("rules_edition", combo.RulesEdition).
				Str("rep_title", combo.RepTitle).
				Str("bgg_id", matchResult.BGGID).
				Str("bgg_name", matchResult.Name).
				Msg("cannot create a mapping entry for a match result missing a bgg game")
			continue
		}

		mappingFile.Mappings[combo.MappingKey()] = bgg.MappingEntry{
			BGGID:        matchResult.BGGID,
			BGGName:      matchResult.Name,
			BGGRank:      matchResult.Game.Rank,
			BGGAvgRating: matchResult.Game.Average,
		}
		mappingFile.Matched++
	}

	if missedMatchCount != 0 {
		gcb.Logger.Warn().Msgf("%d gencon events resulted in no BGG match", missedMatchCount)
	}

	return mappingFile, nil
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
