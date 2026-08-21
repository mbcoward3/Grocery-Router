package grocery_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mbcoward3/grocery-router/internal/grocery"
)

var (
	each = &grocery.Unit{ID: 1, Key: "each", Dimension: "count", ToBaseNumerator: 1, ToBaseDenominator: 1}
	tsp  = &grocery.Unit{ID: 2, Key: "tsp", Dimension: "volume", ToBaseNumerator: 1, ToBaseDenominator: 1}
	tbsp = &grocery.Unit{ID: 3, Key: "tbsp", Dimension: "volume", ToBaseNumerator: 3, ToBaseDenominator: 1}
	cup  = &grocery.Unit{ID: 4, Key: "cup", Dimension: "volume", ToBaseNumerator: 48, ToBaseDenominator: 1}
	oz   = &grocery.Unit{ID: 5, Key: "oz", Dimension: "mass", ToBaseNumerator: 1, ToBaseDenominator: 1}
)

func TestAggregateExactQuantitiesAndTraces(t *testing.T) {
	requirements := []grocery.Requirement{
		requirement(10, 100, 1, "oil", "Oil", "measured", "exact", rational(1, 1), nil, tbsp),
		requirement(11, 101, 1, "oil", "Oil", "measured", "exact", rational(3, 1), nil, tsp),
		requirement(12, 102, 1, "oil", "Oil", "measured", "range", rational(1, 1), rational(2, 1), tbsp),
	}

	lines, err := grocery.Aggregate(requirements)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 {
		t.Fatalf("lines = %d, want 1", len(lines))
	}
	line := lines[0]
	assertRational(t, line.Minimum, 3, 1)
	assertRational(t, line.Maximum, 4, 1)
	if line.QuantityKind != "range" || line.Unit.Key != "tbsp" {
		t.Fatalf("line quantity = %s %s, want range tbsp", line.QuantityKind, line.Unit.Key)
	}
	if len(line.Contributions) != 3 || line.Contributions[1].WeekRecipeID != 11 || line.Contributions[1].Unit.Key != "tsp" {
		t.Fatalf("contributions did not preserve occurrence trace and source unit: %#v", line.Contributions)
	}
}

func TestAggregateSeparatesIncompatibleUnitsAndPackageSizes(t *testing.T) {
	onionEach := requirement(1, 1, 2, "onion", "Onion", "counted", "exact", rational(1, 1), nil, each)
	onionCup := requirement(2, 2, 2, "onion", "Onion", "counted", "exact", rational(1, 1), nil, cup)
	can145 := requirement(1, 3, 3, "tomatoes", "Tomatoes", "counted", "exact", rational(2, 1), nil, nil)
	can145.Package = &grocery.Package{Type: "can", Size: rational(29, 2), SizeUnit: oz}
	can145Again := requirement(2, 4, 3, "tomatoes", "Tomatoes", "counted", "exact", rational(1, 1), nil, nil)
	can145Again.Package = &grocery.Package{Type: "can", Size: rational(29, 2), SizeUnit: oz}
	can28 := requirement(3, 5, 3, "tomatoes", "Tomatoes", "counted", "exact", rational(1, 1), nil, nil)
	can28.Package = &grocery.Package{Type: "can", Size: rational(28, 1), SizeUnit: oz}

	lines, err := grocery.Aggregate([]grocery.Requirement{onionEach, onionCup, can145, can145Again, can28})
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 4 {
		t.Fatalf("lines = %d, want 4 incompatible requirements", len(lines))
	}
	assertRational(t, lines[2].Minimum, 3, 1)
	if len(lines[2].Contributions) != 2 {
		t.Fatal("matching package sizes did not aggregate")
	}
	if lines[2].Key == lines[3].Key {
		t.Fatal("different package sizes unexpectedly shared an aggregation key")
	}
}

func TestAggregatePresenceOptionalAndDuplicateOccurrences(t *testing.T) {
	first := requirement(20, 200, 4, "salt", "Salt", "presence-only", "unspecified", nil, nil, nil)
	first.Optional = true
	second := requirement(21, 200, 4, "salt", "Salt", "presence-only", "exact", rational(1, 1), nil, tsp)
	second.Optional = false

	lines, err := grocery.Aggregate([]grocery.Requirement{first, second})
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || lines[0].QuantityKind != "presence" {
		t.Fatalf("presence lines = %#v", lines)
	}
	if lines[0].Optional {
		t.Fatal("line with a required contribution was marked optional")
	}
	if len(lines[0].Contributions) != 2 || lines[0].Contributions[0].WeekRecipeID == lines[0].Contributions[1].WeekRecipeID {
		t.Fatal("duplicate recipe occurrences were not independently traced")
	}
}

func TestRepresentativeAggregationGolden(t *testing.T) {
	requirements := []grocery.Requirement{
		requirement(1, 10, 1, "oil", "Olive Oil", "measured", "exact", rational(1, 1), nil, tbsp),
		requirement(2, 11, 1, "oil", "Olive Oil", "measured", "exact", rational(3, 1), nil, tsp),
		requirement(1, 12, 2, "salt", "Salt", "presence-only", "unspecified", nil, nil, nil),
		requirement(2, 13, 2, "salt", "Salt", "presence-only", "exact", rational(1, 2), nil, tsp),
	}
	lines, err := grocery.Aggregate(requirements)
	if err != nil {
		t.Fatal(err)
	}
	var output strings.Builder
	for _, line := range lines {
		fmt.Fprintf(&output, "%s | %s | %s | optional=%t | contributions=%d\n",
			line.StoreSectionName, line.GroceryItemName, quantity(line), line.Optional, len(line.Contributions))
	}
	golden, err := os.ReadFile(filepath.Join("testdata", "representative.golden"))
	if err != nil {
		t.Fatal(err)
	}
	if output.String() != string(golden) {
		t.Fatalf("aggregation output:\n%s\nwant:\n%s", output.String(), golden)
	}
}

func requirement(
	weekRecipeID, ingredientID, itemID int64,
	key, name, mode, kind string,
	minimum, maximum *grocery.Rational,
	unit *grocery.Unit,
) grocery.Requirement {
	return grocery.Requirement{
		WeekRecipeID: weekRecipeID, RecipeIngredientID: ingredientID,
		RecipeName: fmt.Sprintf("Recipe %d", weekRecipeID), GroceryItemID: itemID,
		GroceryItemKey: key, GroceryItemName: name, ShoppingMode: mode,
		StoreSectionID: 1, StoreSectionName: "Pantry", QuantityKind: kind,
		Minimum: minimum, Maximum: maximum, Unit: unit,
	}
}

func rational(numerator, denominator int64) *grocery.Rational {
	return &grocery.Rational{Numerator: numerator, Denominator: denominator}
}

func assertRational(t *testing.T, got *grocery.Rational, numerator, denominator int64) {
	t.Helper()
	if got == nil || got.Numerator != numerator || got.Denominator != denominator {
		t.Fatalf("rational = %#v, want %d/%d", got, numerator, denominator)
	}
}

func quantity(line grocery.Line) string {
	if line.QuantityKind == "presence" || line.QuantityKind == "unspecified" {
		return line.QuantityKind
	}
	value := fmt.Sprintf("%d/%d", line.Minimum.Numerator, line.Minimum.Denominator)
	if line.QuantityKind == "range" {
		value += fmt.Sprintf("–%d/%d", line.Maximum.Numerator, line.Maximum.Denominator)
	}
	if line.Package != nil {
		return value + " x " + line.Package.Type
	}
	return value + " " + line.Unit.Key
}
