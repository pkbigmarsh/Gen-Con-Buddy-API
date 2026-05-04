package data

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gencon_buddy_api/cmd/app"
	"github.com/gencon_buddy_api/internal/changelog"
	"github.com/gencon_buddy_api/internal/event"
	"github.com/gencon_buddy_api/internal/search"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wI2L/jsondiff"
	"github.com/xuri/excelize/v2"
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
		if strings.HasSuffix(localFilepath, ".csv") {
			events, err = event.LoadEventCSV(cmd.Context(), localFilepath, gcb.Logger)
		} else if strings.HasSuffix(localFilepath, ".xlsx") {
			events, err = event.LoadEventXLSX(cmd.Context(), localFilepath, gcb.Logger)
		} else {
			return fmt.Errorf("unknown file type in filepath: %s", localFilepath)
		}
	}

	if err != nil {
		gcb.Logger.Err(err).
			Str("download_url", downloadURL).
			Str("local_file", localFilepath).
			Msg("failed to read in the event list for updating")
		return fmt.Errorf("failed to fetch event list for updating: %w", err)
	}

	return processChangeLogEvents(cmd.Context(), gcb, events)
}

func downloadEvents(ctx context.Context, gcb *app.App, downloadURL string) ([]*event.Event, error) {
	gcb.Logger.Info().Str("url", downloadURL).Msg("Fetching events from download page.")

	resp, err := http.Get(downloadURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from [%s]: %w", downloadURL, err)
	}

	if resp == nil {
		return nil, nil
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			gcb.Logger.Err(err).Msg("failed to close the response body")
		}
	}()

	f, err := excelize.OpenReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from [%s]: %w", downloadURL, err)
	}

	return event.ReadEventsFromXLSX(ctx, gcb.Logger, f)
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

	if len(clEntry.CreatedEvents) == 0 && len(clEntry.UpdatedEvents) == 0 && len(clEntry.DeletedEvents) == 0 {
		gcb.Logger.Info().
			Str("change_log_entry_id", clEntry.ID).
			Msg("No events were changed, not creating change log entry")

		// no-op
		return nil
	}

	itemErr, err := gcb.ChangeLogRepo.CreateEntries(ctx, clEntry)
	if err != nil {
		return fmt.Errorf("failed to call opensearch with a create request: %w", err)
	}

	if itemErr != nil {
		return fmt.Errorf("change log entry [%s] failed to be written: %w", clEntry.ID, errors.Join(itemErr...))
	}

	gcb.Logger.Info().
		Int("update_count", len(clEntry.UpdatedEvents)).
		Int("create_count", len(clEntry.CreatedEvents)).
		Int("delete_count", len(clEntry.DeletedEvents)).
		Msgf("Successfully created change log %s", clEntry.ID)

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
