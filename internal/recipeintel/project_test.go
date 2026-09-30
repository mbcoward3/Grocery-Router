package recipeintel

import (
	"encoding/json"
	"testing"

	"github.com/mbcoward3/grocery-router/internal/recipepilot"
)

func TestCandidatesFromSourceArtifactProjectsRecipeContent(t *testing.T) {
	artifact := recipepilot.SourceArtifact{
		Source: recipepilot.ManifestSource{ID: "example"},
		Recipes: []recipepilot.RecipeCandidate{{Recipe: recipepilot.SourceRecipe{
			Name:               json.RawMessage(`"Example &amp; Dinner"`),
			Description:        json.RawMessage(`"A useful recipe"`),
			RecipeIngredient:   json.RawMessage(`["1 onion", "2 cups broth"]`),
			RecipeInstructions: json.RawMessage(`[{"@type":"HowToSection","name":"Sauce","itemListElement":[{"@type":"HowToStep","text":"Cook &amp; stir."}]}]`),
		}}},
	}
	candidates, err := CandidatesFromSourceArtifact(artifact)
	if err != nil {
		t.Fatalf("CandidatesFromSourceArtifact: %v", err)
	}
	candidate := candidates[0]
	if candidate.Name != "Example & Dinner" || len(candidate.Ingredients) != 2 {
		t.Fatalf("candidate = %#v", candidate)
	}
	state, err := candidate.Project("full_recipe")
	if err != nil {
		t.Fatalf("Project: %v", err)
	}
	if len(state.Method) != 1 || state.Method[0].Section != "Sauce" || state.Method[0].Text != "Cook & stir." {
		t.Fatalf("method = %#v", state.Method)
	}
}
