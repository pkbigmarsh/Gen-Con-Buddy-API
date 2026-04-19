package event

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewSortFromString(t *testing.T) {
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
			gotField, gotDir, err := NewSortFromString(tt.input)
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
