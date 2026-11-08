package catalogcandidate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mbcoward3/grocery-router/internal/recipepilot"
)

func TestBuildBatchWritesDeterministicCandidatesAndIndex(t *testing.T) {
	runDir, proposalDir, outputDir := writeFixture(t, 1)
	report, err := BuildBatch(BuildOptions{SourceRunDir: runDir, ProposalDir: proposalDir, OutputDir: outputDir, Agent: AgentProvenance{Provider: "pi", Model: "fixture", PromptSHA256: strings.Repeat("a", 64), InputSHA256: strings.Repeat("b", 64)}, Now: func() time.Time { return fixtureTime() }})
	if err != nil {
		t.Fatalf("BuildBatch() error = %v", err)
	}
	if len(report.Candidates) != 1 {
		t.Fatalf("candidates = %d, want 1", len(report.Candidates))
	}
	candidateData, err := os.ReadFile(filepath.Join(outputDir, "fixture-soup.candidate.json"))
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := DecodeCandidate(candidateData)
	if err != nil {
		t.Fatal(err)
	}
	if candidate.Source.ArtifactSHA256 == "" || candidate.Source.RunIndexSHA256 == "" || candidate.Source.Selected.RecipeRawCanonical == "" {
		t.Fatalf("missing source digests: %+v", candidate.Source)
	}
	if err := candidate.Validate([]string{"1 cup carrots", "2 cups water"}); err != nil {
		t.Fatal(err)
	}
	markdown, err := os.ReadFile(filepath.Join(outputDir, "fixture-soup.review.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(markdown), "## Blockers") || !strings.Contains(string(markdown), "grocery mapping is unresolved") {
		t.Fatalf("markdown did not emphasize blockers:\n%s", markdown)
	}
	index, err := os.ReadFile(filepath.Join(outputDir, "review-index.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(index), "fixture-soup.review.md") {
		t.Fatalf("review index missing candidate: %s", index)
	}
}

func TestBuildBatchRejectsIngredientCoverageDrift(t *testing.T) {
	runDir, proposalDir, outputDir := writeFixture(t, 1)
	path := filepath.Join(proposalDir, "000.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), "2 cups water", "2 cups stock", 1))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = BuildBatch(BuildOptions{SourceRunDir: runDir, ProposalDir: proposalDir, OutputDir: outputDir})
	if err == nil || !strings.Contains(err.Error(), "source-line") {
		t.Fatalf("BuildBatch error = %v, want source-line coverage error", err)
	}
}

func TestBuildBatchRejectsTwentySixCandidates(t *testing.T) {
	runDir, proposalDir, outputDir := writeFixture(t, 26)
	_, err := BuildBatch(BuildOptions{SourceRunDir: runDir, ProposalDir: proposalDir, OutputDir: outputDir})
	if err == nil || !strings.Contains(err.Error(), "maximum") {
		t.Fatalf("BuildBatch error = %v, want max error", err)
	}
}

func writeFixture(t *testing.T, count int) (string, string, string) {
	t.Helper()
	root := t.TempDir()
	runDir := filepath.Join(root, "run")
	proposalDir := filepath.Join(root, "proposals")
	outputDir := filepath.Join(root, "out")
	must(t, os.MkdirAll(runDir, 0o755))
	must(t, os.MkdirAll(proposalDir, 0o755))
	index := recipepilot.RunIndex{FormatVersion: recipepilot.OutputVersion, Implementation: recipepilot.Implementation, StartedAt: fixtureTime(), ManifestPath: "pilot/recipe-sources.yaml", ManifestSHA256: strings.Repeat("c", 64), Sources: count, Succeeded: count}
	for i := 0; i < count; i++ {
		sourceID := "source-" + pad(i)
		file := sourceID + ".json"
		index.Artifacts = append(index.Artifacts, recipepilot.ArtifactIndex{SourceID: sourceID, File: file, Status: "succeeded"})
		raw := json.RawMessage(`{"@type":"Recipe","name":"Fixture Soup","recipeIngredient":["1 cup carrots","2 cups water"],"recipeInstructions":["Simmer."]}`)
		artifact := recipepilot.SourceArtifact{FormatVersion: recipepilot.OutputVersion, Source: recipepilot.ManifestSource{ID: sourceID, URL: "https://example.test/recipe"}, Fetch: &recipepilot.FetchMetadata{RequestedURL: "https://example.test/recipe", FinalURL: "https://example.test/recipe", FetchedAt: fixtureTime(), BodySHA256: strings.Repeat("d", 64)}, JSONLDBlocks: []recipepilot.JSONLDBlock{{ScriptIndex: 0, SHA256: strings.Repeat("e", 64)}}, Recipes: []recipepilot.RecipeCandidate{{ScriptIndex: 0, JSONPath: "$[0]", JSONLD: raw, Recipe: recipepilot.SourceRecipe{RecipeIngredient: json.RawMessage(`["1 cup carrots","2 cups water"]`), Raw: raw}}}}
		writeJSONFile(t, filepath.Join(runDir, file), artifact)
		proposal := proposalFixture("fixture-soup-" + pad(i))
		if count == 1 {
			proposal = proposalFixture("fixture-soup")
		}
		writeJSONFile(t, filepath.Join(proposalDir, pad(i)+".json"), proposal)
	}
	writeJSONFile(t, filepath.Join(runDir, "index.json"), index)
	return runDir, proposalDir, outputDir
}

func proposalFixture(key string) Proposal {
	return Proposal{Key: key, Name: "Fixture Soup", ImageURL: "https://example.test/image.jpg", Yield: "4 servings", HandsOn: &Duration{Min: 10, Max: 10}, Unattended: &Duration{Min: 20, Max: 30}, IngredientSections: []ProposalIngredientSection{{Name: "Soup", Ingredients: []ProposalIngredient{{SourceText: "1 cup carrots", Quantity: Quantity{Kind: "exact", Amount: "1", Unit: "cup"}, ItemPhrase: "carrots", GroceryProposal: GroceryProposal{State: "create", Key: "carrots", Name: "Carrots", StoreSection: "produce", ShoppingMode: "measured"}}, {SourceText: "2 cups water", Quantity: Quantity{Kind: "exact", Amount: "2", Unit: "cup"}, ItemPhrase: "water", NonShopping: true, GroceryProposal: GroceryProposal{State: "unresolved"}}}}}, InstructionSections: []InstructionSection{{Name: "Cook", Steps: []string{"Simmer until hot."}}}, Issues: []Issue{{Severity: "blocker", Field: "ingredients[1]", Message: "water mapping requires review"}}, SourceNotes: []SourceNote{}, Decisions: []Decision{}}
}

func writeJSONFile(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	must(t, os.WriteFile(path, append(data, '\n'), 0o644))
}
func fixtureTime() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }
func pad(i int) string {
	if i < 10 {
		return "00" + string(rune('0'+i))
	}
	return "0" + string(rune('0'+i/10)) + string(rune('0'+i%10))
}
func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
