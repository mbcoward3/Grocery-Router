package qualify_test

import (
	"context"
	"database/sql"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/mbcoward3/grocery-router/internal/catalog/qualify"
	"github.com/mbcoward3/grocery-router/internal/ingest"
	"github.com/mbcoward3/grocery-router/internal/tenant"
	"github.com/mbcoward3/grocery-router/internal/testdatabase"
	"github.com/mbcoward3/grocery-router/internal/week"
	"gopkg.in/yaml.v3"
)

// TestCompatibilityFixture is the spec's slice 1 compatibility fixture: a hand-written
// catalog-format recipe converts, imports into an empty test household, and aggregates in a
// real week to the expected grocery output, which matches the in-memory conversion the
// harness uses.
func TestCompatibilityFixture(t *testing.T) {
	file, err := os.Open(filepath.Join("..", "..", "ingest", "testdata", "catalog", "every-relationship.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	document, err := ingest.ParseCatalogDocument(file)
	file.Close()
	if err != nil {
		t.Fatal(err)
	}

	expectedBytes, err := os.ReadFile(filepath.Join("testdata", "compatibility-grocery.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var expected []qualify.GroceryLine
	if err := yaml.Unmarshal(expectedBytes, &expected); err != nil {
		t.Fatal(err)
	}
	inMemory, err := qualify.GroceryOutput(document)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(inMemory, expected) {
		rendered, _ := yaml.Marshal(inMemory)
		t.Fatalf("in-memory grocery output differs from testdata/compatibility-grocery.yaml:\n%s", rendered)
	}

	content, err := document.Content()
	if err != nil {
		t.Fatal(err)
	}
	// The test household's approval marks this import only; catalog trust is never recorded
	// as verified outside a test database.
	bootstrap := ingest.Document{
		FormatVersion: 1, Key: "catalog-format-example", Status: "verified", ApprovedOn: "2026-11-08",
		Name: content.Name, Source: content.Source, ImageURL: content.ImageURL, Yield: content.Yield,
		HandsOn: content.HandsOn, Unattended: content.Unattended,
		IngredientSections: content.IngredientSections, InstructionSections: content.InstructionSections,
	}
	ctx := context.Background()
	db := testdatabase.Open(t)
	if err := ingest.Import(ctx, db, tenant.CowardHouseholdID, []ingest.Document{bootstrap}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	service := week.NewService(db, nil, tenant.CowardHouseholdID)
	if _, err := service.Generate(ctx, now, 1); err != nil {
		t.Fatal(err)
	}
	checklist, err := service.Checklist(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	sourceLines := ingredientSourceLines(t, db, document)
	var imported []qualify.GroceryLine
	for _, line := range checklist.Lines {
		output := qualify.GroceryLine{Kind: line.QuantityKind, Optional: line.IsOptional == 1,
			Amount:  ratString(line.AmountMinNumerator, line.AmountMinDenominator),
			Maximum: ratString(line.AmountMaxNumerator, line.AmountMaxDenominator),
			Unit:    line.UnitKey.String}
		if err := db.QueryRow(`SELECT gi.key, ss.key FROM grocery_items gi JOIN store_sections ss ON ss.id = gi.store_section_id WHERE gi.id = $1`,
			line.GroceryItemID.Int64).Scan(&output.Item, &output.Section); err != nil {
			t.Fatal(err)
		}
		if line.PackageType.Valid {
			output.Package = &ingest.Package{Type: line.PackageType.String,
				Size: ratString(line.PackageSizeNumerator, line.PackageSizeDenominator), Unit: line.PackageSizeUnitKey.String}
		}
		contributions, err := service.Contributions(ctx, now, line.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, contribution := range contributions {
			output.SourceLines = append(output.SourceLines, sourceLines[contribution.RecipeIngredientID])
		}
		imported = append(imported, output)
	}
	// The checklist lists lines in store order; the aggregated lines themselves must match.
	got, want := sortedLines(imported), sortedLines(expected)
	if !reflect.DeepEqual(got, want) {
		rendered, _ := yaml.Marshal(got)
		t.Fatalf("imported week grocery output differs from the expected output:\n%s", rendered)
	}
}

func sortedLines(lines []qualify.GroceryLine) []qualify.GroceryLine {
	sorted := slices.Clone(lines)
	slices.SortFunc(sorted, func(left, right qualify.GroceryLine) int {
		return strings.Compare(left.Item+"|"+left.Kind, right.Item+"|"+right.Kind)
	})
	return sorted
}

// ingredientSourceLines maps each imported recipe ingredient to its requirement's source line,
// using the import's section and ingredient positions.
func ingredientSourceLines(t *testing.T, db *sql.DB, document ingest.CatalogDocument) map[int64]int {
	t.Helper()
	rows, err := db.Query(`SELECT ri.id, ris.position, ri.position FROM recipe_ingredients ri
		JOIN recipe_ingredient_sections ris ON ris.id = ri.section_id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	result := make(map[int64]int)
	for rows.Next() {
		var id, section, position int64
		if err := rows.Scan(&id, &section, &position); err != nil {
			t.Fatal(err)
		}
		result[id] = document.IngredientSections[section].Requirements[position].Provenance.Line
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return result
}

func ratString(numerator, denominator sql.NullInt64) string {
	if !numerator.Valid {
		return ""
	}
	return big.NewRat(numerator.Int64, denominator.Int64).RatString()
}
