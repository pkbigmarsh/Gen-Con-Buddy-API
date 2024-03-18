package search

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewGenericVa(t *testing.T) {
	tests := []struct {
		name            string
		formattedString string
		wantErr         bool
		wantValues      GenericValue
	}{
		{
			name:            "emtpy string should fail",
			formattedString: "",
			wantErr:         true,
		},
		{
			name:            "single value",
			formattedString: "string",
			wantValues:      []any{"string"},
		},
		{
			name:            "list values",
			formattedString: "1,2,3",
			wantValues:      []any{"1", "2", "3"},
		},
		{
			name:            "single range",
			formattedString: "(a,b)",
			wantValues: []any{Range{
				min: "a",
				max: "b",
			}},
		},
		{
			name:            "list ranges",
			formattedString: "(1,2),(3,4)",
			wantValues: []any{
				Range{
					min: "1",
					max: "2",
				},
				Range{
					min: "3",
					max: "4",
				},
			},
		},
		{
			name:            "mixed types",
			formattedString: "1,(2,3),4,(5,6),(8,9),10",
			wantValues: []any{
				"1",
				Range{
					min: "2",
					max: "3",
				},
				"4",
				Range{
					min: "5",
					max: "6",
				},
				Range{
					min: "8",
					max: "9",
				},
				"10",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValues, gotErr := NewGenericValue(tt.formattedString)
			if tt.wantErr {
				require.Error(t, gotErr)
			} else {
				require.NoError(t, gotErr)
			}

			require.ElementsMatch(t, tt.wantValues, gotValues)
		})
	}
}
