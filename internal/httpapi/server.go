// Package httpapi exposes Grocery Router application services over JSON HTTP.
package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mbcoward3/grocery-router/internal/store"
	"github.com/mbcoward3/grocery-router/internal/week"
)

// Clock returns the local time used to resolve the current Sunday.
type Clock func() time.Time

// Server is the HTTP boundary around corpus reads and week operations.
type Server struct {
	queries *store.Queries
	weeks   *week.Service
	now     Clock
	handler http.Handler
}

// New constructs an API handler. A nil clock uses time.Now.
func New(db *sql.DB, weeks *week.Service, clock Clock) *Server {
	if clock == nil {
		clock = time.Now
	}
	server := &Server{queries: store.New(db), weeks: weeks, now: clock}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/recipes", server.listRecipes)
	mux.HandleFunc("GET /api/recipes/{recipeID}", server.getRecipe)
	mux.HandleFunc("GET /api/week/current", server.currentWeek)
	mux.HandleFunc("POST /api/week/current/generate", server.generateWeek)
	mux.HandleFunc("POST /api/week/current/recipes", server.addRecipe)
	mux.HandleFunc("DELETE /api/week/current/recipes/{occurrenceID}", server.removeRecipe)
	mux.HandleFunc("PUT /api/week/current/recipes/{occurrenceID}", server.swapRecipe)
	mux.HandleFunc("POST /api/week/current/recipes/{occurrenceID}/random-swap", server.randomSwapRecipe)
	mux.HandleFunc("GET /api/week/current/groceries", server.getGroceries)
	mux.HandleFunc("POST /api/week/current/groceries", server.addGroceryLine)
	mux.HandleFunc("PATCH /api/week/current/groceries/{lineID}", server.updateGroceryLine)
	mux.HandleFunc("DELETE /api/week/current/groceries/{lineID}", server.removeGroceryLine)
	mux.HandleFunc("GET /api/week/current/groceries/{lineID}/contributions", server.getGroceryContributions)
	server.handler = requestHeaders(mux)
	return server
}

// Handler returns the complete API handler.
func (server *Server) Handler() http.Handler {
	return server.handler
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
	Name string `json:"name"`
}

type groceryLineUpdate struct {
	Completed    *bool   `json:"completed"`
	OverrideText *string `json:"overrideText"`
}

type groceryResponse struct {
	StartsOn string        `json:"startsOn"`
	Lines    []groceryLine `json:"lines"`
}

