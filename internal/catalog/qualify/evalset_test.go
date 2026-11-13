package qualify_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/mbcoward3/grocery-router/internal/catalog/qualify"
	"github.com/mbcoward3/grocery-router/internal/catalog/registry"
)

func TestParserRejectsMalformedCases(t *testing.T) {
	type mutation struct {
		name   string
		part   qualify.Part
		mutate func(t *testing.T, f *fixture, dir string)
		want   string
	}
	replace := func(file, old, replacement string) func(*testing.T, *fixture, string) {
		return func(t *testing.T, f *fixture, dir string) {
			path := filepath.Join(dir, file)
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(content), old) {
				t.Fatalf("%s does not contain %q:\n%s", file, old, content)
			}
			f.write(path, []byte(strings.Replace(string(content), old, replacement, 1)))
		}
	}
	expected, source := qualify.ExpectedFileName, qualify.SourceFileName
	tests := []mutation{
		{"unknown expected field", qualify.PartMustPass, replace(expected, "reviewed: true", "reviewed: true\nscore: 1"), "field score not found"},
		{"expected format version", qualify.PartMustPass, replace(expected, "format_version: 1\nreviewed", "format_version: 2\nreviewed"), "format_version = 2"},
		{"two YAML documents", qualify.PartMustPass, replace(expected, "reviewed: true", "reviewed: true\n---\nreviewed: true"), "exactly one YAML document"},
		{"unreviewed without draft note", qualify.PartMustPass, replace(expected, "reviewed: true", "reviewed: false"), "draft_note"},
		{"reviewed with draft note", qualify.PartMustPass, replace(expected, "reviewed: true", "reviewed: true\ndraft_note: leftover"), "cannot keep a draft_note"},
		{"unknown outcome", qualify.PartMustPass, replace(expected, "outcome: publish", "outcome: maybe"), "outcome must be"},
		{"must-pass expecting rejection", qualify.PartMustPass, func(_ *testing.T, f *fixture, dir string) {
			f.writeExpected(dir, qualify.Expected{FormatVersion: 1, Reviewed: true, Outcome: qualify.OutcomeReject,
				Reject: &qualify.ExpectedRejection{Codes: []string{"x"}}})
		}, "must-pass case must have outcome publish"},
		{"expected-reject expecting publication", qualify.PartExpectedReject, replace(expected, "outcome: reject", "outcome: publish"), "expected-reject case must have outcome reject"},
		{"rejection without codes", qualify.PartExpectedReject, replace(expected, "codes:\n        - missing_note_dependency", "codes: []"), "needs reject.codes"},
		{"bad rejection code", qualify.PartExpectedReject, replace(expected, "- missing_note_dependency", "- Missing Note"), "not a snake_case code"},
		{"unknown category", qualify.PartMustPass, replace(expected, "reviewed: true", "reviewed: true\ncategories: [spicy]"), "unknown category"},
		{"publish without instruction judgment", qualify.PartMustPass, replace(expected, "instructions_complete: true\n", ""), "instructions_complete"},
		{"publish with incomplete instructions", qualify.PartMustPass, replace(expected, "instructions_complete: true", "instructions_complete: false"), "instructions_complete"},
		{"publish without grocery output", qualify.PartMustPass, func(_ *testing.T, f *fixture, dir string) {
			content, _ := os.ReadFile(filepath.Join(dir, expected))
			text := string(content)
			f.write(filepath.Join(dir, expected), []byte(text[:strings.Index(text, "grocery:")]))
		}, "expected grocery output"},
		{"hand-edited grocery output", qualify.PartMustPass, replace(expected, `amount: 1/2`, `amount: 3/4`), "does not match the aggregation"},
		{"unpublishable expected document", qualify.PartMustPass, replace(expected, "state: intentionally_unspecified", "state: resolved"), "not publishable"},
		{"source lines differ from node", qualify.PartMustPass, replace(source, `"2 large eggs"`, `"3 large eggs"`), "source_lines must equal"},
		{"unknown source field", qualify.PartMustPass, replace(source, `"format_version": 1`, `"format_version": 1, "extra": true`), "unknown field"},
		{"source format version", qualify.PartMustPass, replace(source, `"format_version": 1`, `"format_version": 3`), "format_version = 3"},
		{"unknown site", qualify.PartMustPass, replace(source, `"site": "test-site"`, `"site": "nope"`), "not in the site registry"},
		{"url on another host", qualify.PartMustPass, replace(source, `https://www.example.com/`, `https://evil.example.net/`), "hosts"},
		{"node is not a recipe", qualify.PartMustPass, replace(source, `"@type": "Recipe"`, `"@type": "Article"`), "@type"},
		{"extra file", qualify.PartMustPass, func(_ *testing.T, f *fixture, dir string) { f.write(filepath.Join(dir, "notes.txt"), []byte("x")) }, "exactly"},
		{"missing expected file", qualify.PartMustPass, func(_ *testing.T, _ *fixture, dir string) { _ = os.Remove(filepath.Join(dir, expected)) }, "exactly"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newFixture(t)
			if test.part == qualify.PartExpectedReject {
				f.rejectCase(test.part, "case-a", "missing_note_dependency")
			} else {
				f.publishCase(test.part, "case-a")
			}
			dir := f.caseDir(test.part, "case-a")
			if _, err := qualify.ReadCase(dir, test.part, f.sites); err != nil {
				t.Fatalf("unmutated case is invalid: %v", err)
			}
			test.mutate(t, f, dir)
			_, err := qualify.ReadCase(dir, test.part, f.sites)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("err = %v, want it to mention %q", err, test.want)
			}
		})
	}
}

