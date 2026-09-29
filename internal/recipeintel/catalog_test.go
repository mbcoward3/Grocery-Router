package recipeintel

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckedInCatalogIsCompleteAndValid(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "pilot", "recipe-intelligence.yaml"))
	if err != nil {
		t.Fatalf("read catalog: %v", err)
	}
	catalog, digest, err := ReadCatalog(data)
	if err != nil {
		t.Fatalf("ReadCatalog: %v", err)
	}
	if len(catalog.Questions) != 137 {
		t.Fatalf("question count = %d, want 137", len(catalog.Questions))
	}
	if len(digest) != 64 {
		t.Fatalf("digest = %q", digest)
	}
	grouped, err := catalog.QuestionsByProjection()
	if err != nil {
		t.Fatalf("QuestionsByProjection: %v", err)
	}
	count := 0
	for _, questions := range grouped {
		count += len(questions)
	}
	if count != 136 {
		t.Fatalf("enabled question count = %d, want 136", count)
	}
	for _, projection := range []string{"ingredients", "method", "full_recipe"} {
		if len(grouped[projection]) == 0 {
			t.Errorf("projection %q has no questions", projection)
		}
	}
	if len(grouped["identity"]) != 0 {
		t.Error("identity projection should be folded into full_recipe for evidence-rich classification")
	}
}

func TestCatalogRejectsUnknownFields(t *testing.T) {
	_, _, err := ReadCatalog([]byte("version: 1\nmodel: jev-1.13.0\nunknown: true\nquestions: []\n"))
	if err == nil {
		t.Fatal("ReadCatalog unexpectedly accepted an unknown field")
	}
}
