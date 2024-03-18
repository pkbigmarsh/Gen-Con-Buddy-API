package search

import (
	"fmt"
	"time"
)

// Date implements the Term interface to support
// search for date values. All dates without a timezone
// provided are set to America/Indianapolis GMT-4.
type Date struct {
	field string
	value GenericValue
}

func NewDate(field, vals string) (Date, error) {
	date := Date{}

	if field == "" {
		return date, fmt.Errorf("cannot create a date term without a field")
	}

	if vals == "" {
		return date, fmt.Errorf("cannot create a date term on %s without fields", field)
	}

	date.field = field

	v, err := NewGenericValue(vals)
	if err != nil {
		return date, fmt.Errorf("failed to parse values for date term %s: %w", field, err)
	}

	date.value = v

	return date, nil
}

func (d Date) ToQuery() (any, error) {
	if d.field == "" {
		return nil, fmt.Errorf("cannot build date query without a field")
	}

	if len(d.value) == 0 {
		return nil, fmt.Errorf("cannot build date query %s without any values", d.field)
	}

	var items []any
	for _, val := range d.value {
		switch v := val.(type) {
		case string:
			dateStr, err := convertDate(v)
			if err != nil {
				return nil, fmt.Errorf("cannot build a date query for %s with an invalid date: %w", d.field, err)
			}

			items = append(items, map[string]any{
				"term": map[string]any{d.field: dateStr},
			})
		case Range:
			r, err := dateRangeQuery(d.field, v)
			if err != nil {
				return nil, err
			}

			items = append(items, r)
		}
	}

	if len(items) == 1 {
		return items[0], nil
	}

	return map[string]any{
		"bool": map[string]any{"should": items},
	}, nil
}

func convertDate(dateStr string) (string, error) {
	dateTime, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return "", nil
	}

	indy, err := time.LoadLocation("America/Indianapolis")
	if err != nil {
		return "", fmt.Errorf("failed to load indy time zone: %w", err)
	}

	dateTime = dateTime.In(indy)

	return dateTime.Format(time.RFC3339), nil
}

func dateRangeQuery(field string, r Range) (any, error) {
	rangeMap := make(map[string]any)

	if r.min != "" {
		dateStr, err := convertDate(r.min)
		if err != nil {
			return nil, fmt.Errorf("cannot build a date range query with an invalid date %s: %w", r.min, err)
		}

		op := "gt"
		if r.inclusiveMin {
			op += "e"
		}

		rangeMap[op] = dateStr
	}

	if r.max != "" {
		dateStr, err := convertDate(r.min)
		if err != nil {
			return nil, fmt.Errorf("cannot build a date range query with an invalid date %s: %w", r.max, err)
		}

		op := "lt"
		if r.inclusiveMax {
			op += "e"
		}

		rangeMap[op] = dateStr
	}

	return map[string]any{
		"range": map[string]any{field: rangeMap},
	}, nil
}
