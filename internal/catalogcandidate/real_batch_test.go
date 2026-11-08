package catalogcandidate_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mbcoward3/grocery-router/internal/catalogcandidate"
)

func TestCommittedChickenMarsalaPreservesReviewDifficulties(t *testing.T) {
	path := filepath.Join("..", "..", "catalog", "candidates", "chicken-marsala.candidate.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := catalogcandidate.DecodeCandidate(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := candidate.Validate(nil); err != nil {
		t.Fatal(err)
	}
	if len(candidate.InstructionSections) != 2 {
		t.Fatalf("instruction sections = %d, want 2", len(candidate.InstructionSections))
	}
	var ingredients []catalogcandidate.Ingredient
	for _, section := range candidate.IngredientSections {
		ingredients = append(ingredients, section.Ingredients...)
	}
	if len(ingredients) != 15 {
		t.Fatalf("ingredients = %d, want 15", len(ingredients))
	}
	if len(ingredients[0].Quantity.Equivalents) == 0 || len(ingredients[5].Quantity.Equivalents) == 0 {
		t.Fatal("chicken and butter dual units were not retained")
	}
	if ingredients[7].Quantity.Kind != "unspecified" {
		t.Fatalf("malformed garlic quantity = %q, want unspecified", ingredients[7].Quantity.Kind)
	}
	if !ingredients[14].Optional || !strings.Contains(ingredients[14].SourceText, "optional") {
		t.Fatal("optional parsley garnish was not retained")
	}
	joined := ""
	blockers := 0
	for _, issue := range candidate.Issues {
		joined += " " + issue.Message
		if issue.Severity == "blocker" {
			blockers++
		}
	}
	for _, want := range []string{"dual-unit", "alternative", "source-note", "1 garlic"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("review issues do not mention %q", want)
		}
	}
	if blockers == 0 || len(candidate.SourceNotes) != 5 {
		t.Fatalf("blockers=%d source notes=%d", blockers, len(candidate.SourceNotes))
	}
}
