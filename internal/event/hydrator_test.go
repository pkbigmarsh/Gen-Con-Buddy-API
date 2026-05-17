package event

import (
	"testing"

	"github.com/gencon_buddy_api/internal/bgg"
	"github.com/stretchr/testify/require"
)

func TestHydrateBGG_Hit(t *testing.T) {
	mapping := map[string]bgg.MappingEntry{
		"Wingspan|1st": {BGGID: "266192", BGGRank: 10, BGGAvgRating: 8.1},
	}
	h := NewHydrateBGG(mapping)
	e := &Event{GameSystem: "Wingspan", RulesEdition: "1st"}

	require.NoError(t, h.Hydrate(e))
	require.Equal(t, "266192", e.BggID)
	require.Equal(t, 10, e.BggRank)
	require.InDelta(t, 8.1, e.BggAvgRating, 0.001)
}

func TestHydrateBGG_Miss(t *testing.T) {
	mapping := map[string]bgg.MappingEntry{
		"Wingspan|1st": {BGGID: "266192"},
	}
	h := NewHydrateBGG(mapping)
	e := &Event{GameSystem: "Unknown Game", RulesEdition: "1st"}

	require.NoError(t, h.Hydrate(e))
	require.Empty(t, e.BggID)
	require.Zero(t, e.BggRank)
	require.Zero(t, e.BggAvgRating)
}

func TestHydrateBGG_EmptyMapping(t *testing.T) {
	h := NewHydrateBGG(map[string]bgg.MappingEntry{})
	e := &Event{GameSystem: "Wingspan", RulesEdition: "1st"}

	require.NoError(t, h.Hydrate(e))
	require.Empty(t, e.BggID)
}

func TestHydrateBGG_KeySeparator(t *testing.T) {
	// key is "GameSystem|RulesEdition" — pipe is the separator
	mapping := map[string]bgg.MappingEntry{
		"D&D|5e": {BGGID: "12345"},
	}
	h := NewHydrateBGG(mapping)

	hit := &Event{GameSystem: "D&D", RulesEdition: "5e"}
	require.NoError(t, h.Hydrate(hit))
	require.Equal(t, "12345", hit.BggID)

	miss := &Event{GameSystem: "D&D|5e", RulesEdition: ""}
	require.NoError(t, h.Hydrate(miss))
	require.Empty(t, miss.BggID)
}

func TestHydrateBGG_Name(t *testing.T) {
	require.Equal(t, "HydrateBGG", NewHydrateBGG(nil).Name())
}
