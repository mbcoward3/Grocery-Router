package httpapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mbcoward3/grocery-router/internal/catalogcandidate"
	"github.com/mbcoward3/grocery-router/internal/ingest"
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

func TestCatalogSearchUsesControlledFieldsAndSemanticScores(t *testing.T) {
	profile := json.RawMessage(`{"answers":{
		"composition.center":{"type":"choice","choice":"poultry","confidence":1.0,"disposition":"needs_review"},
		"role.potluck_dish":{"type":"noul","noul":0.91,"disposition":"needs_review"},
		"method.slow_cooker":{"type":"noul","noul":0.05,"disposition":"needs_review"},
		"flavor.spice_aromatic_intensity":{"type":"score","score":2.8,"confidence":0.9,"disposition":"needs_review"}
	}}`)
	recipe := ingest.Document{
		Name: "Sunday Supper",
		IngredientSections: []ingest.IngredientSection{{Ingredients: []ingest.Ingredient{{
			SourceText: "1 cup mushrooms", GroceryItem: ingest.GroceryItem{Name: "Mushrooms"},
		}}}},
		InstructionSections: []ingest.InstructionSection{{Steps: []string{"Rinse with cold water."}}},
	}
	document, err := json.Marshal(recipe)
	if err != nil {
		t.Fatal(err)
	}
	summary := catalogRecipeSummary{Name: recipe.Name, Key: "sunday-supper", Status: "verified", Facets: semanticFacets(profile)}

	for _, query := range []string{"poultry", "thanksgiving", "mushrooms", "spice"} {
		if score := catalogSearchScore(summary, document, profile, query); score == 0 {
			t.Errorf("expected %q to match", query)
		}
	}
	for _, query := range []string{"slow cooker", "cold"} {
		if score := catalogSearchScore(summary, document, profile, query); score != 0 {
			t.Errorf("expected incidental or low-probability %q not to match, score %.2f", query, score)
		}
	}
}

func TestRealCatalogSearchExcludesMetadataAndRanksSemantics(t *testing.T) {
	candidatePaths, err := filepath.Glob("../../catalog/candidates/*.candidate.json")
	if err != nil {
		t.Fatal(err)
	}
	search := func(query string) []scoredCatalogRecipe {
		t.Helper()
		matches := make([]scoredCatalogRecipe, 0)
		for _, candidatePath := range candidatePaths {
			candidateData, readErr := os.ReadFile(candidatePath)
			if readErr != nil {
				t.Fatal(readErr)
			}
			candidate, decodeErr := catalogcandidate.DecodeCandidate(candidateData)
			if decodeErr != nil {
				t.Fatal(decodeErr)
			}
			profilePath := strings.Replace(candidatePath, "candidates", "profiles", 1)
			profilePath = strings.Replace(profilePath, ".candidate.json", ".profile.json", 1)
			profile, readErr := os.ReadFile(profilePath)
			if readErr != nil {
				t.Fatal(readErr)
			}
			summary := catalogRecipeSummary{Name: candidate.Name.Value, Key: candidate.CatalogKey, Status: "reviewable"}
			if score := catalogSearchScore(summary, candidateData, profile, query); score > 0 {
				matches = append(matches, scoredCatalogRecipe{Summary: summary, Score: score})
			}
		}
		sortCatalogSearchResults(matches)
		return matches
	}

	spice := search("spice")
	if len(spice) == 0 || len(spice) >= 14 {
		t.Fatalf("spice result count = %d, want a focused non-empty set", len(spice))
	}
	if spice[0].Summary.Key != "homemade-vegetarian-chili" {
		t.Fatalf("top spice result = %q, want homemade-vegetarian-chili", spice[0].Summary.Key)
	}
	for _, match := range spice {
		if match.Summary.Key == "meatloaf" {
			t.Fatal("meatloaf matched spice through grocery-section metadata")
		}
	}

	cold := search("cold")
	for _, match := range cold {
		if match.Summary.Key == "baked-chicken-breasts" {
			t.Fatal("baked chicken matched cold only because its instructions mention cold water")
		}
	}
	if len(cold) == 0 || cold[0].Summary.Key != "creamy-white-chicken-chili" {
		t.Fatalf("cold results were not ranked by cold-weather probability: %+v", cold)
	}
}
