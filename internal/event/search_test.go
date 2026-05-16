package event

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSort(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantField Field
		wantDir   string
		wantErr   bool
	}{
		{
			name:      "valid field asc",
			input:     "startDateTime.asc",
			wantField: StartDateTime,
			wantDir:   "asc",
		},
		{
			name:      "valid field desc",
			input:     "cost.desc",
			wantField: Cost,
			wantDir:   "desc",
		},
		{
			name:      "valid text field asc",
			input:     "title.asc",
			wantField: Title,
			wantDir:   "asc",
		},
		{
			name:    "filter field is rejected",
			input:   "filter.asc",
			wantErr: true,
		},
		{
			name:    "unknown field is rejected",
			input:   "bogus.asc",
			wantErr: true,
		},
		{
			name:    "invalid direction is rejected",
			input:   "title.up",
			wantErr: true,
		},
		{
			name:    "missing direction",
			input:   "title",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "extra dot in direction is rejected",
			input:   "title.asc.extra",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotField, gotDir, err := ParseSort(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantField, gotField)
			require.Equal(t, tt.wantDir, gotDir)
		})
	}
}

func TestParseSorts(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantSorts []SortEntry
		wantErr   bool
	}{
		{
			name:      "single field asc",
			input:     "startDateTime.asc",
			wantSorts: []SortEntry{{Field: StartDateTime, Dir: "asc"}},
		},
		{
			name:  "two fields",
			input: "startDateTime.asc,title.desc",
			wantSorts: []SortEntry{
				{Field: StartDateTime, Dir: "asc"},
				{Field: Title, Dir: "desc"},
			},
		},
		{
			name:  "three fields",
			input: "cost.asc,title.asc,startDateTime.desc",
			wantSorts: []SortEntry{
				{Field: Cost, Dir: "asc"},
				{Field: Title, Dir: "asc"},
				{Field: StartDateTime, Dir: "desc"},
			},
		},
		{
			name:    "empty string returns error",
			input:   "",
			wantErr: true,
		},
		{
			name:    "trailing comma returns error",
			input:   "startDateTime.asc,",
			wantErr: true,
		},
		{
			name:    "leading comma returns error",
			input:   ",startDateTime.asc",
			wantErr: true,
		},
		{
			name:    "invalid field returns error",
			input:   "startDateTime.asc,bogus.asc",
			wantErr: true,
		},
		{
			name:    "invalid direction returns error",
			input:   "startDateTime.up",
			wantErr: true,
		},
		{
			name:  "whitespace around commas is trimmed",
			input: "startDateTime.asc , title.desc",
			wantSorts: []SortEntry{
				{Field: StartDateTime, Dir: "asc"},
				{Field: Title, Dir: "desc"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSorts(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantSorts, got)
		})
	}
}
