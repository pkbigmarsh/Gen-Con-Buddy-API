package event

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/xuri/excelize/v2"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

var (
	headersToFields = map[string]string{
		"game id":                    "game_id",
		"group":                      "group",
		"title":                      "title",
		"short description":          "short_description",
		"long description":           "long_description",
		"event type":                 "event_type",
		"game system":                "game_system",
		"rules edition":              "rules_edition",
		"minimum players":            "min_players",
		"maximum players":            "max_players",
		"age required":               "age_required",
		"experience required":        "experience_required",
		"materials required":         "materials_required",
		"materials required details": "materials_required_details",
		"start date & time":          "start_date_time",
		"duration":                   "duration",
		"end date & time":            "end_date_time",
		"gm names":                   "gm_names",
		"website":                    "website",
		"email":                      "email",
		"tournament?":                "tournament",
		"round number":               "round_number",
		"total rounds":               "total_rounds",
		"minimum play time":          "minimum_play_time",
		"attendee registration?":     "attendee_registration",
		"cost $":                     "cost",
		"location":                   "location",
		"room name":                  "room_name",
		"table number":               "table_number",
		"special category":           "special_category",
		"tickets available":          "tickets_available",
		"last modified":              "last_modified",

		// GenCon stopped including these columns in the 2024 export;
		// kept here for backward compatibility with older files.
		"year":               "year",
		"materials provided": "materials_provided",
		"also runs":          "also_runs",
		"prize":              "prize",
		"rules complexity":   "rules_complexity",
		"original order":     "original_order",
	}
)

// Reader reads Events from a source, applying optional hydrators to each event as it is parsed.
type Reader interface {
	ReadEvents(context.Context, ...Hydrator) ([]*Event, error)
	Close() error
}

// NewReaderFromPath returns a Reader for the given filepath based on its extension.
func NewReaderFromPath(logger zerolog.Logger, filepath string) (Reader, error) {
	if strings.HasSuffix(filepath, ".xlsx") {
		return NewXLSXReader(logger, filepath)
	}
	if strings.HasSuffix(filepath, ".csv") {
		return NewCSVReader(logger, filepath)
	}
	return nil, fmt.Errorf("unknown file type in filepath: %s", filepath)
}

// hydrateEvent applies each hydrator to the event, logging but not failing on errors.
func hydrateEvent(logger zerolog.Logger, e *Event, hydrators []Hydrator) {
	for _, h := range hydrators {
		if err := h.Hydrate(e); err != nil {
			logger.Err(err).Str("hydrator", h.Name()).Msg("failed to hydrate event")
		}
	}
}

// CSVReader reads events from a CSV file.
type CSVReader struct {
	logger    zerolog.Logger
	parser    Parser
	file      *os.File
	csvReader *csv.Reader
}

func NewCSVReader(logger zerolog.Logger, filepath string) (*CSVReader, error) {
	logger.Info().Msgf("Loading event csv %s", filepath)

	eventFile, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open csv file: %w", err)
	}

	eventReader := csv.NewReader(transform.NewReader(eventFile, charmap.Windows1252.NewDecoder()))
	headers, err := eventReader.Read()
	if err == io.EOF {
		eventFile.Close()
		return nil, fmt.Errorf("event csv file empty")
	}
	if err != nil {
		eventFile.Close()
		return nil, fmt.Errorf("failed to read csv headers: %w", err)
	}

	return &CSVReader{
		logger:    logger,
		parser:    NewHeaderedParser(logger, headers),
		file:      eventFile,
		csvReader: eventReader,
	}, nil
}

func (c *CSVReader) ReadEvents(ctx context.Context, hydrators ...Hydrator) ([]*Event, error) {
	var (
		events = make([]*Event, 0)
		err    error
		row    []string
	)

	for {
		row, err = c.csvReader.Read()
		if err != nil {
			break
		}
		e, parseErr := c.parser.Parse(row)
		if parseErr != nil {
			c.logger.Err(parseErr).Msg("failed to parse row of csv file")
			continue
		}
		hydrateEvent(c.logger, e, hydrators)
		events = append(events, e)
	}

	for msg, count := range c.parser.DataErrors() {
		c.logger.Warn().Msgf("Found data validation %d times | %s", count, msg)
	}

	if err != io.EOF {
		return nil, err
	}

	c.logger.Info().Msgf("Parsed %d valid events from file", len(events))
	return events, nil
}

func (c *CSVReader) Close() error {
	if c.file != nil {
		return c.file.Close()
	}
	return nil
}

// XLSXReader reads events from an Excel (xlsx) file.
type XLSXReader struct {
	logger zerolog.Logger
	parser Parser
	rows   [][]string // data rows only (header stripped)
}

func NewXLSXReader(logger zerolog.Logger, filepath string) (*XLSXReader, error) {
	logger.Info().Msgf("Loading event file %s", filepath)
	f, err := excelize.OpenFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open excel file: %w", err)
	}
	return newXLSXReader(logger, f)
}

func NewXLSXReaderFromReader(logger zerolog.Logger, r io.Reader) (*XLSXReader, error) {
	logger.Info().Msg("Loading events from io.Reader")
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, fmt.Errorf("failed to open excel file: %w", err)
	}
	return newXLSXReader(logger, f)
}

func newXLSXReader(logger zerolog.Logger, f *excelize.File) (*XLSXReader, error) {
	defer func() {
		if err := f.Close(); err != nil {
			logger.Err(err).Msg("Failed to close event xlsx")
		}
	}()

	if f.SheetCount > 1 {
		logger.Warn().Msgf("expecting a single sheet in the event file, but found [%d]. Using the first sheet.", f.SheetCount)
	}

	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read rows from sheet [%s]: %w", sheetName, err)
	}

	if len(rows) <= 1 {
		return nil, fmt.Errorf("invalid event file, not enough rows: %d", len(rows))
	}

	return &XLSXReader{
		logger: logger,
		parser: NewHeaderedParser(logger, rows[0]),
		rows:   rows[1:],
	}, nil
}

func (x *XLSXReader) ReadEvents(ctx context.Context, hydrators ...Hydrator) ([]*Event, error) {
	events := make([]*Event, 0, len(x.rows))

	for i, row := range x.rows {
		e, parseErr := x.parser.Parse(row)
		if parseErr != nil {
			x.logger.Err(parseErr).Msgf("failed to parse row %d of excel file", i+1)
			continue
		}
		hydrateEvent(x.logger, e, hydrators)
		events = append(events, e)
	}

	for msg, count := range x.parser.DataErrors() {
		x.logger.Warn().Msgf("Found data validation %d times | %s", count, msg)
	}

	x.logger.Info().Msgf("Parsed %d valid events from file", len(events))
	return events, nil
}

// Close is a no-op; the xlsx file is read fully in the constructor.
func (x *XLSXReader) Close() error { return nil }
