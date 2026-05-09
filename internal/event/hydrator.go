package event

// Hydrator adds additional information to an event during parsing.
type Hydrator interface {
	Hydrate(*Event) error
	Name() string
}

// HydrateTotalTickets sets the initial total ticket count when events are first created.
type HydrateTotalTickets struct{}

func (h HydrateTotalTickets) Hydrate(e *Event) error {
	e.TotalTickets = e.TicketsAvailable
	return nil
}

func (h HydrateTotalTickets) Name() string { return "HydrateTotalTickets" }

// HydrateBGG applies a pre-built BGG ID mapping to each event.
type HydrateBGG struct {
	mapping map[string]string
}

func NewHydrateBGG(mapping map[string]string) HydrateBGG {
	return HydrateBGG{mapping: mapping}
}

func (h HydrateBGG) Hydrate(e *Event) error {
	if id, ok := h.mapping[bggMappingKey(e.GameSystem, e.RulesEdition)]; ok {
		e.BggID = id
	}
	return nil
}

func (h HydrateBGG) Name() string { return "HydrateBGG" }

// bggMappingKey produces the key used by both HydrateBGG and bgg.LoadMapping.
// Must match bgg.mappingKey — "GameSystem|RulesEdition".
func bggMappingKey(gameSystem, rulesEdition string) string {
	return gameSystem + "|" + rulesEdition
}
