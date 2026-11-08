package ingest

import "sort"

// unitDimensions mirrors the exact unit vocabulary seeded by migration 00001_corpus.sql. The
// units table is global and has no runtime write path, so recipe content can be checked
// against it before any database transaction. TestUnitVocabularyMatchesDatabase keeps the two
// in step: a migration that changes the seed must change this map in the same commit.
var unitDimensions = map[string]string{
	"each":   "count",
	"slice":  "count",
	"clove":  "count",
	"leaf":   "count",
	"bunch":  "count",
	"sprig":  "count",
	"tsp":    "volume",
	"tbsp":   "volume",
	"floz":   "volume",
	"cup":    "volume",
	"pint":   "volume",
	"quart":  "volume",
	"gallon": "volume",
	"ml":     "volume",
	"l":      "volume",
	"oz":     "mass",
	"lb":     "mass",
	"g":      "mass",
	"kg":     "mass",
}

// IsKnownUnit reports whether key names a unit in the seeded unit vocabulary.
func IsKnownUnit(key string) bool {
	_, ok := unitDimensions[key]
	return ok
}

// UnitDimension returns the dimension (count, mass, or volume) of a known unit.
func UnitDimension(key string) (string, bool) {
	dimension, ok := unitDimensions[key]
	return dimension, ok
}

// UnitKeys returns every known unit key in sorted order.
func UnitKeys() []string {
	keys := make([]string, 0, len(unitDimensions))
	for key := range unitDimensions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
