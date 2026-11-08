package ingest

import (
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"time"
)

// RecipeContent is the recipe content every recipe format shares: what a household plans,
// shops for, and cooks from. Approval (bootstrap) and interpretation provenance (catalog) are
// format-specific and live outside it.
type RecipeContent struct {
	Name                string
	Source              Source
	ImageURL            string
	Yield               string
	HandsOn             *Duration
	Unattended          *Duration
	IngredientSections  []IngredientSection
	InstructionSections []InstructionSection
}

// ValidateRecipeContent is the shared recipe content validation used by every recipe format.
// It checks the name, source, yield, durations, ingredient and instruction sections, and
// every quantity, including that each unit exists in the unit vocabulary and each rational
// fits the database's 64-bit numerator and denominator columns. It returns the first problem.
func ValidateRecipeContent(content RecipeContent) error {
	if err := validateRecipeMetadata(content.Name, content.Source, content.ImageURL, content.Yield, content.HandsOn, content.Unattended); err != nil {
		return err
	}
	if len(content.IngredientSections) == 0 {
		return errors.New("at least one ingredient section is required")
	}
	items := make(groceryItemSet)
	for sectionIndex, section := range content.IngredientSections {
		if err := validateIngredientSectionShape(sectionIndex, section.Name, len(section.Ingredients)); err != nil {
			return err
		}
		for ingredientIndex, ingredient := range section.Ingredients {
			location := fmt.Sprintf("ingredient_sections[%d].ingredients[%d]", sectionIndex, ingredientIndex)
			if err := validateIngredient(location, ingredient, items); err != nil {
				return err
			}
		}
	}
	return validateInstructionSections(content.InstructionSections)
}

func validateRecipeMetadata(name string, source Source, imageURL, yield string, handsOn, unattended *Duration) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("name is required")
	}
	if source.Relationship != "source" && source.Relationship != "adapted-from" {
		return fmt.Errorf("invalid source relationship %q", source.Relationship)
	}
	if strings.TrimSpace(source.Attribution) == "" {
		return errors.New("source attribution is required")
	}
	if source.URL != "" {
		parsed, err := url.ParseRequestURI(source.URL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return fmt.Errorf("invalid source URL %q", source.URL)
		}
	}
	if source.CheckedOn != "" {
		if _, err := time.Parse(time.DateOnly, source.CheckedOn); err != nil {
			return errors.New("source checked_on must be YYYY-MM-DD")
		}
	}
	if imageURL != "" && strings.TrimSpace(imageURL) == "" {
		return errors.New("image_url cannot be blank")
	}
	if yield != "" && strings.TrimSpace(yield) == "" {
		return errors.New("yield cannot be blank")
	}
	if err := validateDuration("hands_on", handsOn); err != nil {
		return err
	}
	return validateDuration("unattended", unattended)
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

func validateIngredientSectionShape(index int, name string, ingredients int) error {
	if strings.TrimSpace(name) == "" || ingredients == 0 {
		return fmt.Errorf("ingredient section %d needs a name and ingredients", index)
	}
	return nil
}

// groceryItemSet requires every use of a grocery item key within one recipe to agree.
type groceryItemSet map[string]GroceryItem

func validateIngredient(location string, ingredient Ingredient, items groceryItemSet) error {
	if strings.TrimSpace(ingredient.SourceText) == "" {
		return fmt.Errorf("%s source_text is required", location)
	}
	if err := ingredient.GroceryItem.validate(); err != nil {
		return fmt.Errorf("%s: %w", location, err)
	}
	if existing, ok := items[ingredient.GroceryItem.Key]; ok && existing != ingredient.GroceryItem {
		return fmt.Errorf("%s redefines grocery item %q inconsistently", location, ingredient.GroceryItem.Key)
	}
	items[ingredient.GroceryItem.Key] = ingredient.GroceryItem
	if _, err := ingredient.Quantity.rationals(); err != nil {
		return fmt.Errorf("%s: %w", location, err)
	}
	if ingredient.Preparation != "" && strings.TrimSpace(ingredient.Preparation) == "" {
		return fmt.Errorf("%s preparation cannot be blank", location)
	}
	if ingredient.Note != "" && strings.TrimSpace(ingredient.Note) == "" {
		return fmt.Errorf("%s note cannot be blank", location)
	}
	return nil
}

