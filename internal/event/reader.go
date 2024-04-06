package event

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
)

var (
	headersToFields = map[string]string{
		"year":                   "year",
		"original order":         "original_order",
		"game id":                "game_id",
		"group":                  "group",
		"title":                  "title",
		"short description":      "short_description",
		"long description":       "long_description",
		"event type":             "event_type",
		"game system":            "game_system",
		"rules edition":          "rules_edition",
		"minimum players":        "min_players",
		"maximum players":        "max_players",
		"age required":           "age_required",
		"experience required":    "experience_required",
		"materials provided":     "materials_provided",
		"start date & time":      "start_date_time",
		"duration":               "duration",
		"end date & time":        "end_date_time",
		"gm names":               "gm_names",
		"website":                "website",
		"email":                  "email",
		"tournament?":            "tournament",
		"round number":           "round_number",
		"total rounds":           "total_rounds",
		"minimum play time":      "minimum_play_time",
		"attendee registration?": "attendee_registration",
		"cost $":                 "cost",
		"location":               "location",
		"room name":              "room_name",
		"table number":           "table_number",
		"special category":       "special_category",
		"tickets available":      "tickets_available",
		"last modified":          "last_modified",
		"also runs":              "also_runs",
		"prize":                  "prize",
		"rules complexity":       "rules_complexity",
	}
)

func LoadEventCSV(ctx context.Context, filepath string, logger zerolog.Logger) ([]*Event, error) {
	logger.Info().Msgf("Loading event csv %s", filepath)

	eventFile, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer func() {
		err := eventFile.Close()
		if err != nil {
			logger.Err(err).Msg("Failed to close event csv")
		}
	}()

	eventReader := csv.NewReader(eventFile)
	headers, err := eventReader.Read()
	if err == io.EOF {
		return nil, fmt.Errorf("Event CSV file empty")
	}

	// maps csv record position -> field that they belong to
	indexToFieldMap := make(map[int]string, len(headers))
	for index, header := range headers {
		lcHeader := strings.ToLower(header)
		if field, ok := headersToFields[lcHeader]; ok {
			logger.Debug().Msgf("Mapped csv header [%s] to event field [%s]", header, field)
			indexToFieldMap[index] = field
		} else {
			logger.Warn().Msgf("Failed to find an appropriate field for CSV header %s", header)
		}
	}

	var (
		events []*Event
		row    []string
	)

	for {
		row, err = eventReader.Read()
		if err != nil {
			break
		}
		// logger.Debug().Msgf("Processing event csv row: %v", row)
		newEvent := &Event{}
		for index, value := range row {
			if field, ok := indexToFieldMap[index]; !ok {
				logger.Warn().Msgf("CSV index %d did not match any field", index)
			} else {
				if err := newEvent.SetFieldFromString(field, value); err != nil {
					logger.Err(err).Msgf("Failed to set field [%s] with value [%s], skipping", field, value)
				}
			}
		}

		if newEvent.GameID != "" {
			// logger.Debug().Msgf("Valid Event: %v", newEvent)
			events = append(events, newEvent)
		} else {
			logger.Warn().Msgf("Invalid event row: %v", row)
		}
	}

	logger.Info().Msgf("Parsed %d valid events from file", len(events))

	if err != io.EOF {
		return nil, err
	}

	return events, nil
}
