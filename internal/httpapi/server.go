// Package httpapi exposes Grocery Router application services over JSON HTTP.
package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mbcoward3/grocery-router/internal/store"
	"github.com/mbcoward3/grocery-router/internal/week"
)

// Clock returns the local time used to resolve the current Sunday.
type Clock func() time.Time

// AuthorizeHousehold checks membership after ServeMux has resolved the path value.
type AuthorizeHousehold func(context.Context, string) bool

// Server is the HTTP boundary around corpus reads, week operations, and household settings.
type Server struct {
	queries   *store.Queries
	weeks     *week.Service
	now       Clock
	authorize AuthorizeHousehold
	handler   http.Handler
}

// New constructs an API handler. A nil clock uses time.Now.
func New(db *sql.DB, weeks *week.Service, clock Clock, authorize AuthorizeHousehold) *Server {
	if clock == nil {
		clock = time.Now
	}
	if authorize == nil {
		panic("HTTP API requires an authorizer")
	}
	server := &Server{queries: store.New(db), weeks: weeks, now: clock, authorize: authorize}
	mux := http.NewServeMux()
	mux.Handle("GET /api/v2/households/{householdID}/recipes", server.guard(http.HandlerFunc(server.listRecipes)))
	mux.Handle("GET /api/v2/households/{householdID}/recipes/{recipeID}", server.guard(http.HandlerFunc(server.getRecipe)))
	mux.Handle("GET /api/v2/households/{householdID}/week/current", server.guard(http.HandlerFunc(server.currentWeek)))
	mux.Handle("GET /api/v2/households/{householdID}/weeks/history", server.guard(http.HandlerFunc(server.weekHistory)))
	mux.Handle("GET /api/v2/households/{householdID}/weeks/history/{weekID}", server.guard(http.HandlerFunc(server.historicalWeek)))
	mux.Handle("POST /api/v2/households/{householdID}/week/current/generate", server.guard(http.HandlerFunc(server.generateWeek)))
	mux.Handle("POST /api/v2/households/{householdID}/week/current/close", server.guard(http.HandlerFunc(server.closeWeek)))
	mux.Handle("POST /api/v2/households/{householdID}/week/current/recipes", server.guard(http.HandlerFunc(server.addRecipe)))
	mux.Handle("DELETE /api/v2/households/{householdID}/week/current/recipes/{occurrenceID}", server.guard(http.HandlerFunc(server.removeRecipe)))
	mux.Handle("PUT /api/v2/households/{householdID}/week/current/recipes/{occurrenceID}", server.guard(http.HandlerFunc(server.swapRecipe)))
	mux.Handle("POST /api/v2/households/{householdID}/week/current/recipes/{occurrenceID}/random-swap", server.guard(http.HandlerFunc(server.randomSwapRecipe)))
	mux.Handle("GET /api/v2/households/{householdID}/week/current/groceries", server.guard(http.HandlerFunc(server.getGroceries)))
	mux.Handle("POST /api/v2/households/{householdID}/week/current/groceries", server.guard(http.HandlerFunc(server.addGroceryLine)))
	mux.Handle("PATCH /api/v2/households/{householdID}/week/current/groceries/{lineID}", server.guard(http.HandlerFunc(server.updateGroceryLine)))
	mux.Handle("DELETE /api/v2/households/{householdID}/week/current/groceries/{lineID}", server.guard(http.HandlerFunc(server.removeGroceryLine)))
	mux.Handle("GET /api/v2/households/{householdID}/week/current/groceries/{lineID}/contributions", server.guard(http.HandlerFunc(server.getGroceryContributions)))
	mux.Handle("GET /api/v2/households/{householdID}/grocery-items", server.guard(http.HandlerFunc(server.searchGroceryItems)))
	mux.Handle("GET /api/v2/households/{householdID}/store-sections", server.guard(http.HandlerFunc(server.storeSections)))
	mux.Handle("PUT /api/v2/households/{householdID}/store-sections", server.guard(http.HandlerFunc(server.setStoreSections)))
	mux.Handle("GET /api/v2/households/{householdID}/settings", server.guard(http.HandlerFunc(server.getSettings)))
	mux.Handle("PATCH /api/v2/households/{householdID}/settings", server.guard(http.HandlerFunc(server.updateSettings)))
	server.handler = requestHeaders(mux)
	return server
}

