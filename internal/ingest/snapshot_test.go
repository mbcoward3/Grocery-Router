package ingest_test

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mbcoward3/grocery-router/internal/ingest"
)

var updateGolden = flag.Bool("update", false, "rewrite golden files from the current implementation")

// importSnapshotTables lists every table the bootstrap import writes or reads, in dependency
// order. Rows are compared with their identity values, so insertion order is part of the proof.
var importSnapshotTables = []string{
	"store_sections",
	"units",
	"grocery_items",
	"recipes",
	"recipe_sources",
	"recipe_ingredient_sections",
	"recipe_ingredients",
	"recipe_instruction_sections",
	"recipe_steps",
	"recipe_review_flags",
}

// TestBootstrapCorpusImportSnapshot proves the approved corpus imports to exactly the database
// contents recorded before the content contract was split. Only wall-clock audit columns are
// excluded. Regenerate with `go test ./internal/ingest -run Snapshot -update` only when an
// intentional schema or corpus change is reviewed.
func TestBootstrapCorpusImportSnapshot(t *testing.T) {
	db := openMigratedDB(t)
	documents, err := ingest.ReadDirectory(filepath.Join("..", "..", "corpus", "recipes"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ingest.Import(context.Background(), db, "c0a7a2d8-669b-4e47-91c1-4d9a32f339d5", documents); err != nil {
		t.Fatal(err)
	}

	var snapshot strings.Builder
	for _, table := range importSnapshotTables {
		rows, err := db.Query(`SELECT (to_jsonb(t) - 'created_at' - 'updated_at')::text FROM ` + table + ` t ORDER BY t.id`)
		if err != nil {
			t.Fatalf("snapshot %s: %v", table, err)
		}
		for rows.Next() {
			var row string
			if err := rows.Scan(&row); err != nil {
				t.Fatal(err)
			}
			snapshot.WriteString(table + " " + row + "\n")
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		rows.Close()
	}

	path := filepath.Join("testdata", "bootstrap-import.golden")
	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(snapshot.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := snapshot.String(); got != string(want) {
		gotLines, wantLines := strings.Split(got, "\n"), strings.Split(string(want), "\n")
		for index := 0; index < len(gotLines) && index < len(wantLines); index++ {
			if gotLines[index] != wantLines[index] {
				t.Fatalf("import snapshot differs at line %d:\n got: %s\nwant: %s", index+1, gotLines[index], wantLines[index])
			}
		}
		t.Fatalf("import snapshot has %d lines, want %d", len(gotLines), len(wantLines))
	}
}
