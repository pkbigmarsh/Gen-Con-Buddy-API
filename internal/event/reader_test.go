package event

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

// csvHeaders matches the keys in headersToFields for the fields used in test rows.
const csvHeaders = "Game ID,Title,Game System,Rules Edition,Tickets Available\n"

func writeCSVFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "events_*.csv")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(f.Name()) })
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}

// makeXLSXBytes returns an xlsx file as bytes with the given headers and rows.
func makeXLSXBytes(t *testing.T, headers []string, rows [][]string) []byte {
	t.Helper()
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	for col, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		require.NoError(t, f.SetCellValue(sheet, cell, h))
	}
	for rowIdx, row := range rows {
		for col, val := range row {
			cell, _ := excelize.CoordinatesToCellName(col+1, rowIdx+2)
			require.NoError(t, f.SetCellValue(sheet, cell, val))
		}
	}
	var buf bytes.Buffer
	require.NoError(t, f.Write(&buf))
	return buf.Bytes()
}

func writeXLSXFile(t *testing.T, headers []string, rows [][]string) string {
	t.Helper()
	data := makeXLSXBytes(t, headers, rows)
	f, err := os.CreateTemp("", "events_*.xlsx")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(f.Name()) })
	_, err = f.Write(data)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}

// --- CSVReader ---

func TestCSVReader_readEvents(t *testing.T) {
	path := writeCSVFile(t, csvHeaders+
		"BGM24001,Wingspan,Wingspan,1st,4\n"+
		"BGM24002,Ark Nova,Ark Nova,1st,8\n")

	r, err := NewCSVReader(zerolog.Nop(), path)
	require.NoError(t, err)
	defer r.Close()

	events, err := r.ReadEvents(context.Background())
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Equal(t, "BGM24001", events[0].GameID)
	require.Equal(t, "Wingspan", events[0].GameSystem)
	require.Equal(t, "BGM24002", events[1].GameID)
}

func TestCSVReader_appliesHydrators(t *testing.T) {
	path := writeCSVFile(t, csvHeaders+"BGM24001,Wingspan,Wingspan,1st,4\n")

	r, err := NewCSVReader(zerolog.Nop(), path)
	require.NoError(t, err)
	defer r.Close()

	events, err := r.ReadEvents(context.Background(), HydrateTotalTickets{})
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, events[0].TicketsAvailable, events[0].TotalTickets)
}

func TestCSVReader_bggHydrator(t *testing.T) {
	path := writeCSVFile(t, csvHeaders+
		"BGM24001,Wingspan,Wingspan,1st,4\n"+
		"BGM24002,Ark Nova,Ark Nova,1st,8\n")

	r, err := NewCSVReader(zerolog.Nop(), path)
	require.NoError(t, err)
	defer r.Close()

	events, err := r.ReadEvents(context.Background(), NewHydrateBGG(map[string]string{
		"Wingspan|1st": "266192",
	}))
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Equal(t, "266192", events[0].BggID)
	require.Equal(t, "", events[1].BggID)
}

func TestCSVReader_excludesRowsRejectedByParser(t *testing.T) {
	// Rows where the parser cannot produce a valid Event (e.g. missing Game ID)
	// are logged and skipped; valid rows on either side are unaffected.
	path := writeCSVFile(t, csvHeaders+
		"BGM24001,Wingspan,Wingspan,1st,4\n"+
		",No ID,Game,1st,4\n"+
		"BGM24003,Ark Nova,Ark Nova,1st,8\n")

	r, err := NewCSVReader(zerolog.Nop(), path)
	require.NoError(t, err)
	defer r.Close()

	events, err := r.ReadEvents(context.Background())
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Equal(t, "BGM24001", events[0].GameID)
	require.Equal(t, "BGM24003", events[1].GameID)
}

func TestCSVReader_multipleHydratorsCompose(t *testing.T) {
	// Both hydrators must apply without stomping each other's fields.
	path := writeCSVFile(t, csvHeaders+"BGM24001,Wingspan,Wingspan,1st,4\n")

	r, err := NewCSVReader(zerolog.Nop(), path)
	require.NoError(t, err)
	defer r.Close()

	events, err := r.ReadEvents(context.Background(),
		HydrateTotalTickets{},
		NewHydrateBGG(map[string]string{"Wingspan|1st": "266192"}),
	)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, events[0].TicketsAvailable, events[0].TotalTickets)
	require.Equal(t, "266192", events[0].BggID)
}