// Handler returns the complete API handler.
func (server *Server) Handler() http.Handler {
	return server.handler
}

func (server *Server) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		householdID := request.PathValue("householdID")
		if !server.authorize(request.Context(), householdID) {
			// 404, not 403, so the response does not disclose whether another household exists.
			writeError(response, http.StatusNotFound, "not_found", "The requested resource was not found.")
			return
		}
		next.ServeHTTP(response, request)
	})
}

type recipeSummary struct {
	ID         int64         `json:"id"`
	Key        string        `json:"key"`
	Name       string        `json:"name"`
	ImageURL   *string       `json:"imageUrl"`
	Yield      *string       `json:"yield"`
	HandsOn    durationRange `json:"handsOn"`
	Unattended durationRange `json:"unattended"`
}

type durationRange struct {
	MinimumMinutes *int64 `json:"minimumMinutes"`
	MaximumMinutes *int64 `json:"maximumMinutes"`
}

type weekResponse struct {
	ID       int64              `json:"id"`
	StartsOn string             `json:"startsOn"`
	EndsOn   *string            `json:"endsOn"`
	Recipes  []recipeOccurrence `json:"recipes"`
}

type recipeDetail struct {
	recipeSummary
	Sources      []recipeSource             `json:"sources"`
	Ingredients  []recipeIngredientSection  `json:"ingredientSections"`
	Instructions []recipeInstructionSection `json:"instructionSections"`
}

type recipeSource struct {
	Relationship string  `json:"relationship"`
	Attribution  string  `json:"attribution"`
	URL          *string `json:"url"`
	Primary      bool    `json:"primary"`
}

type recipeIngredientSection struct {
	Name        string             `json:"name"`
	Ingredients []recipeIngredient `json:"ingredients"`
}

type recipeIngredient struct {
	ID           int64   `json:"id"`
	SourceText   string  `json:"sourceText"`
	Preparation  *string `json:"preparation"`
	Optional     bool    `json:"optional"`
	DisplayNote  *string `json:"displayNote"`
	ShoppingItem *string `json:"shoppingItem"`
}

type recipeInstructionSection struct {
	Name  string       `json:"name"`
	Steps []recipeStep `json:"steps"`
}

type recipeStep struct {
	ID          int64  `json:"id"`
	Instruction string `json:"instruction"`
}

type recipeOccurrence struct {
	ID       int64         `json:"id"`
	Position int64         `json:"position"`
	Recipe   recipeSummary `json:"recipe"`
}

type generateRequest struct {
	RecipeCount int `json:"recipeCount"`
}

type recipeRequest struct {
	RecipeID int64 `json:"recipeId"`
}

type manualLineRequest struct {
	Name           string `json:"name"`
	GroceryItemID  int64  `json:"groceryItemId"`
	StoreSectionID int64  `json:"storeSectionId"`
}

type groceryLineUpdate struct {
	StoreSectionID *int64  `json:"storeSectionId"`
	Completed      *bool   `json:"completed"`
	OverrideText   *string `json:"overrideText"`
	// Removed accepts only false: it restores a removed line (Undo). DELETE removes.
	Removed *bool `json:"removed"`
}

type groceryResponse struct {
	StartsOn string        `json:"startsOn"`
	Lines    []groceryLine `json:"lines"`
}

type weekHistorySummary struct {
	ID             int64  `json:"id"`
	StartsOn       string `json:"startsOn"`
	EndsOn         string `json:"endsOn"`
	RecipeCount    int64  `json:"recipeCount"`
	GroceryCount   int64  `json:"groceryCount"`
	CompletedCount int64  `json:"completedCount"`
}

type historicalWeekResponse struct {
	Week      weekResponse    `json:"week"`
	Groceries groceryResponse `json:"groceries"`
}

