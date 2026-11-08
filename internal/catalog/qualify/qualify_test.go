package qualify_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/mbcoward3/grocery-router/internal/catalog/qualify"
	"github.com/mbcoward3/grocery-router/internal/catalog/registry"
	"github.com/mbcoward3/grocery-router/internal/ingest"
	"gopkg.in/yaml.v3"
)

const testRegistry = `version: 1
sites:
  - id: test-site
    name: Test Site
    hosts: [www.example.com]
    sitemaps: [https://www.example.com/sitemap.xml]
    measurement_convention: us
    active: true
    eligibility:
      intended_use: tests
      reviewed_on: 2026-11-08
      notes: test fixture
  - id: other-site
    name: Other Site
    hosts: [other.example.com]
    sitemaps: [https://other.example.com/sitemap.xml]
    measurement_convention: us
    active: false
`

func testSites(t *testing.T) registry.Registry {
	t.Helper()
	sites, err := registry.Parse([]byte(testRegistry))
	if err != nil {
		t.Fatal(err)
	}
	return sites
}

var testLines = []string{
	"½ cup walnuts (plus 2 tablespoons for topping)",
	"2 large eggs",
	"Salt, to taste",
}

// testDocument is a small publishable recipe with an additive line and a presence-only item.
func testDocument(name string) ingest.CatalogDocument {
	walnuts := &ingest.GroceryItem{Key: "walnuts", Name: "Walnuts", StoreSection: ingest.StoreSection{Key: "baking", Name: "Baking"}, ShoppingMode: "measured"}
	eggs := &ingest.GroceryItem{Key: "eggs", Name: "Eggs", StoreSection: ingest.StoreSection{Key: "dairy", Name: "Dairy"}, ShoppingMode: "counted"}
	salt := &ingest.GroceryItem{Key: "salt", Name: "Salt", StoreSection: ingest.StoreSection{Key: "spices", Name: "Spices"}, ShoppingMode: "presence-only"}
	provenance := func(line, start, end int) ingest.Provenance {
		return ingest.Provenance{Type: ingest.ProvenanceIngredientLine, Line: line, Span: ingest.Span{Start: start, End: end}}
	}
	return ingest.CatalogDocument{
		FormatVersion: ingest.CatalogFormatVersion,
		Name:          name,
		Source:        ingest.Source{Relationship: "source", Attribution: "Test Site", URL: "https://www.example.com/" + name + "/"},
		SourceLines:   slices.Clone(testLines),
		IngredientSections: []ingest.CatalogIngredientSection{{
			Name: "Ingredients",
			Requirements: []ingest.Requirement{
				{Provenance: provenance(0, 0, 13), Relationship: ingest.RelationshipAdditive, State: ingest.StateResolved,
					GroceryItem: walnuts, Quantity: &ingest.Quantity{Kind: "exact", Amount: "1/2", Unit: "cup"}},
				{Provenance: provenance(0, 15, 45), Relationship: ingest.RelationshipAdditive, State: ingest.StateResolved,
					GroceryItem: walnuts, Quantity: &ingest.Quantity{Kind: "exact", Amount: "2", Unit: "tbsp"}, Note: "for topping"},
				{Provenance: provenance(1, 0, 12), Relationship: ingest.RelationshipSingle, State: ingest.StateResolved,
					GroceryItem: eggs, Quantity: &ingest.Quantity{Kind: "exact", Amount: "2", Unit: "each"}},
				{Provenance: provenance(2, 0, 14), Relationship: ingest.RelationshipSingle, State: ingest.StateIntentionallyUnspecified,
					GroceryItem: salt, Quantity: &ingest.Quantity{Kind: "unspecified"}, Preparation: "to taste"},
			},
		}},
		InstructionSections: []ingest.InstructionSection{{Name: "Instructions", Steps: []string{
			"Whisk the eggs.", "Fold in ½ cup walnuts and top with the rest.", "Season with salt.",
		}}},
	}
}

