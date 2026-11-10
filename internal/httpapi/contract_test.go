package httpapi_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

// apiPrefix is where internal/httpapi mounts its routes. The contract's paths omit it and
// declare it once as the relative server URL, which the router maps onto every path.
const apiPrefix = "/api/v2"

// loadContract parses and validates api/openapi.yaml once per test binary.
var loadContract = sync.OnceValues(func() (routers.Router, error) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(filepath.Join("..", "..", "api", "openapi.yaml"))
	if err != nil {
		return nil, fmt.Errorf("load contract: %w", err)
	}
	if err := doc.Validate(loader.Context); err != nil {
		return nil, fmt.Errorf("validate contract: %w", err)
	}
	if len(doc.Servers) != 1 || doc.Servers[0].URL != apiPrefix {
		return nil, fmt.Errorf("contract servers = %v, want exactly %q", doc.Servers, apiPrefix)
	}
	router, err := gorillamux.NewRouter(doc)
	if err != nil {
		return nil, fmt.Errorf("route contract: %w", err)
	}
	return router, nil
})

type contractExpectation int

const (
	// conformingRequest is the default: the request and the response must both match the contract.
	conformingRequest contractExpectation = iota
	// nonconformingRequest deliberately violates the contract's request schema; the server's
	// rejection must still be a documented response.
	nonconformingRequest
	// undocumentedOperation deliberately targets a method or path that the contract does not
	// describe, so there is no response definition to check.
	undocumentedOperation
)

type contractExpectationKey struct{}

func withContractExpectation(request *http.Request, expectation contractExpectation) *http.Request {
	return request.WithContext(context.WithValue(request.Context(), contractExpectationKey{}, expectation))
}

// contractHandler wraps handler so that every request it serves, and the response it returns,
// is validated against api/openapi.yaml. Violations fail t.
func contractHandler(t *testing.T, handler http.Handler) http.Handler {
	t.Helper()
	router, err := loadContract()
	if err != nil {
		t.Fatal(err)
	}
	return &contractValidator{t: t, router: router, next: handler}
}

type contractValidator struct {
	t      *testing.T
	router routers.Router
	next   http.Handler
}

func (validator *contractValidator) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	validator.t.Helper()
	expectation, _ := request.Context().Value(contractExpectationKey{}).(contractExpectation)
	recorder := httptest.NewRecorder()
	if err := serveAgainstContract(validator.router, validator.next, recorder, request, expectation); err != nil {
		validator.t.Errorf("%s %s: %v", request.Method, request.URL.Path, err)
	}
	for name, values := range recorder.Header() {
		response.Header()[name] = values
	}
	response.WriteHeader(recorder.Code)
	_, _ = response.Write(recorder.Body.Bytes())
}

// serveAgainstContract serves request with next into recorder and reports every way the
// exchange departs from the contract. It is separate from the wrapper so it can be tested
// without a database.
func serveAgainstContract(router routers.Router, next http.Handler, recorder *httptest.ResponseRecorder, request *http.Request, expectation contractExpectation) error {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		return fmt.Errorf("read request body: %w", err)
	}
	request.Body = io.NopCloser(bytes.NewReader(body))

	route, pathParams, findErr := router.FindRoute(request)
	if expectation == undocumentedOperation {
		next.ServeHTTP(recorder, request)
		if findErr == nil {
			return fmt.Errorf("operation %s is in the contract but the test marked it undocumented", route.Operation.OperationID)
		}
		if recorder.Code != http.StatusNotFound && recorder.Code != http.StatusMethodNotAllowed {
			return fmt.Errorf("undocumented operation answered %d, want 404 or 405", recorder.Code)
		}
		return nil
	}
	if findErr != nil {
		next.ServeHTTP(recorder, request)
		return fmt.Errorf("operation is not in the contract: %w", findErr)
	}

	options := &openapi3filter.Options{
		AuthenticationFunc:    openapi3filter.NoopAuthenticationFunc,
		IncludeResponseStatus: true,
		MultiError:            true,
	}
	validationRequest := request.Clone(request.Context())
	validationRequest.Body = io.NopCloser(bytes.NewReader(body))
	requestInput := &openapi3filter.RequestValidationInput{
		Request: validationRequest, PathParams: pathParams, Route: route, Options: options,
	}
	requestErr := openapi3filter.ValidateRequest(request.Context(), requestInput)

	next.ServeHTTP(recorder, request)

	var problems []error
	switch {
	case expectation == conformingRequest && requestErr != nil:
		problems = append(problems, fmt.Errorf("request does not match contract: %w", requestErr))
	case expectation == nonconformingRequest && requestErr == nil:
		problems = append(problems, errors.New("request was marked nonconforming but matches the contract"))
	}
	responseInput := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: requestInput,
		Status:                 recorder.Code,
		Header:                 recorder.Header(),
		Options:                options,
	}
	responseInput.SetBodyBytes(recorder.Body.Bytes())
	if err := openapi3filter.ValidateResponse(request.Context(), responseInput); err != nil {
		problems = append(problems, fmt.Errorf("%s response %d does not match contract: %w; body %s",
			route.Operation.OperationID, recorder.Code, err, recorder.Body.String()))
	}
	return errors.Join(problems...)
}

