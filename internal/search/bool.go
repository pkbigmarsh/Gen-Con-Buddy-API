package search

import "fmt"

// Bool implements the Term interface
// to support boolean statements
type Bool struct {
	must    []Term
	should  []Term
	mustNot []Term
}

func NewBool() *Bool {
	return &Bool{}
}

func (b *Bool) Must(terms ...Term) *Bool {
	b.must = append(b.must, terms...)
	return b
}

func (b *Bool) Should(terms ...Term) *Bool {
	b.should = append(b.should, terms...)
	return b
}

func (b *Bool) MustNot(terms ...Term) *Bool {
	b.mustNot = append(b.mustNot, terms...)
	return b
}

func (b *Bool) ToQuery() (any, error) {
	if len(b.must) == 0 && len(b.should) == 0 && len(b.mustNot) == 0 {
		return nil, fmt.Errorf("cannot create a bool query without must, should, or mustNot set")
	}

	var (
		mustQueries    []any
		shouldQueries  []any
		mustNotQueries []any
	)

	for _, t := range b.must {
		query, err := t.ToQuery()
		if err != nil {
			return nil, fmt.Errorf("failed to convert search term into a must query: %w", err)
		}

		mustQueries = append(mustQueries, query)
	}

	for _, t := range b.should {
		query, err := t.ToQuery()
		if err != nil {
			return nil, fmt.Errorf("failed to convert search term into a should query: %w", err)
		}

		shouldQueries = append(shouldQueries, query)
	}

	for _, t := range b.mustNot {
		query, err := t.ToQuery()
		if err != nil {
			return nil, fmt.Errorf("failed to convert search term into a mustNot query: %w", err)
		}

		mustNotQueries = append(mustNotQueries, query)
	}

	boolQuery := make(map[string]any)
	if len(mustQueries) != 0 {
		boolQuery["must"] = mustQueries
	}

	if len(shouldQueries) != 0 {
		boolQuery["should"] = shouldQueries
	}

	if len(mustNotQueries) != 0 {
		boolQuery["must_not"] = mustNotQueries
	}

	return map[string]any{
		"bool": boolQuery,
	}, nil
}
