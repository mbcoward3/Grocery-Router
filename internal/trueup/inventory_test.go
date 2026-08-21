package trueup_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mbcoward3/grocery-router/internal/ingest"
	"github.com/mbcoward3/grocery-router/internal/trueup"
)

func TestRepositoryInventory(t *testing.T) {
	root := filepath.Join("..", "..")
	file, err := os.Open(filepath.Join(root, "archive", "trueup", "recipes.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	rows, err := trueup.ReadInventory(root, file)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(rows), 25; got != want {
		t.Fatalf("inventory has %d recipes, want %d PDF recipes", got, want)
	}
	if rows[0].Key != "chicken-and-biscuits-casserole" {
		t.Fatalf("first inventory key = %q", rows[0].Key)
	}
	if rows[len(rows)-1].Key != "tuna-melt" {
		t.Fatalf("last inventory key = %q", rows[len(rows)-1].Key)
	}
}

func TestApprovedCorpusMatchesInventory(t *testing.T) {
	root := filepath.Join("..", "..")
	file, err := os.Open(filepath.Join(root, "archive", "trueup", "recipes.csv"))
	if err != nil {
		t.Fatal(err)
	}
	rows, err := trueup.ReadInventory(root, file)
	file.Close()
	if err != nil {
		t.Fatal(err)
	}
	documents, err := ingest.ReadDirectory(filepath.Join(root, "corpus", "recipes"))
	if err != nil {
		t.Fatal(err)
	}
	if err := trueup.MatchApprovedCorpus(rows, documents); err != nil {
		t.Fatal(err)
	}
}

func TestInventoryRejectsMissingEvidence(t *testing.T) {
	csv := `inventory_key,pdf_name,pdf_pages,linked_url,supporting_evidence,proposed_recipe_name,source_relationship,status,verified_on,disposition_note
soup,Soup,1,,sources/missing.pdf,Soup,source,inventoried,,test
`
	_, err := trueup.ReadInventory(t.TempDir(), strings.NewReader(csv))
	if err == nil || !strings.Contains(err.Error(), "evidence") {
		t.Fatalf("error = %v, want missing evidence", err)
	}
}

func TestInventoryRejectsVerifiedWithoutDate(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "evidence.txt"), []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	csv := `inventory_key,pdf_name,pdf_pages,linked_url,supporting_evidence,proposed_recipe_name,source_relationship,status,verified_on,disposition_note
soup,Soup,1,,evidence.txt,Soup,source,verified,,test
`
	_, err := trueup.ReadInventory(root, strings.NewReader(csv))
	if err == nil || !strings.Contains(err.Error(), "verified_on") {
		t.Fatalf("error = %v, want verified_on failure", err)
	}
}
