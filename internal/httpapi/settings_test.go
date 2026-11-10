package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mbcoward3/grocery-router/internal/testdatabase"
)

func assertSettings(t *testing.T, response *httptest.ResponseRecorder, timezone string, count float64) {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("settings status = %d, body %s", response.Code, response.Body.String())
	}
	body := decodeObject(t, response)
	if body["timezone"] != timezone || body["defaultRecipeCount"] != count {
		t.Fatalf("settings = %#v, want %s and %v", body, timezone, count)
	}
}

func TestHouseholdSettingsAPI(t *testing.T) {
	// Settings need only the migrated household, not the recipe corpus.
	handler := handlerForDB(t, testdatabase.Open(t))
	const path = "/api/v2/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5/settings"

	assertSettings(t, request(t, handler, http.MethodGet, path, ""), "UTC", 5)

	for _, tc := range []struct {
		body        string
		expectation contractExpectation
	}{
		{`{"timezone":"Not/AZone"}`, conformingRequest},
		{`{"timezone":"Local"}`, conformingRequest},
		{`{"defaultRecipeCount":0}`, nonconformingRequest},
		{`{"defaultRecipeCount":22}`, nonconformingRequest},
		// One invalid field rejects the whole change.
		{`{"timezone":"America/Chicago","defaultRecipeCount":0}`, nonconformingRequest},
	} {
		response := requestExpecting(t, handler, tc.expectation, http.MethodPatch, path, tc.body)
		if response.Code != http.StatusUnprocessableEntity {
			t.Fatalf("PATCH %s status = %d, body %s", tc.body, response.Code, response.Body.String())
		}
		assertErrorCode(t, response, "invalid_settings")
	}
	for _, body := range []string{`{}`, `{"surprise":true}`, `{"defaultRecipeCount":"7"}`} {
		response := requestExpecting(t, handler, nonconformingRequest, http.MethodPatch, path, body)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("PATCH %s status = %d, body %s", body, response.Code, response.Body.String())
		}
	}
	assertSettings(t, request(t, handler, http.MethodGet, path, ""), "UTC", 5)

	assertSettings(t, request(t, handler, http.MethodPatch, path, `{"defaultRecipeCount":8}`), "UTC", 8)
	assertSettings(t, request(t, handler, http.MethodPatch, path, `{"timezone":"America/Chicago"}`), "America/Chicago", 8)
	assertSettings(t, request(t, handler, http.MethodGet, path, ""), "America/Chicago", 8)
	assertSettings(t, request(t, handler, http.MethodPatch, path, `{"timezone":"UTC","defaultRecipeCount":21}`), "UTC", 21)

	const other = "/api/v2/households/711e58fa-2c03-44cc-83aa-432eab3158fb/settings"
	for _, call := range []struct{ method, body string }{{http.MethodGet, ""}, {http.MethodPatch, `{"defaultRecipeCount":3}`}} {
		response := request(t, handler, call.method, other, call.body)
		if response.Code != http.StatusNotFound {
			t.Fatalf("%s another household's settings status = %d", call.method, response.Code)
		}
		assertErrorCode(t, response, "not_found")
	}
}

func TestHouseholdRolloverSettingsAPI(t *testing.T) {
	handler := handlerForDB(t, testdatabase.Open(t))
	const path = "/api/v2/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5/settings"
	assertRollover := func(response *httptest.ResponseRecorder, day float64, at, mode string) {
		t.Helper()
		if response.Code != http.StatusOK {
			t.Fatalf("settings status = %d, body %s", response.Code, response.Body.String())
		}
		body := decodeObject(t, response)
		if body["weekStartDay"] != day || body["rolloverTime"] != at || body["rolloverMode"] != mode {
			t.Fatalf("settings = %#v, want %v %s %s", body, day, at, mode)
		}
	}
	// The defaults are the earlier fixed rollover: Sunday 12:00 AM, automatic.
	assertRollover(request(t, handler, http.MethodGet, path, ""), 0, "00:00", "automatic")

	for _, body := range []string{
		`{"weekStartDay":7}`, `{"weekStartDay":-1}`, `{"rolloverTime":"24:00"}`, `{"rolloverTime":"6pm"}`,
		`{"rolloverTime":"6:00"}`, `{"rolloverMode":"weekly"}`, `{"weekStartDay":5,"rolloverMode":"weekly"}`,
	} {
		response := requestExpecting(t, handler, nonconformingRequest, http.MethodPatch, path, body)
		if response.Code != http.StatusUnprocessableEntity {
			t.Fatalf("PATCH %s status = %d, body %s", body, response.Code, response.Body.String())
		}
		assertErrorCode(t, response, "invalid_settings")
	}
	assertRollover(request(t, handler, http.MethodGet, path, ""), 0, "00:00", "automatic")

	assertRollover(request(t, handler, http.MethodPatch, path, `{"weekStartDay":5,"rolloverTime":"18:00"}`), 5, "18:00", "automatic")
	assertRollover(request(t, handler, http.MethodPatch, path, `{"rolloverMode":"manual"}`), 5, "18:00", "manual")
	assertRollover(request(t, handler, http.MethodPatch, path, `{"rolloverTime":"23:59"}`), 5, "23:59", "manual")
	assertSettings(t, request(t, handler, http.MethodGet, path, ""), "UTC", 5)
}

func TestHouseholdSettingsInternalError(t *testing.T) {
	db := testdatabase.Open(t)
	handler := handlerForDB(t, db)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	const path = "/api/v2/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5/settings"
	for _, call := range []struct{ method, body string }{{http.MethodGet, ""}, {http.MethodPatch, `{"defaultRecipeCount":3}`}} {
		response := request(t, handler, call.method, path, call.body)
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("%s settings on a closed database status = %d, body %s", call.method, response.Code, response.Body.String())
		}
		assertErrorCode(t, response, "internal_error")
	}
}
