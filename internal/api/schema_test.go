package api

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestFacetFieldsHaveKeywordSubfields verifies that every facet field that
// targets a .keyword subfield has that subfield defined in the OpenSearch index
// template. Catches schema/code drift before it reaches production.
func TestFacetFieldsHaveKeywordSubfields(t *testing.T) {
	raw, err := os.ReadFile("../../cmd/data/initialize/schema/event_index_template.json")
	require.NoError(t, err)

	var tmpl struct {
		Mappings struct {
			Properties map[string]struct {
				Fields map[string]json.RawMessage `json:"fields"`
			} `json:"properties"`
		} `json:"mappings"`
	}
	require.NoError(t, json.Unmarshal(raw, &tmpl))

	for displayField, osField := range facetFields {
		if !strings.HasSuffix(osField, ".keyword") {
			continue
		}
		baseField := strings.TrimSuffix(osField, ".keyword")
		prop, ok := tmpl.Mappings.Properties[baseField]
		require.Truef(t, ok,
			"index template missing field %q (required by facet %q)", baseField, displayField)
		_, hasKeyword := prop.Fields["keyword"]
		require.Truef(t, hasKeyword,
			"field %q in index template has no .keyword subfield (required by facet %q)", baseField, displayField)
	}
}