// fixture writes evaluation cases to disk and builds a matching correct candidate set.
type fixture struct {
	t          *testing.T
	root       string
	evalDir    string
	holdoutDir string
	candidates qualify.CandidateSet
	sites      registry.Registry
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	root := t.TempDir()
	f := &fixture{t: t, root: root, evalDir: filepath.Join(root, "eval"), holdoutDir: filepath.Join(root, "holdout"),
		candidates: qualify.CandidateSet{}, sites: testSites(t)}
	for _, part := range qualify.CheckedInParts {
		if err := os.MkdirAll(filepath.Join(f.evalDir, string(part)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(f.holdoutDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return f
}

// standard writes a qualifying evaluation set: 3 must-pass, 1 expected-reject, 1 regression,
// 1 development case, and 15 holdout cases, all on the active test site.
func (f *fixture) standard() *fixture {
	for _, id := range []string{"must-a", "must-b", "must-c"} {
		f.publishCase(qualify.PartMustPass, id)
	}
	f.rejectCase(qualify.PartExpectedReject, "reject-a", "missing_note_dependency")
	f.publishCase(qualify.PartRegression, "regression-a")
	f.publishCase(qualify.PartDevelopment, "development-a")
	for index := range 15 {
		f.publishCase(qualify.PartHoldout, "holdout-"+string(rune('a'+index)))
	}
	return f
}

func (f *fixture) caseDir(part qualify.Part, id string) string {
	if part == qualify.PartHoldout {
		return filepath.Join(f.holdoutDir, id)
	}
	return filepath.Join(f.evalDir, string(part), id)
}

func (f *fixture) writeSource(dir, id string, lines []string) {
	f.t.Helper()
	node, _ := json.Marshal(map[string]any{"@context": "https://schema.org", "@type": "Recipe", "name": id, "recipeIngredient": lines})
	source, _ := json.MarshalIndent(map[string]any{
		"format_version": 1, "site": "test-site", "url": "https://www.example.com/" + id + "/", "node": json.RawMessage(node),
	}, "", "  ")
	f.write(filepath.Join(dir, qualify.SourceFileName), source)
}

func (f *fixture) write(path string, content []byte) {
	f.t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) publishCase(part qualify.Part, id string) {
	f.t.Helper()
	document := testDocument(id)
	grocery, err := qualify.GroceryOutput(document)
	if err != nil {
		f.t.Fatal(err)
	}
	complete := true
	expected := qualify.Expected{
		FormatVersion: 1, Reviewed: true, Outcome: qualify.OutcomePublish,
		InstructionsComplete: &complete, Document: &document, Grocery: grocery,
	}
	dir := f.caseDir(part, id)
	f.writeSource(dir, id, document.SourceLines)
	f.writeExpected(dir, expected)
	candidate := testDocument(id)
	f.candidates[id] = qualify.Candidate{FormatVersion: 1, Document: &candidate}
}

func (f *fixture) rejectCase(part qualify.Part, id string, code string) {
	f.t.Helper()
	dir := f.caseDir(part, id)
	f.writeSource(dir, id, testLines)
	f.writeExpected(dir, qualify.Expected{
		FormatVersion: 1, Reviewed: true, Outcome: qualify.OutcomeReject,
		Reject: &qualify.ExpectedRejection{Codes: []string{code}},
	})
	f.candidates[id] = qualify.Candidate{FormatVersion: 1, Rejections: []qualify.Rejection{{Code: code}}}
}

func (f *fixture) writeExpected(dir string, expected qualify.Expected) {
	f.t.Helper()
	content, err := yaml.Marshal(expected)
	if err != nil {
		f.t.Fatal(err)
	}
	f.write(filepath.Join(dir, qualify.ExpectedFileName), content)
}

func (f *fixture) input(withHoldout bool) qualify.Input {
	f.t.Helper()
	set, err := qualify.ReadEvalSet(f.evalDir, f.sites)
	if err != nil {
		f.t.Fatal(err)
	}
	input := qualify.Input{EvalSet: set, Candidates: f.candidates, Sites: f.sites}
	if withHoldout {
		if input.Holdout, err = qualify.ReadHoldout(f.holdoutDir, f.sites); err != nil {
			f.t.Fatal(err)
		}
	}
	return input
}

func run(t *testing.T, input qualify.Input) qualify.Report {
	t.Helper()
	report, err := qualify.Qualify(input)
	if err != nil {
		t.Fatal(err)
	}
	return report
}

func criterion(t *testing.T, report qualify.Report, prefix string) qualify.Criterion {
	t.Helper()
	for _, c := range report.Criteria {
		if strings.HasPrefix(c.Name, prefix) {
			return c
		}
	}
	t.Fatalf("no criterion starting %q", prefix)
	return qualify.Criterion{}
}

func result(t *testing.T, report qualify.Report, id string) qualify.CaseResult {
	t.Helper()
	for _, r := range report.Results {
		if r.Case.ID == id {
			return r
		}
	}
	t.Fatalf("no result for %s", id)
	return qualify.CaseResult{}
}

func failedCriteria(report qualify.Report) []string {
	var names []string
	for _, c := range report.Criteria {
		if !c.Passed {
			names = append(names, c.Name+": "+strings.Join(c.Failures, "; "))
		}
	}
	return names
}

func TestCorrectCandidateSetQualifies(t *testing.T) {
	report := run(t, newFixture(t).standard().input(true))
	if !report.Qualified() {
		var text strings.Builder
		_ = report.WriteText(&text)
		t.Fatalf("correct candidate set did not qualify: %v\n%s", failedCriteria(report), text.String())
	}
	if got := result(t, report, "must-a"); got.Status != qualify.StatusCorrect || len(got.Differences) != 0 {
		t.Fatalf("must-a = %s %v, want correct with no differences", got.Status, got.Differences)
	}
	for _, tally := range report.ByCategory {
		if tally.Name == string(qualify.CategoryAdditive) && tally.Correct != 20 {
			t.Fatalf("additive category correct = %d, want 20", tally.Correct)
		}
	}
}

func TestEmptyCandidateSetFails(t *testing.T) {
	f := newFixture(t).standard()
	f.candidates = qualify.CandidateSet{}
	report := run(t, f.input(true))
	if report.Qualified() {
		t.Fatal("empty candidate set qualified")
	}
	if criterion(t, report, "candidate set is not empty").Passed {
		t.Fatal("empty candidate set criterion passed")
	}
	if criterion(t, report, "at least 15 holdout").Passed {
		t.Fatal("empty holdout result passed")
	}
}

func TestHoldoutIsRequired(t *testing.T) {
	f := newFixture(t).standard()
	for id := range f.candidates {
		if strings.HasPrefix(id, "holdout-") {
			delete(f.candidates, id)
		}
	}
	report := run(t, f.input(false))
	holdout := criterion(t, report, "at least 15 holdout")
	if holdout.Passed || !strings.Contains(strings.Join(holdout.Failures, " "), "no holdout was supplied") {
		t.Fatalf("missing holdout criterion = %+v, want failure", holdout)
	}
}

func TestHoldoutNeedsFifteenPublished(t *testing.T) {
	f := newFixture(t).standard()
	delete(f.candidates, "holdout-a")
	report := run(t, f.input(true))
	if holdout := criterion(t, report, "at least 15 holdout"); holdout.Passed {
		t.Fatal("14 of 15 published holdout recipes passed")
	}
}

func TestHoldoutMaterialErrorFails(t *testing.T) {
	f := newFixture(t).standard()
	f.publishCase(qualify.PartHoldout, "holdout-extra")
	f.candidates["holdout-extra"].Document.IngredientSections[0].Requirements[2].Quantity.Amount = "3"
	report := run(t, f.input(true))
	if holdout := criterion(t, report, "at least 15 holdout"); holdout.Passed {
		t.Fatal("a holdout recipe with a wrong amount passed")
	}
}

func TestMissingRequirementIsMaterialError(t *testing.T) {
	f := newFixture(t).standard()
	candidate := f.candidates["must-a"].Document
	requirements := candidate.IngredientSections[0].Requirements
	// Drop the walnut topping. The remaining walnut line is still publishable as a single line.
	requirements[0].Relationship = ingest.RelationshipSingle
	candidate.IngredientSections[0].Requirements = append(requirements[:1:1], requirements[2:]...)
	if issues := ingest.CheckPublishable(*candidate); len(issues) > 0 {
		t.Fatalf("test candidate should be publishable: %v", issues)
	}

	report := run(t, f.input(true))
	got := result(t, report, "must-a")
	if !got.Published || got.Status != qualify.StatusMaterialErrors {
		t.Fatalf("must-a = %s (published %v), want published with material errors", got.Status, got.Published)
	}
	kinds := func(differences []qualify.Difference) []qualify.DifferenceKind {
		var result []qualify.DifferenceKind
		for _, difference := range differences {
			result = append(result, difference.Kind)
		}
		return result
	}
	if material := kinds(got.MaterialErrors()); !slices.Equal(material, []qualify.DifferenceKind{qualify.MissingRequirement}) {
		t.Fatalf("material errors = %v, want [missing_requirement]", material)
	}
	if !slices.Contains(kinds(got.Differences), qualify.CosmeticRelationship) {
		t.Fatalf("differences = %v, want the relationship change reported as cosmetic", got.Differences)
	}
	if report.Qualified() || criterion(t, report, "every must-pass").Passed {
		t.Fatal("a must-pass recipe with a missing requirement qualified")
	}
}

func TestMaterialDifferenceKinds(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ingest.CatalogDocument)
		want   qualify.DifferenceKind
	}{
		{"amount", func(d *ingest.CatalogDocument) { d.IngredientSections[0].Requirements[2].Quantity.Amount = "3" }, qualify.WrongAmount},
		{"unit", func(d *ingest.CatalogDocument) { d.IngredientSections[0].Requirements[0].Quantity.Unit = "tbsp" }, qualify.WrongUnit},
		{"range", func(d *ingest.CatalogDocument) {
			d.IngredientSections[0].Requirements[2].Quantity = &ingest.Quantity{Kind: "range", Amount: "2", Maximum: "3", Unit: "each"}
		}, qualify.WrongRange},
		{"package", func(d *ingest.CatalogDocument) {
			d.IngredientSections[0].Requirements[2].Quantity = &ingest.Quantity{Kind: "exact", Amount: "2", Package: &ingest.Package{Type: "carton"}}
		}, qualify.WrongPackage},
		{"optional", func(d *ingest.CatalogDocument) { d.IngredientSections[0].Requirements[2].Optional = true }, qualify.WrongOptional},
		{"non-shopping", func(d *ingest.CatalogDocument) { d.IngredientSections[0].Requirements[2].NonShopping = true }, qualify.WrongNonShopping},
		{"unspecified", func(d *ingest.CatalogDocument) {
			r := &d.IngredientSections[0].Requirements[2]
			r.State, r.Quantity = ingest.StateIntentionallyUnspecified, &ingest.Quantity{Kind: "unspecified"}
		}, qualify.WrongUnspecified},
		{"item", func(d *ingest.CatalogDocument) {
			d.IngredientSections[0].Requirements[2].GroceryItem = &ingest.GroceryItem{Key: "egg-whites", Name: "Egg Whites",
				StoreSection: ingest.StoreSection{Key: "dairy", Name: "Dairy"}, ShoppingMode: "counted"}
		}, qualify.WrongItem},
		{"shopping mode", func(d *ingest.CatalogDocument) {
			for index := range d.IngredientSections[0].Requirements[:2] {
				item := *d.IngredientSections[0].Requirements[index].GroceryItem
				item.ShoppingMode = "counted"
				d.IngredientSections[0].Requirements[index].GroceryItem = &item
			}
		}, qualify.WrongShoppingMode},
		{"preparation", func(d *ingest.CatalogDocument) { d.IngredientSections[0].Requirements[2].Preparation = "beaten" }, qualify.WrongPreparation},
		{"missing step", func(d *ingest.CatalogDocument) { d.InstructionSections[0].Steps = d.InstructionSections[0].Steps[1:] }, qualify.MissingStep},
		{"changed step", func(d *ingest.CatalogDocument) { d.InstructionSections[0].Steps[2] = "Season with pepper." }, qualify.WrongStep},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newFixture(t).standard()
			test.mutate(f.candidates["must-a"].Document)
			got := result(t, run(t, f.input(true)), "must-a")
			if !slices.ContainsFunc(got.MaterialErrors(), func(d qualify.Difference) bool { return d.Kind == test.want }) {
				t.Fatalf("material errors = %v, want %s", got.MaterialErrors(), test.want)
			}
		})
	}
}

