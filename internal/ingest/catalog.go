package ingest

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

// CatalogFormatVersion is the catalog content format version this package reads and writes.
// Any change to the meaning of a field requires a new version.
const CatalogFormatVersion = 1

// CatalogDocument is one recipe in the versioned shared catalog content format: the shared
// recipe content plus, for every ingredient requirement, typed provenance, relationship, and
// interpretation state (CATALOG_INGEST_SPEC.md §5.1–§5.3).
//
// It deliberately has no key, status, or approved_on: a catalog key is assigned at
// publication and catalog trust is recorded as release provenance, never as `verified`.
// ParseCatalogDocument rejects those fields as unknown.
type CatalogDocument struct {
	FormatVersion int       `yaml:"format_version" json:"format_version"`
	Name          string    `yaml:"name" json:"name"`
	Source        Source    `yaml:"source" json:"source"`
	ImageURL      string    `yaml:"image_url,omitempty" json:"image_url,omitempty"`
	Yield         string    `yaml:"yield,omitempty" json:"yield,omitempty"`
	HandsOn       *Duration `yaml:"hands_on,omitempty" json:"hands_on,omitempty"`
	Unattended    *Duration `yaml:"unattended,omitempty" json:"unattended,omitempty"`
	// SourceLines are the source node's recipeIngredient strings, exactly as published, in
	// source order. A line's position is its zero-based index here.
	SourceLines         []string                   `yaml:"source_lines" json:"source_lines"`
	IngredientSections  []CatalogIngredientSection `yaml:"ingredient_sections" json:"ingredient_sections"`
	InstructionSections []InstructionSection       `yaml:"instruction_sections" json:"instruction_sections"`
}

// CatalogIngredientSection is an ordered group of requirements. Heading is set when a source
// line with the `heading` relationship (for example "For the sauce:") started the section.
type CatalogIngredientSection struct {
	Name         string        `yaml:"name" json:"name"`
	Heading      *Provenance   `yaml:"heading,omitempty" json:"heading,omitempty"`
	Requirements []Requirement `yaml:"requirements" json:"requirements"`
}

// Requirement is one structured need read from a source line. Its source text is the line its
// provenance names. Requirements in a publishable state (resolved, intentionally_unspecified)
// carry a grocery item and quantity. Requirements in an unpublishable state (unresolved,
// unsupported) carry a Reason instead and no interpretation.
type Requirement struct {
	Provenance   Provenance          `yaml:"provenance" json:"provenance"`
	Relationship Relationship        `yaml:"relationship,omitempty" json:"relationship,omitempty"`
	State        InterpretationState `yaml:"state" json:"state"`
	// Reason is a snake_case code explaining an unresolved or unsupported requirement, for
	// example "incoherent_alternative".
	Reason      string       `yaml:"reason,omitempty" json:"reason,omitempty"`
	GroceryItem *GroceryItem `yaml:"grocery_item,omitempty" json:"grocery_item,omitempty"`
	Quantity    *Quantity    `yaml:"quantity,omitempty" json:"quantity,omitempty"`
	Preparation string       `yaml:"preparation,omitempty" json:"preparation,omitempty"`
	Optional    bool         `yaml:"optional,omitempty" json:"optional,omitempty"`
	NonShopping bool         `yaml:"non_shopping,omitempty" json:"non_shopping,omitempty"`
	Note        string       `yaml:"note,omitempty" json:"note,omitempty"`
}

// Provenance says where a requirement or heading came from.
type Provenance struct {
	Type ProvenanceType `yaml:"type" json:"type"`
	// Line is the zero-based position of the source line in CatalogDocument.SourceLines.
	Line int `yaml:"line" json:"line"`
	// Span is the part of the source line the requirement or heading was read from.
	Span Span `yaml:"span" json:"span"`
}

// Span is a non-empty, half-open [Start, End) range of Unicode code points within one source
// line. Code points, not bytes, so "½ cup" spans are stable across encodings.
type Span struct {
	Start int `yaml:"start" json:"start"`
	End   int `yaml:"end" json:"end"`
}

// ProvenanceType names what kind of source text produced a requirement.
type ProvenanceType string

