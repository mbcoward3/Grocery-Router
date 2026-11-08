package recipeintel

import (
	"encoding/json"
	"fmt"
	"html"
	"sort"
	"strings"

	"github.com/mbcoward3/grocery-router/internal/ingest"
	"github.com/mbcoward3/grocery-router/internal/recipepilot"
)

// Candidate is a source or approved recipe normalized only enough for assessment.
type Candidate struct {
	ID           string
	SourceKind   string
	SourceID     string
	Candidate    int
	Name         string
	Description  []string
	Categories   []string
	Keywords     []string
	Yield        []string
	Times        map[string][]string
	Ingredients  []string
	Instructions []Instruction
}

// Instruction retains an optional section and one source instruction text.
type Instruction struct {
	Section string `json:"section,omitempty"`
	Text    string `json:"text"`
}

// ProjectedState limits Jev context to fields relevant to a question group.
type ProjectedState struct {
	Recipe      map[string]any `json:"recipe"`
	Ingredients []string       `json:"ingredients,omitempty"`
	Method      []Instruction  `json:"method,omitempty"`
}

// CandidatesFromSourceArtifact projects every extracted Recipe node in one artifact.
func CandidatesFromSourceArtifact(artifact recipepilot.SourceArtifact) ([]Candidate, error) {
	if artifact.Error != "" {
		return nil, fmt.Errorf("source artifact %q failed acquisition: %s", artifact.Source.ID, artifact.Error)
	}
	result := make([]Candidate, 0, len(artifact.Recipes))
	for index, source := range artifact.Recipes {
		candidate, err := candidateFromSourceRecipe(artifact.Source.ID, index, source.Recipe)
		if err != nil {
			return nil, fmt.Errorf("source %q recipe %d: %w", artifact.Source.ID, index, err)
		}
		result = append(result, candidate)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("source artifact %q has no recipes", artifact.Source.ID)
	}
	return result, nil
}

func candidateFromSourceRecipe(sourceID string, index int, recipe recipepilot.SourceRecipe) (Candidate, error) {
	name, err := rawTextValues(recipe.Name)
	if err != nil {
		return Candidate{}, fmt.Errorf("decode name: %w", err)
	}
	if len(name) == 0 {
		name, err = rawTextValues(recipe.Headline)
		if err != nil {
			return Candidate{}, fmt.Errorf("decode headline: %w", err)
		}
	}
	ingredients, err := rawTextValues(recipe.RecipeIngredient)
	if err != nil {
		return Candidate{}, fmt.Errorf("decode ingredients: %w", err)
	}
	instructions, err := rawInstructions(recipe.RecipeInstructions)
	if err != nil {
		return Candidate{}, fmt.Errorf("decode instructions: %w", err)
	}
	candidate := Candidate{
		ID: fmt.Sprintf("source--%s--%d", sourceID, index), SourceKind: "source_artifact", SourceID: sourceID,
		Candidate: index, Name: first(name), Ingredients: ingredients, Instructions: instructions,
		Times: make(map[string][]string),
	}
	fields := []struct {
		raw         json.RawMessage
		destination *[]string
	}{
		{recipe.Description, &candidate.Description},
		{recipe.RecipeCategory, &candidate.Categories},
		{recipe.Keywords, &candidate.Keywords},
		{recipe.RecipeYield, &candidate.Yield},
	}
	for _, field := range fields {
		values, decodeErr := rawTextValues(field.raw)
		if decodeErr != nil {
			return Candidate{}, decodeErr
		}
		*field.destination = values
	}
	for key, raw := range map[string]json.RawMessage{
		"prep": recipe.PrepTime, "cook": recipe.CookTime, "total": recipe.TotalTime,
	} {
		values, decodeErr := rawTextValues(raw)
		if decodeErr != nil {
			return Candidate{}, decodeErr
		}
		if len(values) > 0 {
			candidate.Times[key] = values
		}
	}
	return candidate, nil
}

