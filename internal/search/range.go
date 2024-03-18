package search

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	rangePattern = regexp.MustCompile(`[[(](.*?,.+?|.+?,.*?)[\])]`)
)

type Range struct {
	min          string
	inclusiveMin bool
	max          string
	inclusiveMax bool
}

func NewRange(v string) (Range, error) {
	if !rangePattern.MatchString(v) {
		return Range{}, fmt.Errorf("range value must have 1 or 2 comma seperated values, bounded by inclusive [] or exclusive brackets, but got %s", v)
	}

	values := strings.Split(v[1:len(v)-1], ",")

	r := Range{
		min:          values[0],
		inclusiveMin: v[0] == '[',
		inclusiveMax: v[len(v)-1] == ']',
	}

	if len(values) > 1 {
		r.max = values[1]
	}

	return r, nil
}
