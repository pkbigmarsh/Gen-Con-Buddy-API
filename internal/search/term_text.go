package search

import (
	"fmt"
	"strings"
)

// Text implements the Term interface
// to support searching on text fields
type Text struct {
	field  string
	values []string
}

func NewText(field, vals string) (Text, error) {
	text := Text{}
	if field == "" {
		return text, fmt.Errorf("cannot create a text term without a field")
	}

	if vals == "" {
		return text, fmt.Errorf("cannot create a text term on %s without fields", field)
	}

	text.field = field
	text.values = strings.Split(vals, ",")

	return text, nil
}

func (t Text) ToQuery() (any, error) {
	if t.field == "" {
		return nil, fmt.Errorf("cannot create a text query without a field")
	}

	if len(t.values) == 0 {
		return nil, fmt.Errorf("cannot create a text query on %s without and values", t.field)
	}

	return map[string]any{
		"match": map[string]any{t.field: strings.Join(t.values, " ")},
	}, nil
}
