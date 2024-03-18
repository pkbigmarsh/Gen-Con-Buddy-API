package search

import (
	"fmt"
	"strconv"
)

// Number implements the Term interface to support
// search for number values
type Number struct {
	field string
	value GenericValue
}

func NewNumber(field, val string) (Number, error) {
	if field == "" {
		return Number{}, fmt.Errorf("cannot create a number term without a field")
	}

	num := Number{field: field}

	if val == "" {
		num.value = []any{"0"}
		return num, nil
	}

	v, err := NewGenericValue(val)
	if err != nil {
		return Number{}, fmt.Errorf("failed to parse values for number term %s: %w", field, err)
	}

	num.value = v

	return num, nil
}

func (n Number) ToQuery() (any, error) {
	if n.field == "" {
		return nil, fmt.Errorf("cannot build a number query without a field")
	}

	if len(n.value) == 0 {
		return nil, fmt.Errorf("cannot build a number query without any values")
	}

	var shoulds []any

	for _, val := range n.value {
		switch v := val.(type) {
		case string:
			if _, err := strconv.ParseFloat(v, 64); err != nil {
				return nil, fmt.Errorf("cannot build a number query with a non-number value %s: %w", v, err)
			}
			shoulds = append(shoulds, map[string]any{
				"term": map[string]any{n.field: v},
			})
		case Range:
			r, err := numberRangeQuery(n.field, v)
			if err != nil {
				return nil, fmt.Errorf("cannot build a number query with an invalid range: %w", err)
			}

			shoulds = append(shoulds, r)
		}
	}

	return map[string]any{
		"bool": map[string]any{"should": shoulds},
	}, nil
}

func numberRangeQuery(field string, r Range) (any, error) {
	rangeMap := make(map[string]any)

	if r.min != "" {
		if _, err := strconv.ParseFloat(r.min, 64); err != nil {
			return nil, fmt.Errorf("cannot build a string number query with a non-number value %s: %w", r.min, err)
		}

		op := "gt"
		if r.inclusiveMin {
			op += "e"
		}

		rangeMap[op] = r.min
	}

	if r.max != "" {
		if _, err := strconv.ParseFloat(r.max, 64); err != nil {
			return nil, fmt.Errorf("cannot build a string number query with a non-number value %s: %w", r.max, err)
		}
		op := "lt"
		if r.inclusiveMax {
			op += "e"
		}

		rangeMap[op] = r.max
	}

	return map[string]any{
		"range": map[string]any{field: rangeMap},
	}, nil
}
