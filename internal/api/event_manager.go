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

// GetKeywordFacets returns distinct values and counts for any keyword field or subfield.
func (m EventManager) GetKeywordFacets(ctx context.Context, field string, size int) ([]gcbapi.KeywordFacet, error) {
	facets, err := m.repo.GetKeywordFacets(ctx, field, size)
	if err != nil {
		return nil, err
	}

	result := make([]gcbapi.KeywordFacet, len(facets))
	for i, f := range facets {
		result[i] = gcbapi.KeywordFacet{Value: f.Value, Count: f.Count}
	}
	return result, nil
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
