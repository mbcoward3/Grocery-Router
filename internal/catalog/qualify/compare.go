package qualify

import (
	"fmt"
	"math/big"
	"reflect"
	"slices"
	"strings"
	"unicode"

	"github.com/mbcoward3/grocery-router/internal/ingest"
)

// DifferenceKind names one kind of difference between a published candidate and the
// expected document.
type DifferenceKind string

// Material differences: they change what is bought or cooked (§6.2).
const (
	MissingRequirement   DifferenceKind = "missing_requirement"
	ExtraRequirement     DifferenceKind = "extra_requirement"
	WrongItem            DifferenceKind = "wrong_item"
	WrongShoppingMode    DifferenceKind = "wrong_shopping_mode"
	WrongAmount          DifferenceKind = "wrong_amount"
	WrongUnit            DifferenceKind = "wrong_unit"
	WrongRange           DifferenceKind = "wrong_range"
	WrongPackage         DifferenceKind = "wrong_package"
	WrongOptional        DifferenceKind = "wrong_optional"
	WrongUnspecified     DifferenceKind = "wrong_unspecified"
	WrongNonShopping     DifferenceKind = "wrong_non_shopping"
	WrongPreparation     DifferenceKind = "wrong_preparation"
	MissingStep          DifferenceKind = "missing_step"
	WrongStep            DifferenceKind = "wrong_step"
	WrongSourceLines     DifferenceKind = "wrong_source_lines"
	WrongGroceryOutput   DifferenceKind = "wrong_grocery_output"
	UnexpectedPublishing DifferenceKind = "unexpected_publication"
)

// Cosmetic differences: rewording or layout that preserves meaning.
const (
	CosmeticItemName        DifferenceKind = "item_name"
	CosmeticStoreSection    DifferenceKind = "store_section"
	CosmeticRelationship    DifferenceKind = "relationship"
	CosmeticSpan            DifferenceKind = "span"
	CosmeticNote            DifferenceKind = "note"
	CosmeticPreparationText DifferenceKind = "preparation_wording"
	CosmeticStepText        DifferenceKind = "step_wording"
	CosmeticSections        DifferenceKind = "section_layout"
	CosmeticMetadata        DifferenceKind = "metadata"
)

var materialKinds = map[DifferenceKind]bool{
	MissingRequirement: true, ExtraRequirement: true, WrongItem: true, WrongShoppingMode: true,
	WrongAmount: true, WrongUnit: true, WrongRange: true, WrongPackage: true, WrongOptional: true,
	WrongUnspecified: true, WrongNonShopping: true, WrongPreparation: true, MissingStep: true,
	WrongStep: true, WrongSourceLines: true, WrongGroceryOutput: true, UnexpectedPublishing: true,
}

// Material reports whether a difference kind is a material error.
func (k DifferenceKind) Material() bool { return materialKinds[k] }

// Difference is one difference between a published candidate and its expectation.
type Difference struct {
	Kind   DifferenceKind
	Path   string
	Detail string
}

// Material reports whether the difference is a material error.
func (d Difference) Material() bool { return d.Kind.Material() }

func (d Difference) String() string {
	text := string(d.Kind)
	if d.Path != "" {
		text = d.Path + ": " + text
	}
	if d.Detail != "" {
		text += ": " + d.Detail
	}
	return text
}

type located struct {
	path        string
	section     string
	requirement ingest.Requirement
}

// Compare lists every difference between a published candidate document and the expected
// document, including the expected grocery output. Requirements are matched per source line
// by grocery item, then in order.
func Compare(expected Expected, candidate ingest.CatalogDocument) []Difference {
	var differences []Difference
	add := func(kind DifferenceKind, path, format string, args ...any) {
		differences = append(differences, Difference{Kind: kind, Path: path, Detail: fmt.Sprintf(format, args...)})
	}
	want := *expected.Document

	if !slices.Equal(want.SourceLines, candidate.SourceLines) {
		add(WrongSourceLines, "source_lines", "candidate source lines differ from the source node's recipeIngredient list")
		return differences
	}
	compareMetadata(want, candidate, add)

	wantByLine := requirementsByLine(want)
	gotByLine := requirementsByLine(candidate)
	for line := range want.SourceLines {
		compareLine(fmt.Sprintf("source_lines[%d]", line), wantByLine[line], gotByLine[line], add)
	}
	compareSteps(want, candidate, add)

	materialSoFar := slices.ContainsFunc(differences, Difference.Material)
	output, err := GroceryOutput(candidate)
	switch {
	case err != nil:
		add(WrongGroceryOutput, "grocery", "candidate does not aggregate: %v", err)
	case !materialSoFar && !reflect.DeepEqual(output, expected.Grocery):
		add(WrongGroceryOutput, "grocery", "candidate grocery output differs from the expected output")
	}
	return differences
}

