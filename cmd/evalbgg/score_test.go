package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSimilarityScore(t *testing.T) {
	tests := []struct {
		a, b string
		min  float64 // score must be >= min
		max  float64 // score must be <= max
	}{
		{"axis & allies", "axis & allies", 1.0, 1.0},
		{"", "", 1.0, 1.0},
		{"axis & allies", "completely different", 0.0, 0.3},
		{"axis & allies 1942", "axis & allies 1942", 1.0, 1.0},
		{"wingspan", "wingspam", 0.8, 1.0}, // one char different
	}
	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			score := similarityScore(tt.a, tt.b)
			require.GreaterOrEqual(t, score, tt.min)
			require.LessOrEqual(t, score, tt.max)
		})
	}
}

func TestJaccardScore(t *testing.T) {
	tests := []struct {
		a, b string
		min  float64
		max  float64
	}{
		{"axis allies 1942", "axis allies 1942", 1.0, 1.0},
		{"axis allies", "axis allies 1942", 0.6, 0.8},
		{"wingspan", "ark nova", 0.0, 0.1},
		{"", "", 1.0, 1.0},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			score := jaccardScore(tt.a, tt.b)
			require.GreaterOrEqual(t, score, tt.min)
			require.LessOrEqual(t, score, tt.max)
		})
	}
}
