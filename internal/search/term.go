package search

// Term defines a category of supports OpenSearch
// search terms
type Term interface {
	ToQuery() (any, error)
}
