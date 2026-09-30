package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"

	"github.com/mbcoward3/grocery-router/internal/catalog"
	"github.com/mbcoward3/grocery-router/internal/catalogcandidate"
	"github.com/mbcoward3/grocery-router/internal/ingest"
	"github.com/mbcoward3/grocery-router/internal/store"
)

type catalogRecipeSummary struct {
	CatalogID            int64         `json:"catalogId"`
	Key                  string        `json:"key"`
	Name                 string        `json:"name"`
	Status               string        `json:"status"`
	HouseholdState       *string       `json:"householdState"`
	MaterializedRecipeID *int64        `json:"materializedRecipeId"`
	ImageURL             *string       `json:"imageUrl"`
	Yield                *string       `json:"yield"`
	HandsOn              durationRange `json:"handsOn"`
	Unattended           durationRange `json:"unattended"`
	Facets               []string      `json:"facets"`
}

type catalogRecipeDetail struct {
	catalogRecipeSummary
	Sources      []recipeSource             `json:"sources"`
	Ingredients  []recipeIngredientSection  `json:"ingredientSections"`
	Instructions []recipeInstructionSection `json:"instructionSections"`
}

type catalogMembershipRequest struct {
	State string `json:"state"`
}

type compactSemanticProfile struct {
	Answers map[string]compactSemanticAnswer `json:"answers"`
}

type compactSemanticAnswer struct {
	Type        string   `json:"type"`
	Noul        *float64 `json:"noul"`
	Choice      string   `json:"choice"`
	Score       *float64 `json:"score"`
	Confidence  float64  `json:"confidence"`
	Disposition string   `json:"disposition"`
}

func (server *Server) listCatalogRecipes(response http.ResponseWriter, request *http.Request) {
	rows, err := server.queries.ListCatalogRecipes(request.Context(), server.householdID)
	if err != nil {
		writeInternalError(response, err)
		return
	}
	query := strings.ToLower(strings.TrimSpace(request.URL.Query().Get("q")))
	results := make([]scoredCatalogRecipe, 0, len(rows))
	for _, row := range rows {
		summary, err := catalogSummaryFromRow(row)
		if err != nil {
			writeInternalError(response, err)
			return
		}
		score := catalogSearchScore(summary, row.RecipeDocument, row.SemanticProfile, query)
		if query != "" && score == 0 {
			continue
		}
		results = append(results, scoredCatalogRecipe{Summary: summary, Score: score})
	}
	if query != "" {
		sortCatalogSearchResults(results)
	}
	result := make([]catalogRecipeSummary, 0, len(results))
	for _, match := range results {
		result = append(result, match.Summary)
	}
	writeJSON(response, http.StatusOK, map[string]any{"recipes": result})
}

func (server *Server) getCatalogRecipe(response http.ResponseWriter, request *http.Request) {
	id, ok := pathID(response, request, "catalogRecipeID")
	if !ok {
		return
	}
	row, err := server.queries.GetCatalogRecipe(request.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(response, http.StatusNotFound, "catalog_recipe_not_found", "The shared recipe was not found.")
		return
	}
	if err != nil {
		writeInternalError(response, err)
		return
	}
	membership, membershipErr := server.queries.GetHouseholdCatalogMembership(request.Context(), store.GetHouseholdCatalogMembershipParams{HouseholdID: server.householdID, CatalogRecipeID: id})
	if membershipErr != nil && !errors.Is(membershipErr, sql.ErrNoRows) {
		writeInternalError(response, membershipErr)
		return
	}
	detail, err := catalogDetailFromRow(row)
	if err != nil {
		writeInternalError(response, err)
		return
	}
	if membershipErr == nil {
		detail.HouseholdState = &membership.State
		detail.MaterializedRecipeID = &membership.MaterializedRecipeID
	}
	writeJSON(response, http.StatusOK, detail)
}

func (server *Server) setCatalogMembership(response http.ResponseWriter, request *http.Request) {
	id, ok := pathID(response, request, "catalogRecipeID")
	if !ok {
		return
	}
	var input catalogMembershipRequest
	if !decodeJSON(response, request, &input) {
		return
	}
	result, err := server.catalog.AddToHousehold(request.Context(), server.householdID, id, input.State)
	switch {
	case errors.Is(err, catalog.ErrNotVerified):
		writeError(response, http.StatusConflict, "catalog_recipe_not_verified", "This shared recipe still requires human approval.")
	case errors.Is(err, sql.ErrNoRows):
		writeError(response, http.StatusNotFound, "catalog_recipe_not_found", "The shared recipe was not found.")
	case err != nil:
		writeError(response, http.StatusUnprocessableEntity, "catalog_adoption_failed", err.Error())
	default:
		writeJSON(response, http.StatusOK, map[string]any{"catalogRecipeId": id, "state": result.Membership.State, "recipeId": result.MaterializedRecipe.ID})
	}
}

