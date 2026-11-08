package ingest

import "sort"

// unitVocabulary mirrors the exact unit vocabulary seeded by migration 00001_corpus.sql. The
// units table is global and has no runtime write path, so recipe content can be checked
// against it before any database transaction. TestUnitVocabularyMatchesDatabase keeps the two
// in step: a migration that changes the seed must change this map in the same commit.
var unitVocabulary = map[string]unitDefinition{
	"each":   {"count", 1, 1},
	"slice":  {"count", 1, 1},
	"clove":  {"count", 1, 1},
	"leaf":   {"count", 1, 1},
	"bunch":  {"count", 1, 1},
	"sprig":  {"count", 1, 1},
	"tsp":    {"volume", 1, 1},
	"tbsp":   {"volume", 3, 1},
	"floz":   {"volume", 6, 1},
	"cup":    {"volume", 48, 1},
	"pint":   {"volume", 96, 1},
	"quart":  {"volume", 192, 1},
	"gallon": {"volume", 768, 1},
	"ml":     {"volume", 32000000, 157725491},
	"l":      {"volume", 32000000000, 157725491},
	"oz":     {"mass", 1, 1},
	"lb":     {"mass", 16, 1},
	"g":      {"mass", 1600000, 45359237},
	"kg":     {"mass", 1600000000, 45359237},
}

// unitDefinition is one seeded unit: its dimension and exact ratio to the dimension's base
// unit (tsp for volume, oz for mass, each for count).
type unitDefinition struct {
	dimension         string
	toBaseNumerator   int64
	toBaseDenominator int64
}

// IsKnownUnit reports whether key names a unit in the seeded unit vocabulary.
func IsKnownUnit(key string) bool {
	_, ok := unitVocabulary[key]
	return ok
}

// UnitDimension returns the dimension (count, mass, or volume) of a known unit.
func UnitDimension(key string) (string, bool) {
	unit, ok := unitVocabulary[key]
	return unit.dimension, ok
}

// UnitToBase returns the exact ratio from a known unit to its dimension's base unit, as the
// database's units.to_base_numerator and units.to_base_denominator store it.
func UnitToBase(key string) (numerator, denominator int64, ok bool) {
	unit, ok := unitVocabulary[key]
	return unit.toBaseNumerator, unit.toBaseDenominator, ok
}

// UnitKeys returns every known unit key in sorted order.
func UnitKeys() []string {
	keys := make([]string, 0, len(unitVocabulary))
	for key := range unitVocabulary {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
