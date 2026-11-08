package ingest_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/mbcoward3/grocery-router/internal/ingest"
	"gopkg.in/yaml.v3"
)

const catalogFixture = "every-relationship.yaml"

func loadCatalog(t *testing.T) ingest.CatalogDocument {
	t.Helper()
	file, err := os.Open(filepath.Join("testdata", "catalog", catalogFixture))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	document, err := ingest.ParseCatalogDocument(file)
	if err != nil {
		t.Fatal(err)
	}
	return document
}

// Fixture requirement positions.
func walnutBase(d *ingest.CatalogDocument) *ingest.Requirement    { return req(d, 0, 0) }
func walnutTopping(d *ingest.CatalogDocument) *ingest.Requirement { return req(d, 0, 1) }
func butterMore(d *ingest.CatalogDocument) *ingest.Requirement    { return req(d, 0, 3) }
func thyme(d *ingest.CatalogDocument) *ingest.Requirement         { return req(d, 1, 0) }
func rosemary(d *ingest.CatalogDocument) *ingest.Requirement      { return req(d, 1, 1) }
func chicken(d *ingest.CatalogDocument) *ingest.Requirement       { return req(d, 1, 2) }
func beef(d *ingest.CatalogDocument) *ingest.Requirement          { return req(d, 1, 5) }
func salt(d *ingest.CatalogDocument) *ingest.Requirement          { return req(d, 1, 6) }

func req(d *ingest.CatalogDocument, section, index int) *ingest.Requirement {
	return &d.IngredientSections[section].Requirements[index]
}

func validationCodes(t *testing.T, err error) []ingest.IssueCode {
	t.Helper()
	if err == nil {
		return nil
	}
	var validation *ingest.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("error %v is not a *ValidationError", err)
	}
	return issueCodes(validation.Issues)
}

func issueCodes(issues []ingest.Issue) []ingest.IssueCode {
	codes := make([]ingest.IssueCode, 0, len(issues))
	for _, issue := range issues {
		codes = append(codes, issue.Code)
	}
	return codes
}

func TestCatalogFixtureIsValidAndPublishable(t *testing.T) {
	document := loadCatalog(t)
	if document.FormatVersion != ingest.CatalogFormatVersion {
		t.Fatalf("format_version = %d", document.FormatVersion)
	}
	if issues := ingest.CheckPublishable(document); len(issues) != 0 {
		t.Fatalf("CheckPublishable() = %v, want publishable", issues)
	}

	// "½ cup walnuts (plus 2 tablespoons for topping)" is two additive requirements for the
	// same item, each with its own span in the original line.
	base, topping := walnutBase(&document), walnutTopping(&document)
	line := document.SourceLines[base.Provenance.Line]
	runes := []rune(line)
	if got := string(runes[base.Provenance.Span.Start:base.Provenance.Span.End]); got != "½ cup walnuts" {
		t.Fatalf("base span text = %q", got)
	}
	if got := string(runes[topping.Provenance.Span.Start:topping.Provenance.Span.End]); got != "plus 2 tablespoons for topping" {
		t.Fatalf("topping span text = %q", got)
	}
	if base.Relationship != ingest.RelationshipAdditive || topping.Relationship != ingest.RelationshipAdditive ||
		base.GroceryItem.Key != "walnuts" || topping.GroceryItem.Key != "walnuts" ||
		base.Quantity.Amount != "1/2" || topping.Quantity.Amount != "2" || topping.Quantity.Unit != "tbsp" {
		t.Fatalf("walnut requirements = %#v, %#v", base, topping)
	}
	ingredient, ok := topping.Ingredient(line)
	if !ok || ingredient.SourceText != line || ingredient.GroceryItem.Key != "walnuts" {
		t.Fatalf("Ingredient() = %#v, %v", ingredient, ok)
	}

	// "plus more" without an amount is a second, intentionally unspecified requirement.
	more := butterMore(&document)
	if more.State != ingest.StateIntentionallyUnspecified || more.Quantity.Kind != "unspecified" || more.GroceryItem.Key != "butter" {
		t.Fatalf("butter plus more = %#v", more)
	}
}

