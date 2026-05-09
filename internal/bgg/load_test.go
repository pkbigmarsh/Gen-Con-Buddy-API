package bgg

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeMappingFile(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	f, err := os.CreateTemp("", "bgg_mapping_*.json")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(f.Name()) })
	_, err = f.Write(data)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}

func TestLoadMapping(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		want    map[string]string
		wantErr bool
	}{
		{
			name: "two entries",
			setup: func(t *testing.T) string {
				return writeMappingFile(t, map[string]any{
					"mappings": []any{
						map[string]any{"game_system": "Wingspan", "rules_edition": "1st", "bgg_id": "266192", "bgg_name": "Wingspan"},
						map[string]any{"game_system": "Ark Nova", "rules_edition": "1st", "bgg_id": "342942", "bgg_name": "Ark Nova"},
					},
				})
			},
			want: map[string]string{
				"Wingspan|1st": "266192",
				"Ark Nova|1st": "342942",
			},
		},
		{
			name: "empty mappings array",
			setup: func(t *testing.T) string {
				return writeMappingFile(t, map[string]any{"mappings": []any{}})
			},
			want: map[string]string{},
		},
		{
			name: "file not found",
			setup: func(t *testing.T) string {
				return "/nonexistent/bgg_mapping.json"
			},
			wantErr: true,
		},
		{
			name: "invalid JSON",
			setup: func(t *testing.T) string {
				f, err := os.CreateTemp("", "bgg_mapping_*.json")
				require.NoError(t, err)
				t.Cleanup(func() { os.Remove(f.Name()) })
				f.WriteString("not valid json")
				f.Close()
				return f.Name()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			m, err := LoadMapping(path)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, m)
		})
	}
}