type groceryLine struct {
	ID                int64    `json:"id"`
	SectionID         int64    `json:"sectionId"`
	Section           string   `json:"section"`
	Name              string   `json:"name"`
	Quantity          *string  `json:"quantity"`
	GeneratedQuantity *string  `json:"generatedQuantity"`
	Origin            string   `json:"origin"`
	Optional          bool     `json:"optional"`
	Removed           bool     `json:"removed"`
	Completed         bool     `json:"completed"`
	HasContributions  bool     `json:"hasContributions"`
	RecipeNames       []string `json:"recipeNames"`
}

type groceryContribution struct {
	RecipeID    int64   `json:"recipeId"`
	RecipeName  string  `json:"recipeName"`
	SourceText  string  `json:"sourceText"`
	Quantity    *string `json:"quantity"`
	Preparation *string `json:"preparation"`
	Optional    bool    `json:"optional"`
}

type errorEnvelope struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (server *Server) listRecipes(response http.ResponseWriter, request *http.Request) {
	recipes, err := server.queries.ListVerifiedRecipes(request.Context(), request.PathValue("householdID"))
	if err != nil {
		writeInternalError(response, request, err)
		return
	}
	result := make([]recipeSummary, 0, len(recipes))
	for _, recipe := range recipes {
		result = append(result, summaryFromRecipe(recipe))
	}
	writeJSON(response, http.StatusOK, map[string]any{"recipes": result})
}

func (server *Server) getRecipe(response http.ResponseWriter, request *http.Request) {
	recipeID, ok := pathID(response, request, "recipeID")
	if !ok {
		return
	}
	recipe, err := server.queries.GetVerifiedRecipe(request.Context(), store.GetVerifiedRecipeParams{HouseholdID: request.PathValue("householdID"), ID: recipeID})
	if errors.Is(err, sql.ErrNoRows) {
		writeError(response, http.StatusNotFound, "recipe_not_found", "The recipe was not found.")
		return
	}
	if err != nil {
		writeInternalError(response, request, err)
		return
	}
	sources, err := server.queries.ListRecipeSources(request.Context(), store.ListRecipeSourcesParams{HouseholdID: request.PathValue("householdID"), RecipeID: recipeID})
	if err != nil {
		writeInternalError(response, request, err)
		return
	}
	ingredientRows, err := server.queries.ListRecipeIngredients(request.Context(), store.ListRecipeIngredientsParams{HouseholdID: request.PathValue("householdID"), RecipeID: recipeID})
	if err != nil {
		writeInternalError(response, request, err)
		return
	}
	stepRows, err := server.queries.ListRecipeSteps(request.Context(), store.ListRecipeStepsParams{HouseholdID: request.PathValue("householdID"), RecipeID: recipeID})
	if err != nil {
		writeInternalError(response, request, err)
		return
	}
	detail := recipeDetail{
		recipeSummary: summaryFromRecipe(recipe), Sources: make([]recipeSource, 0, len(sources)),
		Ingredients: make([]recipeIngredientSection, 0), Instructions: make([]recipeInstructionSection, 0),
	}
	for _, source := range sources {
		detail.Sources = append(detail.Sources, recipeSource{
			Relationship: source.Relationship, Attribution: source.Attribution,
			URL: nullableString(source.Url), Primary: source.IsPrimary == 1,
		})
	}
	for _, row := range ingredientRows {
		if len(detail.Ingredients) == 0 || detail.Ingredients[len(detail.Ingredients)-1].Name != row.SectionName {
			detail.Ingredients = append(detail.Ingredients, recipeIngredientSection{Name: row.SectionName, Ingredients: make([]recipeIngredient, 0)})
		}
		var shoppingItem *string
		if row.GroceryItemName.Valid {
			shoppingItem = &row.GroceryItemName.String
		}
		section := &detail.Ingredients[len(detail.Ingredients)-1]
		section.Ingredients = append(section.Ingredients, recipeIngredient{
			ID: row.ID, SourceText: row.SourceText, Preparation: nullableString(row.Preparation),
			Optional: row.IsOptional == 1, DisplayNote: nullableString(row.DisplayNote), ShoppingItem: shoppingItem,
		})
	}
	for _, row := range stepRows {
		if len(detail.Instructions) == 0 || detail.Instructions[len(detail.Instructions)-1].Name != row.SectionName {
			detail.Instructions = append(detail.Instructions, recipeInstructionSection{Name: row.SectionName, Steps: make([]recipeStep, 0)})
		}
		section := &detail.Instructions[len(detail.Instructions)-1]
		section.Steps = append(section.Steps, recipeStep{ID: row.ID, Instruction: row.Instruction})
	}
	writeJSON(response, http.StatusOK, detail)
}