func TestCatalogDocumentRoundTripsThroughYAMLAndJSON(t *testing.T) {
	document := loadCatalog(t)

	encoded, err := yaml.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	fromYAML, err := ingest.ParseCatalogDocument(bytes.NewReader(encoded))
	if err != nil {
		t.Fatalf("parse marshaled YAML: %v", err)
	}
	if !reflect.DeepEqual(fromYAML, document) {
		t.Fatal("YAML round trip changed the document")
	}

	encoded, err = json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	fromJSON, err := ingest.ParseCatalogDocument(bytes.NewReader(encoded))
	if err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	if !reflect.DeepEqual(fromJSON, document) {
		t.Fatal("JSON round trip changed the document")
	}
}

func TestParseCatalogDocumentIsStrict(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("testdata", "catalog", catalogFixture))
	if err != nil {
		t.Fatal(err)
	}
	fixture := strings.ReplaceAll(string(contents), "\r\n", "\n")
	cases := []struct {
		name  string
		input string
		want  string
	}{
		// Catalog trust is release provenance, never bootstrap approval (invariant 3).
		{"bootstrap status", strings.Replace(fixture, "format_version: 1", "format_version: 1\nstatus: verified", 1), "field status not found"},
		{"bootstrap approved_on", strings.Replace(fixture, "format_version: 1", "format_version: 1\napproved_on: 2026-11-08", 1), "field approved_on not found"},
		{"bootstrap key", strings.Replace(fixture, "format_version: 1", "format_version: 1\nkey: example", 1), "field key not found"},
		{"unknown requirement field", strings.Replace(fixture, "state: resolved", "state: resolved\n        confidence: high", 1), "field confidence not found"},
		{"second YAML document", fixture + "\n---\nname: extra\n", "exactly one YAML document"},
		{"format version", strings.Replace(fixture, "format_version: 1", "format_version: 2", 1), string(ingest.IssueUnsupportedFormatVersion)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ingest.ParseCatalogDocument(strings.NewReader(tc.input))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ParseCatalogDocument() = %v, want error containing %q", err, tc.want)
			}
		})
	}
}

func TestCatalogRelationships(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ingest.CatalogDocument)
		want   ingest.IssueCode // empty means valid
	}{
		{"single", func(*ingest.CatalogDocument) {}, ""},
		{"additive", func(*ingest.CatalogDocument) {}, ""},
		{"compound", func(*ingest.CatalogDocument) {}, ""},
		{"alternative", func(*ingest.CatalogDocument) {}, ""},
		{"equivalent", func(*ingest.CatalogDocument) {}, ""},
		{"divided", func(*ingest.CatalogDocument) {}, ""},
		{"heading section", func(*ingest.CatalogDocument) {}, ""},
		{"heading on a requirement", func(d *ingest.CatalogDocument) {
			beef(d).Relationship = ingest.RelationshipHeading
		}, ingest.IssueHeadingRequirement},
		{"unknown relationship", func(d *ingest.CatalogDocument) {
			beef(d).Relationship = "either"
		}, ingest.IssueUnknownRelationship},
		{"missing relationship on resolved", func(d *ingest.CatalogDocument) {
			beef(d).Relationship = ""
		}, ingest.IssueMissingRelationship},
		{"requirements of one line disagree", func(d *ingest.CatalogDocument) {
			walnutTopping(d).Relationship = ingest.RelationshipCompound
		}, ingest.IssueRelationshipConflict},
		{"additive with one requirement", func(d *ingest.CatalogDocument) {
			d.IngredientSections[0].Requirements = slices.Delete(d.IngredientSections[0].Requirements, 1, 2)
		}, ingest.IssueRelationshipCardinality},
		{"additive with two items", func(d *ingest.CatalogDocument) {
			item := *walnutTopping(d).GroceryItem
			item.Key, item.Name = "pecans", "Pecans"
			walnutTopping(d).GroceryItem = &item
		}, ingest.IssueRelationshipCardinality},
		{"compound with one item twice", func(d *ingest.CatalogDocument) {
			rosemary(d).GroceryItem = thyme(d).GroceryItem
		}, ingest.IssueRelationshipCardinality},
		{"compound with one requirement", func(d *ingest.CatalogDocument) {
			d.IngredientSections[1].Requirements = slices.Delete(d.IngredientSections[1].Requirements, 1, 2)
		}, ingest.IssueRelationshipCardinality},
		{"single with two requirements", func(d *ingest.CatalogDocument) {
			for _, r := range []*ingest.Requirement{thyme(d), rosemary(d)} {
				r.Relationship = ingest.RelationshipSingle
			}
		}, ingest.IssueRelationshipCardinality},
		{"alternative keeps only the first option", func(d *ingest.CatalogDocument) {
			second := *chicken(d)
			second.Provenance.Span = ingest.Span{Start: 21, End: 29}
			thighs := *second.GroceryItem
			thighs.Key, thighs.Name = "chicken-thighs", "Chicken Thighs"
			second.GroceryItem = &thighs
			d.IngredientSections[1].Requirements = slices.Insert(d.IngredientSections[1].Requirements, 3, second)
		}, ingest.IssueRelationshipCardinality},
		{"divided with two requirements", func(d *ingest.CatalogDocument) {
			sugar := req(d, 1, 4)
			d.IngredientSections[1].Requirements = slices.Insert(d.IngredientSections[1].Requirements, 5, *sugar)
		}, ingest.IssueRelationshipCardinality},
		{"heading line also yields a requirement", func(d *ingest.CatalogDocument) {
			first := *walnutBase(d)
			first.Provenance = ingest.Provenance{Type: ingest.ProvenanceIngredientLine, Line: 0, Span: ingest.Span{Start: 0, End: 3}}
			first.Relationship = ingest.RelationshipSingle
			d.IngredientSections[0].Requirements = slices.Insert(d.IngredientSections[0].Requirements, 0, first)
		}, ingest.IssueHeadingLineHasRequirement},
		{"same heading twice", func(d *ingest.CatalogDocument) {
			heading := *d.IngredientSections[0].Heading
			d.IngredientSections[1].Heading = &heading
		}, ingest.IssueDuplicateHeading},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			document := loadCatalog(t)
			tc.mutate(&document)
			assertValidation(t, document, tc.want)
		})
	}
	for _, relationship := range ingest.Relationships {
		if !relationship.Valid() {
			t.Errorf("%q is not valid", relationship)
		}
	}
}

