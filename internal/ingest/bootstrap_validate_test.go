package ingest_test

import (
	"strings"
	"testing"

	"github.com/mbcoward3/grocery-router/internal/ingest"
)

// TestBootstrapValidatorKeepsEveryRejection pins each rejection the bootstrap validator made
// before the content contract was split (catalog spec invariant 3). Each case mutates one
// approved corpus recipe and must still fail with the same reason.
func TestBootstrapValidatorKeepsEveryRejection(t *testing.T) {
	firstIngredient := func(d *ingest.Document) *ingest.Ingredient {
		return &d.IngredientSections[0].Ingredients[0]
	}
	measured := func(d *ingest.Document) *ingest.Quantity {
		// The second ingredient is "1 cup milk": an exact quantity with a unit.
		return &d.IngredientSections[0].Ingredients[1].Quantity
	}
	withReview := func(d *ingest.Document, decision ingest.ReviewDecision) {
		d.Review = append(d.Review, decision)
	}

	cases := []struct {
		name   string
		mutate func(*ingest.Document)
		want   string
	}{
		{"format version", func(d *ingest.Document) { d.FormatVersion = 2 }, "format_version = 2, want 1"},
		{"key", func(d *ingest.Document) { d.Key = "Not A Key" }, "invalid key"},
		{"empty name", func(d *ingest.Document) { d.Name = "  " }, "name is required"},
		{"draft status", func(d *ingest.Document) { d.Status = "draft" }, "bootstrap status must be verified"},
		{"missing status", func(d *ingest.Document) { d.Status = "" }, "bootstrap status must be verified"},
		{"missing approved_on", func(d *ingest.Document) { d.ApprovedOn = "" }, "approved_on must be YYYY-MM-DD"},
		{"malformed approved_on", func(d *ingest.Document) { d.ApprovedOn = "2026-13-01" }, "approved_on must be YYYY-MM-DD"},
		{"source relationship", func(d *ingest.Document) { d.Source.Relationship = "inspired-by" }, "invalid source relationship"},
		{"source attribution", func(d *ingest.Document) { d.Source.Attribution = "" }, "source attribution is required"},
		{"source URL scheme", func(d *ingest.Document) { d.Source.URL = "ftp://example.com/r" }, "invalid source URL"},
		{"source URL relative", func(d *ingest.Document) { d.Source.URL = "/recipes/1" }, "invalid source URL"},
		{"source checked_on", func(d *ingest.Document) { d.Source.CheckedOn = "yesterday" }, "source checked_on must be YYYY-MM-DD"},
		{"hands_on negative", func(d *ingest.Document) { d.HandsOn = &ingest.Duration{Min: -1, Max: 5} }, "hands_on range is invalid"},
		{"hands_on inverted", func(d *ingest.Document) { d.HandsOn = &ingest.Duration{Min: 10, Max: 5} }, "hands_on range is invalid"},
		{"unattended inverted", func(d *ingest.Document) { d.Unattended = &ingest.Duration{Min: 10, Max: 5} }, "unattended range is invalid"},
		{"no ingredient sections", func(d *ingest.Document) { d.IngredientSections = nil }, "at least one ingredient section is required"},
		{"unnamed ingredient section", func(d *ingest.Document) { d.IngredientSections[0].Name = "" }, "ingredient section 0 needs a name and ingredients"},
		{"empty ingredient section", func(d *ingest.Document) { d.IngredientSections[0].Ingredients = nil }, "ingredient section 0 needs a name and ingredients"},
		{"empty source text", func(d *ingest.Document) { firstIngredient(d).SourceText = " " }, "source_text is required"},
		{"grocery item key", func(d *ingest.Document) { firstIngredient(d).GroceryItem.Key = "Bad Key" }, "grocery item needs valid key and name"},
		{"grocery item name", func(d *ingest.Document) { firstIngredient(d).GroceryItem.Name = "" }, "grocery item needs valid key and name"},
		{"store section key", func(d *ingest.Document) { firstIngredient(d).GroceryItem.StoreSection.Key = "" }, "needs valid store section"},
		{"store section name", func(d *ingest.Document) { firstIngredient(d).GroceryItem.StoreSection.Name = "" }, "needs valid store section"},
		{"shopping mode", func(d *ingest.Document) { firstIngredient(d).GroceryItem.ShoppingMode = "bulk" }, "invalid shopping mode"},
		{"inconsistent grocery item", func(d *ingest.Document) {
			second := &d.IngredientSections[0].Ingredients[1]
			second.GroceryItem.Key = firstIngredient(d).GroceryItem.Key
		}, "redefines grocery item"},
		{"quantity kind", func(d *ingest.Document) { measured(d).Kind = "approximate" }, "invalid quantity kind"},
		{"unspecified with amount", func(d *ingest.Document) { measured(d).Kind = "unspecified" }, "unspecified quantity cannot carry amount, unit, or package"},
		{"missing amount", func(d *ingest.Document) { measured(d).Amount = "" }, "invalid amount"},
		{"malformed amount", func(d *ingest.Document) { measured(d).Amount = "one" }, "invalid amount"},
		{"zero amount", func(d *ingest.Document) { measured(d).Amount = "0" }, "value must be positive"},
		{"negative amount", func(d *ingest.Document) { measured(d).Amount = "-1" }, "value must be positive"},
		{"three-part amount", func(d *ingest.Document) { measured(d).Amount = "1 1/2 3" }, "invalid number"},
		{"malformed mixed number", func(d *ingest.Document) { measured(d).Amount = "1 half" }, "invalid mixed number"},
		{"range without maximum", func(d *ingest.Document) { measured(d).Kind = "range" }, "invalid maximum"},
		{"range inverted", func(d *ingest.Document) {
			measured(d).Kind, measured(d).Amount, measured(d).Maximum = "range", "2", "1"
		}, "quantity minimum exceeds maximum"},
		{"exact with maximum", func(d *ingest.Document) { measured(d).Maximum = "2" }, "exact quantity cannot carry maximum"},
		{"neither unit nor package", func(d *ingest.Document) { measured(d).Unit = "" }, "needs exactly one of unit or package"},
		{"unit and package", func(d *ingest.Document) {
			measured(d).Package = &ingest.Package{Type: "carton"}
		}, "needs exactly one of unit or package"},
		{"malformed unit key", func(d *ingest.Document) { measured(d).Unit = "Cup" }, "invalid unit key"},
		{"package type", func(d *ingest.Document) { firstIngredient(d).Quantity.Package.Type = " " }, "package type is required"},
		{"package size without unit", func(d *ingest.Document) { firstIngredient(d).Quantity.Package.Unit = "" }, "package size and unit must appear together"},
		{"package unit without size", func(d *ingest.Document) { firstIngredient(d).Quantity.Package.Size = "" }, "package size and unit must appear together"},
		{"package size", func(d *ingest.Document) { firstIngredient(d).Quantity.Package.Size = "big" }, "invalid package size"},
		{"package unit key", func(d *ingest.Document) { firstIngredient(d).Quantity.Package.Unit = "Ounce" }, "invalid package unit key"},
		{"no instruction sections", func(d *ingest.Document) { d.InstructionSections = nil }, "at least one instruction section is required"},
		{"unnamed instruction section", func(d *ingest.Document) { d.InstructionSections[0].Name = "" }, "instruction section 0 needs a name and steps"},
		{"empty instruction section", func(d *ingest.Document) { d.InstructionSections[0].Steps = nil }, "instruction section 0 needs a name and steps"},
		{"blank step", func(d *ingest.Document) { d.InstructionSections[0].Steps[0] = " " }, "steps[0] is empty"},
		{"review field", func(d *ingest.Document) {
			withReview(d, ingest.ReviewDecision{Kind: "rewritten", Note: "n", Approved: true})
		}, "needs field and note"},
		{"review note", func(d *ingest.Document) {
			withReview(d, ingest.ReviewDecision{Field: "f", Kind: "rewritten", Approved: true})
		}, "needs field and note"},
		{"review kind", func(d *ingest.Document) {
			withReview(d, ingest.ReviewDecision{Field: "f", Kind: "guessed", Note: "n", Approved: true})
		}, "invalid kind"},
		{"review not approved", func(d *ingest.Document) {
			withReview(d, ingest.ReviewDecision{Field: "f", Kind: "rewritten", Note: "n"})
		}, "is not approved"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			document := approvedDocument(t)
			if err := document.Validate(); err != nil {
				t.Fatalf("baseline document invalid: %v", err)
			}
			tc.mutate(&document)
			err := document.Validate()
			if err == nil {
				t.Fatalf("Validate() accepted the document, want rejection containing %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tc.want)
			}
			if !strings.HasPrefix(err.Error(), `recipe "`) {
				t.Fatalf("Validate() = %v, want recipe key prefix", err)
			}
		})
	}
}
