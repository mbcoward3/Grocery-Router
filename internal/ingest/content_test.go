package ingest_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/mbcoward3/grocery-router/internal/ingest"
	"github.com/mbcoward3/grocery-router/internal/store"
)

func TestSharedContentValidationIgnoresApproval(t *testing.T) {
	document := approvedDocument(t)
	document.Status, document.ApprovedOn, document.Key = "draft", "", ""
	if err := ingest.ValidateRecipeContent(document.Content()); err != nil {
		t.Fatalf("shared content validation rejected unapproved content: %v", err)
	}
	if err := document.Validate(); err == nil {
		t.Fatal("bootstrap validator accepted an unapproved document")
	}
}

func TestSharedContentValidationQuantities(t *testing.T) {
	const (
		maxInt64     = "9223372036854775807"
		overInt64    = "9223372036854775808"
		negOverInt64 = "-9223372036854775809"
	)
	cases := []struct {
		name     string
		quantity ingest.Quantity
		want     string // empty means valid
	}{
		{"known unit", ingest.Quantity{Kind: "exact", Amount: "1", Unit: "cup"}, ""},
		{"every count unit", ingest.Quantity{Kind: "exact", Amount: "2", Unit: "clove"}, ""},
		{"metric unit", ingest.Quantity{Kind: "exact", Amount: "250", Unit: "ml"}, ""},
		{"unknown unit", ingest.Quantity{Kind: "exact", Amount: "1", Unit: "pinch"}, `unknown unit "pinch"`},
		{"unknown unit in range", ingest.Quantity{Kind: "range", Amount: "1", Maximum: "2", Unit: "handful"}, `unknown unit "handful"`},
		{"known package unit", ingest.Quantity{Kind: "exact", Amount: "1", Package: &ingest.Package{Type: "can", Size: "15", Unit: "oz"}}, ""},
		{"unknown package unit", ingest.Quantity{Kind: "exact", Amount: "1", Package: &ingest.Package{Type: "can", Size: "15", Unit: "ounces"}}, `unknown package unit "ounces"`},
		{"package without size", ingest.Quantity{Kind: "exact", Amount: "1", Package: &ingest.Package{Type: "jar"}}, ""},
		{"largest numerator", ingest.Quantity{Kind: "exact", Amount: maxInt64, Unit: "g"}, ""},
		{"numerator over 64 bits", ingest.Quantity{Kind: "exact", Amount: overInt64, Unit: "g"}, "exceeds 64-bit"},
		{"largest denominator", ingest.Quantity{Kind: "exact", Amount: "1/" + maxInt64, Unit: "g"}, ""},
		{"denominator over 64 bits", ingest.Quantity{Kind: "exact", Amount: "1/" + overInt64, Unit: "g"}, "exceeds 64-bit"},
		{"reducible fraction fits", ingest.Quantity{Kind: "exact", Amount: "2/" + "18446744073709551614", Unit: "g"}, ""},
		{"decimal expands past 64 bits", ingest.Quantity{Kind: "exact", Amount: "1e30", Unit: "g"}, "exceeds 64-bit"},
		{"mixed number over 64 bits", ingest.Quantity{Kind: "exact", Amount: maxInt64 + " 1/2", Unit: "g"}, "exceeds 64-bit"},
		{"maximum over 64 bits", ingest.Quantity{Kind: "range", Amount: "1", Maximum: overInt64, Unit: "g"}, "invalid maximum"},
		{"package size over 64 bits", ingest.Quantity{Kind: "exact", Amount: "1", Package: &ingest.Package{Type: "can", Size: overInt64, Unit: "oz"}}, "invalid package size"},
		{"negative beyond 64 bits stays non-positive", ingest.Quantity{Kind: "exact", Amount: negOverInt64, Unit: "g"}, "value must be positive"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			document := approvedDocument(t)
			document.IngredientSections[0].Ingredients[1].Quantity = tc.quantity
			err := ingest.ValidateRecipeContent(document.Content())
			if tc.want == "" {
				if err != nil {
					t.Fatalf("ValidateRecipeContent() = %v, want valid", err)
				}
				if err := document.Validate(); err != nil {
					t.Fatalf("bootstrap Validate() = %v, want valid", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ValidateRecipeContent() = %v, want error containing %q", err, tc.want)
			}
			if err := document.Validate(); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("bootstrap Validate() = %v, want error containing %q", err, tc.want)
			}
		})
	}
}

func TestSharedContentValidationRejectsBlankOptionalText(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ingest.Document)
		want   string
	}{
		{"yield", func(d *ingest.Document) { d.Yield = " " }, "yield cannot be blank"},
		{"image_url", func(d *ingest.Document) { d.ImageURL = "\t" }, "image_url cannot be blank"},
		{"preparation", func(d *ingest.Document) { d.IngredientSections[0].Ingredients[0].Preparation = " " }, "preparation cannot be blank"},
		{"note", func(d *ingest.Document) { d.IngredientSections[0].Ingredients[0].Note = " " }, "note cannot be blank"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			document := approvedDocument(t)
			tc.mutate(&document)
			if err := ingest.ValidateRecipeContent(document.Content()); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ValidateRecipeContent() = %v, want %q", err, tc.want)
			}
		})
	}
}

// TestUnitVocabularyMatchesDatabase keeps the in-code unit vocabulary identical to the units
// seeded by migrations, so validation before a transaction agrees with the import's lookup.
func TestUnitVocabularyMatchesDatabase(t *testing.T) {
	db := openMigratedDB(t)
	units, err := store.New(db).ListUnits(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var databaseKeys []string
	for _, unit := range units {
		databaseKeys = append(databaseKeys, unit.Key)
		dimension, ok := ingest.UnitDimension(unit.Key)
		if !ok || dimension != unit.Dimension {
			t.Errorf("unit %q: vocabulary dimension %q (known=%v), database %q", unit.Key, dimension, ok, unit.Dimension)
		}
		numerator, denominator, _ := ingest.UnitToBase(unit.Key)
		if numerator != unit.ToBaseNumerator || denominator != unit.ToBaseDenominator {
			t.Errorf("unit %q: vocabulary ratio %d/%d, database %d/%d", unit.Key, numerator, denominator, unit.ToBaseNumerator, unit.ToBaseDenominator)
		}
	}
	slices.Sort(databaseKeys)
	if !slices.Equal(databaseKeys, ingest.UnitKeys()) {
		t.Fatalf("unit vocabulary = %v, database = %v", ingest.UnitKeys(), databaseKeys)
	}
}
