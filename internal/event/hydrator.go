package event

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