func (server *Server) currentWeek(response http.ResponseWriter, request *http.Request) {
	view, err := server.weeks.ForHousehold(request.PathValue("householdID")).Current(request.Context(), server.now())
	server.writeWeekResult(response, request, view, err)
}

func (server *Server) weekHistory(response http.ResponseWriter, request *http.Request) {
	rows, err := server.weeks.ForHousehold(request.PathValue("householdID")).PastWeeks(request.Context(), server.now())
	if err != nil {
		writeInternalError(response, request, err)
		return
	}
	result := make([]weekHistorySummary, 0, len(rows))
	for _, row := range rows {
		result = append(result, weekHistorySummary{
			ID: row.ID, StartsOn: row.StartsOn, EndsOn: row.EndsOn.String, RecipeCount: row.RecipeCount,
			GroceryCount: row.GroceryCount, CompletedCount: row.CompletedCount,
		})
	}
	writeJSON(response, http.StatusOK, map[string]any{"weeks": result})
}

func (server *Server) historicalWeek(response http.ResponseWriter, request *http.Request) {
	weekID, ok := pathID(response, request, "weekID")
	if !ok {
		return
	}
	view, err := server.weeks.ForHousehold(request.PathValue("householdID")).PastWeek(request.Context(), server.now(), weekID)
	if errors.Is(err, week.ErrPastWeek) {
		writeError(response, http.StatusNotFound, "past_week_not_found", "The past week was not found.")
		return
	}
	if err != nil {
		writeInternalError(response, request, err)
		return
	}
	writeJSON(response, http.StatusOK, historicalWeekResponse{
		Week:      responseFromView(view.View),
		Groceries: groceryResponseFromChecklist(view.Checklist, view.Week.StartsOn),
	})
}

func (server *Server) generateWeek(response http.ResponseWriter, request *http.Request) {
	var input generateRequest
	if !decodeJSON(response, request, &input) {
		return
	}
	view, err := server.weeks.ForHousehold(request.PathValue("householdID")).Generate(request.Context(), server.now(), input.RecipeCount)
	server.writeWeekResult(response, request, view, err)
}

// closeWeek ends the current week now. It never generates the next one.
func (server *Server) closeWeek(response http.ResponseWriter, request *http.Request) {
	err := server.weeks.ForHousehold(request.PathValue("householdID")).Close(request.Context(), server.now())
	switch {
	case errors.Is(err, week.ErrNoCurrentWeek):
		writeError(response, http.StatusNotFound, "no_current_week", "The current week has not been generated.")
	case err != nil:
		writeInternalError(response, request, err)
	default:
		response.WriteHeader(http.StatusNoContent)
	}
}

func (server *Server) addRecipe(response http.ResponseWriter, request *http.Request) {
	var input recipeRequest
	if !decodeJSON(response, request, &input) {
		return
	}
	if input.RecipeID <= 0 {
		writeError(response, http.StatusBadRequest, "invalid_request", "recipeId must be positive")
		return
	}
	view, err := server.weeks.ForHousehold(request.PathValue("householdID")).Add(request.Context(), server.now(), input.RecipeID)
	server.writeWeekResult(response, request, view, err)
}

func (server *Server) removeRecipe(response http.ResponseWriter, request *http.Request) {
	occurrenceID, ok := pathID(response, request, "occurrenceID")
	if !ok {
		return
	}
	view, err := server.weeks.ForHousehold(request.PathValue("householdID")).Remove(request.Context(), server.now(), occurrenceID)
	server.writeWeekResult(response, request, view, err)
}

