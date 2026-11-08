// Package qualify reads the catalog evaluation set and scores candidate catalog documents
// against it under the qualification criteria of CATALOG_INGEST_SPEC.md §6.3.
package qualify

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/mbcoward3/grocery-router/internal/catalog/registry"
	"github.com/mbcoward3/grocery-router/internal/ingest"
	"gopkg.in/yaml.v3"
)

// CaseFormatVersion is the evaluation case format version (source.json and expected.yaml).
const CaseFormatVersion = 1

// Case file names inside one case directory.
const (
	SourceFileName   = "source.json"
	ExpectedFileName = "expected.yaml"
	readmeFileName   = "README.md"
)

// Part is one part of the evaluation set (§4 slice 1).
type Part string

// Evaluation parts. PartHoldout lives outside the repository and is supplied by path.
const (
	PartMustPass       Part = "must-pass"
	PartExpectedReject Part = "expected-reject"
	PartRegression     Part = "regression"
	PartDevelopment    Part = "development"
	PartHoldout        Part = "holdout"
)

// CheckedInParts are the parts stored under catalog/eval/, in report order.
var CheckedInParts = []Part{PartMustPass, PartExpectedReject, PartRegression, PartDevelopment}

// Outcome is what a case expects the pipeline to do with its source node.
type Outcome string

// Expected outcomes.
const (
	OutcomePublish Outcome = "publish"
	OutcomeReject  Outcome = "reject"
)

// Category is a reporting category (§6.3).
type Category string

// Reporting categories in report order.
const (
	CategoryAdditive      Category = "additive"
	CategoryCompound      Category = "compound"
	CategoryAlternative   Category = "alternative"
	CategoryAICreatedItem Category = "ai_created_item"
	CategoryNonShopping   Category = "non_shopping"
	CategoryPresenceOnly  Category = "presence_only"
)

// Categories lists every reporting category in report order.
var Categories = []Category{
	CategoryAdditive, CategoryCompound, CategoryAlternative,
	CategoryAICreatedItem, CategoryNonShopping, CategoryPresenceOnly,
}

