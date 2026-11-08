package main

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestCatalogQualifyFailsOnEmptyCandidateSet(t *testing.T) {
	root := filepath.Join("..", "..")
	var stdout strings.Builder
	err := run([]string{"catalog-qualify", "--root", root, "--candidates", t.TempDir()}, &stdout, &stdout)
	if !errors.Is(err, errNotQualified) {
		t.Fatalf("catalog-qualify err = %v, want errNotQualified", err)
	}
	report := stdout.String()
	for _, want := range []string{
		"NOT QUALIFIED",
		"[FAIL] candidate set is not empty",
		"no holdout was supplied",
		"must-pass/love-and-lemons-banana-bread [love-and-lemons]: no_candidate (unreviewed draft)",
		"By category",
	} {
		if !strings.Contains(report, want) {
			t.Errorf("report does not contain %q:\n%s", want, report)
		}
	}
}