func TestCatalogInterpretationStates(t *testing.T) {
	unresolved := func(state ingest.InterpretationState, reason string) func(*ingest.CatalogDocument) {
		return func(d *ingest.CatalogDocument) {
			*beef(d) = ingest.Requirement{Provenance: beef(d).Provenance, State: state, Reason: reason}
		}
	}
	cases := []struct {
		name   string
		mutate func(*ingest.CatalogDocument)
		want   ingest.IssueCode
	}{
		{"resolved", func(*ingest.CatalogDocument) {}, ""},
		{"intentionally unspecified", func(*ingest.CatalogDocument) {}, ""},
		{"unresolved with reason", unresolved(ingest.StateUnresolved, "incoherent_alternative"), ""},
		{"unsupported with reason", unresolved(ingest.StateUnsupported, "metric_cup"), ""},
		{"unresolved keeps relationship", func(d *ingest.CatalogDocument) {
			unresolved(ingest.StateUnresolved, "unknown_item")(d)
			beef(d).Relationship = ingest.RelationshipSingle
		}, ""},
		{"unknown state", func(d *ingest.CatalogDocument) { beef(d).State = "guessed" }, ingest.IssueUnknownState},
		{"missing state", func(d *ingest.CatalogDocument) { beef(d).State = "" }, ingest.IssueUnknownState},
		{"resolved with unspecified quantity", func(d *ingest.CatalogDocument) {
			beef(d).Quantity = &ingest.Quantity{Kind: "unspecified"}
		}, ingest.IssueStateQuantityMismatch},
		{"intentionally unspecified with an amount", func(d *ingest.CatalogDocument) {
			salt(d).Quantity = &ingest.Quantity{Kind: "exact", Amount: "1", Unit: "tsp"}
		}, ingest.IssueStateQuantityMismatch},
		{"resolved without grocery item", func(d *ingest.CatalogDocument) { beef(d).GroceryItem = nil }, ingest.IssueMissingInterpretation},
		{"resolved without quantity", func(d *ingest.CatalogDocument) { beef(d).Quantity = nil }, ingest.IssueMissingInterpretation},
		{"resolved with reason", func(d *ingest.CatalogDocument) { beef(d).Reason = "unknown_item" }, ingest.IssueUnexpectedReason},
		{"unresolved without reason", unresolved(ingest.StateUnresolved, ""), ingest.IssueMissingReason},
		{"unsupported without reason", unresolved(ingest.StateUnsupported, ""), ingest.IssueMissingReason},
		{"reason is not a code", unresolved(ingest.StateUnresolved, "Could not parse!"), ingest.IssueInvalidReason},
		{"unresolved with grocery item", func(d *ingest.CatalogDocument) {
			item := beef(d).GroceryItem
			unresolved(ingest.StateUnresolved, "unknown_unit")(d)
			beef(d).GroceryItem = item
		}, ingest.IssueUninterpretedContent},
		{"unsupported with quantity", func(d *ingest.CatalogDocument) {
			unresolved(ingest.StateUnsupported, "metric_cup")(d)
			beef(d).Quantity = &ingest.Quantity{Kind: "exact", Amount: "1", Unit: "cup"}
		}, ingest.IssueUninterpretedContent},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			document := loadCatalog(t)
			tc.mutate(&document)
			assertValidation(t, document, tc.want)
		})
	}
	for _, state := range ingest.InterpretationStates {
		want := state == ingest.StateResolved || state == ingest.StateIntentionallyUnspecified
		if !state.Valid() || state.Publishable() != want {
			t.Errorf("%q: Valid()=%v Publishable()=%v", state, state.Valid(), state.Publishable())
		}
	}
}