var (
	caseIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	codePattern   = regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`)
)

// Source is a case's source.json: one Schema.org Recipe node and where it came from.
type Source struct {
	FormatVersion int    `json:"format_version"`
	Site          string `json:"site"`
	URL           string `json:"url"`
	// Note records how the node was captured, for example when it was reconstructed from
	// pilot data rather than copied from stored evidence.
	Note string `json:"note,omitempty"`
	// Node is the raw Schema.org Recipe JSON object, kept byte for byte.
	Node json.RawMessage `json:"node"`
	// Ingredients is the node's recipeIngredient list, decoded from Node.
	Ingredients []string `json:"-"`
}

// Expected is a case's expected.yaml.
type Expected struct {
	FormatVersion int  `yaml:"format_version"`
	Reviewed      bool `yaml:"reviewed"`
	// DraftNote explains what a reviewer must check. It is required while reviewed is false
	// and forbidden once the household has reviewed the case.
	DraftNote string `yaml:"draft_note,omitempty"`
	// Note is a permanent explanation that survives review.
	Note    string  `yaml:"note,omitempty"`
	Outcome Outcome `yaml:"outcome"`
	// Categories adds reporting categories the expected document cannot show, such as
	// ai_created_item, or that a rejected case exercises.
	Categories []Category `yaml:"categories,omitempty"`
	// InstructionsComplete is the household's judgment that the source instructions are
	// complete enough to cook from. Required, and true, for outcome publish.
	InstructionsComplete *bool  `yaml:"instructions_complete,omitempty"`
	InstructionsNote     string `yaml:"instructions_note,omitempty"`
	// Document is the expected interpretation in the catalog content format (outcome publish).
	Document *ingest.CatalogDocument `yaml:"document,omitempty"`
	// Grocery is the expected grocery output for Document, produced by the existing
	// aggregation (outcome publish).
	Grocery []GroceryLine `yaml:"grocery,omitempty"`
	// Reject is the expected rejection (outcome reject).
	Reject *ExpectedRejection `yaml:"reject,omitempty"`
}

// ExpectedRejection lists the issue or reason codes a rejection must include.
type ExpectedRejection struct {
	Codes []string `yaml:"codes"`
}

// Case is one parsed evaluation case.
type Case struct {
	ID       string
	Part     Part
	Dir      string
	Source   Source
	Expected Expected
}

// Site returns the case's registry site ID.
func (c Case) Site() string { return c.Source.Site }

// CaseCategories returns the reporting categories a case exercises: those its expected
// document shows plus those its expected.yaml lists.
func (c Case) CaseCategories() []Category {
	seen := make(map[Category]bool)
	for _, category := range c.Expected.Categories {
		seen[category] = true
	}
	if document := c.Expected.Document; document != nil {
		for _, section := range document.IngredientSections {
			for _, requirement := range section.Requirements {
				switch requirement.Relationship {
				case ingest.RelationshipAdditive:
					seen[CategoryAdditive] = true
				case ingest.RelationshipCompound:
					seen[CategoryCompound] = true
				case ingest.RelationshipAlternative:
					seen[CategoryAlternative] = true
				}
				if requirement.NonShopping {
					seen[CategoryNonShopping] = true
				}
				if requirement.GroceryItem != nil && requirement.GroceryItem.ShoppingMode == "presence-only" {
					seen[CategoryPresenceOnly] = true
				}
			}
		}
	}
	var categories []Category
	for _, category := range Categories {
		if seen[category] {
			categories = append(categories, category)
		}
	}
	return categories
}

// EvalSet is the checked-in evaluation set.
type EvalSet struct {
	Cases []Case
}

// ReadEvalSet strictly reads a checked-in evaluation set: a directory holding exactly the
// four part directories (and an optional README.md), each holding case directories. Every
// case's site must be in the registry.
func ReadEvalSet(dir string, sites registry.Registry) (EvalSet, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return EvalSet{}, fmt.Errorf("read evaluation set: %w", err)
	}
	present := make(map[Part]bool)
	for _, entry := range entries {
		if !entry.IsDir() && entry.Name() == readmeFileName {
			continue
		}
		part := Part(entry.Name())
		if !entry.IsDir() || !slices.Contains(CheckedInParts, part) {
			return EvalSet{}, fmt.Errorf("evaluation set %s: unexpected entry %q", dir, entry.Name())
		}
		present[part] = true
	}
	var set EvalSet
	for _, part := range CheckedInParts {
		if !present[part] {
			return EvalSet{}, fmt.Errorf("evaluation set %s: missing part directory %q", dir, part)
		}
		cases, err := readPart(filepath.Join(dir, string(part)), part, sites)
		if err != nil {
			return EvalSet{}, err
		}
		set.Cases = append(set.Cases, cases...)
	}
	if err := checkUniqueIDs(set.Cases); err != nil {
		return EvalSet{}, err
	}
	return set, nil
}

// ReadHoldout strictly reads the operator's holdout: a directory of case directories (and an
// optional README.md) in the same case format, kept outside the repository.
func ReadHoldout(dir string, sites registry.Registry) ([]Case, error) {
	cases, err := readPart(dir, PartHoldout, sites)
	if err != nil {
		return nil, err
	}
	return cases, checkUniqueIDs(cases)
}

func readPart(dir string, part Part, sites registry.Registry) ([]Case, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read %s cases: %w", part, err)
	}
	var cases []Case
	for _, entry := range entries {
		if !entry.IsDir() && entry.Name() == readmeFileName {
			continue
		}
		if !entry.IsDir() || !caseIDPattern.MatchString(entry.Name()) {
			return nil, fmt.Errorf("%s: %q is not a case directory with a lowercase kebab-case ID", part, entry.Name())
		}
		parsed, err := ReadCase(filepath.Join(dir, entry.Name()), part, sites)
		if err != nil {
			return nil, err
		}
		cases = append(cases, parsed)
	}
	sort.Slice(cases, func(i, j int) bool { return cases[i].ID < cases[j].ID })
	return cases, nil
}

func checkUniqueIDs(cases []Case) error {
	seen := make(map[string]Part)
	for _, c := range cases {
		if previous, ok := seen[c.ID]; ok {
			return fmt.Errorf("case %q appears in both %s and %s", c.ID, previous, c.Part)
		}
		seen[c.ID] = c.Part
	}
	return nil
}

// ReadCase strictly reads one case directory. It must hold exactly source.json and
// expected.yaml.
func ReadCase(dir string, part Part, sites registry.Registry) (Case, error) {
	id := filepath.Base(dir)
	fail := func(err error) (Case, error) { return Case{}, fmt.Errorf("%s case %q: %w", part, id, err) }
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fail(err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	if !slices.Equal(names, []string{ExpectedFileName, SourceFileName}) {
		return fail(fmt.Errorf("case directory must contain exactly %s and %s, found %v", SourceFileName, ExpectedFileName, names))
	}
	sourceBytes, err := os.ReadFile(filepath.Join(dir, SourceFileName))
	if err != nil {
		return fail(err)
	}
	source, err := ParseSource(sourceBytes, sites)
	if err != nil {
		return fail(err)
	}
	expectedBytes, err := os.ReadFile(filepath.Join(dir, ExpectedFileName))
	if err != nil {
		return fail(err)
	}
	expected, err := ParseExpected(expectedBytes, part)
	if err != nil {
		return fail(err)
	}
	if document := expected.Document; document != nil && !slices.Equal(document.SourceLines, source.Ingredients) {
		return fail(errors.New("expected document source_lines must equal the node's recipeIngredient list exactly"))
	}
	return Case{ID: id, Part: part, Dir: dir, Source: source, Expected: expected}, nil
}

// ParseSource strictly parses source.json and checks it against the site registry.
func ParseSource(content []byte, sites registry.Registry) (Source, error) {
	var source Source
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&source); err != nil {
		return Source{}, fmt.Errorf("decode %s: %w", SourceFileName, err)
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return Source{}, fmt.Errorf("%s must contain exactly one JSON object", SourceFileName)
	}
	if source.FormatVersion != CaseFormatVersion {
		return Source{}, fmt.Errorf("%s format_version = %d, want %d", SourceFileName, source.FormatVersion, CaseFormatVersion)
	}
	site, ok := sites.Site(source.Site)
	if !ok {
		return Source{}, fmt.Errorf("site %q is not in the site registry", source.Site)
	}
	parsed, err := url.Parse(source.URL)
	if err != nil || parsed.Scheme != "https" || !site.HasHost(parsed.Host) {
		return Source{}, fmt.Errorf("url %q must be an https URL on one of site %s's hosts %v", source.URL, site.ID, site.Hosts)
	}
	if source.Note != "" && strings.TrimSpace(source.Note) == "" {
		return Source{}, errors.New("note cannot be blank")
	}

	var node struct {
		Type             any               `json:"@type"`
		RecipeIngredient []json.RawMessage `json:"recipeIngredient"`
	}
	if len(source.Node) == 0 || source.Node[0] != '{' {
		return Source{}, errors.New("node must be a JSON object")
	}
	if err := json.Unmarshal(source.Node, &node); err != nil {
		return Source{}, fmt.Errorf("decode node: %w", err)
	}
	if !isRecipeType(node.Type) {
		return Source{}, errors.New(`node @type must be "Recipe" or an array containing it`)
	}
	if len(node.RecipeIngredient) == 0 {
		return Source{}, errors.New("node needs a non-empty recipeIngredient array")
	}
	for index, raw := range node.RecipeIngredient {
		var line string
		if err := json.Unmarshal(raw, &line); err != nil {
			return Source{}, fmt.Errorf("node recipeIngredient[%d] is not a string", index)
		}
		source.Ingredients = append(source.Ingredients, line)
	}
	return source, nil
}

func isRecipeType(value any) bool {
	switch typed := value.(type) {
	case string:
		return typed == "Recipe"
	case []any:
		for _, item := range typed {
			if item == "Recipe" {
				return true
			}
		}
	}
	return false
}

// ParseExpected strictly parses expected.yaml for a case in the given part.
func ParseExpected(content []byte, part Part) (Expected, error) {
	var expected Expected
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)
	if err := decoder.Decode(&expected); err != nil {
		return Expected{}, fmt.Errorf("decode %s: %w", ExpectedFileName, err)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return Expected{}, fmt.Errorf("%s must contain exactly one YAML document", ExpectedFileName)
	}
	if err := expected.validate(part); err != nil {
		return Expected{}, fmt.Errorf("%s: %w", ExpectedFileName, err)
	}
	return expected, nil
}

func (e Expected) validate(part Part) error {
	if e.FormatVersion != CaseFormatVersion {
		return fmt.Errorf("format_version = %d, want %d", e.FormatVersion, CaseFormatVersion)
	}
	switch {
	case !e.Reviewed && strings.TrimSpace(e.DraftNote) == "":
		return errors.New("an unreviewed case needs a draft_note saying what to review")
	case e.Reviewed && e.DraftNote != "":
		return errors.New("a reviewed case cannot keep a draft_note; move lasting context to note")
	}
	for name, text := range map[string]string{"note": e.Note, "instructions_note": e.InstructionsNote} {
		if text != "" && strings.TrimSpace(text) == "" {
			return fmt.Errorf("%s cannot be blank", name)
		}
	}
	seen := make(map[Category]bool)
	for _, category := range e.Categories {
		if !slices.Contains(Categories, category) {
			return fmt.Errorf("unknown category %q", category)
		}
		if seen[category] {
			return fmt.Errorf("duplicate category %q", category)
		}
		seen[category] = true
	}

	switch e.Outcome {
	case OutcomePublish:
		return e.validatePublish(part)
	case OutcomeReject:
		return e.validateReject(part)
	default:
		return fmt.Errorf("outcome must be %q or %q, got %q", OutcomePublish, OutcomeReject, e.Outcome)
	}
}

func (e Expected) validatePublish(part Part) error {
	if part == PartExpectedReject {
		return errors.New("an expected-reject case must have outcome reject")
	}
	if e.Reject != nil {
		return errors.New("outcome publish cannot carry reject")
	}
	if e.InstructionsComplete == nil || !*e.InstructionsComplete {
		return errors.New("outcome publish needs instructions_complete: true; incomplete instructions mean the case expects rejection")
	}
	if e.Document == nil {
		return errors.New("outcome publish needs the expected document")
	}
	if issues := ingest.CheckPublishable(*e.Document); len(issues) > 0 {
		return fmt.Errorf("expected document is not publishable: %v", (&ingest.ValidationError{Issues: issues}).Error())
	}
	if len(e.Grocery) == 0 {
		return errors.New("outcome publish needs the expected grocery output")
	}
	output, err := GroceryOutput(*e.Document)
	if err != nil {
		return fmt.Errorf("aggregate expected document: %w", err)
	}
	if !reflect.DeepEqual(output, e.Grocery) {
		rendered, _ := yaml.Marshal(output)
		return fmt.Errorf("expected grocery output does not match the aggregation of the expected document; aggregation gives:\n%s", rendered)
	}
	return nil
}

func (e Expected) validateReject(part Part) error {
	if part == PartMustPass {
		return errors.New("a must-pass case must have outcome publish")
	}
	if e.Document != nil || e.Grocery != nil {
		return errors.New("outcome reject cannot carry an expected document or grocery output")
	}
	if e.Reject == nil || len(e.Reject.Codes) == 0 {
		return errors.New("outcome reject needs reject.codes")
	}
	seen := make(map[string]bool)
	for _, code := range e.Reject.Codes {
		if !codePattern.MatchString(code) {
			return fmt.Errorf("reject code %q is not a snake_case code", code)
		}
		if seen[code] {
			return fmt.Errorf("duplicate reject code %q", code)
		}
		seen[code] = true
	}
	return nil
}