func validateInstructionSections(sections []InstructionSection) error {
	if len(sections) == 0 {
		return errors.New("at least one instruction section is required")
	}
	for sectionIndex, section := range sections {
		if strings.TrimSpace(section.Name) == "" || len(section.Steps) == 0 {
			return fmt.Errorf("instruction section %d needs a name and steps", sectionIndex)
		}
		for stepIndex, step := range section.Steps {
			if strings.TrimSpace(step) == "" {
				return fmt.Errorf("instruction_sections[%d].steps[%d] is empty", sectionIndex, stepIndex)
			}
		}
	}
	return nil
}

func (g GroceryItem) validate() error {
	if !documentKeyPattern.MatchString(g.Key) || strings.TrimSpace(g.Name) == "" {
		return errors.New("grocery item needs valid key and name")
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

// Rationals parses a quantity's amount, maximum, and package size as exact rationals after
// the same checks shared validation applies. Values that are absent from the quantity are nil.
func (q Quantity) Rationals() (minimum, maximum, packageSize *big.Rat, err error) {
	values, err := q.rationals()
	return values.Minimum, values.Maximum, values.PackageSize, err
}

// rationals parses and checks a quantity. Every returned rational fits 64-bit storage and
// every unit key is in the unit vocabulary.
func (q Quantity) rationals() (quantityRationals, error) {
	var result quantityRationals
	if q.Kind != "exact" && q.Kind != "range" && q.Kind != "unspecified" {
		return result, fmt.Errorf("invalid quantity kind %q", q.Kind)
	}
	if q.Kind == "unspecified" {
		if q.Amount != "" || q.Maximum != "" || q.Unit != "" || q.Package != nil {
			return result, errors.New("unspecified quantity cannot carry amount, unit, or package")
		}
		return result, nil
	}
	minimum, err := parseStoredRational(q.Amount)
	if err != nil {
		return result, fmt.Errorf("invalid amount: %w", err)
	}
	result.Minimum = minimum
	if q.Kind == "range" {
		maximum, err := parseStoredRational(q.Maximum)
		if err != nil {
			return result, fmt.Errorf("invalid maximum: %w", err)
		}
		if minimum.Cmp(maximum) > 0 {
			return result, errors.New("quantity minimum exceeds maximum")
		}
		result.Maximum = maximum
	} else if q.Maximum != "" {
		return result, errors.New("exact quantity cannot carry maximum")
	}
	if (q.Unit == "") == (q.Package == nil) {
		return result, errors.New("numeric quantity needs exactly one of unit or package")
	}
	if q.Unit != "" {
		if !documentKeyPattern.MatchString(q.Unit) {
			return result, fmt.Errorf("invalid unit key %q", q.Unit)
		}
		if !IsKnownUnit(q.Unit) {
			return result, fmt.Errorf("unknown unit %q", q.Unit)
		}
	}
	if q.Package != nil {
		if strings.TrimSpace(q.Package.Type) == "" {
			return result, errors.New("package type is required")
		}
		if (q.Package.Size == "") != (q.Package.Unit == "") {
			return result, errors.New("package size and unit must appear together")
		}
		if q.Package.Size != "" {
			size, err := parseStoredRational(q.Package.Size)
			if err != nil {
				return result, fmt.Errorf("invalid package size: %w", err)
			}
			if !documentKeyPattern.MatchString(q.Package.Unit) {
				return result, fmt.Errorf("invalid package unit key %q", q.Package.Unit)
			}
			if !IsKnownUnit(q.Package.Unit) {
				return result, fmt.Errorf("unknown package unit %q", q.Package.Unit)
			}
			result.PackageSize = size
		}
	}
	return result, nil
}

// parseStoredRational parses a positive rational that the database can store exactly.
func parseStoredRational(value string) (*big.Rat, error) {
	result, err := parsePositiveRational(value)
	if err != nil {
		return nil, err
	}
	if !fitsInt64Storage(result) {
		return nil, fmt.Errorf("value %s exceeds 64-bit numerator or denominator storage", result.RatString())
	}
	return result, nil
}

// fitsInt64Storage reports whether a rational's reduced numerator and denominator each fit a
// PostgreSQL BIGINT column, which is how nullRational stores it.
func fitsInt64Storage(value *big.Rat) bool {
	return value.Num().IsInt64() && value.Denom().IsInt64()
}

func parsePositiveRational(value string) (*big.Rat, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("value is required")
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
		return nil, errors.New("value must be positive")
	}
	return result, nil
}