func catalogSummaryFromRow(row store.ListCatalogRecipesRow) (catalogRecipeSummary, error) {
	summary, err := catalogSummary(row.ID, row.Key, row.Name, row.Status, row.RecipeDocument, row.SemanticProfile)
	if err != nil {
		return catalogRecipeSummary{}, err
	}
	if row.MembershipState.Valid {
		summary.HouseholdState = &row.MembershipState.String
	}
	if row.MaterializedRecipeID.Valid {
		summary.MaterializedRecipeID = &row.MaterializedRecipeID.Int64
	}
	return summary, nil
}

func catalogSummary(id int64, key, name, status string, document, profile json.RawMessage) (catalogRecipeSummary, error) {
	result := catalogRecipeSummary{CatalogID: id, Key: key, Name: name, Status: status, Facets: semanticFacets(profile)}
	if status == "reviewable" {
		candidate, candidateErr := catalogcandidate.DecodeCandidate(document)
		if candidateErr == nil {
			result.ImageURL = stringPointer(candidate.ImageURL.Value)
			result.Yield = stringPointer(candidate.Yield.Value)
			result.HandsOn = candidateDuration(candidate.HandsOn.Value)
			result.Unattended = candidateDuration(candidate.Unattended.Value)
			return result, nil
		}
		// Programmatic review fixtures and future review migrations may use the
		// approved document shape while the outer catalog status remains reviewable.
		var doc ingest.Document
		if err := json.Unmarshal(document, &doc); err != nil {
			return result, candidateErr
		}
		result.ImageURL = stringPointer(doc.ImageURL)
		result.Yield = stringPointer(doc.Yield)
		result.HandsOn = ingestDuration(doc.HandsOn)
		result.Unattended = ingestDuration(doc.Unattended)
		return result, nil
	}
	var doc ingest.Document
	if err := json.Unmarshal(document, &doc); err != nil {
		return result, err
	}
	result.ImageURL = stringPointer(doc.ImageURL)
	result.Yield = stringPointer(doc.Yield)
	result.HandsOn = ingestDuration(doc.HandsOn)
	result.Unattended = ingestDuration(doc.Unattended)
	return result, nil
}

func catalogDetailFromRow(row store.CatalogRecipe) (catalogRecipeDetail, error) {
	summary, err := catalogSummary(row.ID, row.Key, row.Name, row.Status, row.RecipeDocument, row.SemanticProfile)
	if err != nil {
		return catalogRecipeDetail{}, err
	}
	detail := catalogRecipeDetail{catalogRecipeSummary: summary, Sources: []recipeSource{{Relationship: "source", Attribution: row.SourceAttribution, URL: nullableString(row.SourceUrl), Primary: true}}, Ingredients: []recipeIngredientSection{}, Instructions: []recipeInstructionSection{}}
	if row.Status == "reviewable" {
		candidate, candidateErr := catalogcandidate.DecodeCandidate(row.RecipeDocument)
		if candidateErr == nil {
			id := int64(1)
			for _, section := range candidate.IngredientSections {
				out := recipeIngredientSection{Name: section.Name, Ingredients: []recipeIngredient{}}
				for _, ingredient := range section.Ingredients {
					item := ingredient.GroceryProposal.Name
					out.Ingredients = append(out.Ingredients, recipeIngredient{ID: id, SourceText: candidateIngredientDisplay(ingredient), Preparation: stringPointer(ingredient.Preparation), Optional: ingredient.Optional, ShoppingItem: stringPointer(item)})
					id++
				}
				detail.Ingredients = append(detail.Ingredients, out)
			}
			for _, section := range candidate.InstructionSections {
				out := recipeInstructionSection{Name: section.Name, Steps: []recipeStep{}}
				for _, step := range section.Steps {
					out.Steps = append(out.Steps, recipeStep{ID: id, Instruction: step})
					id++
				}
				detail.Instructions = append(detail.Instructions, out)
			}
			return detail, nil
		}
	}
	var doc ingest.Document
	if err := json.Unmarshal(row.RecipeDocument, &doc); err != nil {
		return detail, err
	}
	id := int64(1)
	for _, section := range doc.IngredientSections {
		out := recipeIngredientSection{Name: section.Name, Ingredients: []recipeIngredient{}}
		for _, ingredient := range section.Ingredients {
			out.Ingredients = append(out.Ingredients, recipeIngredient{ID: id, SourceText: ingredient.SourceText, Preparation: stringPointer(ingredient.Preparation), Optional: ingredient.Optional, DisplayNote: stringPointer(ingredient.Note), ShoppingItem: stringPointer(ingredient.GroceryItem.Name)})
			id++
		}
		detail.Ingredients = append(detail.Ingredients, out)
	}
	for _, section := range doc.InstructionSections {
		out := recipeInstructionSection{Name: section.Name, Steps: []recipeStep{}}
		for _, step := range section.Steps {
			out.Steps = append(out.Steps, recipeStep{ID: id, Instruction: step})
			id++
		}
		detail.Instructions = append(detail.Instructions, out)
	}
	return detail, nil
}