func (server *Server) swapRecipe(response http.ResponseWriter, request *http.Request) {
	occurrenceID, ok := pathID(response, request, "occurrenceID")
	if !ok {
		return
	}
	var input recipeRequest
	if !decodeJSON(response, request, &input) {
		return
	}
	if input.RecipeID <= 0 {
		writeError(response, http.StatusBadRequest, "invalid_request", "recipeId must be positive")
		return
	}
	view, err := server.weeks.ForHousehold(request.PathValue("householdID")).Swap(request.Context(), server.now(), occurrenceID, input.RecipeID)
	server.writeWeekResult(response, request, view, err)
}

func (server *Server) randomSwapRecipe(response http.ResponseWriter, request *http.Request) {
	occurrenceID, ok := pathID(response, request, "occurrenceID")
	if !ok {
		return
	}
	view, err := server.weeks.ForHousehold(request.PathValue("householdID")).RandomSwap(request.Context(), server.now(), occurrenceID)
	server.writeWeekResult(response, request, view, err)
}

func (server *Server) getGroceries(response http.ResponseWriter, request *http.Request) {
	server.writeGroceries(response, request)
}

func (server *Server) addGroceryLine(response http.ResponseWriter, request *http.Request) {
	var input manualLineRequest
	if !decodeJSON(response, request, &input) {
		return
	}
	if _, err := server.weeks.ForHousehold(request.PathValue("householdID")).AddManualLine(request.Context(), server.now(), input.Name, week.ManualLineOptions{GroceryItemID: input.GroceryItemID, StoreSectionID: input.StoreSectionID}); err != nil {
		server.writeGroceryError(response, request, err)
		return
	}
	server.writeGroceries(response, request)
}

func (server *Server) updateGroceryLine(response http.ResponseWriter, request *http.Request) {
	lineID, ok := pathID(response, request, "lineID")
	if !ok {
		return
	}
	var input groceryLineUpdate
	if !decodeJSON(response, request, &input) {
		return
	}
	updates := 0
	if input.Completed != nil {
		updates++
	}
	if input.OverrideText != nil {
		updates++
	}
	if input.StoreSectionID != nil {
		updates++
	}
	if input.Removed != nil {
		updates++
	}
	if updates != 1 || (input.Removed != nil && *input.Removed) {
		writeError(response, http.StatusBadRequest, "invalid_request", "Provide exactly one supported grocery line update.")
		return
	}
	var err error
	if input.Completed != nil {
		err = server.weeks.ForHousehold(request.PathValue("householdID")).SetLineCompleted(request.Context(), server.now(), lineID, *input.Completed)
	} else if input.Removed != nil {
		err = server.weeks.ForHousehold(request.PathValue("householdID")).SetLineRemoved(request.Context(), server.now(), lineID, false)
	} else if input.StoreSectionID != nil {
		err = server.weeks.ForHousehold(request.PathValue("householdID")).SetLineSection(request.Context(), server.now(), lineID, *input.StoreSectionID)
	} else {
		err = server.weeks.ForHousehold(request.PathValue("householdID")).SetLineOverride(request.Context(), server.now(), lineID, *input.OverrideText)
	}
	if err != nil {
		server.writeGroceryError(response, request, err)
		return
	}
	server.writeGroceries(response, request)
}

func (server *Server) removeGroceryLine(response http.ResponseWriter, request *http.Request) {
	lineID, ok := pathID(response, request, "lineID")
	if !ok {
		return
	}
	if err := server.weeks.ForHousehold(request.PathValue("householdID")).SetLineRemoved(request.Context(), server.now(), lineID, true); err != nil {
		server.writeGroceryError(response, request, err)
		return
	}
	server.writeGroceries(response, request)
}

