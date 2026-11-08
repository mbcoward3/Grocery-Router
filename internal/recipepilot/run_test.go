package recipepilot

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type stubFetcher struct {
	calls []string
}

func (fetcher *stubFetcher) Fetch(_ context.Context, rawURL string) (FetchResult, error) {
	fetcher.calls = append(fetcher.calls, rawURL)
	if len(fetcher.calls) == 1 {
		return FetchResult{Metadata: FetchMetadata{RequestedURL: rawURL, FetchedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)}}, errors.New("source refused request")
	}
	return FetchResult{
		Metadata: FetchMetadata{
			RequestedURL: rawURL, FinalURL: rawURL, FetchedAt: time.Date(2026, 1, 2, 3, 4, 6, 0, time.UTC),
			Status: 200, MediaType: "text/html", DecodedBytes: 82, BodySHA256: "digest",
		},
		Body: []byte(`<script type="application/ld+json">{"@type":"Recipe","name":"Kept"}</script>`),
	}, nil
}

func TestRunnerWritesCompleteInspectableRunAfterSourceFailure(t *testing.T) {
	root := t.TempDir()
	manifestPath := filepath.Join(root, "manifest.yaml")
	manifest := "version: 1\nsources:\n  - id: first\n    url: https://example.com/first\n  - id: second\n    url: https://example.com/second\n"
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	fetcher := &stubFetcher{}
	runner := Runner{
		Fetcher: fetcher,
		Now: func() time.Time {
			return time.Date(2026, 2, 3, 4, 5, 6, 7, time.UTC)
		},
	}
	report, err := runner.Run(t.Context(), manifestPath, filepath.Join(root, "output"))
	if err == nil {
		t.Fatal("Run unexpectedly succeeded")
	}
	if len(fetcher.calls) != 2 {
		t.Fatalf("fetch calls = %d, want 2", len(fetcher.calls))
	}
	if report.Index.Failed != 1 || report.Index.Succeeded != 1 {
		t.Fatalf("index counts = %#v", report.Index)
	}
	for _, name := range []string{"first.json", "second.json", "index.json"} {
		if _, err := os.Stat(filepath.Join(report.Directory, name)); err != nil {
			t.Errorf("missing %s: %v", name, err)
		}
	}
	data, err := os.ReadFile(filepath.Join(report.Directory, "second.json"))
	if err != nil {
		t.Fatalf("read second artifact: %v", err)
	}
	var artifact SourceArtifact
	if err := json.Unmarshal(data, &artifact); err != nil {
		t.Fatalf("decode second artifact: %v", err)
	}
	if len(artifact.Recipes) != 1 || string(artifact.Recipes[0].Recipe.Name) != `"Kept"` {
		t.Fatalf("second recipes = %#v", artifact.Recipes)
	}
}