// CandidateFromDocument projects an approved bootstrap recipe without changing it.
func CandidateFromDocument(document ingest.Document) Candidate {
	candidate := Candidate{
		ID: "corpus--" + document.Key, SourceKind: "approved_corpus", SourceID: document.Key,
		Name: document.Name, Yield: nonEmpty(document.Yield), Times: make(map[string][]string),
	}
	if document.HandsOn != nil {
		candidate.Times["hands_on_minutes"] = []string{formatRange(document.HandsOn.Min, document.HandsOn.Max)}
	}
	if document.Unattended != nil {
		candidate.Times["unattended_minutes"] = []string{formatRange(document.Unattended.Min, document.Unattended.Max)}
	}
	for _, section := range document.IngredientSections {
		for _, ingredient := range section.Ingredients {
			candidate.Ingredients = append(candidate.Ingredients, ingredient.SourceText)
		}
	}
	for _, section := range document.InstructionSections {
		for _, step := range section.Steps {
			candidate.Instructions = append(candidate.Instructions, Instruction{Section: section.Name, Text: step})
		}
	}
	return candidate
}

// Project returns one focused state shape for a catalog projection.
func (candidate Candidate) Project(name string) (ProjectedState, error) {
	identity := map[string]any{
		"name": candidate.Name, "description": candidate.Description, "source_categories": candidate.Categories,
		"source_keywords": candidate.Keywords, "yield": candidate.Yield, "times": candidate.Times,
	}
	state := ProjectedState{Recipe: identity}
	switch name {
	case "identity":
	case "ingredients":
		state.Ingredients = candidate.Ingredients
	case "method":
		state.Ingredients = candidate.Ingredients
		state.Method = candidate.Instructions
	case "full_recipe":
		state.Ingredients = candidate.Ingredients
		state.Method = candidate.Instructions
	default:
		return ProjectedState{}, fmt.Errorf("unknown projection %q", name)
	}
	return state, nil
}

func rawTextValues(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	var result []string
	collectText(value, &result)
	return compactStrings(result), nil
}

func collectText(value any, destination *[]string) {
	switch typed := value.(type) {
	case string:
		*destination = append(*destination, typed)
	case []any:
		for _, item := range typed {
			collectText(item, destination)
		}
	case map[string]any:
		for _, key := range []string{"name", "text", "value"} {
			if field, ok := typed[key]; ok {
				collectText(field, destination)
				return
			}
		}
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			collectText(typed[key], destination)
		}
	case json.Number:
		*destination = append(*destination, typed.String())
	case float64:
		*destination = append(*destination, fmt.Sprint(typed))
	}
}

func rawInstructions(raw json.RawMessage) ([]Instruction, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	var result []Instruction
	collectInstructions(value, "", &result)
	return result, nil
}

func collectInstructions(value any, section string, destination *[]Instruction) {
	switch typed := value.(type) {
	case string:
		text := cleanText(typed)
		if text != "" {
			*destination = append(*destination, Instruction{Section: section, Text: text})
		}
	case []any:
		for _, item := range typed {
			collectInstructions(item, section, destination)
		}
	case map[string]any:
		kind := strings.ToLower(firstAnyString(typed["@type"]))
		if strings.Contains(kind, "section") {
			if name := firstAnyString(typed["name"]); name != "" {
				section = cleanText(name)
			}
			if children, ok := typed["itemListElement"]; ok {
				collectInstructions(children, section, destination)
				return
			}
		}
		if text := firstAnyString(typed["text"]); text != "" {
			collectInstructions(text, section, destination)
			return
		}
		if children, ok := typed["itemListElement"]; ok {
			collectInstructions(children, section, destination)
		}
	}
}

func firstAnyString(value any) string {
	var values []string
	collectText(value, &values)
	return first(compactStrings(values))
}

func compactStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = cleanText(value)
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func cleanText(value string) string {
	return strings.Join(strings.Fields(html.UnescapeString(value)), " ")
}
func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
func nonEmpty(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return []string{value}
}
func formatRange(minimum, maximum int64) string {
	if minimum == maximum {
		return fmt.Sprint(minimum)
	}
	return fmt.Sprintf("%d-%d", minimum, maximum)
}
