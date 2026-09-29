package recipeintel

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPClientSendsAuthenticatedTypedRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("Authorization"); got != "Bearer secret" {
			t.Errorf("Authorization = %q", got)
		}
		var body EvaluationRequest
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"model":"jev-1.13.0","answers":{"role.main":{"type":"noul","noul":0.9}},"usage":{"input_tokens":10,"output_tokens":2}}`))
	}))
	defer server.Close()
	client := HTTPClient{Endpoint: server.URL, APIKey: "secret", Client: server.Client()}
	result, err := client.Evaluate(t.Context(), EvaluationRequest{
		State: map[string]string{"name": "Dinner"}, Model: "jev-1.13.0",
		Questions: map[string]APIQuestion{"role.main": {Type: "noul", Instructions: "Is this a main?"}},
	})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if result.Model != "jev-1.13.0" || result.Usage.InputTokens != 10 {
		t.Fatalf("result = %#v", result)
	}
}

func TestHTTPClientRequiresAPIKeyBeforeNetwork(t *testing.T) {
	_, err := (HTTPClient{}).Evaluate(t.Context(), EvaluationRequest{})
	if err == nil {
		t.Fatal("Evaluate unexpectedly accepted an empty API key")
	}
}
