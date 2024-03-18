package search

import (
	"fmt"
	"strings"
)

// Keyword implements the Term interface
// to support searching on keyword fields
type Keyword struct {
	field  string
	values []string
}

func NewKeyword(field, vals string) (Keyword, error) {
	keyword := Keyword{}
	if field == "" {
		return keyword, fmt.Errorf("cannot create a keyword term without a field")
	}

	if vals == "" {
		return keyword, fmt.Errorf("cannot create a keyword term on %s without fields", field)
	}

	keyword.field = field
	keyword.values = strings.Split(vals, ",")

	return keyword, nil
}

func (k Keyword) ToQuery() (any, error) {
	if k.field == "" {
		return nil, fmt.Errorf("cannot create a keyword query without a field")
	}

	if len(k.values) == 0 {
		return nil, fmt.Errorf("cannot create a keyword query on %s without and values", k.field)
	}

	if len(k.values) == 1 {
		return map[string]any{
			"term": map[string]any{k.field: k.values[0]},
		}, nil
	}

	return map[string]any{
		"terms": map[string]any{k.field: k.values},
	}, nil
}