func requirementsByLine(document ingest.CatalogDocument) map[int][]located {
	result := make(map[int][]located)
	for sectionIndex, section := range document.IngredientSections {
		for requirementIndex, requirement := range section.Requirements {
			result[requirement.Provenance.Line] = append(result[requirement.Provenance.Line], located{
				path:        fmt.Sprintf("ingredient_sections[%d].requirements[%d]", sectionIndex, requirementIndex),
				section:     section.Name,
				requirement: requirement,
			})
		}
	}
	return result
}

func compareMetadata(want, got ingest.CatalogDocument, add func(DifferenceKind, string, string, ...any)) {
	fields := []struct {
		name      string
		want, got any
	}{
		{"name", want.Name, got.Name},
		{"source", want.Source, got.Source},
		{"image_url", want.ImageURL, got.ImageURL},
		{"yield", want.Yield, got.Yield},
		{"hands_on", want.HandsOn, got.HandsOn},
		{"unattended", want.Unattended, got.Unattended},
	}
	for _, field := range fields {
		if !reflect.DeepEqual(field.want, field.got) {
			add(CosmeticMetadata, field.name, "expected %v, got %v", deref(field.want), deref(field.got))
		}
	}
	headings := func(document ingest.CatalogDocument) []string {
		var names []string
		for _, section := range document.IngredientSections {
			names = append(names, section.Name)
		}
		return names
	}
	if !slices.Equal(headings(want), headings(got)) {
		add(CosmeticSections, "ingredient_sections", "expected sections %q, got %q", headings(want), headings(got))
	}
}

func deref(value any) any {
	if duration, ok := value.(*ingest.Duration); ok && duration != nil {
		return *duration
	}
	return value
}

func compareLine(path string, want, got []located, add func(DifferenceKind, string, string, ...any)) {
	usedWant := make([]bool, len(want))
	usedGot := make([]bool, len(got))
	type pair struct{ want, got located }
	var pairs []pair
	for wantIndex, expected := range want {
		for gotIndex, actual := range got {
			if !usedGot[gotIndex] && sameItem(expected.requirement, actual.requirement) {
				usedWant[wantIndex], usedGot[gotIndex] = true, true
				pairs = append(pairs, pair{expected, actual})
				break
			}
		}
	}
	gotIndex := 0
	for wantIndex, expected := range want {
		if usedWant[wantIndex] {
			continue
		}
		for gotIndex < len(got) && usedGot[gotIndex] {
			gotIndex++
		}
		if gotIndex < len(got) {
			usedGot[gotIndex] = true
			add(WrongItem, path, "expected %s, got %s", itemKey(expected.requirement), itemKey(got[gotIndex].requirement))
			continue
		}
		add(MissingRequirement, path, "expected %s %s is missing", itemKey(expected.requirement), describeQuantity(expected.requirement.Quantity))
	}
	for index, actual := range got {
		if !usedGot[index] {
			add(ExtraRequirement, path, "unexpected %s %s", itemKey(actual.requirement), describeQuantity(actual.requirement.Quantity))
		}
	}
	for _, matched := range pairs {
		compareRequirement(path+" "+itemKey(matched.want.requirement), matched.want, matched.got, add)
	}
}

func sameItem(left, right ingest.Requirement) bool {
	return left.GroceryItem != nil && right.GroceryItem != nil && left.GroceryItem.Key == right.GroceryItem.Key
}

func itemKey(requirement ingest.Requirement) string {
	if requirement.GroceryItem == nil {
		return "(" + string(requirement.State) + ")"
	}
	return requirement.GroceryItem.Key
}

func compareRequirement(path string, want, got located, add func(DifferenceKind, string, string, ...any)) {
	w, g := want.requirement, got.requirement
	if w.GroceryItem.Name != g.GroceryItem.Name {
		add(CosmeticItemName, path, "expected %q, got %q", w.GroceryItem.Name, g.GroceryItem.Name)
	}
	if w.GroceryItem.StoreSection != g.GroceryItem.StoreSection {
		add(CosmeticStoreSection, path, "expected %s, got %s", w.GroceryItem.StoreSection.Key, g.GroceryItem.StoreSection.Key)
	}
	if w.GroceryItem.ShoppingMode != g.GroceryItem.ShoppingMode {
		add(WrongShoppingMode, path, "expected %s, got %s", w.GroceryItem.ShoppingMode, g.GroceryItem.ShoppingMode)
	}
	compareQuantity(path, *w.Quantity, *g.Quantity, add)
	if w.Optional != g.Optional {
		add(WrongOptional, path, "expected optional=%v, got %v", w.Optional, g.Optional)
	}
	if w.NonShopping != g.NonShopping {
		add(WrongNonShopping, path, "expected non_shopping=%v, got %v", w.NonShopping, g.NonShopping)
	}
	switch {
	case normalizeText(w.Preparation) != normalizeText(g.Preparation):
		add(WrongPreparation, path, "expected %q, got %q", w.Preparation, g.Preparation)
	case w.Preparation != g.Preparation:
		add(CosmeticPreparationText, path, "expected %q, got %q", w.Preparation, g.Preparation)
	}
	if w.Note != g.Note {
		add(CosmeticNote, path, "expected %q, got %q", w.Note, g.Note)
	}
	if w.Relationship != g.Relationship {
		add(CosmeticRelationship, path, "expected %s, got %s", w.Relationship, g.Relationship)
	}
	if w.Provenance.Span != g.Provenance.Span {
		add(CosmeticSpan, path, "expected [%d,%d), got [%d,%d)", w.Provenance.Span.Start, w.Provenance.Span.End, g.Provenance.Span.Start, g.Provenance.Span.End)
	}
	if want.section != got.section {
		add(CosmeticSections, path, "expected section %q, got %q", want.section, got.section)
	}
}

