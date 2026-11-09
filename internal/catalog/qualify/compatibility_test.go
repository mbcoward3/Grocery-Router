package qualify_test

import (
	"context"
	"database/sql"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"slices"
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
	service := week.NewService(db, nil).ForHousehold(tenant.CowardHouseholdID)
	if _, err := service.Generate(ctx, now, 1); err != nil {
		t.Fatal(err)
	}
	checklist, err := service.Checklist(ctx, now)
	if err != nil {
		t.Fatal(err)
	}
	sourceLines := ingredientSourceLines(t, db, document)
	// A week lists one row per grocery item; amounts that cannot be added stay separate within
	// the row. Compare item by item so every evaluation line is accounted for.
	imported := make(map[string]*itemRow)
	for _, line := range checklist.Lines {
		var item, section string
		if err := db.QueryRow(`SELECT gi.key, ss.key FROM grocery_items gi JOIN store_sections ss ON ss.id = gi.store_section_id WHERE gi.id = $1`,
			line.GroceryItemID.Int64).Scan(&item, &section); err != nil {
			t.Fatal(err)
		}
		row := &itemRow{Item: item, Section: section, Optional: line.IsOptional == 1,
			Presence:    line.QuantityKind == "presence",
			Unspecified: line.QuantityKind != "presence" && checklist.Unspecified[line.ID]}
		if line.QuantityKind == "exact" || line.QuantityKind == "range" {
			first := qualify.GroceryLine{Kind: line.QuantityKind,
				Amount:  ratString(line.AmountMinNumerator, line.AmountMinDenominator),
				Maximum: ratString(line.AmountMaxNumerator, line.AmountMaxDenominator), Unit: line.UnitKey.String}
			if line.PackageType.Valid {
				first.Package = &ingest.Package{Type: line.PackageType.String,
					Size: ratString(line.PackageSizeNumerator, line.PackageSizeDenominator), Unit: line.PackageSizeUnitKey.String}
			}
			row.Quantities = append(row.Quantities, first)
		}
		for _, extra := range checklist.Quantities[line.ID] {
			quantity := qualify.GroceryLine{Kind: extra.QuantityKind,
				Amount:  big.NewRat(extra.AmountMinNumerator, extra.AmountMinDenominator).RatString(),
				Maximum: ratString(extra.AmountMaxNumerator, extra.AmountMaxDenominator), Unit: extra.UnitKey.String}
			if extra.PackageType.Valid {
				quantity.Package = &ingest.Package{Type: extra.PackageType.String,
					Size: ratString(extra.PackageSizeNumerator, extra.PackageSizeDenominator), Unit: extra.PackageSizeUnitKey.String}
			}
			row.Quantities = append(row.Quantities, quantity)
		}
		contributions, err := service.Contributions(ctx, now, line.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, contribution := range contributions {
			row.SourceLines = append(row.SourceLines, sourceLines[contribution.RecipeIngredientID])
		}
		slices.Sort(row.SourceLines)
		imported[item] = row
	}
	if want := byItem(expected); !reflect.DeepEqual(imported, want) {
		rendered, _ := yaml.Marshal(imported)
		t.Fatalf("imported week grocery rows differ from the expected output:\n%s", rendered)
	}
}

// itemRow is one week checklist row in evaluation terms.
type itemRow struct {
	Item        string                `yaml:"item"`
	Section     string                `yaml:"section"`
	Presence    bool                  `yaml:"presence,omitempty"`
	Unspecified bool                  `yaml:"unspecified,omitempty"`
	Optional    bool                  `yaml:"optional,omitempty"`
	Quantities  []qualify.GroceryLine `yaml:"quantities,omitempty"`
	SourceLines []int                 `yaml:"source_lines,flow"`
}

// byItem groups evaluation lines the way a week groups them: one row per item, its exact and
// range amounts in order, and a flag for any requirement without an amount.
func byItem(lines []qualify.GroceryLine) map[string]*itemRow {
	rows := make(map[string]*itemRow)
	for _, line := range lines {
		row, ok := rows[line.Item]
		if !ok {
			row = &itemRow{Item: line.Item, Section: line.Section, Optional: true}
			rows[line.Item] = row
		}
		row.Optional = row.Optional && line.Optional
		switch line.Kind {
		case "presence":
			row.Presence = true
		case "unspecified":
			row.Unspecified = true
		default:
			row.Quantities = append(row.Quantities, qualify.GroceryLine{Kind: line.Kind,
				Amount: line.Amount, Maximum: line.Maximum, Unit: line.Unit, Package: line.Package})
		}
		row.SourceLines = append(row.SourceLines, line.SourceLines...)
		slices.Sort(row.SourceLines)
	}
	return rows
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