func TestCosmeticDifferencesDoNotFail(t *testing.T) {
	f := newFixture(t).standard()
	candidate := f.candidates["must-a"].Document
	candidate.Name = "Renamed"
	candidate.InstructionSections[0].Steps[0] = "whisk  the eggs"
	candidate.IngredientSections[0].Requirements[1].Note = "topping"
	candidate.IngredientSections[0].Requirements[2].GroceryItem = &ingest.GroceryItem{Key: "eggs", Name: "Large Eggs",
		StoreSection: ingest.StoreSection{Key: "dairy", Name: "Dairy"}, ShoppingMode: "counted"}
	report := run(t, f.input(true))
	got := result(t, report, "must-a")
	if got.Status != qualify.StatusCorrect || len(got.Differences) != 4 {
		t.Fatalf("must-a = %s with %v, want correct with four cosmetic differences", got.Status, got.Differences)
	}
	if !report.Qualified() {
		t.Fatalf("cosmetic differences failed qualification: %v", failedCriteria(report))
	}
}

func TestExpectedRejectThatPublishesFails(t *testing.T) {
	f := newFixture(t).standard()
	published := testDocument("reject-a")
	f.candidates["reject-a"] = qualify.Candidate{FormatVersion: 1, Document: &published}
	report := run(t, f.input(true))
	got := result(t, report, "reject-a")
	if !got.Published || got.Status != qualify.StatusMaterialErrors || got.MaterialErrors()[0].Kind != qualify.UnexpectedPublishing {
		t.Fatalf("reject-a = %s %v, want unexpected publication", got.Status, got.Differences)
	}
	if report.Qualified() || criterion(t, report, "every expected-reject").Passed {
		t.Fatal("an expected-reject recipe that published qualified")
	}
}

