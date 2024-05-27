package search

import (
	"fmt"
)

// Prefix implements the Term interface
// to support prefix searching on text fields
type Prefix struct {
	field string
	value string
}

func NewPrefix(field, val string) (Prefix, error) {
	prefix := Prefix{}
	if field == "" {
		return prefix, fmt.Errorf("cannot create a prefix term without a field")
	}

	prefix.field = field
	prefix.value = val

	return prefix, nil
}

func (t Prefix) ToQuery() (any, error) {
	if t.field == "" {
		return nil, fmt.Errorf("cannot create a prefix query without a field")
	}

	return map[string]any{
		"prefix": map[string]any{t.field: t.value},
	}, nil
}
