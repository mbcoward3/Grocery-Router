package httpapi

import (
	"encoding/json"
	"testing"

	"github.com/mbcoward3/grocery-router/internal/catalogcandidate"
)

func TestSemanticFacetsAreNotTruncated(t *testing.T) {
	answers := make(map[string]any)
	for _, key := range []string{
		"utility.batch_cooking", "utility.crowd_friendly", "utility.freezer_friendly",
		"utility.leftovers", "utility.make_ahead", "utility.meal_prep",
		"utility.portable", "utility.quick_serving", "utility.scales_up",
		"utility.weeknight_friendly",
	} {
		answers[key] = map[string]any{"type": "noul", "noul": 0.95, "disposition": "needs_review"}
	}
	profile, err := json.Marshal(map[string]any{"answers": answers})
	if err != nil {
		t.Fatal(err)
	}

	facets := semanticFacets(profile)
	if len(facets) != 10 {
		t.Fatalf("facet count = %d, want 10: %v", len(facets), facets)
	}
}

func TestCandidateIngredientDisplayUsesStructuredProposal(t *testing.T) {
	tests := []struct {
		name       string
		ingredient catalogcandidate.Ingredient
		want       string
	}{
		{
			name: "measured with preparation",
			ingredient: catalogcandidate.Ingredient{
				SourceText:  "1/2 cup thickened / heavy cream ( (Note 4))",
				ItemPhrase:  "Heavy Cream",
				Preparation: "divided",
				Quantity:    catalogcandidate.Quantity{Kind: "exact", Amount: "1/2", Unit: "cup"},
			},
			want: "½ cup heavy cream, divided",
		},
		{
			name: "sized packages",
			ingredient: catalogcandidate.Ingredient{
				ItemPhrase: "Black Beans",
				Quantity: catalogcandidate.Quantity{Kind: "exact", Amount: "2", Package: &catalogcandidate.Package{
					Count: "2", Type: "can", Size: "15", Unit: "oz",
				}},
			},
			want: "2 × 15 oz cans black beans",
		},
		{
			name: "unspecified",
			ingredient: catalogcandidate.Ingredient{
				ItemPhrase: "Salt",
				Quantity:   catalogcandidate.Quantity{Kind: "unspecified"},
			},
			want: "salt",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := candidateIngredientDisplay(tt.ingredient); got != tt.want {
				t.Fatalf("display = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCatalogSearchUsesFullSemanticProfileAndUserLanguage(t *testing.T) {
	profile := json.RawMessage(`{"answers":{
		"composition.center":{"type":"choice","choice":"poultry","disposition":"needs_review"},
		"role.potluck_dish":{"type":"noul","noul":0.91,"disposition":"needs_review"},
		"method.slow_cooker":{"type":"noul","noul":0.05,"disposition":"needs_review"}
	}}`)
	summary := catalogRecipeSummary{Name: "Sunday Supper", Key: "sunday-supper", Facets: semanticFacets(profile)}
	document := json.RawMessage(`{"ingredient_sections":[{"ingredients":[{"source_text":"1 cup mushrooms"}]}]}`)

	for _, query := range []string{"poultry", "chicken", "thanksgiving", "mushrooms"} {
		if !catalogSearchMatch(summary, document, "Example Kitchen", query) {
			t.Errorf("expected %q to match", query)
		}
	}
	if catalogSearchMatch(summary, document, "Example Kitchen", "slow cooker") {
		t.Error("a low-probability semantic answer must not become searchable")
	}
}
