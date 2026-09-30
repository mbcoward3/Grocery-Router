package httpapi

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/mbcoward3/grocery-router/internal/catalogcandidate"
	"github.com/mbcoward3/grocery-router/internal/ingest"
)

const minimumCatalogSearchScore = 20.0

type catalogSearchDocument struct {
	Title        string
	Ingredients  []string
	Instructions []string
}

type semanticSignal struct {
	Key        string
	Type       string
	Noul       float64
	Choice     string
	Score      float64
	Confidence float64
}

type semanticConcept struct {
	Key     string
	Type    string
	Choice  string
	Minimum float64
	Weight  float64
}

// Query vocabulary is global and versioned with the application. It translates
// ordinary cooking language into existing assessment dimensions; it is not
// duplicated into every recipe.
var semanticConcepts = map[string][]semanticConcept{
	"spice":     {{Key: "flavor.spice_aromatic_intensity", Type: "score", Minimum: 1.75, Weight: 16}},
	"spiced":    {{Key: "flavor.spice_aromatic_intensity", Type: "score", Minimum: 1.75, Weight: 16}},
	"aromatic":  {{Key: "flavor.spice_aromatic_intensity", Type: "score", Minimum: 1.75, Weight: 16}},
	"spicy":     {{Key: "flavor.heat", Type: "score", Minimum: 1.25, Weight: 18}},
	"heat":      {{Key: "flavor.heat", Type: "score", Minimum: 1.25, Weight: 18}},
	"poultry":   {{Key: "composition.center", Type: "choice", Choice: "poultry", Weight: 55}},
	"cold":      {{Key: "utility.cold_weather", Type: "noul", Minimum: .8, Weight: 52}, {Key: "identity.temperature", Type: "choice", Choice: "cold", Weight: 55}},
	"winter":    {{Key: "utility.cold_weather", Type: "noul", Minimum: .8, Weight: 52}},
	"summer":    {{Key: "utility.warm_weather", Type: "noul", Minimum: .65, Weight: 52}},
	"weeknight": {{Key: "utility.weeknight_friendly", Type: "noul", Minimum: .65, Weight: 52}},
	"hands-off": {{Key: "search.hands_off", Type: "noul", Minimum: .65, Weight: 52}},
	"hands off": {{Key: "search.hands_off", Type: "noul", Minimum: .65, Weight: 52}},
	"thanksgiving": {
		{Key: "tradition.holiday_celebration", Type: "noul", Minimum: .55, Weight: 60},
		{Key: "role.potluck_dish", Type: "noul", Minimum: .75, Weight: 24},
		{Key: "utility.crowd_friendly", Type: "noul", Minimum: .8, Weight: 18},
		{Key: "utility.make_ahead", Type: "noul", Minimum: .8, Weight: 14},
	},
}

func buildCatalogSearchDocument(status string, data json.RawMessage) catalogSearchDocument {
	if status == "reviewable" {
		if candidate, err := catalogcandidate.DecodeCandidate(data); err == nil {
			document := catalogSearchDocument{Title: candidate.Name.Value}
			for _, section := range candidate.IngredientSections {
				for _, ingredient := range section.Ingredients {
					document.Ingredients = append(document.Ingredients, ingredient.ItemPhrase, ingredient.GroceryProposal.Name)
				}
			}
			for _, section := range candidate.InstructionSections {
				document.Instructions = append(document.Instructions, section.Steps...)
			}
			return document
		}
	}
	var recipe ingest.Document
	if json.Unmarshal(data, &recipe) != nil {
		return catalogSearchDocument{}
	}
	document := catalogSearchDocument{Title: recipe.Name}
	for _, section := range recipe.IngredientSections {
		for _, ingredient := range section.Ingredients {
			document.Ingredients = append(document.Ingredients, ingredient.GroceryItem.Name, ingredient.SourceText)
		}
	}
	for _, section := range recipe.InstructionSections {
		document.Instructions = append(document.Instructions, section.Steps...)
	}
	return document
}

func catalogSearchScore(summary catalogRecipeSummary, document, profile json.RawMessage, query string) float64 {
	q := normalizeSearchText(query)
	if q == "" {
		return minimumCatalogSearchScore
	}
	searchDocument := buildCatalogSearchDocument(summary.Status, document)
	signals := semanticSignals(profile)
	score := lexicalSearchScore(searchDocument, q)
	for _, concept := range queryConcepts(q) {
		score += scoreSemanticConcept(signals, concept)
	}
	for _, signal := range signals {
		label := signal.Key
		if dot := strings.Index(label, "."); dot >= 0 {
			label = label[dot+1:]
		}
		label = normalizeSearchText(label)
		if signal.Type == "choice" {
			label = normalizeSearchText(signal.Choice)
		}
		if strings.Contains(label, q) || strings.Contains(q, label) {
			switch signal.Type {
			case "choice":
				score += 38 * confidenceOrOne(signal.Confidence)
			case "noul":
				if signal.Noul >= .65 {
					score += 38 * signal.Noul
				}
			}
		}
	}
	if score < minimumCatalogSearchScore {
		return 0
	}
	return score
}

func lexicalSearchScore(document catalogSearchDocument, query string) float64 {
	score := 0.0
	if strings.Contains(normalizeSearchText(document.Title), query) {
		score += 100
	}
	for _, ingredient := range document.Ingredients {
		if strings.Contains(normalizeSearchText(ingredient), query) {
			score += 55
		}
	}
	for _, instruction := range document.Instructions {
		if strings.Contains(normalizeSearchText(instruction), query) {
			score += 5
		}
	}
	return score
}

func semanticSignals(data json.RawMessage) map[string]semanticSignal {
	var profile compactSemanticProfile
	if json.Unmarshal(data, &profile) != nil {
		return nil
	}
	result := make(map[string]semanticSignal, len(profile.Answers))
	for key, answer := range profile.Answers {
		signal := semanticSignal{Key: key, Type: answer.Type, Choice: answer.Choice, Confidence: answer.Confidence}
		if answer.Noul != nil {
			signal.Noul = *answer.Noul
		}
		if answer.Score != nil {
			signal.Score = *answer.Score
		}
		result[key] = signal
	}
	return result
}

func queryConcepts(query string) []semanticConcept {
	result := append([]semanticConcept(nil), semanticConcepts[query]...)
	for word, concepts := range semanticConcepts {
		if word != query && strings.Contains(query, word) {
			result = append(result, concepts...)
		}
	}
	return result
}

func scoreSemanticConcept(signals map[string]semanticSignal, concept semanticConcept) float64 {
	signal, ok := signals[concept.Key]
	if !ok || signal.Type != concept.Type {
		return 0
	}
	switch concept.Type {
	case "choice":
		if signal.Choice == concept.Choice {
			return concept.Weight * confidenceOrOne(signal.Confidence)
		}
	case "noul":
		if signal.Noul >= concept.Minimum {
			return concept.Weight * signal.Noul
		}
	case "score":
		if signal.Score >= concept.Minimum {
			return concept.Weight * signal.Score * confidenceOrOne(signal.Confidence)
		}
	}
	return 0
}

func confidenceOrOne(value float64) float64 {
	if value <= 0 {
		return 1
	}
	return value
}

func normalizeSearchText(value string) string {
	value = strings.ToLower(strings.ReplaceAll(value, "_", " "))
	return strings.Join(strings.Fields(value), " ")
}

type scoredCatalogRecipe struct {
	Summary catalogRecipeSummary
	Score   float64
}

func sortCatalogSearchResults(results []scoredCatalogRecipe) {
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Summary.Name < results[j].Summary.Name
		}
		return results[i].Score > results[j].Score
	})
}
