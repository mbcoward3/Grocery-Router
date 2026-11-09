package httpapi

import (
	"database/sql"
	"testing"

	"github.com/mbcoward3/grocery-router/internal/store"
)

func TestLineQuantityTextJoinsAmountsThatCannotBeAdded(t *testing.T) {
	number := func(value int64) sql.NullInt64 { return sql.NullInt64{Int64: value, Valid: true} }
	text := func(value string) sql.NullString { return sql.NullString{String: value, Valid: true} }
	garlic := store.ListShoppingLinesRow{QuantityKind: "exact", AmountMinNumerator: number(8), AmountMinDenominator: number(1), UnitSymbol: text("cloves")}
	tablespoon := []store.ListShoppingLineQuantitiesRow{{QuantityKind: "exact", AmountMinNumerator: 1, AmountMinDenominator: 1, UnitSymbol: text("tbsp")}}
	pepper := store.ListShoppingLinesRow{QuantityKind: "exact", AmountMinNumerator: number(2), AmountMinDenominator: number(1), UnitSymbol: text("tsp")}
	for _, test := range []struct {
		name        string
		row         store.ListShoppingLinesRow
		extra       []store.ListShoppingLineQuantitiesRow
		unspecified bool
		want        string
	}{
		{"single amount", pepper, nil, false, "2 tsp"},
		{"amount plus unspecified", pepper, nil, true, "2 tsp + more"},
		{"incompatible units", garlic, tablespoon, false, "8 cloves + 1 tbsp"},
		{"everything", garlic, tablespoon, true, "8 cloves + 1 tbsp + more"},
	} {
		if got := lineQuantityText(test.row, test.extra, test.unspecified); got == nil || *got != test.want {
			t.Errorf("%s: got %v, want %q", test.name, got, test.want)
		}
	}
	if got := lineQuantityText(store.ListShoppingLinesRow{QuantityKind: "unspecified"}, nil, true); got != nil {
		t.Errorf("unspecified only: got %q, want no amount", *got)
	}
}
