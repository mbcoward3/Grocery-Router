package ingest

import (
	"bytes"
	"fmt"
	"io"
	"math/big"
	"net/url"
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

type Source struct {
	Relationship string `yaml:"relationship"`
	Attribution  string `yaml:"attribution"`
	URL          string `yaml:"url,omitempty"`
	CheckedOn    string `yaml:"checked_on,omitempty"`
}

type Duration struct {
	Min int64 `yaml:"min"`
	Max int64 `yaml:"max"`
}

type IngredientSection struct {
	Name        string       `yaml:"name"`
	Ingredients []Ingredient `yaml:"ingredients"`
}

type Ingredient struct {
	SourceText  string      `yaml:"source_text"`
	GroceryItem GroceryItem `yaml:"grocery_item"`
	Quantity    Quantity    `yaml:"quantity"`
	Preparation string      `yaml:"preparation,omitempty"`
	Optional    bool        `yaml:"optional,omitempty"`
	NonShopping bool        `yaml:"non_shopping,omitempty"`
	Note        string      `yaml:"note,omitempty"`
}

type GroceryItem struct {
	Key          string       `yaml:"key"`
	Name         string       `yaml:"name"`
	StoreSection StoreSection `yaml:"store_section"`
	ShoppingMode string       `yaml:"shopping_mode"`
}

type StoreSection struct {
	Key  string `yaml:"key"`
	Name string `yaml:"name"`
}

type Quantity struct {
	Kind    string   `yaml:"kind"`
	Amount  string   `yaml:"amount,omitempty"`
	Maximum string   `yaml:"maximum,omitempty"`
	Unit    string   `yaml:"unit,omitempty"`
	Package *Package `yaml:"package,omitempty"`
}

type Package struct {
	Type string `yaml:"type"`
	Size string `yaml:"size,omitempty"`
	Unit string `yaml:"unit,omitempty"`
}

type InstructionSection struct {
	Name  string   `yaml:"name"`
	Steps []string `yaml:"steps"`
}

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

// Validate enforces file-level completeness before any database transaction starts.
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
	if strings.TrimSpace(d.Name) == "" {
		return prefix("name is required")
	}
	if d.Status != "verified" {
		return prefix("bootstrap status must be verified, got %q", d.Status)
	}
	if _, err := time.Parse(time.DateOnly, d.ApprovedOn); err != nil {
		return prefix("approved_on must be YYYY-MM-DD")
	}
	if d.Source.Relationship != "source" && d.Source.Relationship != "adapted-from" {
		return prefix("invalid source relationship %q", d.Source.Relationship)
	}
	if strings.TrimSpace(d.Source.Attribution) == "" {
		return prefix("source attribution is required")
	}
	if d.Source.URL != "" {
		parsed, err := url.ParseRequestURI(d.Source.URL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return prefix("invalid source URL %q", d.Source.URL)
		}
	}
	if d.Source.CheckedOn != "" {
		if _, err := time.Parse(time.DateOnly, d.Source.CheckedOn); err != nil {
			return prefix("source checked_on must be YYYY-MM-DD")
		}
	}
	if err := validateDuration("hands_on", d.HandsOn); err != nil {
		return prefix("%v", err)
	}
	if err := validateDuration("unattended", d.Unattended); err != nil {
		return prefix("%v", err)
	}
	if len(d.IngredientSections) == 0 {
		return prefix("at least one ingredient section is required")
	}
	items := make(map[string]GroceryItem)
	ingredientCount := 0
	for sectionIndex, section := range d.IngredientSections {
		if strings.TrimSpace(section.Name) == "" || len(section.Ingredients) == 0 {
			return prefix("ingredient section %d needs a name and ingredients", sectionIndex)
		}
		for ingredientIndex, ingredient := range section.Ingredients {
			ingredientCount++
			location := fmt.Sprintf("ingredient_sections[%d].ingredients[%d]", sectionIndex, ingredientIndex)
			if strings.TrimSpace(ingredient.SourceText) == "" {
				return prefix("%s source_text is required", location)
			}
			if err := ingredient.GroceryItem.validate(); err != nil {
				return prefix("%s: %v", location, err)
			}
			if existing, ok := items[ingredient.GroceryItem.Key]; ok && existing != ingredient.GroceryItem {
				return prefix("%s redefines grocery item %q inconsistently", location, ingredient.GroceryItem.Key)
			}
			items[ingredient.GroceryItem.Key] = ingredient.GroceryItem
			if _, err := ingredient.Quantity.rationals(); err != nil {
				return prefix("%s: %v", location, err)
			}
		}
	}
	if ingredientCount == 0 {
		return prefix("ingredients are required")
	}
	if len(d.InstructionSections) == 0 {
		return prefix("at least one instruction section is required")
	}
	for sectionIndex, section := range d.InstructionSections {
		if strings.TrimSpace(section.Name) == "" || len(section.Steps) == 0 {
			return prefix("instruction section %d needs a name and steps", sectionIndex)
		}
		for stepIndex, step := range section.Steps {
			if strings.TrimSpace(step) == "" {
				return prefix("instruction_sections[%d].steps[%d] is empty", sectionIndex, stepIndex)
			}
		}
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

func validateDuration(name string, duration *Duration) error {
	if duration == nil {
		return nil
	}
	if duration.Min < 0 || duration.Max < duration.Min {
		return fmt.Errorf("%s range is invalid", name)
	}
	return nil
}

func (g GroceryItem) validate() error {
	if !documentKeyPattern.MatchString(g.Key) || strings.TrimSpace(g.Name) == "" {
		return fmt.Errorf("grocery item needs valid key and name")
	}
	if !documentKeyPattern.MatchString(g.StoreSection.Key) || strings.TrimSpace(g.StoreSection.Name) == "" {
		return fmt.Errorf("grocery item %q needs valid store section", g.Key)
	}
	if g.ShoppingMode != "measured" && g.ShoppingMode != "counted" && g.ShoppingMode != "presence-only" {
		return fmt.Errorf("grocery item %q has invalid shopping mode %q", g.Key, g.ShoppingMode)
	}
	return nil
}

type quantityRationals struct {
	Minimum     *big.Rat
	Maximum     *big.Rat
	PackageSize *big.Rat
}

func (q Quantity) rationals() (quantityRationals, error) {
	var result quantityRationals
	if q.Kind != "exact" && q.Kind != "range" && q.Kind != "unspecified" {
		return result, fmt.Errorf("invalid quantity kind %q", q.Kind)
	}
	if q.Kind == "unspecified" {
		if q.Amount != "" || q.Maximum != "" || q.Unit != "" || q.Package != nil {
			return result, fmt.Errorf("unspecified quantity cannot carry amount, unit, or package")
		}
		return result, nil
	}
	minimum, err := parsePositiveRational(q.Amount)
	if err != nil {
		return result, fmt.Errorf("invalid amount: %w", err)
	}
	result.Minimum = minimum
	if q.Kind == "range" {
		maximum, err := parsePositiveRational(q.Maximum)
		if err != nil {
			return result, fmt.Errorf("invalid maximum: %w", err)
		}
		if minimum.Cmp(maximum) > 0 {
			return result, fmt.Errorf("quantity minimum exceeds maximum")
		}
		result.Maximum = maximum
	} else if q.Maximum != "" {
		return result, fmt.Errorf("exact quantity cannot carry maximum")
	}
	if (q.Unit == "") == (q.Package == nil) {
		return result, fmt.Errorf("numeric quantity needs exactly one of unit or package")
	}
	if q.Unit != "" && !documentKeyPattern.MatchString(q.Unit) {
		return result, fmt.Errorf("invalid unit key %q", q.Unit)
	}
	if q.Package != nil {
		if strings.TrimSpace(q.Package.Type) == "" {
			return result, fmt.Errorf("package type is required")
		}
		if (q.Package.Size == "") != (q.Package.Unit == "") {
			return result, fmt.Errorf("package size and unit must appear together")
		}
		if q.Package.Size != "" {
			size, err := parsePositiveRational(q.Package.Size)
			if err != nil {
				return result, fmt.Errorf("invalid package size: %w", err)
			}
			if !documentKeyPattern.MatchString(q.Package.Unit) {
				return result, fmt.Errorf("invalid package unit key %q", q.Package.Unit)
			}
			result.PackageSize = size
		}
	}
	return result, nil
}

func parsePositiveRational(value string) (*big.Rat, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("value is required")
	}
	var result *big.Rat
	parts := strings.Fields(value)
	if len(parts) == 2 {
		whole, ok := new(big.Rat).SetString(parts[0])
		if !ok {
			return nil, fmt.Errorf("invalid mixed number %q", value)
		}
		fraction, ok := new(big.Rat).SetString(parts[1])
		if !ok {
			return nil, fmt.Errorf("invalid mixed number %q", value)
		}
		result = new(big.Rat).Add(whole, fraction)
	} else if len(parts) == 1 {
		var ok bool
		result, ok = new(big.Rat).SetString(value)
		if !ok {
			return nil, fmt.Errorf("invalid number %q", value)
		}
	} else {
		return nil, fmt.Errorf("invalid number %q", value)
	}
	if result.Sign() <= 0 {
		return nil, fmt.Errorf("value must be positive")
	}
	return result, nil
}
