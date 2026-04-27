package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/gencon_buddy_api/cmd/app"
	"github.com/gencon_buddy_api/internal/changelog"
	"github.com/gencon_buddy_api/internal/event"
	"github.com/gencon_buddy_api/internal/search"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wI2L/jsondiff"
)

const (
	flagDownloadURL = "download_url"
	// defaultDownloadURL for the gencon event downloads.
	// Found on https://www.gencon.com/gen-con-indy/how-to-find-events.
	defaultDownloadURL = "https://www.gencon.com/downloads/events.zip"
)

var (
	UpdateCmd = &cobra.Command{
		Use:   "update",
		Short: "Fetch and update the latest events from GenCon",
		Long:  "Fetches the latestevent data from gencon, unzips the data, updates existing events, inserts new events, and deletes removed events",
		RunE:  update,
	}
)

func init() {
	UpdateCmd.Flags().String(flagDownloadURL, defaultDownloadURL, "Remote url to download the GenCon events from. Default value is [https://www.gencon.com/downloads/events.zip].")
	viper.BindPFlag("DOWNLOAD_URL", UpdateCmd.Flags().Lookup(flagDownloadURL))
}

func update(cmd *cobra.Command, _ []string) error {
	downloadURL, err := cmd.Flags().GetString(flagDownloadURL)
	if err != nil {
		return fmt.Errorf("failed to read %s flag: %w", flagDownloadURL, err)
	}

	localFilepath, err := cmd.Flags().GetString(filepathFlag)
	if err != nil {
		return fmt.Errorf("failed to read %s flag: %w", filepathFlag, err)
	}

	if downloadURL != defaultDownloadURL && localFilepath != "" {
		return fmt.Errorf("cannot use both --download_url and --local_file")
	}

	gcb := app.GetAppFromContext(cmd.Context())
	if gcb == nil {
		return fmt.Errorf("failed to load gcp app context")
	}

	var events []*event.Event

	if downloadURL != defaultDownloadURL {
		events, err = downloadEvents(cmd.Context(), gcb, downloadURL)
	} else {
		events, err = event.LoadEventCSV(cmd.Context(), localFilepath, gcb.Logger)
	}

	if err != nil {
		gcb.Logger.Err(err).
			Str("download_url", downloadURL).
			Str("local_file", localFilepath).
			Msg("failed to read in the event list for updating")
		return fmt.Errorf("failed to fetch event list for updating: %w", err)
	}

	bggMapping := loadBGGMapping(cmd, gcb.Logger)
	for _, e := range events {
		if id, ok := bggMapping[e.GameSystem+"|"+e.RulesEdition]; ok {
			e.BggID = id
		}
	}

	return processChangeLogEvents(cmd.Context(), gcb, events)
}

