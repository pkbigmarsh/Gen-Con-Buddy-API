package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Axis & Allies", "axis & allies"},
		{"Axis & Allies: 1942", "axis & allies 1942"},  // colon becomes space, collapsed
		{"Ticket to Ride!", "ticket to ride"},
		{"  Extra   Spaces  ", "extra spaces"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			require.Equal(t, tt.want, normalize(tt.input))
		})
	}
}

func TestIsInformativeEdition(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"1st", false},
		{"2nd", false},
		{"Global 1942 2nd", true},
		{"Prison Outbreak", true},
		{"1941", true},
		{"Revised", false},
		{"Classic", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			require.Equal(t, tt.want, isInformativeEdition(tt.input))
		})
	}
}

func TestExtractTitleDerived(t *testing.T) {
	tests := []struct {
		system string
		title  string
		want   string
	}{
		{
			system: "Axis & Allies",
			title:  "Axis & Allies Global 1942",
			want:   "global 1942",
		},
		{
			system: "Terraforming Mars",
			title:  "Terraforming Mars: Ares Expedition Mini Tournament",
			want:   "ares expedition",
		},
		{
			system: "Wingspan",
			title:  "Wingspan Tournament Finals",
			want:   "", // all tokens are stopwords or system tokens
		},
		{
			system: "Twilight Imperium",
			title:  "Twilight Imperium with Prophecy of Kings",
			want:   "prophecy kings",
		},
	}
	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			require.Equal(t, tt.want, extractTitleDerived(tt.system, tt.title))
		})
	}
}