// TestContractHelperRejectsDrift checks the validator itself against canned handlers, so its
// logic runs without PostgreSQL.
func TestContractHelperRejectsDrift(t *testing.T) {
	router, err := loadContract()
	if err != nil {
		t.Fatal(err)
	}
	const groceries = apiPrefix + "/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5/week/current/groceries"
	const closeWeek = apiPrefix + "/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5/week/current/close"
	const line = `{"id":1,"sectionId":2,"section":"Produce","name":"Onion","quantity":null,"generatedQuantity":"1",` +
		`"origin":"generated","optional":false,"removed":false,"completed":false,"hasContributions":true,"recipeNames":["Soup"]}`
	canned := func(status int, contentType, body string) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
			response.Header().Set("Content-Type", contentType)
			response.WriteHeader(status)
			_, _ = io.WriteString(response, body)
		})
	}
	const jsonType = "application/json; charset=utf-8"
	cases := []struct {
		name        string
		method      string
		path        string
		body        string
		expectation contractExpectation
		handler     http.Handler
		wantErr     bool
	}{
		{"conforming groceries", http.MethodGet, groceries, "", conformingRequest,
			canned(http.StatusOK, jsonType, `{"startsOn":"2026-08-16","lines":[`+line+`]}`), false},
		{"renamed line field", http.MethodGet, groceries, "", conformingRequest,
			canned(http.StatusOK, jsonType, `{"startsOn":"2026-08-16","lines":[`+
				`{"id":1,"sectionID":2,"section":"Produce","name":"Onion","quantity":null,"generatedQuantity":"1",`+
				`"origin":"generated","optional":false,"removed":false,"completed":false,"hasContributions":true,"recipeNames":[]}]}`), true},
		{"invalid date", http.MethodGet, groceries, "", conformingRequest,
			canned(http.StatusOK, jsonType, `{"startsOn":"August 16","lines":[]}`), true},
		{"undocumented status", http.MethodGet, groceries, "", conformingRequest,
			canned(http.StatusTeapot, jsonType, `{"error":{"code":"x","message":"y"}}`), true},
		{"plain text error", http.MethodGet, groceries, "", conformingRequest,
			canned(http.StatusNotFound, "text/plain; charset=utf-8", "404 page not found\n"), true},
		{"documented error", http.MethodGet, groceries, "", conformingRequest,
			canned(http.StatusNotFound, jsonType, `{"error":{"code":"no_current_week","message":"m"}}`), false},
		{"path outside contract", http.MethodGet, apiPrefix + "/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5/nowhere", "", conformingRequest,
			canned(http.StatusOK, jsonType, `{}`), true},
		{"path without server prefix", http.MethodGet, "/households/c0a7a2d8-669b-4e47-91c1-4d9a32f339d5/week/current/groceries", "", conformingRequest,
			canned(http.StatusOK, jsonType, `{"startsOn":"2026-08-16","lines":[]}`), true},
		{"invalid request body", http.MethodPost, groceries, `{"name":"Milk","surprise":true}`, conformingRequest,
			canned(http.StatusBadRequest, jsonType, `{"error":{"code":"invalid_json","message":"m"}}`), true},
		{"expected invalid request body", http.MethodPost, groceries, `{"name":"Milk","surprise":true}`, nonconformingRequest,
			canned(http.StatusBadRequest, jsonType, `{"error":{"code":"invalid_json","message":"m"}}`), false},
		{"nonconforming request that conforms", http.MethodPost, groceries, `{"name":"Milk"}`, nonconformingRequest,
			canned(http.StatusOK, jsonType, `{"startsOn":"2026-08-16","lines":[]}`), true},
		{"documented empty success", http.MethodPost, closeWeek, "", conformingRequest,
			canned(http.StatusNoContent, "", ""), false},
		{"empty success with an undocumented body", http.MethodPost, closeWeek, "", conformingRequest,
			canned(http.StatusOK, jsonType, `{}`), true},
		{"undocumented method", http.MethodDelete, groceries, "", undocumentedOperation,
			canned(http.StatusMethodNotAllowed, "text/plain", "Method Not Allowed\n"), false},
		{"undocumented operation answered with success", http.MethodDelete, groceries, "", undocumentedOperation,
			canned(http.StatusOK, jsonType, `{}`), true},
		{"documented operation marked undocumented", http.MethodGet, groceries, "", undocumentedOperation,
			canned(http.StatusOK, jsonType, `{"startsOn":"2026-08-16","lines":[]}`), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(tc.method, tc.path, bytes.NewReader([]byte(tc.body)))
			if tc.body != "" {
				request.Header.Set("Content-Type", "application/json")
			}
			err := serveAgainstContract(router, tc.handler, httptest.NewRecorder(), request, tc.expectation)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, want error %t", err, tc.wantErr)
			}
		})
	}
}