// loadBGGMapping reads the mapping file produced by match-bgg and returns a
// map keyed by "GameSystem|RulesEdition" → BGG ID string.
// If the file does not exist, it logs a warning and returns an empty map.
func loadBGGMapping(cmd *cobra.Command, logger zerolog.Logger) map[string]string {
	path, err := cmd.Flags().GetString("bgg-mapping")
	if err != nil {
		logger.Warn().Err(err).Msg("failed to read bgg-mapping flag; events will have no bggId")
		return map[string]string{}
	}
	if path == "" {
		return map[string]string{}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		logger.Warn().Str("path", path).Msg("bgg mapping file not found; events will have no bggId")
		return map[string]string{}
	}

	var file struct {
		Mappings []struct {
			GameSystem   string `json:"game_system"`
			RulesEdition string `json:"rules_edition"`
			BGGID        string `json:"bgg_id"`
		} `json:"mappings"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		logger.Warn().Err(err).Str("path", path).Msg("failed to parse bgg mapping file; events will have no bggId")
		return map[string]string{}
	}

	m := make(map[string]string, len(file.Mappings))
	for _, e := range file.Mappings {
		m[e.GameSystem+"|"+e.RulesEdition] = e.BGGID
	}
	return m
}

func downloadEvents(ctx context.Context, gcb *app.App, downloadURL string) ([]*event.Event, error) {
	// TODO when the link is available
	return []*event.Event{}, nil
}

func processChangeLogEvents(ctx context.Context, gcb *app.App, eventList []*event.Event) error {
	clEntry := changelog.NewEntry()

	gcb.Logger.Info().
		Str("change_log_entry_id", clEntry.ID).
		Str("change_log_date", clEntry.Date).
		Int("event_count", len(eventList)).
		Msg("Creating new change log entry")

	var (
		count     = 0
		fetchList []string
		batchMap  = make(map[string]*event.Event)
	)

	for _, e := range eventList {
		e.LastChangeLogModification = clEntry.ID
		fetchList = append(fetchList, e.GameID)
		batchMap[e.GameID] = e
		count++

		if count >= gcb.BatchSize {
			if err := processChangeLogBatch(ctx, gcb, clEntry, fetchList, batchMap); err != nil {
				gcb.Logger.Warn().
					Err(err).
					Str("change_log_entry_id", clEntry.ID).
					Msg("failed to process a batch of changes")
			}

			count = 0
			fetchList = []string{}
			batchMap = make(map[string]*event.Event)
		}
	}

	if count > 0 {
		if err := processChangeLogBatch(ctx, gcb, clEntry, fetchList, batchMap); err != nil {
			gcb.Logger.Warn().
				Err(err).
				Str("change_log_entry_id", clEntry.ID).
				Msg("failed to process a batch of changes")
		}
	}

	// wait for OS refresh window
	// definitely hacky, but I don't want to add a waitfor configuration option right now
	time.Sleep(time.Second * 2)

	if err := processChangeLogDeletions(ctx, gcb, clEntry); err != nil {
		gcb.Logger.Warn().
			Err(err).
			Str("change_log_entry_id", clEntry.ID).
			Msg("failed to mark deleted events as deleted")
	}

	itemErr, err := gcb.ChangeLogRepo.CreateEntries(ctx, clEntry)
	if err != nil {
		return fmt.Errorf("failed to call opensearch with a create request: %w", err)
	}

	if itemErr != nil {
		return fmt.Errorf("change log entry [%s] failed to be written: %w", clEntry.ID, errors.Join(itemErr...))
	}

	return nil
}

func processChangeLogBatch(ctx context.Context, gcb *app.App, clEntry *changelog.Entry, eventIds []string, eventBatch map[string]*event.Event) error {
	var (
		writeEvents  []*event.Event
		updateEvents []*event.Event
	)

	fetchedEvents, err := gcb.EventRepo.FetchEvents(ctx, eventIds...)
	if err != nil {
		return err
	}

	for id := range fetchedEvents.Missing {
		if e, ok := eventBatch[id]; !ok {
			gcb.Logger.Warn().
				Str("game_id", id).
				Str("change_log_entry_id", clEntry.ID).
				Msg("Event listed in list to fetch, but not the batch of events provided. Skipping")
		} else {
			writeEvents = append(writeEvents, e)
			clEntry.CreatedEvents = append(clEntry.CreatedEvents, id)
		}
	}

	if len(writeEvents) > 0 {
		createErrs, reqErr := gcb.EventRepo.CreateEvents(ctx, writeEvents)
		if reqErr != nil {
			return fmt.Errorf("failed to create the new events in the change log entry: %w", reqErr)
		}

		if len(createErrs) != 0 {
			gcb.Logger.Warn().
				Err(errors.Join(createErrs...)).
				Str("change_log_entry_id", clEntry.ID).
				Msg("Failed to create some new events for change log")
		}
	}

	for id, e := range fetchedEvents.Found {
		updateEvent, ok := eventBatch[id]
		if !ok {
			gcb.Logger.Warn().
				Str("game_id", id).
				Str("change_log_entry_id", clEntry.ID).
				Msg("Event listed in list to fetch, but not the batch of events provided. Skipping")

			continue
		}

		p, err := jsondiff.Compare(e, updateEvent, jsondiff.Ignores(event.EventJsonCmpIgnoredFields...))
		if err != nil {
			gcb.Logger.Err(err).Msg("failed to call jsondiff")
		}

		if len(p) > 0 {
			// Only include the entry as an update, if some field changed
			clEntry.UpdatedEvents = append(clEntry.UpdatedEvents, id)
		}

		// update every event to always set the lastChangeLogModification
		// to latest. This lets us search for what events were deleted.
		updateEvents = append(updateEvents, updateEvent)
	}

	if len(updateEvents) > 0 {
		updateErrs, reqErr := gcb.EventRepo.UpdateEvents(ctx, updateEvents)
		if reqErr != nil {
			return fmt.Errorf("failed to update the new events in the change log entry: %w", reqErr)
		}

		if len(updateErrs) != 0 {
			gcb.Logger.Warn().
				Err(errors.Join(updateErrs...)).
				Str("change_log_entry_id", clEntry.ID).
				Msg("Failed to create some new events for change log")
		}
	}

	return nil
}

func processChangeLogDeletions(ctx context.Context, gcb *app.App, clEntry *changelog.Entry) error {
	changeLogSearchTerm, err := event.NewSearchField(string(event.LastChangeLogModification), clEntry.ID)
	if err != nil {
		return fmt.Errorf("could not build search term for change log id: %w", err)
	}

	alreadyDeletedSearchTerm, err := event.NewSearchField(string(event.Deleted), "true")
	if err != nil {
		return fmt.Errorf("cound not build deleted search term: %w", err)
	}

	// find events that have not been modified by the current change log
	// and are not already deleted.
	deletedEventsSearchRequest := event.SearchRequest{
		Terms: []search.Term{
			search.NewBool().MustNot(
				changeLogSearchTerm,
				alreadyDeletedSearchTerm,
			),
		},
		Page:  0,
		Limit: 1000,
	}

	var (
		results      event.SearchResponse
		deleteEvents []*event.Event
	)

	results, err = gcb.EventRepo.Search(Cmd.Context(), deletedEventsSearchRequest)

	for err == nil && len(results.Events) == 1000 {
		for _, e := range results.Events {
			e.Deleted = true
			e.LastChangeLogModification = clEntry.ID
			deleteEvents = append(deleteEvents, e)
			clEntry.DeletedEvents = append(clEntry.DeletedEvents, e.GameID)
		}

		deletedEventsSearchRequest.Page += 1
		results, err = gcb.EventRepo.Search(Cmd.Context(), deletedEventsSearchRequest)
	}

	if err != nil {
		gcb.Logger.Warn().Err(err).
			Str("change_log_entry_id", clEntry.ID).
			Msg("failed to search for events that need deleting")
	}

	for _, e := range results.Events {
		e.Deleted = true
		e.LastChangeLogModification = clEntry.ID
		deleteEvents = append(deleteEvents, e)
		clEntry.DeletedEvents = append(clEntry.DeletedEvents, e.GameID)
	}

	if len(deleteEvents) == 0 {
		// nothing to update, short circuit
		return nil
	}

	writeErrs, err := gcb.EventRepo.UpdateEvents(ctx, deleteEvents)
	if err != nil {
		gcb.Logger.Warn().Err(err).
			Str("change_log_entry_id", clEntry.ID).
			Msg("failed to add the deleted flag to the deleted events")
	}

	if len(writeErrs) > 0 {
		gcb.Logger.Warn().Err(err).
			Str("change_log_entry_id", clEntry.ID).
			Msg("failed to add the deleted flag to specific events")
	}

	return nil
}
