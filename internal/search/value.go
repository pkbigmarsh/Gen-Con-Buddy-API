package search

import (
	"fmt"
	"strings"
)

type GenericValue []any

func NewGenericValue(val string) (GenericValue, error) {
	if val == "" {
		return nil, fmt.Errorf("cannot have an empty generic value")
	}

	var (
		values     GenericValue
		rangeIndex = rangePattern.FindStringIndex(val)
		commaIndex = strings.Index(val, ",")
	)

	if rangeIndex == nil {
		for _, s := range strings.Split(val, ",") {
			values = append(values, s)
		}

		return values, nil
	}

	if commaIndex == -1 {
		return nil, fmt.Errorf("cannot find range value or list value with value string %s", val)
	}

	for val != "" {
		if rangeIndex != nil && rangeIndex[0] == 0 {
			rangeStr := val[rangeIndex[0]:rangeIndex[1]]
			rng, err := NewRange(rangeStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse range value %s", rangeStr)
			}

			values = append(values, rng)

			if rangeIndex[1] == len(val) {
				val = ""
			} else {
				val = val[rangeIndex[1]+1:]
			}
		} else if commaIndex > 0 {
			values = append(values, val[:commaIndex])
			val = val[commaIndex+1:]
		} else if rangeIndex == nil && commaIndex == -1 {
			values = append(values, val)
			val = ""
		}

		rangeIndex = rangePattern.FindStringIndex(val)
		commaIndex = strings.Index(val, ",")
	}

	return values, nil
}