func TestCatalogProvenance(t *testing.T) {
	// "½ cup walnuts (plus 2 tablespoons for topping)" is 46 code points but 47 bytes.
	walnutLine := func(d *ingest.CatalogDocument) string { return d.SourceLines[1] }
	cases := []struct {
		name   string
		mutate func(*ingest.CatalogDocument)
		want   ingest.IssueCode
	}{
		{"span ends at the last code point", func(d *ingest.CatalogDocument) {
			walnutTopping(d).Provenance.Span.End = len([]rune(walnutLine(d)))
		}, ""},
		{"span covers the whole line", func(d *ingest.CatalogDocument) {
			beef(d).Provenance.Span = ingest.Span{Start: 0, End: len([]rune(d.SourceLines[7]))}
		}, ""},
		{"single code point span", func(d *ingest.CatalogDocument) {
			walnutBase(d).Provenance.Span = ingest.Span{Start: 0, End: 1}
		}, ""},
		{"span end counted in bytes", func(d *ingest.CatalogDocument) {
			walnutTopping(d).Provenance.Span.End = len(walnutLine(d))
		}, ingest.IssueSpanOutOfBounds},
		{"span past the line", func(d *ingest.CatalogDocument) {
			beef(d).Provenance.Span.End = len([]rune(d.SourceLines[7])) + 1
		}, ingest.IssueSpanOutOfBounds},
		{"empty span", func(d *ingest.CatalogDocument) {
			beef(d).Provenance.Span = ingest.Span{Start: 3, End: 3}
		}, ingest.IssueSpanOutOfBounds},
		{"inverted span", func(d *ingest.CatalogDocument) {
			beef(d).Provenance.Span = ingest.Span{Start: 5, End: 2}
		}, ingest.IssueSpanOutOfBounds},
		{"negative span start", func(d *ingest.CatalogDocument) {
			beef(d).Provenance.Span.Start = -1
		}, ingest.IssueSpanOutOfBounds},
		{"heading span past its line", func(d *ingest.CatalogDocument) {
			d.IngredientSections[0].Heading.Span.End = 14
		}, ingest.IssueSpanOutOfBounds},
		{"line before the first", func(d *ingest.CatalogDocument) {
			beef(d).Provenance.Line = -1
		}, ingest.IssueSourceLineOutOfRange},
		{"line after the last", func(d *ingest.CatalogDocument) {
			salt(d).Provenance.Line = len(d.SourceLines)
		}, ingest.IssueSourceLineOutOfRange},
		{"instruction step is reserved", func(d *ingest.CatalogDocument) {
			beef(d).Provenance.Type = ingest.ProvenanceInstructionStep
		}, ingest.IssueReservedProvenanceType},
		{"instruction step heading is reserved", func(d *ingest.CatalogDocument) {
			d.IngredientSections[0].Heading.Type = ingest.ProvenanceInstructionStep
		}, ingest.IssueReservedProvenanceType},
		{"unknown provenance type", func(d *ingest.CatalogDocument) {
			beef(d).Provenance.Type = "ai_guess"
		}, ingest.IssueUnknownProvenanceType},
		{"missing provenance type", func(d *ingest.CatalogDocument) {
			beef(d).Provenance.Type = ""
		}, ingest.IssueUnknownProvenanceType},
		{"requirements out of source order", func(d *ingest.CatalogDocument) {
			requirements := d.IngredientSections[1].Requirements
			requirements[4], requirements[5] = requirements[5], requirements[4]
		}, ingest.IssueSourceOrder},
		{"heading after its requirements", func(d *ingest.CatalogDocument) {
			heading := ingest.Provenance{Type: ingest.ProvenanceIngredientLine, Line: 8, Span: ingest.Span{Start: 0, End: 4}}
			d.IngredientSections[1].Heading = &heading
		}, ingest.IssueSourceOrder},
		{"line split across sections", func(d *ingest.CatalogDocument) {
			moved := *walnutTopping(d)
			d.IngredientSections[0].Requirements = slices.Delete(d.IngredientSections[0].Requirements, 1, 2)
			d.IngredientSections[1].Requirements = slices.Insert(d.IngredientSections[1].Requirements, 0, moved)
		}, ingest.IssueSourceOrder},
		{"blank source line", func(d *ingest.CatalogDocument) {
			d.SourceLines = append(d.SourceLines, " ")
		}, ingest.IssueBlankSourceLine},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			document := loadCatalog(t)
			tc.mutate(&document)
			assertValidation(t, document, tc.want)
		})
	}
}