func TestNewCSVReader_emptyFile(t *testing.T) {
	path := writeCSVFile(t, "")
	_, err := NewCSVReader(zerolog.Nop(), path)
	require.Error(t, err)
}

func TestNewCSVReader_fileNotFound(t *testing.T) {
	_, err := NewCSVReader(zerolog.Nop(), "/nonexistent/events.csv")
	require.Error(t, err)
}

// --- XLSXReader ---

var xlsxHeaders = []string{"Game ID", "Title", "Game System", "Rules Edition", "Tickets Available"}

func TestXLSXReader_readEvents(t *testing.T) {
	path := writeXLSXFile(t, xlsxHeaders, [][]string{
		{"BGM24001", "Wingspan", "Wingspan", "1st", "4"},
		{"BGM24002", "Ark Nova", "Ark Nova", "1st", "8"},
	})

	r, err := NewXLSXReader(zerolog.Nop(), path)
	require.NoError(t, err)

	events, err := r.ReadEvents(context.Background())
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Equal(t, "BGM24001", events[0].GameID)
	require.Equal(t, "BGM24002", events[1].GameID)
}

func TestXLSXReader_appliesHydrators(t *testing.T) {
	path := writeXLSXFile(t, xlsxHeaders, [][]string{
		{"BGM24001", "Wingspan", "Wingspan", "1st", "4"},
	})

	r, err := NewXLSXReader(zerolog.Nop(), path)
	require.NoError(t, err)

	events, err := r.ReadEvents(context.Background(), HydrateTotalTickets{})
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, events[0].TicketsAvailable, events[0].TotalTickets)
}

func TestNewXLSXReaderFromReader(t *testing.T) {
	data := makeXLSXBytes(t, xlsxHeaders, [][]string{
		{"BGM24001", "Wingspan", "Wingspan", "1st", "4"},
	})

	r, err := NewXLSXReaderFromReader(zerolog.Nop(), bytes.NewReader(data))
	require.NoError(t, err)

	events, err := r.ReadEvents(context.Background())
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "BGM24001", events[0].GameID)
}

func TestXLSXReader_multipleHydratorsCompose(t *testing.T) {
	path := writeXLSXFile(t, xlsxHeaders, [][]string{
		{"BGM24001", "Wingspan", "Wingspan", "1st", "4"},
	})

	r, err := NewXLSXReader(zerolog.Nop(), path)
	require.NoError(t, err)

	events, err := r.ReadEvents(context.Background(),
		HydrateTotalTickets{},
		NewHydrateBGG(map[string]string{"Wingspan|1st": "266192"}),
	)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, events[0].TicketsAvailable, events[0].TotalTickets)
	require.Equal(t, "266192", events[0].BggID)
}

func TestNewXLSXReader_headerOnlyFileReturnsError(t *testing.T) {
	path := writeXLSXFile(t, xlsxHeaders, nil)
	_, err := NewXLSXReader(zerolog.Nop(), path)
	require.Error(t, err)
}

func TestNewXLSXReader_fileNotFound(t *testing.T) {
	_, err := NewXLSXReader(zerolog.Nop(), "/nonexistent/events.xlsx")
	require.Error(t, err)
}

// --- NewReaderFromPath ---

func TestNewReaderFromPath(t *testing.T) {
	csvPath := writeCSVFile(t, csvHeaders+"BGM24001,Wingspan,Wingspan,1st,4\n")
	xlsxPath := writeXLSXFile(t, xlsxHeaders, [][]string{{"BGM24001", "Wingspan", "Wingspan", "1st", "4"}})

	tests := []struct {
		name        string
		path        string
		wantType    string
		wantErr     bool
	}{
		{"csv file", csvPath, "csv", false},
		{"xlsx file", xlsxPath, "xlsx", false},
		{"unknown extension", "events.tsv", "", true},
		{"no extension", "events", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewReaderFromPath(zerolog.Nop(), tt.path)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			defer r.Close()
			switch tt.wantType {
			case "csv":
				require.IsType(t, &CSVReader{}, r)
			case "xlsx":
				require.IsType(t, &XLSXReader{}, r)
			}
		})
	}
}
