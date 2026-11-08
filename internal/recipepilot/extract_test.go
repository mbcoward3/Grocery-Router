package recipepilot

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestExtractPreservesRecipeJSONLDShapesAndProvenance(t *testing.T) {
	html := `<!doctype html><html><head>
<link rel="alternate canonical" href="/canonical-recipe">
<script type="application/ld+json; charset=utf-8">{
  "@context":"https://schema.org",
  "@graph":[
    {"@type":"WebPage","name":"Page"},
    {"@type":["Thing","Recipe"],"@id":"#recipe","name":"Soup","recipeYield":["4 bowls",4],
     "recipeIngredient":["1 onion"],"recipeInstructions":{"@type":"HowToStep","text":"Cook"},
     "customSourceField":{"kept":true}}
  ]
}</script>
<script type="application/ld+json">{"@type":"Recipe","name":"Second","image":{"url":"image.jpg"}}</script>
<script type="application/ld+json">{broken</script>
</head></html>`
	extraction, err := Extract([]byte(html), "https://example.com/original")
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(extraction.Recipes) != 2 {
		t.Fatalf("recipes = %d, want 2", len(extraction.Recipes))
	}
	if got := extraction.Recipes[0].JSONPath; got != "$/@graph[1]" {
		t.Errorf("JSONPath = %q", got)
	}
	if got := extraction.Canonical[0]; got.Href != "/canonical-recipe" || got.Resolved != "https://example.com/canonical-recipe" {
		t.Errorf("canonical = %#v", got)
	}
	if len(extraction.JSONLDBlocks) != 3 || extraction.JSONLDBlocks[2].ParseError == "" || len(extraction.Warnings) != 1 {
		t.Errorf("blocks/warnings = %#v / %#v", extraction.JSONLDBlocks, extraction.Warnings)
	}
	first := extraction.Recipes[0].Recipe
	if string(first.RecipeYield) != `["4 bowls",4]` {
		t.Errorf("recipeYield = %s", first.RecipeYield)
	}
	if !strings.Contains(string(first.Raw), `"customSourceField":{"kept":true}`) {
		t.Errorf("raw recipe lost unknown field: %s", first.Raw)
	}
	var complete map[string]any
	if err := json.Unmarshal(extraction.Recipes[0].JSONLD, &complete); err != nil {
		t.Fatalf("complete JSON-LD is invalid: %v", err)
	}
}

func TestExtractReportsNoRecipe(t *testing.T) {
	html := `<script type="application/ld+json">{"@type":"WebPage"}</script>`
	extraction, err := Extract([]byte(html), "https://example.com/")
	if err == nil || !strings.Contains(err.Error(), "no Schema.org Recipe") {
		t.Fatalf("Extract error = %v", err)
	}
	if len(extraction.JSONLDBlocks) != 1 {
		t.Fatalf("blocks = %d", len(extraction.JSONLDBlocks))
	}
}
