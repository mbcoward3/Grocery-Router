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
	WeekStartDay       int    `json:"weekStartDay"`
	RolloverTime       string `json:"rolloverTime"`
	RolloverMode       string `json:"rolloverMode"`
}

type settingsUpdate struct {
	Timezone           *string `json:"timezone"`
	DefaultRecipeCount *int    `json:"defaultRecipeCount"`
	WeekStartDay       *int    `json:"weekStartDay"`
	RolloverTime       *string `json:"rolloverTime"`
	RolloverMode       *string `json:"rolloverMode"`
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
	if input.Timezone == nil && input.DefaultRecipeCount == nil && input.WeekStartDay == nil &&
		input.RolloverTime == nil && input.RolloverMode == nil {
		writeError(response, http.StatusBadRequest, "invalid_request", "Provide at least one setting.")
		return
	}
	change := settings.Change{
		Timezone: input.Timezone, DefaultRecipeCount: input.DefaultRecipeCount,
		WeekStartDay: input.WeekStartDay, RolloverMode: input.RolloverMode,
	}
	if input.RolloverTime != nil {
		minutes, err := settings.ParseClock(*input.RolloverTime)
		if err != nil {
			server.writeSettingsResult(response, request, settings.Settings{}, err)
			return
		}
		change.RolloverMinutes = &minutes
	}
	// Saving goes through the week service, which moves the open week's close time when the
	// rollover changes.
	updated, err := server.weeks.ForHousehold(request.PathValue("householdID")).SaveSettings(request.Context(), server.now(), change)
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
		writeJSON(response, http.StatusOK, settingsResponse{
			Timezone: current.Timezone, DefaultRecipeCount: current.DefaultRecipeCount,
			WeekStartDay: current.WeekStartDay, RolloverTime: settings.FormatClock(current.RolloverMinutes),
			RolloverMode: current.RolloverMode,
		})
	}
}
