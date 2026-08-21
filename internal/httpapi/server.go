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
	mux.HandleFunc("GET /api/week/current", server.currentWeek)
	mux.HandleFunc("POST /api/week/current/generate", server.generateWeek)
	mux.HandleFunc("POST /api/week/current/recipes", server.addRecipe)
	mux.HandleFunc("DELETE /api/week/current/recipes/{occurrenceID}", server.removeRecipe)
	mux.HandleFunc("PUT /api/week/current/recipes/{occurrenceID}", server.swapRecipe)
	mux.HandleFunc("POST /api/week/current/recipes/{occurrenceID}/random-swap", server.randomSwapRecipe)
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