func (server *Server) getGroceryContributions(response http.ResponseWriter, request *http.Request) {
	lineID, ok := pathID(response, request, "lineID")
	if !ok {
		return
	}
	rows, err := server.weeks.ForHousehold(request.PathValue("householdID")).Contributions(request.Context(), server.now(), lineID)
	if err != nil {
		server.writeGroceryError(response, request, err)
		return
	}
	result := make([]groceryContribution, 0, len(rows))
	for _, row := range rows {
		result = append(result, groceryContribution{
			RecipeID: row.RecipeID, RecipeName: row.RecipeName, SourceText: row.SourceText,
			Quantity: quantityText(row.QuantityKind, row.AmountMinNumerator, row.AmountMinDenominator,
				row.AmountMaxNumerator, row.AmountMaxDenominator, row.UnitSymbol,
				row.PackageType, row.PackageSizeNumerator, row.PackageSizeDenominator, row.PackageSizeUnitSymbol),
			Preparation: nullableString(row.Preparation), Optional: row.IsOptional == 1,
		})
	}
	writeJSON(response, http.StatusOK, map[string]any{"contributions": result})
}

func (server *Server) writeGroceries(response http.ResponseWriter, request *http.Request) {
	checklist, err := server.weeks.ForHousehold(request.PathValue("householdID")).Checklist(request.Context(), server.now())
	if err != nil {
		server.writeGroceryError(response, request, err)
		return
	}
	writeJSON(response, http.StatusOK, groceryResponseFromChecklist(checklist, checklist.StartsOn))
}

func groceryResponseFromChecklist(checklist week.Checklist, startsOn string) groceryResponse {
	result := groceryResponse{StartsOn: startsOn, Lines: make([]groceryLine, 0, len(checklist.Lines))}
	for _, row := range checklist.Lines {
		generated := lineQuantityText(row, checklist.Quantities[row.ID], checklist.Unspecified[row.ID])
		quantity := generated
		if row.OverrideText.Valid {
			quantity = &row.OverrideText.String
		}
		recipeNames := checklist.RecipeNames[row.ID]
		if recipeNames == nil {
			recipeNames = make([]string, 0)
		}
		result.Lines = append(result.Lines, groceryLine{
			ID: row.ID, SectionID: row.StoreSectionID, Section: row.StoreSectionName, Name: row.DisplayName,
			Quantity: quantity, GeneratedQuantity: generated, Origin: row.Origin,
			Optional: row.IsOptional == 1, Removed: row.IsRemoved == 1, Completed: row.IsCompleted == 1,
			HasContributions: row.Origin == "generated", RecipeNames: recipeNames,
		})
	}
	return result
}

func (server *Server) writeGroceryError(response http.ResponseWriter, request *http.Request, err error) {
	var validation week.ValidationError
	switch {
	case errors.Is(err, week.ErrNoCurrentWeek):
		writeError(response, http.StatusNotFound, "no_current_week", "The current week has not been generated.")
	case errors.As(err, &validation):
		writeError(response, http.StatusUnprocessableEntity, "grocery_operation_failed", validation.Message)
	default:
		writeInternalError(response, request, err)
	}
}

func (server *Server) writeWeekResult(response http.ResponseWriter, request *http.Request, view week.View, err error) {
	var validation week.ValidationError
	switch {
	case errors.Is(err, week.ErrNoCurrentWeek):
		writeError(response, http.StatusNotFound, "no_current_week", "The current week has not been generated.")
	case errors.Is(err, week.ErrOccurrence):
		writeError(response, http.StatusNotFound, "occurrence_not_found", "The recipe occurrence was not found in the current week.")
	case errors.As(err, &validation):
		writeError(response, http.StatusUnprocessableEntity, "week_operation_failed", validation.Message)
	case err != nil:
		writeInternalError(response, request, err)
	default:
		writeJSON(response, http.StatusOK, responseFromView(view))
	}
}

