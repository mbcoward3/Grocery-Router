// Package ingest owns the recipe content contract. Shared recipe content validation
// (ValidateRecipeContent) is used by two formats: the approved Markdown bootstrap corpus
// (Document), which is validated with human approval and imported into a household, and the
// versioned shared catalog content format (CatalogDocument), which adds per-requirement
// provenance, relationship, and interpretation state.
package ingest

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var documentKeyPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// Document is the strict YAML front matter in one approved Markdown bootstrap recipe.
type Document struct {
	FormatVersion       int                  `yaml:"format_version"`
	Key                 string               `yaml:"key"`
	Name                string               `yaml:"name"`
	Status              string               `yaml:"status"`
	ApprovedOn          string               `yaml:"approved_on"`
	Source              Source               `yaml:"source"`
	ImageURL            string               `yaml:"image_url,omitempty"`
	Yield               string               `yaml:"yield,omitempty"`
	HandsOn             *Duration            `yaml:"hands_on,omitempty"`
	Unattended          *Duration            `yaml:"unattended,omitempty"`
	IngredientSections  []IngredientSection  `yaml:"ingredient_sections"`
	InstructionSections []InstructionSection `yaml:"instruction_sections"`
	Review              []ReviewDecision     `yaml:"review,omitempty"`
	Body                string               `yaml:"-"`
}

// Source records the approved recipe's relationship to its source evidence.
type Source struct {
	Relationship string `yaml:"relationship"`
	Attribution  string `yaml:"attribution"`
	URL          string `yaml:"url,omitempty"`
	CheckedOn    string `yaml:"checked_on,omitempty"`
}

// Duration is an inclusive duration range in minutes.
type Duration struct {
	Min int64 `yaml:"min"`
	Max int64 `yaml:"max"`
}

// IngredientSection is an ordered group of recipe ingredients.
type IngredientSection struct {
	Name        string       `yaml:"name"`
	Ingredients []Ingredient `yaml:"ingredients"`
}

// Ingredient preserves one source requirement and its approved grocery mapping.
type Ingredient struct {
	SourceText  string      `yaml:"source_text"`
	GroceryItem GroceryItem `yaml:"grocery_item"`
	Quantity    Quantity    `yaml:"quantity"`
	Preparation string      `yaml:"preparation,omitempty"`
	Optional    bool        `yaml:"optional,omitempty"`
	NonShopping bool        `yaml:"non_shopping,omitempty"`
	Note        string      `yaml:"note,omitempty"`
}

// GroceryItem is the canonical shoppable identity for an ingredient.
type GroceryItem struct {
	Key          string       `yaml:"key"`
	Name         string       `yaml:"name"`
	StoreSection StoreSection `yaml:"store_section"`
	ShoppingMode string       `yaml:"shopping_mode"`
}

// StoreSection identifies where a grocery item is grouped for shopping.
type StoreSection struct {
	Key  string `yaml:"key"`
	Name string `yaml:"name"`
}

// Quantity is an exact measured, counted, or unspecified recipe requirement.
type Quantity struct {
	Kind    string   `yaml:"kind"`
	Amount  string   `yaml:"amount,omitempty"`
	Maximum string   `yaml:"maximum,omitempty"`
	Unit    string   `yaml:"unit,omitempty"`
	Package *Package `yaml:"package,omitempty"`
}

// Package preserves a source package type and optional exact size.
type Package struct {
	Type string `yaml:"type"`
	Size string `yaml:"size,omitempty"`
	Unit string `yaml:"unit,omitempty"`
}

// InstructionSection is an ordered group of cooking steps.
type InstructionSection struct {
	Name  string   `yaml:"name"`
	Steps []string `yaml:"steps"`
}

// ReviewDecision records a human-approved true-up interpretation.
type ReviewDecision struct {
	Field    string `yaml:"field"`
	Kind     string `yaml:"kind"`
	Note     string `yaml:"note"`
	Approved bool   `yaml:"approved"`
}

