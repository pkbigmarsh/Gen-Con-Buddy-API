package api

import (
	"context"

	"github.com/gencon_buddy_api/gcbapi"
	"github.com/gencon_buddy_api/internal/event"
	"github.com/rs/zerolog"
)

// EventManager handles the inbetween of internal event interactions and external event shapes
type EventManager struct {
	logger *zerolog.Logger
	repo   *event.EventRepo
}

// NewEventManager instantiates a new EventManager
func NewEventManager(logger *zerolog.Logger, repo *event.EventRepo) EventManager {
	return EventManager{
		logger: logger,
		repo:   repo,
	}
}

// Search for events given the search request
func (m EventManager) Search(ctx context.Context, search event.SearchRequest) (int64, []gcbapi.Event, error) {
	resp, err := m.repo.Search(ctx, search)
	if err != nil {
		return 0, nil, err
	}

	extEvents := make([]gcbapi.Event, len(resp.Events))
	for i, evt := range resp.Events {
		extEvents[i] = evt.Externalize()
	}

	return resp.TotalEvents, extEvents, nil
}