func responseFromView(view week.View) weekResponse {
	result := weekResponse{
		ID: view.Week.ID, StartsOn: view.Week.StartsOn, EndsOn: nullableString(view.EndsOn),
		Recipes: make([]recipeOccurrence, 0, len(view.Recipes)),
	}
	for _, occurrence := range view.Recipes {
		result.Recipes = append(result.Recipes, recipeOccurrence{
			ID:       occurrence.ID,
			Position: occurrence.Position,
			Recipe: recipeSummary{
				ID: occurrence.RecipeID, Key: occurrence.RecipeKey, Name: occurrence.RecipeName,
				ImageURL: nullableString(occurrence.ImageUrl), Yield: nullableString(occurrence.YieldText),
				HandsOn: durationRange{
					MinimumMinutes: nullableInt(occurrence.HandsOnMinMinutes),
					MaximumMinutes: nullableInt(occurrence.HandsOnMaxMinutes),
				},
				Unattended: durationRange{
					MinimumMinutes: nullableInt(occurrence.UnattendedMinMinutes),
					MaximumMinutes: nullableInt(occurrence.UnattendedMaxMinutes),
				},
			},
		})
	}
	return result
}

func summaryFromRecipe(recipe store.Recipe) recipeSummary {
	return recipeSummary{
		ID: recipe.ID, Key: recipe.Key, Name: recipe.Name,
		ImageURL: nullableString(recipe.ImageUrl), Yield: nullableString(recipe.YieldText),
		HandsOn: durationRange{
			MinimumMinutes: nullableInt(recipe.HandsOnMinMinutes),
			MaximumMinutes: nullableInt(recipe.HandsOnMaxMinutes),
		},
		Unattended: durationRange{
			MinimumMinutes: nullableInt(recipe.UnattendedMinMinutes),
			MaximumMinutes: nullableInt(recipe.UnattendedMaxMinutes),
		},
	}
}

func nullableString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func nullableInt(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	return &value.Int64
}

func quantityText(
	kind string,
	minNumerator, minDenominator, maxNumerator, maxDenominator sql.NullInt64,
	unitSymbol, packageType sql.NullString,
	packageSizeNumerator, packageSizeDenominator sql.NullInt64,
	packageSizeUnitSymbol sql.NullString,
) *string {
	if kind == "presence" || kind == "unspecified" || !minNumerator.Valid {
		return nil
	}
	minimum := formatRational(minNumerator.Int64, minDenominator.Int64)
	amount := minimum
	if kind == "range" && maxNumerator.Valid {
		amount += "–" + formatRational(maxNumerator.Int64, maxDenominator.Int64)
	}
	if packageType.Valid {
		noun := packageType.String
		if minNumerator.Int64 != minDenominator.Int64 && !strings.HasSuffix(noun, "s") {
			noun += "s"
		}
		if packageSizeNumerator.Valid {
			size := formatRational(packageSizeNumerator.Int64, packageSizeDenominator.Int64)
			if packageSizeUnitSymbol.Valid {
				size += " " + packageSizeUnitSymbol.String
			}
			amount += " × " + size
		}
		value := amount + " " + noun
		return &value
	}
	if unitSymbol.Valid {
		amount += " " + unitSymbol.String
	}
	return &amount
}

// lineQuantityText joins a row's amounts that cannot be added exactly ("8 cloves + 1 tbsp") and
// adds "more" when a recipe also gave no amount, so the row never understates what is needed.
func lineQuantityText(row store.ListShoppingLinesRow, extra []store.ListShoppingLineQuantitiesRow, unspecified bool) *string {
	first := quantityText(row.QuantityKind, row.AmountMinNumerator, row.AmountMinDenominator,
		row.AmountMaxNumerator, row.AmountMaxDenominator, row.UnitSymbol,
		row.PackageType, row.PackageSizeNumerator, row.PackageSizeDenominator, row.PackageSizeUnitSymbol)
	if first == nil {
		return nil
	}
	parts := []string{*first}
	for _, quantity := range extra {
		text := quantityText(quantity.QuantityKind,
			sql.NullInt64{Int64: quantity.AmountMinNumerator, Valid: true}, sql.NullInt64{Int64: quantity.AmountMinDenominator, Valid: true},
			quantity.AmountMaxNumerator, quantity.AmountMaxDenominator, quantity.UnitSymbol,
			quantity.PackageType, quantity.PackageSizeNumerator, quantity.PackageSizeDenominator, quantity.PackageSizeUnitSymbol)
		if text != nil {
			parts = append(parts, *text)
		}
	}
	if unspecified {
		parts = append(parts, "more")
	}
	joined := strings.Join(parts, " + ")
	return &joined
}

