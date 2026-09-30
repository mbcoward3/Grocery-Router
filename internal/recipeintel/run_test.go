package recipeintel

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mbcoward3/grocery-router/internal/recipepilot"
)

type fakeEvaluator struct{ calls int }

func (evaluator *fakeEvaluator) Evaluate(_ context.Context, request EvaluationRequest) (EvaluationResponse, error) {
	evaluator.calls++
	answers := make(map[string]json.RawMessage, len(request.Questions))
	for key, question := range request.Questions {
		var value any
		switch question.Type {
		case "noul":
			value = map[string]any{"type": "noul", "noul": 0.8}
		case "choice":
			criteria := question.Criteria.(map[string]string)
			choice := ""
			probabilities := make(map[string]float64, len(criteria))
			for option := range criteria {
				if choice == "" {
					choice = option
				}
				probabilities[option] = 0
			}
			probabilities[choice] = 1
			value = map[string]any{"type": "choice", "choice": choice, "probabilities": probabilities, "confidence": 1.0}
		case "score":
			levels := question.Criteria.([]string)
			legend := make(map[string]string, len(levels))
			probabilities := make(map[string]float64, len(levels))
			for index, level := range levels {
				key := string(rune('0' + index))
				legend[key] = level
				probabilities[key] = 0
			}
			probabilities["0"] = 1
			value = map[string]any{"type": "score", "score": 0.0, "legend": legend, "probabilities": probabilities, "confidence": 1.0}
		}
		raw, _ := json.Marshal(value)
		answers[key] = raw
	}
	return EvaluationResponse{Model: "jev-1.13.0", Answers: answers, Usage: Usage{InputTokens: 10, OutputTokens: 2}}, nil
}

func TestRunnerEvaluatesAllProjectionsAndPublishesReview(t *testing.T) {
	root := t.TempDir()
	catalogPath := filepath.Join(root, "catalog.yaml")
	catalog := `version: 1
model: jev-1.13.0
questions:
  - key: role.main
    version: 1
    projection: full_recipe
    materialization: facet
    type: noul
    instructions: Is it a main?
    criteria: {true: Yes, false: No}
  - key: identity.temperature
    version: 1
    projection: full_recipe
    materialization: facet
    type: choice
    instructions: Temperature?
    criteria: {hot: Hot, cold: Cold}
  - key: composition.center
    version: 1
    projection: ingredients
    materialization: facet
    type: choice
    instructions: Center?
    criteria: {vegetables: Vegetables, other: Other}
  - key: effort.overall
    version: 1
    projection: method
    materialization: facet
    type: score
    instructions: Effort?
    criteria: [Easy, Hard]
`
	if err := os.WriteFile(catalogPath, []byte(catalog), 0o644); err != nil {
		t.Fatal(err)
	}
	sourceRun := filepath.Join(root, "source-run")
	if err := os.Mkdir(sourceRun, 0o755); err != nil {
		t.Fatal(err)
	}
	artifact := recipepilot.SourceArtifact{
		FormatVersion: 1, Source: recipepilot.ManifestSource{ID: "dinner", URL: "https://example.com"},
		Recipes: []recipepilot.RecipeCandidate{{Recipe: recipepilot.SourceRecipe{
			Name: json.RawMessage(`"Dinner"`), RecipeIngredient: json.RawMessage(`["1 onion"]`),
			RecipeInstructions: json.RawMessage(`["Cook it."]`),
		}}},
	}
	writeTestJSON(t, filepath.Join(sourceRun, "dinner.json"), artifact)
	writeTestJSON(t, filepath.Join(sourceRun, "index.json"), recipepilot.RunIndex{
		Artifacts: []recipepilot.ArtifactIndex{{SourceID: "dinner", File: "dinner.json", Status: "succeeded"}},
	})
	evaluator := &fakeEvaluator{}
	runner := Runner{Evaluator: evaluator, Now: func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 6, time.UTC) }}
	report, err := runner.Run(t.Context(), RunConfig{CatalogPath: catalogPath, SourcePath: sourceRun, OutputRoot: filepath.Join(root, "output")})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if evaluator.calls != 3 {
		t.Fatalf("evaluator calls = %d, want 3", evaluator.calls)
	}
	if report.Index.Succeeded != 1 || report.Index.InputTokens != 30 {
		t.Fatalf("index = %#v", report.Index)
	}
	for _, name := range []string{"index.json", "review.md", "source--dinner--0.json"} {
		if _, err := os.Stat(filepath.Join(report.Directory, name)); err != nil {
			t.Errorf("missing %s: %v", name, err)
		}
	}
}

func writeTestJSON(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
