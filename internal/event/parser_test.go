package event

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestHeaderParser_validRow(t *testing.T) {
	p := NewHeaderedParser(zerolog.Nop(), []string{"Game ID", "Title", "Game System", "Rules Edition"})

	e, err := p.Parse([]string{"BGM24001", "Wingspan", "Wingspan", "1st"})
	require.NoError(t, err)
	require.Equal(t, "BGM24001", e.GameID)
	require.Equal(t, "Wingspan", e.Title)
	require.Equal(t, "Wingspan", e.GameSystem)
	require.Equal(t, "1st", e.RulesEdition)
}

func TestHeaderParser_missingGameIDColumn(t *testing.T) {
	// No "Game ID" header → every row will have an empty GameID → error.
	p := NewHeaderedParser(zerolog.Nop(), []string{"Title", "Game System"})

	_, err := p.Parse([]string{"Wingspan", "Wingspan"})
	require.Error(t, err)
}

func TestHeaderParser_emptyGameID(t *testing.T) {
	p := NewHeaderedParser(zerolog.Nop(), []string{"Game ID", "Title"})

	_, err := p.Parse([]string{"", "Wingspan"})
	require.Error(t, err)
}

func TestHeaderParser_unknownHeadersIgnored(t *testing.T) {
	// Unknown headers should not prevent parsing of known fields.
	p := NewHeaderedParser(zerolog.Nop(), []string{"Game ID", "Totally Unknown Column"})

	e, err := p.Parse([]string{"BGM24001", "ignored value"})
	require.NoError(t, err)
	require.Equal(t, "BGM24001", e.GameID)
}

func TestHeaderParser_dataErrorsAggregate(t *testing.T) {
	// "Minimum Players" expects an integer; the same invalid value across multiple
	// rows should aggregate under one error string rather than producing N entries.
	p := NewHeaderedParser(zerolog.Nop(), []string{"Game ID", "Minimum Players"})

	_, _ = p.Parse([]string{"BGM24001", "not-a-number"})
	_, _ = p.Parse([]string{"BGM24002", "not-a-number"})

	errs := p.DataErrors()
	require.Len(t, errs, 1) // one distinct error string
	for _, count := range errs {
		require.Equal(t, 2, count) // seen on two rows
	}
}

func TestHeaderParser_shortRow(t *testing.T) {
	// A row shorter than the header list should not panic.
	p := NewHeaderedParser(zerolog.Nop(), []string{"Game ID", "Title", "Game System"})

	e, err := p.Parse([]string{"BGM24001"})
	require.NoError(t, err)
	require.Equal(t, "BGM24001", e.GameID)
	require.Equal(t, "", e.Title)
}
