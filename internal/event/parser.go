package event

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog"
)

// Parser reads in an ordered list of string fields and converts them to an Event.
// It collects data validation errors as an aggregate keyed by error string.
type Parser interface {
	Parse([]string) (*Event, error)
	DataErrors() map[string]int
}

// HeaderParser implements Parser with a defined set of headers mapped to internal event fields.
type HeaderParser struct {
	logger          zerolog.Logger
	indexToFieldMap map[int]string
	dataErrors      map[string]int
}

func NewHeaderedParser(logger zerolog.Logger, headers []string) *HeaderParser {
	indexToFieldMap := make(map[int]string, len(headers))
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

func (h *HeaderParser) Parse(fields []string) (*Event, error) {
	newEvent := &Event{}
	for index, value := range fields {
		if field, ok := h.indexToFieldMap[index]; !ok {
			h.logger.Debug().Msgf("index %d did not match any field", index)
		} else {
			if err := newEvent.SetFieldFromString(field, value); err != nil {
				err = fmt.Errorf("validation error for field [%s]: %s", field, err)
				h.dataErrors[err.Error()]++
			}
		}
	}
	if newEvent.GameID == "" {
		h.logger.Warn().Msgf("Invalid event field set: %v", fields)
		return nil, fmt.Errorf("failed to parse event")
	}
	return newEvent, nil
}

func (h *HeaderParser) DataErrors() map[string]int {
	return h.dataErrors
}