const (
	// ProvenanceIngredientLine is one recipeIngredient source line: the only requirement
	// source in this phase.
	ProvenanceIngredientLine ProvenanceType = "ingredient_line"
	// ProvenanceInstructionStep is reserved. Instruction text never creates requirements in
	// this phase, so validation rejects it.
	ProvenanceInstructionStep ProvenanceType = "instruction_step"
)

// Relationship says how one source line divides into requirements (spec §5.1).
type Relationship string

const (
	// RelationshipSingle means one requirement ("3 lb boneless beef chuck").
	RelationshipSingle Relationship = "single"
	// RelationshipAdditive means two or more requirements for the same item
	// ("½ cup walnuts (plus 2 tablespoons for topping)").
	RelationshipAdditive Relationship = "additive"
	// RelationshipCompound means one requirement per distinct item ("1 tsp each thyme and rosemary").
	RelationshipCompound Relationship = "compound"
	// RelationshipAlternative means one requirement, the first option, only if coherent
	// ("2 chicken breasts or 4 thighs").
	RelationshipAlternative Relationship = "alternative"
	// RelationshipEquivalent means one requirement; the other measure is a note ("2 tbsp / 30g butter").
	RelationshipEquivalent Relationship = "equivalent"
	// RelationshipDivided means one requirement for the total ("1 cup sugar, divided").
	RelationshipDivided Relationship = "divided"
	// RelationshipHeading means zero requirements; the line starts an ingredient section
	// ("For the sauce:"). It is recorded as CatalogIngredientSection.Heading, never on a
	// requirement.
	RelationshipHeading Relationship = "heading"
)

// Relationships lists every relationship in specification order.
var Relationships = []Relationship{
	RelationshipSingle, RelationshipAdditive, RelationshipCompound, RelationshipAlternative,
	RelationshipEquivalent, RelationshipDivided, RelationshipHeading,
}

// Valid reports whether r is a known relationship.
func (r Relationship) Valid() bool {
	for _, known := range Relationships {
		if r == known {
			return true
		}
	}
	return false
}

// InterpretationState says how fully a requirement was interpreted (spec §5.3).
type InterpretationState string

const (
	// StateResolved means the requirement is fully structured with an exact or range quantity. Publishable.
	StateResolved InterpretationState = "resolved"
	// StateIntentionallyUnspecified means the source explicitly gives no amount ("to taste",
	// "plus more for greasing"); quantity kind is unspecified. Publishable.
	StateIntentionallyUnspecified InterpretationState = "intentionally_unspecified"
	// StateUnresolved means the requirement could not be determined. Not publishable.
	StateUnresolved InterpretationState = "unresolved"
	// StateUnsupported means the requirement is understood but not representable. Not publishable.
	StateUnsupported InterpretationState = "unsupported"
)

// InterpretationStates lists every interpretation state in specification order.
var InterpretationStates = []InterpretationState{
	StateResolved, StateIntentionallyUnspecified, StateUnresolved, StateUnsupported,
}

// Valid reports whether s is a known interpretation state.
func (s InterpretationState) Valid() bool {
	for _, known := range InterpretationStates {
		if s == known {
			return true
		}
	}
	return false
}

// Publishable reports whether a requirement in state s may be published.
func (s InterpretationState) Publishable() bool {
	return s == StateResolved || s == StateIntentionallyUnspecified
}

// IssueCode is a stable machine-readable reason a catalog document is invalid or not
// publishable. Codes are part of the format contract: add new ones, never repurpose them.
type IssueCode string

// Catalog validation issue codes.
const (
	IssueUnsupportedFormatVersion IssueCode = "unsupported_format_version"
	// IssueInvalidContent is a failure of the shared recipe content validation.
	IssueInvalidContent            IssueCode = "invalid_content"
	IssueBlankSourceLine           IssueCode = "blank_source_line"
	IssueUnknownProvenanceType     IssueCode = "unknown_provenance_type"
	IssueReservedProvenanceType    IssueCode = "reserved_provenance_type"
	IssueSourceLineOutOfRange      IssueCode = "source_line_out_of_range"
	IssueSpanOutOfBounds           IssueCode = "span_out_of_bounds"
	IssueUnknownRelationship       IssueCode = "unknown_relationship"
	IssueHeadingRequirement        IssueCode = "heading_requirement"
	IssueMissingRelationship       IssueCode = "missing_relationship"
	IssueRelationshipConflict      IssueCode = "relationship_conflict"
	IssueRelationshipCardinality   IssueCode = "relationship_cardinality"
	IssueUnknownState              IssueCode = "unknown_state"
	IssueStateQuantityMismatch     IssueCode = "state_quantity_mismatch"
	IssueMissingInterpretation     IssueCode = "missing_interpretation"
	IssueUninterpretedContent      IssueCode = "uninterpreted_requirement_content"
	IssueMissingReason             IssueCode = "missing_reason"
	IssueUnexpectedReason          IssueCode = "unexpected_reason"
	IssueInvalidReason             IssueCode = "invalid_reason"
	IssueHeadingLineHasRequirement IssueCode = "heading_line_has_requirements"
	IssueDuplicateHeading          IssueCode = "duplicate_heading"
	IssueSourceOrder               IssueCode = "source_order"
)