// ParseDocument reads strict YAML front matter and requires the readable Markdown body to be
// the exact checked rendering of that structured data.
func ParseDocument(input io.Reader) (Document, error) {
	contents, err := io.ReadAll(input)
	if err != nil {
		return Document{}, fmt.Errorf("read recipe document: %w", err)
	}
	document, _, err := decodeDocument(contents)
	if err != nil {
		return Document{}, err
	}
	expectedBody := RenderBody(document)
	if strings.TrimSpace(document.Body) != strings.TrimSpace(expectedBody) {
		return Document{}, fmt.Errorf("recipe %q: readable Markdown does not match structured front matter", document.Key)
	}
	return document, nil
}

// RewriteReadableBody preserves strict front matter and replaces the Markdown body with its
// deterministic human-readable rendering.
func RewriteReadableBody(input io.Reader) ([]byte, error) {
	contents, err := io.ReadAll(input)
	if err != nil {
		return nil, fmt.Errorf("read recipe document: %w", err)
	}
	document, frontMatterEnd, err := decodeDocument(contents)
	if err != nil {
		return nil, err
	}
	result := append([]byte(nil), bytes.TrimPrefix(contents, []byte("\xef\xbb\xbf"))[:frontMatterEnd]...)
	result = append(result, '\n')
	result = append(result, RenderBody(document)...)
	return result, nil
}

func decodeDocument(contents []byte) (Document, int, error) {
	contents = bytes.TrimPrefix(contents, []byte("\xef\xbb\xbf"))
	if !bytes.HasPrefix(contents, []byte("---\n")) {
		return Document{}, 0, fmt.Errorf("recipe document must start with YAML front matter")
	}
	end := bytes.Index(contents[4:], []byte("\n---\n"))
	if end < 0 {
		return Document{}, 0, fmt.Errorf("recipe document has no closing front matter delimiter")
	}
	end += 4
	var document Document
	decoder := yaml.NewDecoder(bytes.NewReader(contents[4:end]))
	decoder.KnownFields(true)
	if err := decoder.Decode(&document); err != nil {
		return Document{}, 0, fmt.Errorf("decode recipe front matter: %w", err)
	}
	frontMatterEnd := end + 5
	document.Body = strings.TrimSpace(string(contents[frontMatterEnd:]))
	if err := document.Validate(); err != nil {
		return Document{}, 0, err
	}
	return document, frontMatterEnd, nil
}

// Validate is the bootstrap validator. It enforces the bootstrap file identity and human
// approval (format version, key, `status: verified`, `approved_on`, approved review decisions)
// and calls the shared ValidateRecipeContent for everything else, before any database
// transaction starts.
func (d Document) Validate() error {
	prefix := func(format string, args ...any) error {
		return fmt.Errorf("recipe %q: %s", d.Key, fmt.Sprintf(format, args...))
	}
	if d.FormatVersion != 1 {
		return prefix("format_version = %d, want 1", d.FormatVersion)
	}
	if !documentKeyPattern.MatchString(d.Key) {
		return prefix("invalid key")
	}
	if d.Status != "verified" {
		return prefix("bootstrap status must be verified, got %q", d.Status)
	}
	if _, err := time.Parse(time.DateOnly, d.ApprovedOn); err != nil {
		return prefix("approved_on must be YYYY-MM-DD")
	}
	if err := ValidateRecipeContent(d.Content()); err != nil {
		return prefix("%v", err)
	}
	for index, decision := range d.Review {
		if strings.TrimSpace(decision.Field) == "" || strings.TrimSpace(decision.Note) == "" {
			return prefix("review decision %d needs field and note", index)
		}
		if decision.Kind != "backfilled" && decision.Kind != "rewritten" && decision.Kind != "conflict-resolved" {
			return prefix("review decision %d has invalid kind %q", index, decision.Kind)
		}
		if !decision.Approved {
			return prefix("review decision %d is not approved", index)
		}
	}
	return nil
}

// Content returns the shared recipe content of an approved bootstrap document.
func (d Document) Content() RecipeContent {
	return RecipeContent{
		Name:                d.Name,
		Source:              d.Source,
		ImageURL:            d.ImageURL,
		Yield:               d.Yield,
		HandsOn:             d.HandsOn,
		Unattended:          d.Unattended,
		IngredientSections:  d.IngredientSections,
		InstructionSections: d.InstructionSections,
	}
}
