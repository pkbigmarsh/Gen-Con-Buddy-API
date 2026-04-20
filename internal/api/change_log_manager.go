package api

import (
	"context"
	"fmt"

	"github.com/gencon_buddy_api/gcbapi"
	"github.com/gencon_buddy_api/internal/changelog"
	"github.com/gencon_buddy_api/internal/event"
	"github.com/rs/zerolog"
)

// ChangeLogManger handles the inbetween of internal change log entries
// and external change log entries
type ChangeLogManager struct {
	logger        *zerolog.Logger
	changeLogRepo *changelog.Repo
	eventRepo     *event.EventRepo
}

// NewChangeLogManager instantiates a new [ChangeLogManager]
func NewChangeLogManager(loger *zerolog.Logger, changeLogRepo *changelog.Repo, eventRepo *event.EventRepo) ChangeLogManager {
	return ChangeLogManager{
		logger:        loger,
		changeLogRepo: changeLogRepo,
		eventRepo:     eventRepo,
	}
}

// ListChangeLogSummaries fetches a number of change log entries, and summarizes them before returning
func (m ChangeLogManager) ListChangeLogSummaries(ctx context.Context, numEntries int) ([]gcbapi.ChangeLogSummary, error) {
	entries, err := m.changeLogRepo.List(ctx, changelog.ListEntriesRequest{
		Limit: numEntries,
	})

	if err != nil {
		return nil, err
	}

	summaries := make([]gcbapi.ChangeLogSummary, len(entries))
	for i, e := range entries {
		summaries[i] = gcbapi.ChangeLogSummary{
			ID:           e.ID,
			Date:         e.Date,
			UpdatedCount: len(e.UpdatedEvents),
			DeletedCount: len(e.DeletedEvents),
			CreatedCount: len(e.CreatedEvents),
		}
	}

	return summaries, nil
}

// FetchChangeLogEntry fetches the desired change log and hydrates the event data
func (m ChangeLogManager) FetchChangeLogEntry(ctx context.Context, id string) (gcbapi.ChangeLogEntry, error) {
	fetchResponse, err := m.changeLogRepo.FetchEntries(ctx, id)
	if err != nil {
		return gcbapi.ChangeLogEntry{}, err
	}

	if len(fetchResponse.Missing) > 0 {
		return gcbapi.ChangeLogEntry{}, fmt.Errorf("could not find change log entry [%s]", id)
	}

	if len(fetchResponse.Found) != 1 {
		m.logger.Warn().Msgf("expected a single change log entry for id [%s], instead found %d", id, len(fetchResponse.Found))
	}

	entry, ok := fetchResponse.Found[id]
	if !ok {
		return gcbapi.ChangeLogEntry{}, fmt.Errorf("found results for change log entry id [%s], but no entry in result map", id)
	}

	respEntry := gcbapi.ChangeLogEntry{
		ID:            id,
		Date:          entry.Date,
		UpdatedEvents: make([]gcbapi.Event, 0, len(entry.UpdatedEvents)),
		DeletedEvents: make([]gcbapi.Event, 0, len(entry.DeletedEvents)),
		CreatedEvents: make([]gcbapi.Event, 0, len(entry.CreatedEvents)),
	}

	var eventFetchResponse event.FetchEventsResponse
	if len(entry.CreatedEvents) > 0 {
		eventFetchResponse, err = m.eventRepo.FetchEvents(ctx, entry.CreatedEvents...)
		if err != nil {
			return respEntry, fmt.Errorf("failed to fetch created events for change log [%s]: %w", id, err)
		}

		for _, id := range entry.CreatedEvents {
			e, ok := eventFetchResponse.Found[id]
			if !ok {
				continue
			}

			if e == nil {
				m.logger.Warn().
					Str("event_id", id).
					Msg("event repo returned nil for created event, skipping")
				continue
			}

			respEntry.CreatedEvents = append(respEntry.CreatedEvents, e.Externalize())
		}
	}

	if len(entry.UpdatedEvents) > 0 {
		eventFetchResponse, err = m.eventRepo.FetchEvents(ctx, entry.UpdatedEvents...)
		if err != nil {
			return respEntry, fmt.Errorf("failed to fetch updated events for change log [%s]: %w", id, err)
		}

		for _, id := range entry.UpdatedEvents {
			e, ok := eventFetchResponse.Found[id]
			if !ok {
				continue
			}

			if e == nil {
				m.logger.Warn().
					Str("event_id", id).
					Msg("event repo returned nil for updated event, skipping")
				continue
			}

			respEntry.UpdatedEvents = append(respEntry.UpdatedEvents, e.Externalize())
		}
	}

	if len(entry.DeletedEvents) > 0 {
		eventFetchResponse, err = m.eventRepo.FetchEvents(ctx, entry.DeletedEvents...)
		if err != nil {
			return respEntry, fmt.Errorf("failed to fetch deleted events for change log [%s]: %w", id, err)
		}

		for _, id := range entry.DeletedEvents {
			e, ok := eventFetchResponse.Found[id]
			if !ok {
				continue
			}

			if e == nil {
				m.logger.Warn().
					Str("event_id", id).
					Msg("event repo returned nil for deleted event, skipping")
				continue
			}

			respEntry.DeletedEvents = append(respEntry.DeletedEvents, e.Externalize())
		}
	}

	return respEntry, nil
}
