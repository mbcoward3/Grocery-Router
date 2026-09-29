package recipeintel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// DefaultEndpoint is TypeSafe's production System One evaluation endpoint.
const DefaultEndpoint = "https://api.typesafe.ai/v1/systemone"

// EvaluationRequest is one projected state and its parallel typed questions.
type EvaluationRequest struct {
	State     any                    `json:"state"`
	Model     string                 `json:"model"`
	Questions map[string]APIQuestion `json:"questions"`
}

// EvaluationResponse preserves the model identity, raw typed answers, and usage.
type EvaluationResponse struct {
	Model   string                     `json:"model"`
	Answers map[string]json.RawMessage `json:"answers"`
	Usage   Usage                      `json:"usage"`
}

// Usage records the token accounting returned by TypeSafe.
type Usage struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
}

// Answer is the common decoded view of Noul, Choice, and Score answers.
type Answer struct {
	Type          string             `json:"type"`
	Noul          *float64           `json:"noul,omitempty"`
	Choice        string             `json:"choice,omitempty"`
	Score         *float64           `json:"score,omitempty"`
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
	Confidence    *float64           `json:"confidence,omitempty"`
	Legend        map[string]string  `json:"legend,omitempty"`
}

// Evaluator permits live and fake TypeSafe implementations.
type Evaluator interface {
	Evaluate(context.Context, EvaluationRequest) (EvaluationResponse, error)
}

// HTTPClient calls the TypeSafe API without exposing its API key in artifacts.
type HTTPClient struct {
	Endpoint   string
	APIKey     string
	Client     *http.Client
	MaxRetries int
}

// Evaluate sends one request and validates that every typed answer is present.
func (client HTTPClient) Evaluate(ctx context.Context, request EvaluationRequest) (EvaluationResponse, error) {
	if strings.TrimSpace(client.APIKey) == "" {
		return EvaluationResponse{}, fmt.Errorf("TYPESAFE_API_KEY is required")
	}
	endpoint := client.Endpoint
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	httpClient := client.Client
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 90 * time.Second}
	}
	body, err := json.Marshal(request)
	if err != nil {
		return EvaluationResponse{}, fmt.Errorf("encode TypeSafe request: %w", err)
	}
	maxRetries := client.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}
	for attempt := 0; ; attempt++ {
		result, retryAfter, err := client.attempt(ctx, httpClient, endpoint, body)
		if err == nil {
			if err := validateResponse(request, result); err != nil {
				return EvaluationResponse{}, err
			}
			return result, nil
		}
		var retryable *retryableError
		if !errors.As(err, &retryable) || attempt >= maxRetries {
			return EvaluationResponse{}, err
		}
		wait := retryAfter
		if wait <= 0 {
			wait = time.Duration(1<<attempt) * time.Second
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return EvaluationResponse{}, ctx.Err()
		case <-timer.C:
		}
	}
}

func (client HTTPClient) attempt(ctx context.Context, httpClient *http.Client, endpoint string, body []byte) (EvaluationResponse, time.Duration, error) {
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return EvaluationResponse{}, 0, fmt.Errorf("create TypeSafe request: %w", err)
	}
	httpRequest.Header.Set("Authorization", "Bearer "+client.APIKey)
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("User-Agent", "grocery-router-recipe-intelligence/1")
	response, err := httpClient.Do(httpRequest)
	if err != nil {
		return EvaluationResponse{}, 0, fmt.Errorf("call TypeSafe: %w", err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil {
		return EvaluationResponse{}, 0, fmt.Errorf("read TypeSafe response: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		message := strings.TrimSpace(string(responseBody))
		if len(message) > 1000 {
			message = message[:1000]
		}
		statusErr := fmt.Errorf("TypeSafe returned %s: %s", response.Status, message)
		if response.StatusCode == http.StatusTooManyRequests || response.StatusCode == 529 {
			return EvaluationResponse{}, parseRetryAfter(response.Header.Get("Retry-After")), &retryableError{err: statusErr}
		}
		return EvaluationResponse{}, 0, statusErr
	}
	var result EvaluationResponse
	decoder := json.NewDecoder(bytes.NewReader(responseBody))
	if err := decoder.Decode(&result); err != nil {
		return EvaluationResponse{}, 0, fmt.Errorf("decode TypeSafe response: %w", err)
	}
	return result, 0, nil
}

type retryableError struct{ err error }

func (err *retryableError) Error() string { return err.err.Error() }
func (err *retryableError) Unwrap() error { return err.err }

func parseRetryAfter(value string) time.Duration {
	seconds, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || seconds < 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func validateResponse(request EvaluationRequest, response EvaluationResponse) error {
	if strings.TrimSpace(response.Model) == "" {
		return fmt.Errorf("TypeSafe response has no model")
	}
	if len(response.Answers) != len(request.Questions) {
		return fmt.Errorf("TypeSafe returned %d answers for %d questions", len(response.Answers), len(request.Questions))
	}
	for key, question := range request.Questions {
		raw, ok := response.Answers[key]
		if !ok {
			return fmt.Errorf("TypeSafe response is missing answer %q", key)
		}
		var answer Answer
		if err := json.Unmarshal(raw, &answer); err != nil {
			return fmt.Errorf("decode TypeSafe answer %q: %w", key, err)
		}
		if answer.Type != question.Type {
			return fmt.Errorf("TypeSafe answer %q type = %q, want %q", key, answer.Type, question.Type)
		}
		switch answer.Type {
		case "noul":
			if answer.Noul == nil || *answer.Noul < 0 || *answer.Noul > 1 {
				return fmt.Errorf("TypeSafe answer %q has invalid noul", key)
			}
		case "choice":
			if answer.Choice == "" || answer.Confidence == nil || len(answer.Probabilities) == 0 {
				return fmt.Errorf("TypeSafe answer %q is incomplete", key)
			}
		case "score":
			if answer.Score == nil || answer.Confidence == nil || len(answer.Probabilities) == 0 || len(answer.Legend) == 0 {
				return fmt.Errorf("TypeSafe answer %q is incomplete", key)
			}
		}
	}
	return nil
}