func TestUnpublishableCandidateIsRejected(t *testing.T) {
	f := newFixture(t).standard()
	candidate := f.candidates["reject-a"]
	unresolved := testDocument("reject-a")
	unresolved.IngredientSections[0].Requirements[2] = ingest.Requirement{
		Provenance: unresolved.IngredientSections[0].Requirements[2].Provenance,
		State:      ingest.StateUnresolved, Reason: "ambiguous_unit",
	}
	candidate.Document, candidate.Rejections = &unresolved, nil
	f.candidates["reject-a"] = candidate
	got := result(t, run(t, f.input(true)), "reject-a")
	if got.Published || got.Status != qualify.StatusRejectedAsExpected {
		t.Fatalf("reject-a = %s, want rejected", got.Status)
	}
	if !slices.Contains(got.RejectCodes, "ambiguous_unit") || !slices.Equal(got.MissingRejectCodes, []string{"missing_note_dependency"}) {
		t.Fatalf("codes = %v missing %v", got.RejectCodes, got.MissingRejectCodes)
	}
}

func TestAcceptedRegressionHonoredOnlyWhenListed(t *testing.T) {
	f := newFixture(t).standard()
	for _, id := range []string{"must-a", "regression-a"} {
		f.candidates[id] = qualify.Candidate{FormatVersion: 1, Rejections: []qualify.Rejection{{Code: "incoherent_alternative"}}}
	}

	unlisted := run(t, f.input(true))
	if unlisted.Qualified() || criterion(t, unlisted, "every must-pass").Passed || criterion(t, unlisted, "every regression").Passed {
		t.Fatal("rejected must-pass and regression recipes qualified without being listed")
	}

	input := f.input(true)
	input.Accepted = qualify.AcceptedRegressions{FormatVersion: 1, Accepted: []qualify.AcceptedRegression{{Case: "must-b", Reason: "unrelated"}}}
	if run(t, input).Qualified() {
		t.Fatal("listing a different case honored the rejected ones")
	}

	input.Accepted.Accepted = []qualify.AcceptedRegression{
		{Case: "must-a", Reason: "stricter alternative rule"},
		{Case: "regression-a", Reason: "stricter alternative rule"},
	}
	listed := run(t, input)
	if got := result(t, listed, "must-a"); got.Status != qualify.StatusAcceptedRegression || got.AcceptedReason == "" {
		t.Fatalf("must-a = %s, want accepted regression", got.Status)
	}
	// must-a no longer counts toward the active site's minimum, but must-b, must-c, the
	// development case, and the holdout still do.
	if !listed.Qualified() {
		t.Fatalf("listed accepted regressions did not qualify: %v", failedCriteria(listed))
	}

	input.Accepted.Accepted = []qualify.AcceptedRegression{{Case: "reject-a", Reason: "not allowed"}}
	if _, err := qualify.Qualify(input); err == nil {
		t.Fatal("an accepted regression naming an expected-reject case was allowed")
	}
}

