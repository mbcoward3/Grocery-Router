package evidence_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/mbcoward3/grocery-router/internal/catalog/evidence"
	"github.com/mbcoward3/grocery-router/internal/catalog/safefetch"
)

func response(body string) safefetch.Response {
	return safefetch.Response{
		RequestedURL: "https://www.a.test/r", FinalURL: "https://www.a.test/r/", Status: 200,
		MediaType: "text/html", Header: http.Header{"Etag": {`"v1"`}}, Body: []byte(body),
	}
}

func TestEvidenceRoundTripsCompressedBody(t *testing.T) {
	body := strings.Repeat("<p>soup</p>", 1000)
	stored, err := evidence.New("a", response(body))
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.CompressedBody) >= len(body) || stored.ETag != `"v1"` || stored.DecodedBytes != int64(len(body)) {
		t.Fatalf("evidence = %d bytes etag %q", len(stored.CompressedBody), stored.ETag)
	}
	store := evidence.NewMemoryStore()
	inserted, err := store.Insert(context.Background(), stored)
	if err != nil || inserted.ID == 0 {
		t.Fatal(err)
	}
	loaded, err := store.Get(context.Background(), inserted.ID)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := loaded.Body(safefetch.PageProfile.MaxBodyBytes)
	if err != nil || string(decoded) != body {
		t.Fatalf("Body: %v", err)
	}
	if _, err := loaded.Body(10); err == nil {
		t.Fatal("limit ignored")
	}
	loaded.BodySHA256 = strings.Repeat("0", 64)
	if _, err := loaded.Body(safefetch.PageProfile.MaxBodyBytes); err == nil {
		t.Fatal("digest mismatch accepted")
	}
	if _, err := store.Get(context.Background(), 99); !errors.Is(err, evidence.ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
}

func TestEvidenceRequiresSuccess(t *testing.T) {
	failed := response("x")
	failed.Status = 404
	if _, err := evidence.New("a", failed); err == nil {
		t.Fatal("404 became evidence")
	}
}
