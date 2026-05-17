package event

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog"
)

// Parser reads in an order list of string fields, and converts those to an event.
// A Parser will collect data validation errors as an aggregate based on the error string.
type Parser interface {
	Parse([]string) (*Event, error)
	DataErrors() map[string]int
}

// HeaderParser implements the parser interface with a defined set of headers.
// The headers are mapped to the internal event fields.
type HeaderParser struct {
	logger zerolog.Logger
	// maps the expected field order to event fields, based on the header order
	indexToFieldMap map[int]string
	dataErrors      map[string]int
}

// NewHeaderedParser instantiates a HeaderedParser
func NewHeaderedParser(logger zerolog.Logger, headers []string) *HeaderParser {
	indexToFieldMap := make(map[int]string, len(headers[0]))
	for index, header := range headers {
		lcHeader := strings.ToLower(header)
		if field, ok := headersToFields[lcHeader]; ok {
			logger.Debug().Msgf("Header [%s] to event field [%s]", header, field)
			indexToFieldMap[index] = field
		} else {
			logger.Warn().Msgf("Failed to find an appropriate field for header %s", header)
		}
	}

	return &HeaderParser{
		logger:          logger,
		indexToFieldMap: indexToFieldMap,
		dataErrors:      make(map[string]int),
	}
}

// Parse the ordered field list into an event
func (h *HeaderParser) Parse(fields []string) (*Event, error) {
	var (
		newEvent = &Event{}
	)

	for index, value := range fields {
		if field, ok := h.indexToFieldMap[index]; !ok {
			h.logger.Debug().Msgf("XLSX index %d did not match any field", index)
		} else {
			if err := newEvent.SetFieldFromString(field, strings.TrimSpace(value)); err != nil {
				// logger.Warn().Str("field", field).Str("value", value).Msg("Validation error")
				err = fmt.Errorf("validation error for field [%s]: %s", field, err)
				h.dataErrors[err.Error()] = h.dataErrors[err.Error()] + 1
			}
		}
	}

	if newEvent.GameID == "" {
		h.logger.Warn().Msgf("Invalid event field set: %v", fields)
		return nil, fmt.Errorf("failed to parse event")
	}

	return newEvent, nil
}

// DataErrosr returns the collected data error aggregations so far
func (h *HeaderParser) DataErrors() map[string]int {
	return h.dataErrors
}
