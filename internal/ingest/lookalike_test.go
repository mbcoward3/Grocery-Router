package ingest_test

import (
	"strings"
	"testing"

	"github.com/mbcoward3/grocery-router/internal/ingest"
)

func lookAlikeCorpus(names ...string) []ingest.Document {
	section := ingest.IngredientSection{}
	for _, name := range names {
		key := strings.ReplaceAll(strings.ToLower(name), " ", "-")
		section.Ingredients = append(section.Ingredients, ingest.Ingredient{GroceryItem: ingest.GroceryItem{Key: key, Name: name}})
	}
	return []ingest.Document{{IngredientSections: []ingest.IngredientSection{section}}}
}

func TestLookAlikeItemsNeedADecision(t *testing.T) {
	documents := lookAlikeCorpus("Onion", "Yellow Onion", "Tomato", "Tomatoes", "Milk", "Garlic")
	pairs := ingest.FindLookAlikes(documents)
	if len(pairs) != 2 || pairs[0].Items != [2]string{"onion", "yellow-onion"} || pairs[1].Items != [2]string{"tomato", "tomatoes"} {
		t.Fatalf("look-alikes = %#v", pairs)
	}

	// A new look-alike with no recorded decision fails the audit.
	reviews, err := ingest.ParseItemReviews(strings.NewReader("pairs:\n  - {items: [onion, yellow-onion], decision: distinct}\n"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ingest.CheckLookAlikes(documents, reviews); err == nil || !strings.Contains(err.Error(), "Tomatoes (tomatoes)") {
		t.Fatalf("missing decision error = %v", err)
	}

	// Undecided pairs pass but are reported.
	reviews, err = ingest.ParseItemReviews(strings.NewReader(
		"pairs:\n  - {items: [onion, yellow-onion], decision: group, group: onions}\n  - {items: [tomato, tomatoes], decision: undecided}\n"))
	if err != nil {
		t.Fatal(err)
	}
	undecided, err := ingest.CheckLookAlikes(documents, reviews)
	if err != nil || len(undecided) != 1 || undecided[0].Items[1] != "tomatoes" {
		t.Fatalf("undecided = %#v, %v", undecided, err)
	}
}

func TestItemReviewsRejectInvalidEntries(t *testing.T) {
	for name, input := range map[string]string{
		"unknown decision": "pairs:\n  - {items: [a, b], decision: maybe}\n",
		"unsorted":         "pairs:\n  - {items: [b, a], decision: distinct}\n",
		"one item":         "pairs:\n  - {items: [a], decision: distinct}\n",
		"duplicate":        "pairs:\n  - {items: [a, b], decision: distinct}\n  - {items: [a, b], decision: alias}\n",
		"group needs name": "pairs:\n  - {items: [a, b], decision: group}\n",
		"stray group":      "pairs:\n  - {items: [a, b], decision: distinct, group: x}\n",
		"unknown field":    "pairs:\n  - {items: [a, b], decision: distinct, reason: x}\n",
	} {
		if _, err := ingest.ParseItemReviews(strings.NewReader(input)); err == nil {
			t.Errorf("%s: parsed without error", name)
		}
	}
}
