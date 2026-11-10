package httpapi

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"

	"github.com/mbcoward3/grocery-router/internal/store"
	"github.com/mbcoward3/grocery-router/internal/week"
)

// TestGroceryResponseMatchesContractSchema checks the checklist encoding against the contract's
// Groceries schema without a database; the HTTP tests check the same through real requests.
func TestGroceryResponseMatchesContractSchema(t *testing.T) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(filepath.Join("..", "..", "api", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	schema := doc.Components.Schemas["Groceries"]
	if schema == nil {
		t.Fatal("contract has no Groceries schema")
	}
	number := func(value int64) sql.NullInt64 { return sql.NullInt64{Int64: value, Valid: true} }
	checklist := week.Checklist{
		Lines: []store.ListShoppingLinesRow{
			{
				ID: 1, StoreSectionID: 2, StoreSectionName: "Produce", DisplayName: "Onion", Origin: "generated",
				QuantityKind: "exact", AmountMinNumerator: number(2), AmountMinDenominator: number(1),
				OverrideText: sql.NullString{String: "3", Valid: true}, IsCompleted: 1,
			},
			{ID: 2, StoreSectionID: 3, StoreSectionName: "Household", DisplayName: "Paper towels", Origin: "manual", QuantityKind: "presence"},
		},
		RecipeNames: map[int64][]string{1: {"Soup"}},
	}
	encoded, err := json.Marshal(groceryResponseFromChecklist(checklist, "2026-08-16"))
	if err != nil {
		t.Fatal(err)
	}
	var value any
	if err := json.Unmarshal(encoded, &value); err != nil {
		t.Fatal(err)
	}
	if err := schema.Value.VisitJSON(value, openapi3.EnableJSONSchema2020(), openapi3.VisitAsResponse()); err != nil {
		t.Fatalf("groceries response %s does not match contract: %v", encoded, err)
	}
}