func TestCatalogUsesSharedContentValidation(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ingest.CatalogDocument)
		detail string
	}{
		{"name", func(d *ingest.CatalogDocument) { d.Name = "" }, "name is required"},
		{"source", func(d *ingest.CatalogDocument) { d.Source.URL = "not a url" }, "invalid source URL"},
		{"yield", func(d *ingest.CatalogDocument) { d.Yield = " " }, "yield cannot be blank"},
		{"duration", func(d *ingest.CatalogDocument) { d.HandsOn = &ingest.Duration{Min: 5, Max: 1} }, "hands_on range is invalid"},
		{"empty section", func(d *ingest.CatalogDocument) { d.IngredientSections[0].Requirements = nil }, "needs a name and ingredients"},
		{"no ingredient sections", func(d *ingest.CatalogDocument) { d.IngredientSections = nil }, "at least one ingredient section is required"},
		{"no instructions", func(d *ingest.CatalogDocument) { d.InstructionSections = nil }, "at least one instruction section is required"},
		{"unknown unit", func(d *ingest.CatalogDocument) {
			beef(d).Quantity = &ingest.Quantity{Kind: "exact", Amount: "3", Unit: "pound"}
		}, `unknown unit "pound"`},
		{"unknown package unit", func(d *ingest.CatalogDocument) {
			beef(d).Quantity = &ingest.Quantity{Kind: "exact", Amount: "1", Package: &ingest.Package{Type: "pack", Size: "3", Unit: "pounds"}}
		}, `unknown package unit "pounds"`},
		{"amount beyond 64 bits", func(d *ingest.CatalogDocument) {
			beef(d).Quantity = &ingest.Quantity{Kind: "exact", Amount: "9223372036854775808", Unit: "lb"}
		}, "exceeds 64-bit"},
		{"denominator beyond 64 bits", func(d *ingest.CatalogDocument) {
			beef(d).Quantity = &ingest.Quantity{Kind: "range", Amount: "1", Maximum: "3 1/9223372036854775808", Unit: "lb"}
		}, "exceeds 64-bit"},
		{"inconsistent grocery item", func(d *ingest.CatalogDocument) {
			item := *beef(d).GroceryItem
			item.Key = "salt"
			beef(d).GroceryItem = &item
		}, `redefines grocery item "salt"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			document := loadCatalog(t)
			tc.mutate(&document)
			err := document.Validate()
			assertCodes(t, validationCodes(t, err), ingest.IssueInvalidContent)
			if !strings.Contains(err.Error(), tc.detail) {
				t.Fatalf("Validate() = %v, want detail %q", err, tc.detail)
			}
		})
	}
}

func TestCheckPublishableReportsEveryReason(t *testing.T) {
	t.Run("unresolved and undisposed lines", func(t *testing.T) {
		document := loadCatalog(t)
		// Line 7: unresolved, so the line has requirements but none publishable.
		*beef(&document) = ingest.Requirement{Provenance: beef(&document).Provenance, State: ingest.StateUnsupported, Reason: "metric_cup"}
		// Line 9: a source line no requirement accounts for.
		document.SourceLines = append(document.SourceLines, "1 lemon, zested")
		// Line 3: one compound part unresolved; the other still disposes of the line.
		*rosemary(&document) = ingest.Requirement{Provenance: rosemary(&document).Provenance, Relationship: ingest.RelationshipCompound, State: ingest.StateUnresolved, Reason: "unknown_item"}

		if err := document.Validate(); err != nil {
			t.Fatalf("Validate() = %v; an unpublishable document is still valid", err)
		}
		issues := ingest.CheckPublishable(document)
		assertCodes(t, issueCodes(issues),
			ingest.IssueUnpublishableState,
			ingest.IssueUnpublishableState,
			ingest.IssueUnpublishableSourceLine,
			ingest.IssueUndisposedSourceLine,
		)
		paths := map[string]ingest.IssueCode{}
		for _, issue := range issues {
			paths[issue.Path] = issue.Code
		}
		if paths["source_lines[7]"] != ingest.IssueUnpublishableSourceLine || paths["source_lines[9]"] != ingest.IssueUndisposedSourceLine ||
			paths["source_lines[3]"] != "" || paths["ingredient_sections[1].requirements[5]"] != ingest.IssueUnpublishableState {
			t.Fatalf("issues = %v", issues)
		}
		if !strings.Contains(issues[0].Detail+issues[1].Detail, "metric_cup") {
			t.Fatalf("unpublishable state detail should carry the reason: %v", issues)
		}
	})

	t.Run("empty candidate", func(t *testing.T) {
		document := ingest.CatalogDocument{
			FormatVersion: ingest.CatalogFormatVersion,
			Name:          "Empty",
			Source:        ingest.Source{Relationship: "source", Attribution: "Example"},
			SourceLines:   []string{"1 onion"},
		}
		assertCodes(t, issueCodes(ingest.CheckPublishable(document)),
			ingest.IssueInvalidContent,
			ingest.IssueInvalidContent,
			ingest.IssueNoRequirements,
			ingest.IssueNoInstructionSteps,
			ingest.IssueUndisposedSourceLine,
		)
	})

	t.Run("invalid provenance is also no disposition", func(t *testing.T) {
		document := loadCatalog(t)
		beef(&document).Provenance.Type = ingest.ProvenanceInstructionStep
		salt(&document).Provenance.Span.End = 99
		assertCodes(t, issueCodes(ingest.CheckPublishable(document)),
			ingest.IssueReservedProvenanceType,
			ingest.IssueSpanOutOfBounds,
			ingest.IssueUndisposedSourceLine,
			ingest.IssueUndisposedSourceLine,
		)
	})

	t.Run("heading line is a disposition", func(t *testing.T) {
		document := loadCatalog(t)
		document.IngredientSections[0].Heading = nil
		assertCodes(t, issueCodes(ingest.CheckPublishable(document)), ingest.IssueUndisposedSourceLine)
	})
}

func assertValidation(t *testing.T, document ingest.CatalogDocument, want ingest.IssueCode) {
	t.Helper()
	codes := validationCodes(t, document.Validate())
	if want == "" {
		if len(codes) != 0 {
			t.Fatalf("Validate() codes = %v, want valid", codes)
		}
		return
	}
	if !slices.Contains(codes, want) {
		t.Fatalf("Validate() codes = %v, want %q", codes, want)
	}
	// An invalid document is never publishable, for the same reason.
	if publishCodes := issueCodes(ingest.CheckPublishable(document)); !slices.Contains(publishCodes, want) {
		t.Fatalf("CheckPublishable() codes = %v, want %q", publishCodes, want)
	}
}

// assertCodes compares issue codes as a multiset.
func assertCodes(t *testing.T, got []ingest.IssueCode, want ...ingest.IssueCode) {
	t.Helper()
	got, want = slices.Clone(got), slices.Clone(want)
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("issue codes = %v, want %v", got, want)
	}
}
