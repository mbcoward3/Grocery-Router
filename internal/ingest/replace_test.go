package ingest_test

import (
	"testing"

	"github.com/mbcoward3/grocery-router/internal/ingest"
)

func TestGroceryItemKeyFollowsCorpusKeyRule(t *testing.T) {
	cases := map[string]string{
		"Grains of Paradise":   "grains-of-paradise",
		"  Wine & Beer ":       "wine-and-beer",
		"Baker's yeast":        "bakers-yeast",
		"7-Up":                 "7-up",
		"Half-and-half (1/2)":  "half-and-half-1-2",
		"Jalapeño":             "jalape-o",
		"!!!":                  "",
		"Crème fraîche":        "cr-me-fra-che",
		"Chicken thighs, raw.": "chicken-thighs-raw",
	}
	for name, want := range cases {
		if got := ingest.GroceryItemKey(name); got != want {
			t.Errorf("GroceryItemKey(%q) = %q, want %q", name, got, want)
		}
	}
}