func semanticFacets(data json.RawMessage) []string {
	var profile compactSemanticProfile
	if json.Unmarshal(data, &profile) != nil {
		return []string{}
	}
	facets := make([]string, 0)
	for key, answer := range profile.Answers {
		if answer.Type == "choice" && answer.Choice != "" && answer.Choice != "unclear" {
			facets = append(facets, strings.ReplaceAll(answer.Choice, "_", " "))
		} else if answer.Type == "noul" && answer.Noul != nil && *answer.Noul >= 0.75 {
			label := key
			if dot := strings.Index(label, "."); dot >= 0 {
				label = label[dot+1:]
			}
			facets = append(facets, strings.ReplaceAll(label, "_", " "))
		}
	}
	sort.Strings(facets)
	return facets
}

func candidateIngredientDisplay(ingredient catalogcandidate.Ingredient) string {
	item := strings.ToLower(strings.TrimSpace(ingredient.ItemPhrase))
	if item == "" {
		return ingredient.SourceText
	}
	quantity := ingredient.Quantity
	prefix := ""
	switch {
	case quantity.Package != nil:
		count := quantity.Package.Count
		if count == "" {
			count = quantity.Amount
		}
		packageType := quantity.Package.Type
		if count != "1" {
			packageType = pluralPackageType(packageType)
		}
		if quantity.Package.Size != "" {
			prefix = formatDisplayNumber(count) + " × " + formatDisplayNumber(quantity.Package.Size) + " " + quantity.Package.Unit + " " + packageType
		} else {
			prefix = formatDisplayNumber(count) + " " + packageType
		}
	case quantity.Kind == "range":
		prefix = formatDisplayNumber(quantity.Amount) + "–" + formatDisplayNumber(quantity.Maximum)
		if quantity.Unit != "" && quantity.Unit != "each" {
			prefix += " " + quantity.Unit
		}
	case quantity.Kind == "exact":
		prefix = formatDisplayNumber(quantity.Amount)
		if quantity.Unit != "" && quantity.Unit != "each" {
			prefix += " " + quantity.Unit
		}
	}
	result := strings.TrimSpace(prefix + " " + item)
	if preparation := strings.TrimSpace(ingredient.Preparation); preparation != "" {
		result += ", " + preparation
	}
	return result
}

func formatDisplayNumber(value string) string {
	replacer := strings.NewReplacer(
		"1/8", "⅛", "1/4", "¼", "1/3", "⅓", "3/8", "⅜", "1/2", "½",
		"5/8", "⅝", "2/3", "⅔", "3/4", "¾", "7/8", "⅞",
	)
	return replacer.Replace(value)
}

func pluralPackageType(value string) string {
	if strings.HasSuffix(value, "s") {
		return value
	}
	if strings.HasSuffix(value, "x") || strings.HasSuffix(value, "ch") || strings.HasSuffix(value, "sh") {
		return value + "es"
	}
	return value + "s"
}

func stringPointer(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}
func candidateDuration(value *catalogcandidate.Duration) durationRange {
	if value == nil {
		return durationRange{}
	}
	return durationRange{MinimumMinutes: &value.Min, MaximumMinutes: &value.Max}
}
func ingestDuration(value *ingest.Duration) durationRange {
	if value == nil {
		return durationRange{}
	}
	return durationRange{MinimumMinutes: &value.Min, MaximumMinutes: &value.Max}
}