func TestActiveSiteNeedsThreePublishedRecipes(t *testing.T) {
	f := newFixture(t)
	f.publishCase(qualify.PartMustPass, "must-a")
	f.publishCase(qualify.PartMustPass, "must-b")
	report := run(t, f.input(false))
	sites := criterion(t, report, "each active site")
	if sites.Passed || !strings.Contains(strings.Join(sites.Failures, " "), "test-site: 2") {
		t.Fatalf("site criterion = %+v, want test-site failure", sites)
	}
}

func TestUnreviewedCaseBlocksQualification(t *testing.T) {
	f := newFixture(t).standard()
	path := filepath.Join(f.caseDir(qualify.PartMustPass, "must-b"), qualify.ExpectedFileName)
	content, _ := os.ReadFile(path)
	content = []byte(strings.Replace(string(content), "reviewed: true", "reviewed: false\ndraft_note: check me", 1))
	f.write(path, content)
	report := run(t, f.input(true))
	if report.Qualified() || criterion(t, report, "every evaluation and holdout case has been reviewed").Passed {
		t.Fatal("an unreviewed case did not block qualification")
	}
}

func TestQualifyRejectsUnknownCandidate(t *testing.T) {
	f := newFixture(t).standard()
	f.candidates["no-such-case"] = qualify.Candidate{FormatVersion: 1, Rejections: []qualify.Rejection{{Code: "x"}}}
	if _, err := qualify.Qualify(f.input(true)); err == nil {
		t.Fatal("a candidate naming no case was accepted")
	}
}