func formatRational(numerator, denominator int64) string {
	if denominator <= 0 {
		return ""
	}
	whole := numerator / denominator
	remainder := numerator % denominator
	if remainder == 0 {
		return strconv.FormatInt(whole, 10)
	}
	fraction := fmt.Sprintf("%d/%d", remainder, denominator)
	if whole == 0 {
		return fraction
	}
	return fmt.Sprintf("%d %s", whole, fraction)
}

func pathID(response http.ResponseWriter, request *http.Request, name string) (int64, bool) {
	value, err := strconv.ParseInt(request.PathValue(name), 10, 64)
	if err != nil || value <= 0 {
		writeError(response, http.StatusBadRequest, "invalid_request", fmt.Sprintf("%s must be a positive integer", name))
		return 0, false
	}
	return value, true
}

func decodeJSON(response http.ResponseWriter, request *http.Request, destination any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(response, request.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		writeError(response, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON with only supported fields.")
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(response, http.StatusBadRequest, "invalid_json", "Request body must contain one JSON object.")
		return false
	}
	return true
}

// writeInternalError logs the cause server-side and returns only a generic body. It logs the
// method and path, never the body, cookies, or query string.
func writeInternalError(response http.ResponseWriter, request *http.Request, err error) {
	slog.ErrorContext(request.Context(), "request failed",
		"method", request.Method, "path", request.URL.Path, "error", err)
	writeError(response, http.StatusInternalServerError, "internal_error", "The request could not be completed.")
}

func writeError(response http.ResponseWriter, status int, code, message string) {
	writeJSON(response, status, errorEnvelope{Error: apiError{Code: code, Message: message}})
}

func writeJSON(response http.ResponseWriter, status int, body any) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(body)
}

func requestHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Cache-Control", "no-store")
		response.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(response, request)
	})
}

type storeSectionResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Key  string `json:"key"`
}

func (server *Server) storeSections(response http.ResponseWriter, request *http.Request) {
	sections, err := server.queries.ListStoreSections(request.Context(), request.PathValue("householdID"))
	if err != nil {
		server.writeGroceryError(response, request, err)
		return
	}
	result := make([]storeSectionResponse, 0, len(sections))
	for _, section := range sections {
		result = append(result, storeSectionResponse{section.ID, section.Name, section.Key})
	}
	writeJSON(response, http.StatusOK, map[string]any{"sections": result})
}
func (server *Server) setStoreSections(response http.ResponseWriter, request *http.Request) {
	var input struct {
		SectionIDs []int64 `json:"sectionIds"`
		Reset      bool    `json:"reset"`
	}
	if !decodeJSON(response, request, &input) {
		return
	}
	if err := server.weeks.ForHousehold(request.PathValue("householdID")).SetSectionOrder(request.Context(), input.SectionIDs, input.Reset); err != nil {
		server.writeGroceryError(response, request, err)
		return
	}
	server.storeSections(response, request)
}
func (server *Server) searchGroceryItems(response http.ResponseWriter, request *http.Request) {
	matches, err := server.weeks.ForHousehold(request.PathValue("householdID")).SearchItems(request.Context(), request.URL.Query().Get("q"))
	if err != nil {
		server.writeGroceryError(response, request, err)
		return
	}
	type item struct {
		ID        *int64 `json:"id,omitempty"`
		Name      string `json:"name"`
		SectionID int64  `json:"sectionId"`
		Section   string `json:"section"`
	}
	result := make([]item, 0, len(matches))
	for _, match := range matches {
		entry := item{Name: match.Name, SectionID: match.SectionID, Section: match.Section}
		if match.GroceryItemID != 0 {
			id := match.GroceryItemID
			entry.ID = &id
		}
		result = append(result, entry)
	}
	writeJSON(response, http.StatusOK, map[string]any{"items": result})
}