func TestReadEvalSetLayout(t *testing.T) {
	f := newFixture(t)
	f.publishCase(qualify.PartMustPass, "case-a")
	if _, err := qualify.ReadEvalSet(f.evalDir, f.sites); err != nil {
		t.Fatal(err)
	}

	f.publishCase(qualify.PartDevelopment, "case-a")
	if _, err := qualify.ReadEvalSet(f.evalDir, f.sites); err == nil || !strings.Contains(err.Error(), "appears in both") {
		t.Fatalf("duplicate case ID err = %v", err)
	}
	if err := os.RemoveAll(f.caseDir(qualify.PartDevelopment, "case-a")); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Join(f.evalDir, "holdout"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := qualify.ReadEvalSet(f.evalDir, f.sites); err == nil || !strings.Contains(err.Error(), "unexpected entry") {
		t.Fatalf("a holdout directory inside the evaluation set was accepted: %v", err)
	}
	if err := os.Remove(filepath.Join(f.evalDir, "holdout")); err != nil {
		t.Fatal(err)
	}

	if err := os.Remove(filepath.Join(f.evalDir, string(qualify.PartRegression))); err != nil {
		t.Fatal(err)
	}
	if _, err := qualify.ReadEvalSet(f.evalDir, f.sites); err == nil || !strings.Contains(err.Error(), "missing part") {
		t.Fatalf("missing part err = %v", err)
	}

	f.publishCase(qualify.PartHoldout, "Bad_ID")
	if _, err := qualify.ReadHoldout(f.holdoutDir, f.sites); err == nil {
		t.Fatal("a non-kebab-case holdout ID was accepted")
	}
}

// TestRepositoryEvaluationSet keeps the checked-in evaluation set parseable against the real
// site registry, with every expected grocery output matching the aggregation.
func TestRepositoryEvaluationSet(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	sites, err := registry.ReadFile(filepath.Join(root, filepath.FromSlash(registry.DefaultPath)))
	if err != nil {
		t.Fatal(err)
	}
	set, err := qualify.ReadEvalSet(filepath.Join(root, "catalog", "eval"), sites)
	if err != nil {
		t.Fatal(err)
	}
	parts := make(map[qualify.Part][]string)
	for _, c := range set.Cases {
		parts[c.Part] = append(parts[c.Part], c.ID)
	}
	for part, want := range map[qualify.Part]string{
		qualify.PartMustPass:       "love-and-lemons-banana-bread",
		qualify.PartExpectedReject: "recipetin-eats-chicken-marsala",
		qualify.PartDevelopment:    "chef-jean-pierre-sausage-and-peppers",
	} {
		if !slices.Contains(parts[part], want) {
			t.Errorf("%s cases = %v, want %s", part, parts[part], want)
		}
	}
	if len(parts[qualify.PartRegression]) != 0 {
		t.Errorf("regression cases = %v, want none at the start of slice 1", parts[qualify.PartRegression])
	}
}
