package event

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHydrateTotalTickets(t *testing.T) {
	tests := []struct {
		name      string
		available int64
		existing  int64
		want      int64
	}{
		{"sets from available", 4, 0, 4},
		{"overwrites existing value", 8, 4, 8},
		{"zero available", 0, 10, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Event{TicketsAvailable: tt.available, TotalTickets: tt.existing}
			require.NoError(t, HydrateTotalTickets{}.Hydrate(e))
			require.Equal(t, tt.want, e.TotalTickets)
		})
	}
}

func TestHydrateTotalTickets_Name(t *testing.T) {
	require.Equal(t, "HydrateTotalTickets", HydrateTotalTickets{}.Name())
}

func TestHydrateBGG(t *testing.T) {
	mapping := map[string]string{
		"Wingspan|1st":  "266192",
		"Ark Nova|2023": "342942",
	}
	h := NewHydrateBGG(mapping)

	tests := []struct {
		name         string
		gameSystem   string
		rulesEdition string
		wantBggID    string
	}{
		{"exact match", "Wingspan", "1st", "266192"},
		{"match with year edition", "Ark Nova", "2023", "342942"},
		{"no match — wrong edition", "Wingspan", "2nd", ""},
		{"no match — unknown game", "Ticket to Ride", "1st", ""},
		{"empty mapping key", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Event{GameSystem: tt.gameSystem, RulesEdition: tt.rulesEdition}
			require.NoError(t, h.Hydrate(e))
			require.Equal(t, tt.wantBggID, e.BggID)
		})
	}
}

func TestHydrateBGG_Name(t *testing.T) {
	require.Equal(t, "HydrateBGG", NewHydrateBGG(nil).Name())
}

func TestHydrateBGG_doesNotClearExistingBggID(t *testing.T) {
	// An event already enriched by a previous pass should not lose its ID.
	e := &Event{GameSystem: "Unknown", RulesEdition: "1st", BggID: "12345"}
	require.NoError(t, NewHydrateBGG(map[string]string{}).Hydrate(e))
	require.Equal(t, "12345", e.BggID)
}
