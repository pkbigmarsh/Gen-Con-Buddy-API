package search

import (
	"fmt"
)

// Exists implements the [Term] interface
// to support searching on if a field exists or not.
type Exists struct {
	field string
}

// NewExists creates an exists query
func NewExists(field string) Exists {
	return Exists{field: field}
}

// ToQuery implements the [Term] interface
// converting this Exists query into an OpenSearch query.
func (e Exists) ToQuery() (any, error) {
	if e.field == "" {
		return nil, fmt.Errorf("cannot create an exists query without a field")
	}

	return map[string]any{
		"exists": map[string]any{"field": e.field},
	}, nil
}
