package event

import "github.com/gencon_buddy_api/internal/bgg"

// Hydrator adds additional information to an event
type Hydrator interface {
	// Hydrate modifies the provided event, adding extra information as applicable
	Hydrate(*Event) error
	// Name of the hydrator, used for logging
	Name() string
}

// HydrateTotalTickets is used when the events are first created to set their initial ticket count.
type HydrateTotalTickets struct{}

// Hydrate ...
func (h HydrateTotalTickets) Hydrate(e *Event) error {
	e.TotalTickets = e.TicketsAvailable
	return nil
}

// Name ...
func (h HydrateTotalTickets) Name() string {
	return "HydrateTotalTickets"
}

// HydrateBGG sets BGG fields on events from a precomputed bgg_mapping.json mapping.
type HydrateBGG struct {
	mapping map[string]bgg.MappingEntry
}

// NewHydrateBGG creates a HydrateBGG from the given "GameSystem|RulesEdition" → MappingEntry map.
func NewHydrateBGG(mapping map[string]bgg.MappingEntry) HydrateBGG {
	return HydrateBGG{mapping: mapping}
}

// Hydrate ...
func (h HydrateBGG) Hydrate(e *Event) error {
	entry, ok := h.mapping[e.GameSystem+"|"+e.RulesEdition]
	if !ok {
		return nil
	}
	e.BggID = entry.BGGID
	e.BggRank = entry.BGGRank
	e.BggAvgRating = entry.BGGAvgRating
	return nil
}

// Name ...
func (h HydrateBGG) Name() string {
	return "HydrateBGG"
}
