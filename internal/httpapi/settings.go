package httpapi

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/mbcoward3/grocery-router/internal/settings"
)

type settingsResponse struct {
	Timezone           string `json:"timezone"`
	DefaultRecipeCount int    `json:"defaultRecipeCount"`
}

type settingsUpdate struct {
	Timezone           *string `json:"timezone"`
	DefaultRecipeCount *int    `json:"defaultRecipeCount"`
}

func (server *Server) getSettings(response http.ResponseWriter, request *http.Request) {
	current, err := settings.Load(request.Context(), server.queries, request.PathValue("householdID"))
	server.writeSettingsResult(response, request, current, err)
}

func (server *Server) updateSettings(response http.ResponseWriter, request *http.Request) {
	var input settingsUpdate
	if !decodeJSON(response, request, &input) {
		return
	}
	// An empty change is a malformed request (400, minProperties in the contract), rejected here
	// before settings.Change.Validate, whose own empty-change check serves other callers.
	if input.Timezone == nil && input.DefaultRecipeCount == nil {
		writeError(response, http.StatusBadRequest, "invalid_request", "Provide timezone, defaultRecipeCount, or both.")
		return
	}
	updated, err := settings.Update(request.Context(), server.queries, request.PathValue("householdID"),
		settings.Change{Timezone: input.Timezone, DefaultRecipeCount: input.DefaultRecipeCount})
	server.writeSettingsResult(response, request, updated, err)
}

func (server *Server) writeSettingsResult(response http.ResponseWriter, request *http.Request, current settings.Settings, err error) {
	var validation settings.ValidationError
	switch {
	case errors.Is(err, sql.ErrNoRows):
		writeError(response, http.StatusNotFound, "not_found", "The requested resource was not found.")
	case errors.As(err, &validation):
		writeError(response, http.StatusUnprocessableEntity, "invalid_settings", validation.Message)
	case err != nil:
		writeInternalError(response, request, err)
	default:
		writeJSON(response, http.StatusOK, settingsResponse{Timezone: current.Timezone, DefaultRecipeCount: current.DefaultRecipeCount})
	}
}