func TestReadCandidateSet(t *testing.T) {
	dir := t.TempDir()
	document := testDocument("must-a")
	content, _ := yaml.Marshal(qualify.Candidate{FormatVersion: 1, Document: &document})
	if err := os.WriteFile(filepath.Join(dir, "must-a.yaml"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	set, err := qualify.ReadCandidateSet(dir)
	if err != nil {
		t.Fatal(err)
	}
	if published, codes := set["must-a"].Evaluate(); !published {
		t.Fatalf("round-tripped candidate did not publish: %v", codes)
	}
	for name, body := range map[string]string{
		"bad name.yaml": "format_version: 1\nrejections: [{code: x}]\n",
		"empty.yaml":    "format_version: 1\n",
		"unknown.yaml":  "format_version: 1\nrejections: [{code: x}]\nscore: 3\n",
		"version.yaml":  "format_version: 2\nrejections: [{code: x}]\n",
		"code.yaml":     "format_version: 1\nrejections: [{code: Not-Snake}]\n",
	} {
		bad := t.TempDir()
		if err := os.WriteFile(filepath.Join(bad, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := qualify.ReadCandidateSet(bad); err == nil {
			t.Errorf("%s: malformed candidate accepted", name)
		}
	}
}

func TestReadAcceptedRegressions(t *testing.T) {
	for name, body := range map[string]string{
		"valid":     "format_version: 1\naccepted:\n  - {case: must-a, reason: why}\n",
		"no reason": "format_version: 1\naccepted:\n  - {case: must-a, reason: ' '}\n",
		"duplicate": "format_version: 1\naccepted:\n  - {case: must-a, reason: a}\n  - {case: must-a, reason: b}\n",
		"unknown":   "format_version: 1\naccepted: []\nextra: 1\n",
		"version":   "format_version: 0\naccepted: []\n",
	} {
		path := filepath.Join(t.TempDir(), "accepted.yaml")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := qualify.ReadAcceptedRegressions(path)
		if (err == nil) != (name == "valid") {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}