// Publication predicate issue codes (spec §6.4 clauses 3, 4, and 5). Clause 1 and the
// provenance part of clause 5 are reported with the validation codes above.
const (
	// IssueNoRequirements means clause 3 fails: the document has no requirement.
	IssueNoRequirements IssueCode = "no_requirements"
	// IssueNoInstructionSteps means clause 3 fails: the document has no instruction step.
	IssueNoInstructionSteps IssueCode = "no_instruction_steps"
	// IssueUndisposedSourceLine means clause 4 fails: a non-heading source line yields no requirement.
	IssueUndisposedSourceLine IssueCode = "undisposed_source_line"
	// IssueUnpublishableSourceLine means clause 4 fails: a source line yields requirements but none in
	// a publishable state.
	IssueUnpublishableSourceLine IssueCode = "unpublishable_source_line"
	// IssueUnpublishableState means clause 5 fails: a requirement is unresolved or unsupported.
	IssueUnpublishableState IssueCode = "unpublishable_state"
)

// Issue is one machine-readable problem with a catalog document.
type Issue struct {
	Code IssueCode `json:"code"`
	// Path locates the problem, for example "ingredient_sections[0].requirements[1]".
	Path string `json:"path"`
	// Detail is human-readable context. It is not part of the stable contract.
	Detail string `json:"detail,omitempty"`
}

func (i Issue) String() string {
	text := string(i.Code)
	if i.Path != "" {
		text = i.Path + ": " + text
	}
	if i.Detail != "" {
		text += ": " + i.Detail
	}
	return text
}

// ValidationError reports every issue that makes a catalog document invalid.
type ValidationError struct {
	Issues []Issue
}

func (e *ValidationError) Error() string {
	parts := make([]string, len(e.Issues))
	for index, issue := range e.Issues {
		parts[index] = issue.String()
	}
	return "invalid catalog document: " + strings.Join(parts, "; ")
}

var reasonPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`)

// ParseCatalogDocument strictly decodes one catalog document from YAML (or JSON, which YAML
// accepts) and validates it. Unknown fields, including bootstrap approval fields such as
// status and approved_on, are rejected.
func ParseCatalogDocument(input io.Reader) (CatalogDocument, error) {
	contents, err := io.ReadAll(input)
	if err != nil {
		return CatalogDocument{}, fmt.Errorf("read catalog document: %w", err)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(contents))
	decoder.KnownFields(true)
	var document CatalogDocument
	if err := decoder.Decode(&document); err != nil {
		return CatalogDocument{}, fmt.Errorf("decode catalog document: %w", err)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return CatalogDocument{}, errors.New("catalog document must contain exactly one YAML document")
	}
	if err := document.Validate(); err != nil {
		return CatalogDocument{}, err
	}
	return document, nil
}

// Validate is catalog validation: the shared recipe content validation plus the catalog
// rules (known format version and enums, valid ingredient_line provenance with spans inside
// their source line, state-consistent interpretation, relationship cardinality, and source
// order). It returns a *ValidationError listing every issue, or nil.
//
// A valid document may still be unpublishable; see CheckPublishable.
func (d CatalogDocument) Validate() error {
	if issues := d.validationIssues(); len(issues) > 0 {
		return &ValidationError{Issues: issues}
	}
	return nil
}

// Ingredient returns the requirement as a shared-content ingredient read from sourceText, or
// false when the requirement carries no interpretation (it is unresolved or unsupported).
func (r Requirement) Ingredient(sourceText string) (Ingredient, bool) {
	if r.GroceryItem == nil || r.Quantity == nil {
		return Ingredient{}, false
	}
	return Ingredient{
		SourceText:  sourceText,
		GroceryItem: *r.GroceryItem,
		Quantity:    *r.Quantity,
		Preparation: r.Preparation,
		Optional:    r.Optional,
		NonShopping: r.NonShopping,
		Note:        r.Note,
	}, true
}

// SourceText returns the source line a requirement or heading's provenance names, or false
// when the provenance does not name a line of this document.
func (d CatalogDocument) SourceText(provenance Provenance) (string, bool) {
	if provenance.Type != ProvenanceIngredientLine || provenance.Line < 0 || provenance.Line >= len(d.SourceLines) {
		return "", false
	}
	return d.SourceLines[provenance.Line], true
}

type lineUse struct {
	section      int
	requirements []Requirement
	paths        []string
	heading      string // path of the section heading that names this line, if any
}

func (d CatalogDocument) validationIssues() []Issue {
	var issues []Issue
	add := func(code IssueCode, path, format string, args ...any) {
		issues = append(issues, Issue{Code: code, Path: path, Detail: fmt.Sprintf(format, args...)})
	}

	if d.FormatVersion != CatalogFormatVersion {
		add(IssueUnsupportedFormatVersion, "format_version", "format_version = %d, want %d", d.FormatVersion, CatalogFormatVersion)
	}
	if err := validateRecipeMetadata(d.Name, d.Source, d.ImageURL, d.Yield, d.HandsOn, d.Unattended); err != nil {
		add(IssueInvalidContent, "", "%v", err)
	}
	for index, line := range d.SourceLines {
		if strings.TrimSpace(line) == "" {
			add(IssueBlankSourceLine, fmt.Sprintf("source_lines[%d]", index), "source line is blank")
		}
	}

	uses := make(map[int]*lineUse)
	use := func(line, section int) *lineUse {
		if uses[line] == nil {
			uses[line] = &lineUse{section: section}
		}
		return uses[line]
	}
	lastLine := -1
	order := func(path string, provenance Provenance, strictlyAfter bool) {
		if provenance.Line < lastLine || (strictlyAfter && provenance.Line == lastLine) {
			add(IssueSourceOrder, path, "source line %d appears after source line %d", provenance.Line, lastLine)
		}
		lastLine = max(lastLine, provenance.Line)
	}

	if len(d.IngredientSections) == 0 {
		add(IssueInvalidContent, "ingredient_sections", "at least one ingredient section is required")
	}
	items := make(groceryItemSet)
	for sectionIndex, section := range d.IngredientSections {
		sectionPath := fmt.Sprintf("ingredient_sections[%d]", sectionIndex)
		if err := validateIngredientSectionShape(sectionIndex, section.Name, len(section.Requirements)); err != nil {
			add(IssueInvalidContent, sectionPath, "%v", err)
		}
		if section.Heading != nil {
			headingPath := sectionPath + ".heading"
			if d.validateProvenance(headingPath, *section.Heading, add) {
				lineUse := use(section.Heading.Line, sectionIndex)
				if lineUse.heading != "" {
					add(IssueDuplicateHeading, headingPath, "source line %d already heads %s", section.Heading.Line, lineUse.heading)
				}
				lineUse.heading = headingPath
				order(headingPath, *section.Heading, true)
			}
		}
		for requirementIndex, requirement := range section.Requirements {
			path := fmt.Sprintf("%s.requirements[%d]", sectionPath, requirementIndex)
			provenanceValid := d.validateProvenance(path, requirement.Provenance, add)
			d.validateRequirement(path, requirement, provenanceValid, items, add)
			if provenanceValid {
				lineUse := use(requirement.Provenance.Line, sectionIndex)
				if lineUse.section != sectionIndex {
					add(IssueSourceOrder, path, "source line %d is split across ingredient sections", requirement.Provenance.Line)
				}
				lineUse.requirements = append(lineUse.requirements, requirement)
				lineUse.paths = append(lineUse.paths, path)
				order(path, requirement.Provenance, false)
			}
		}
	}

	for line := range d.SourceLines {
		lineUse := uses[line]
		if lineUse == nil {
			continue
		}
		linePath := fmt.Sprintf("source_lines[%d]", line)
		if lineUse.heading != "" && len(lineUse.requirements) > 0 {
			add(IssueHeadingLineHasRequirement, linePath, "heading line also yields %d requirement(s)", len(lineUse.requirements))
		}
		validateLineRelationship(linePath, lineUse.requirements, add)
	}

	if err := validateInstructionSections(d.InstructionSections); err != nil {
		add(IssueInvalidContent, "instruction_sections", "%v", err)
	}
	return issues
}

// validateProvenance reports provenance problems and whether the provenance names a valid
// span of one of the document's source lines.
func (d CatalogDocument) validateProvenance(path string, provenance Provenance, add func(IssueCode, string, string, ...any)) bool {
	path += ".provenance"
	switch provenance.Type {
	case ProvenanceIngredientLine:
	case ProvenanceInstructionStep:
		add(IssueReservedProvenanceType, path, "instruction_step provenance is reserved; instruction text never creates requirements")
		return false
	default:
		add(IssueUnknownProvenanceType, path, "unknown provenance type %q", provenance.Type)
		return false
	}
	text, ok := d.SourceText(provenance)
	if !ok {
		add(IssueSourceLineOutOfRange, path, "line %d is not one of %d source lines", provenance.Line, len(d.SourceLines))
		return false
	}
	length := utf8.RuneCountInString(text)
	if provenance.Span.Start < 0 || provenance.Span.Start >= provenance.Span.End || provenance.Span.End > length {
		add(IssueSpanOutOfBounds, path, "span [%d,%d) is not a non-empty range within %d code points", provenance.Span.Start, provenance.Span.End, length)
		return false
	}
	return true
}

func (d CatalogDocument) validateRequirement(path string, requirement Requirement, provenanceValid bool, items groceryItemSet, add func(IssueCode, string, string, ...any)) {
	switch {
	case requirement.Relationship == RelationshipHeading:
		add(IssueHeadingRequirement, path, "a heading yields zero requirements; record it as the section heading")
	case requirement.Relationship != "" && !requirement.Relationship.Valid():
		add(IssueUnknownRelationship, path, "unknown relationship %q", requirement.Relationship)
	}
	if !requirement.State.Valid() {
		add(IssueUnknownState, path, "unknown interpretation state %q", requirement.State)
		return
	}

	if requirement.State.Publishable() {
		if requirement.Relationship == "" {
			add(IssueMissingRelationship, path, "%s requirement needs a relationship", requirement.State)
		}
		if requirement.Reason != "" {
			add(IssueUnexpectedReason, path, "%s requirement cannot carry reason %q", requirement.State, requirement.Reason)
		}
		if requirement.GroceryItem == nil || requirement.Quantity == nil {
			add(IssueMissingInterpretation, path, "%s requirement needs a grocery item and quantity", requirement.State)
			return
		}
		wantUnspecified := requirement.State == StateIntentionallyUnspecified
		if (requirement.Quantity.Kind == "unspecified") != wantUnspecified {
			add(IssueStateQuantityMismatch, path, "%s requirement cannot have quantity kind %q", requirement.State, requirement.Quantity.Kind)
		}
		if !provenanceValid {
			return
		}
		sourceText, _ := d.SourceText(requirement.Provenance)
		ingredient, _ := requirement.Ingredient(sourceText)
		if err := validateIngredient(path, ingredient, items); err != nil {
			add(IssueInvalidContent, path, "%v", err)
		}
		return
	}

	switch {
	case requirement.Reason == "":
		add(IssueMissingReason, path, "%s requirement needs a reason", requirement.State)
	case !reasonPattern.MatchString(requirement.Reason):
		add(IssueInvalidReason, path, "reason %q is not a snake_case code", requirement.Reason)
	}
	if requirement.GroceryItem != nil || requirement.Quantity != nil || requirement.Preparation != "" ||
		requirement.Optional || requirement.NonShopping || requirement.Note != "" {
		add(IssueUninterpretedContent, path, "%s requirement cannot carry an interpretation", requirement.State)
	}
}

// validateLineRelationship checks that every requirement of one source line agrees on the
// relationship and, once every requirement on the line is interpreted, that the requirement
// count and items fit it.
func validateLineRelationship(path string, requirements []Requirement, add func(IssueCode, string, string, ...any)) {
	var relationship Relationship
	interpreted := true
	for _, requirement := range requirements {
		if requirement.Relationship != "" {
			if relationship != "" && requirement.Relationship != relationship {
				add(IssueRelationshipConflict, path, "requirements disagree: %q and %q", relationship, requirement.Relationship)
				return
			}
			relationship = requirement.Relationship
		}
		if !requirement.State.Publishable() || requirement.GroceryItem == nil {
			interpreted = false
		}
	}
	if !interpreted || !relationship.Valid() {
		return
	}
	count := len(requirements)
	items := make(map[string]bool, count)
	for _, requirement := range requirements {
		items[requirement.GroceryItem.Key] = true
	}
	switch relationship {
	case RelationshipSingle, RelationshipAlternative, RelationshipEquivalent, RelationshipDivided:
		if count != 1 {
			add(IssueRelationshipCardinality, path, "%s line yields %d requirements, want 1", relationship, count)
		}
	case RelationshipAdditive:
		if count < 2 || len(items) != 1 {
			add(IssueRelationshipCardinality, path, "additive line needs two or more requirements for one item, got %d for %d item(s)", count, len(items))
		}
	case RelationshipCompound:
		if count < 2 || len(items) != count {
			add(IssueRelationshipCardinality, path, "compound line needs one requirement per distinct item, got %d for %d item(s)", count, len(items))
		}
	}
}

// CheckPublishable evaluates spec §6.4 clauses 1, 3, 4, and 5 of the publication predicate and
// returns every failing reason; an empty result means those clauses hold. Clauses 2 and 6–9
// (language, alternative/note/instruction checks, vocabulary snapshot, site state, and
// manifest) belong to later slices and are not evaluated here.
func CheckPublishable(d CatalogDocument) []Issue {
	// Clause 1, and the provenance part of clause 5.
	issues := d.validationIssues()
	add := func(code IssueCode, path, format string, args ...any) {
		issues = append(issues, Issue{Code: code, Path: path, Detail: fmt.Sprintf(format, args...)})
	}

	// Clause 3.
	requirementCount, stepCount := 0, 0
	for _, section := range d.IngredientSections {
		requirementCount += len(section.Requirements)
	}
	for _, section := range d.InstructionSections {
		for _, step := range section.Steps {
			if strings.TrimSpace(step) != "" {
				stepCount++
			}
		}
	}
	if requirementCount == 0 {
		add(IssueNoRequirements, "ingredient_sections", "recipe has no requirements")
	}
	if stepCount == 0 {
		add(IssueNoInstructionSteps, "instruction_sections", "recipe has no instruction steps")
	}

	// Clauses 4 and 5: every line has a publishable disposition and every requirement has a
	// publishable state. Only provenance that names a valid span counts as a disposition.
	headings := make(map[int]bool)
	yielded := make(map[int]int)
	publishable := make(map[int]int)
	for sectionIndex, section := range d.IngredientSections {
		if section.Heading != nil && d.validSpan(*section.Heading) {
			headings[section.Heading.Line] = true
		}
		for requirementIndex, requirement := range section.Requirements {
			path := fmt.Sprintf("ingredient_sections[%d].requirements[%d]", sectionIndex, requirementIndex)
			if !requirement.State.Publishable() {
				detail := string(requirement.State)
				if requirement.Reason != "" {
					detail += ": " + requirement.Reason
				}
				add(IssueUnpublishableState, path, "%s", detail)
			}
			if !d.validSpan(requirement.Provenance) {
				continue
			}
			yielded[requirement.Provenance.Line]++
			if requirement.State.Publishable() {
				publishable[requirement.Provenance.Line]++
			}
		}
	}
	for line := range d.SourceLines {
		path := fmt.Sprintf("source_lines[%d]", line)
		switch {
		case headings[line] && yielded[line] == 0:
		case yielded[line] == 0:
			add(IssueUndisposedSourceLine, path, "source line yields no requirement and is not a heading")
		case publishable[line] == 0:
			add(IssueUnpublishableSourceLine, path, "source line yields no resolved or intentionally_unspecified requirement")
		}
	}
	return issues
}

func (d CatalogDocument) validSpan(provenance Provenance) bool {
	text, ok := d.SourceText(provenance)
	return ok && provenance.Span.Start >= 0 && provenance.Span.Start < provenance.Span.End &&
		provenance.Span.End <= utf8.RuneCountInString(text)
}