func compareQuantity(path string, want, got ingest.Quantity, add func(DifferenceKind, string, string, ...any)) {
	if (want.Kind == "unspecified") != (got.Kind == "unspecified") {
		add(WrongUnspecified, path, "expected %s, got %s", describeQuantity(&want), describeQuantity(&got))
		return
	}
	if want.Kind == "unspecified" {
		return
	}
	wantMin, wantMax, wantSize, _ := want.Rationals()
	gotMin, gotMax, gotSize, _ := got.Rationals()
	if want.Kind != got.Kind || !equalRat(wantMax, gotMax) {
		add(WrongRange, path, "expected %s, got %s", describeQuantity(&want), describeQuantity(&got))
	}
	if !equalRat(wantMin, gotMin) {
		add(WrongAmount, path, "expected %s, got %s", describeQuantity(&want), describeQuantity(&got))
	}
	if want.Unit != got.Unit {
		add(WrongUnit, path, "expected unit %q, got %q", want.Unit, got.Unit)
	}
	samePackage := (want.Package == nil) == (got.Package == nil)
	if samePackage && want.Package != nil {
		samePackage = want.Package.Type == got.Package.Type && want.Package.Unit == got.Package.Unit && equalRat(wantSize, gotSize)
	}
	if !samePackage {
		add(WrongPackage, path, "expected %s, got %s", describeQuantity(&want), describeQuantity(&got))
	}
}

func equalRat(left, right *big.Rat) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Cmp(right) == 0
}

func describeQuantity(quantity *ingest.Quantity) string {
	if quantity == nil {
		return "(no quantity)"
	}
	if quantity.Kind == "unspecified" {
		return "unspecified"
	}
	amount := quantity.Amount
	if quantity.Kind == "range" {
		amount += "-" + quantity.Maximum
	}
	if quantity.Package != nil {
		pkg := quantity.Package.Type
		if quantity.Package.Size != "" {
			pkg = quantity.Package.Size + " " + quantity.Package.Unit + " " + pkg
		}
		return amount + " × " + pkg
	}
	return amount + " " + quantity.Unit
}

// compareSteps aligns the flattened instruction steps by longest common subsequence of
// normalized text. An unmatched expected step is missing; an unmatched candidate step is a
// wrong step. Matches that differ only in case, spacing, or punctuation are cosmetic.
func compareSteps(want, got ingest.CatalogDocument, add func(DifferenceKind, string, string, ...any)) {
	wantSteps, gotSteps := flattenSteps(want), flattenSteps(got)
	n, m := len(wantSteps), len(gotSteps)
	lengths := make([][]int, n+1)
	for i := range lengths {
		lengths[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if normalizeText(wantSteps[i]) == normalizeText(gotSteps[j]) {
				lengths[i][j] = lengths[i+1][j+1] + 1
			} else {
				lengths[i][j] = max(lengths[i+1][j], lengths[i][j+1])
			}
		}
	}
	i, j := 0, 0
	for i < n || j < m {
		switch {
		case i < n && j < m && normalizeText(wantSteps[i]) == normalizeText(gotSteps[j]):
			if wantSteps[i] != gotSteps[j] {
				add(CosmeticStepText, fmt.Sprintf("steps[%d]", i), "expected %q, got %q", wantSteps[i], gotSteps[j])
			}
			i++
			j++
		case j < m && (i == n || lengths[i][j+1] >= lengths[i+1][j]):
			add(WrongStep, fmt.Sprintf("candidate steps[%d]", j), "unexpected step %q", gotSteps[j])
			j++
		default:
			add(MissingStep, fmt.Sprintf("steps[%d]", i), "expected step %q is missing", wantSteps[i])
			i++
		}
	}
	sectionNames := func(document ingest.CatalogDocument) []string {
		var names []string
		for _, section := range document.InstructionSections {
			names = append(names, section.Name)
		}
		return names
	}
	if !slices.Equal(sectionNames(want), sectionNames(got)) {
		add(CosmeticSections, "instruction_sections", "expected sections %q, got %q", sectionNames(want), sectionNames(got))
	}
}

func flattenSteps(document ingest.CatalogDocument) []string {
	var steps []string
	for _, section := range document.InstructionSections {
		steps = append(steps, section.Steps...)
	}
	return steps
}

// normalizeText folds case and treats every run of characters other than letters and digits
// as one separator, so only rewording that changes words or numbers changes meaning.
func normalizeText(text string) string {
	words := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	return strings.Join(words, " ")
}