type groceryLine struct {
	ID                int64   `json:"id"`
	Section           string  `json:"section"`
	Name              string  `json:"name"`
	Quantity          *string `json:"quantity"`
	GeneratedQuantity *string `json:"generatedQuantity"`
	Origin            string  `json:"origin"`
	Optional          bool    `json:"optional"`
	Removed           bool    `json:"removed"`
	Completed         bool    `json:"completed"`
	HasContributions  bool    `json:"hasContributions"`
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
	recipes, err := server.queries.ListVerifiedRecipes(request.Context())
	if err != nil {
		writeInternalError(response, err)
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
	recipe, err := server.queries.GetVerifiedRecipe(request.Context(), recipeID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(response, http.StatusNotFound, "recipe_not_found", "The recipe was not found.")
		return
	}
	if err != nil {
		writeInternalError(response, err)
		return
	}
	sources, err := server.queries.ListRecipeSources(request.Context(), recipeID)
	if err != nil {
		writeInternalError(response, err)
		return
	}
	ingredientRows, err := server.queries.ListRecipeIngredients(request.Context(), recipeID)
	if err != nil {
		writeInternalError(response, err)
		return
	}
	stepRows, err := server.queries.ListRecipeSteps(request.Context(), recipeID)
	if err != nil {
		writeInternalError(response, err)
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
	view, err := server.weeks.Current(request.Context(), server.now())
	server.writeWeekResult(response, view, err)
}

func (server *Server) generateWeek(response http.ResponseWriter, request *http.Request) {
	var input generateRequest
	if !decodeJSON(response, request, &input) {
		return
	}
	view, err := server.weeks.Generate(request.Context(), server.now(), input.RecipeCount)
	server.writeWeekResult(response, view, err)
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
	view, err := server.weeks.Add(request.Context(), server.now(), input.RecipeID)
	server.writeWeekResult(response, view, err)
}

func (server *Server) removeRecipe(response http.ResponseWriter, request *http.Request) {
	occurrenceID, ok := pathID(response, request, "occurrenceID")
	if !ok {
		return
	}
	view, err := server.weeks.Remove(request.Context(), server.now(), occurrenceID)
	server.writeWeekResult(response, view, err)
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
	view, err := server.weeks.Swap(request.Context(), server.now(), occurrenceID, input.RecipeID)
	server.writeWeekResult(response, view, err)
}

func (server *Server) randomSwapRecipe(response http.ResponseWriter, request *http.Request) {
	occurrenceID, ok := pathID(response, request, "occurrenceID")
	if !ok {
		return
	}
	view, err := server.weeks.RandomSwap(request.Context(), server.now(), occurrenceID)
	server.writeWeekResult(response, view, err)
}

func (server *Server) getGroceries(response http.ResponseWriter, request *http.Request) {
	server.writeGroceries(response, request)
}

func (server *Server) addGroceryLine(response http.ResponseWriter, request *http.Request) {
	var input manualLineRequest
	if !decodeJSON(response, request, &input) {
		return
	}
	if _, err := server.weeks.AddManualLine(request.Context(), server.now(), input.Name); err != nil {
		server.writeGroceryError(response, err)
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
	if (input.Completed == nil) == (input.OverrideText == nil) {
		writeError(response, http.StatusBadRequest, "invalid_request", "Provide exactly one supported grocery line update.")
		return
	}
	var err error
	if input.Completed != nil {
		err = server.weeks.SetLineCompleted(request.Context(), server.now(), lineID, *input.Completed)
	} else {
		err = server.weeks.SetLineOverride(request.Context(), server.now(), lineID, *input.OverrideText)
	}
	if err != nil {
		server.writeGroceryError(response, err)
		return
	}
	server.writeGroceries(response, request)
}

func (server *Server) removeGroceryLine(response http.ResponseWriter, request *http.Request) {
	lineID, ok := pathID(response, request, "lineID")
	if !ok {
		return
	}
	if err := server.weeks.SetLineRemoved(request.Context(), server.now(), lineID, true); err != nil {
		server.writeGroceryError(response, err)
		return
	}
	server.writeGroceries(response, request)
}

func (server *Server) getGroceryContributions(response http.ResponseWriter, request *http.Request) {
	lineID, ok := pathID(response, request, "lineID")
	if !ok {
		return
	}
	rows, err := server.weeks.Contributions(request.Context(), server.now(), lineID)
	if err != nil {
		server.writeGroceryError(response, err)
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
	checklist, err := server.weeks.Checklist(request.Context(), server.now())
	if err != nil {
		server.writeGroceryError(response, err)
		return
	}
	result := groceryResponse{StartsOn: week.CurrentSunday(server.now()), Lines: make([]groceryLine, 0, len(checklist.Lines))}
	for _, row := range checklist.Lines {
		generated := quantityText(row.QuantityKind, row.AmountMinNumerator, row.AmountMinDenominator,
			row.AmountMaxNumerator, row.AmountMaxDenominator, row.UnitSymbol,
			row.PackageType, row.PackageSizeNumerator, row.PackageSizeDenominator, row.PackageSizeUnitSymbol)
		quantity := generated
		if row.OverrideText.Valid {
			quantity = &row.OverrideText.String
		}
		result.Lines = append(result.Lines, groceryLine{
			ID: row.ID, Section: row.StoreSectionName, Name: row.DisplayName,
			Quantity: quantity, GeneratedQuantity: generated, Origin: row.Origin,
			Optional: row.IsOptional == 1, Removed: row.IsRemoved == 1, Completed: row.IsCompleted == 1,
			HasContributions: row.Origin == "generated",
		})
	}
	writeJSON(response, http.StatusOK, result)
}

func (server *Server) writeGroceryError(response http.ResponseWriter, err error) {
	if errors.Is(err, week.ErrNoCurrentWeek) {
		writeError(response, http.StatusNotFound, "no_current_week", "The current week has not been generated.")
		return
	}
	writeError(response, http.StatusUnprocessableEntity, "grocery_operation_failed", err.Error())
}

func (server *Server) writeWeekResult(response http.ResponseWriter, view week.View, err error) {
	switch {
	case errors.Is(err, week.ErrNoCurrentWeek):
		writeError(response, http.StatusNotFound, "no_current_week", "The current week has not been generated.")
	case errors.Is(err, week.ErrOccurrence):
		writeError(response, http.StatusNotFound, "occurrence_not_found", "The recipe occurrence was not found in the current week.")
	case err != nil:
		// Domain validation messages are safe and useful to this trusted local client.
		writeError(response, http.StatusUnprocessableEntity, "week_operation_failed", err.Error())
	default:
		writeJSON(response, http.StatusOK, responseFromView(view))
	}
}

func responseFromView(view week.View) weekResponse {
	result := weekResponse{ID: view.Week.ID, StartsOn: view.Week.StartsOn, Recipes: make([]recipeOccurrence, 0, len(view.Recipes))}
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

func writeInternalError(response http.ResponseWriter, err error) {
	_ = err // Detailed errors belong in server logs once runtime logging is introduced.
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
