package qualify

import (
	"fmt"
	"math/big"

	"github.com/mbcoward3/grocery-router/internal/grocery"
	"github.com/mbcoward3/grocery-router/internal/ingest"
)

// GroceryLine is one generated grocery line in the evaluation format: what the existing
// aggregation (internal/grocery) produces for one recipe in an otherwise empty week.
// Amounts are canonical reduced rationals ("5/8", "3/2", "2").
type GroceryLine struct {
	// Item is the grocery item key.
	Item string `yaml:"item"`
	// Section is the store section key.
	Section string `yaml:"section"`
	// Kind is exact, range, unspecified, or presence (presence-only items).
	Kind     string          `yaml:"kind"`
	Amount   string          `yaml:"amount,omitempty"`
	Maximum  string          `yaml:"maximum,omitempty"`
	Unit     string          `yaml:"unit,omitempty"`
	Package  *ingest.Package `yaml:"package,omitempty"`
	Optional bool            `yaml:"optional,omitempty"`
	// SourceLines are the source line positions of every requirement behind the line, in
	// contribution order, so a reviewer can see that nothing was dropped.
	SourceLines []int `yaml:"source_lines,flow"`
}

// GroceryRequirements converts a publishable catalog document into the requirements the
// grocery aggregator takes, exactly as if the recipe had been imported into a household and
// added once to an empty week: non-shopping requirements are left off the list, item and
// section identities are numbered by first use, and units carry the seeded exact ratios.
//
// RecipeIngredientID is the requirement's 1-based position in document order.
func GroceryRequirements(document ingest.CatalogDocument) ([]grocery.Requirement, error) {
	converted, err := convert(document)
	return converted.requirements, err
}

type conversion struct {
	requirements []grocery.Requirement
	itemKeys     map[int64]string
	sectionKeys  map[int64]string
	sourceLines  map[int64]int // RecipeIngredientID -> source line
}

func convert(document ingest.CatalogDocument) (conversion, error) {
	result := conversion{itemKeys: map[int64]string{}, sectionKeys: map[int64]string{}, sourceLines: map[int64]int{}}
	content, err := document.Content()
	if err != nil {
		return result, err
	}
	var provenance []ingest.Provenance
	for _, section := range document.IngredientSections {
		for _, requirement := range section.Requirements {
			provenance = append(provenance, requirement.Provenance)
		}
	}
	items := make(map[string]int64)
	sections := make(map[string]int64)
	units := make(map[string]*grocery.Unit)
	unit := func(key string) (*grocery.Unit, error) {
		if existing, ok := units[key]; ok {
			return existing, nil
		}
		dimension, _ := ingest.UnitDimension(key)
		numerator, denominator, ok := ingest.UnitToBase(key)
		if !ok {
			return nil, fmt.Errorf("unknown unit %q", key)
		}
		created := &grocery.Unit{
			ID: int64(len(units) + 1), Key: key, Dimension: dimension,
			ToBaseNumerator: numerator, ToBaseDenominator: denominator,
		}
		units[key] = created
		return created, nil
	}

	ordinal := int64(0)
	for _, section := range content.IngredientSections {
		for _, ingredient := range section.Ingredients {
			ordinal++
			result.sourceLines[ordinal] = provenance[ordinal-1].Line
			if ingredient.NonShopping {
				continue
			}
			item := ingredient.GroceryItem
			if items[item.Key] == 0 {
				items[item.Key] = int64(len(items) + 1)
				result.itemKeys[items[item.Key]] = item.Key
			}
			if sections[item.StoreSection.Key] == 0 {
				sections[item.StoreSection.Key] = int64(len(sections) + 1)
				result.sectionKeys[sections[item.StoreSection.Key]] = item.StoreSection.Key
			}
			minimum, maximum, packageSize, err := ingredient.Quantity.Rationals()
			if err != nil {
				return result, fmt.Errorf("requirement %d: %w", ordinal, err)
			}
			requirement := grocery.Requirement{
				WeekRecipeID: 1, RecipeIngredientID: ordinal, RecipeName: content.Name,
				GroceryItemID: items[item.Key], GroceryItemKey: item.Key, GroceryItemName: item.Name,
				ShoppingMode: item.ShoppingMode, StoreSectionID: sections[item.StoreSection.Key],
				StoreSectionName: item.StoreSection.Name, QuantityKind: ingredient.Quantity.Kind,
				Minimum: rational(minimum), Maximum: rational(maximum), Optional: ingredient.Optional,
			}
			if ingredient.Quantity.Unit != "" {
				if requirement.Unit, err = unit(ingredient.Quantity.Unit); err != nil {
					return result, err
				}
			}
			if pkg := ingredient.Quantity.Package; pkg != nil {
				requirement.Package = &grocery.Package{Type: pkg.Type, Size: rational(packageSize)}
				if pkg.Unit != "" {
					if requirement.Package.SizeUnit, err = unit(pkg.Unit); err != nil {
						return result, err
					}
				}
			}
			result.requirements = append(result.requirements, requirement)
		}
	}
	return result, nil
}

// GroceryOutput aggregates a publishable catalog document with grocery.Aggregate and renders
// the result in the evaluation format.
func GroceryOutput(document ingest.CatalogDocument) ([]GroceryLine, error) {
	converted, err := convert(document)
	if err != nil {
		return nil, err
	}
	lines, err := grocery.Aggregate(converted.requirements)
	if err != nil {
		return nil, err
	}
	return RenderGroceryLines(lines, func(line grocery.Line) (string, string) {
		return converted.itemKeys[line.GroceryItemID], converted.sectionKeys[line.StoreSectionID]
	}, func(contribution grocery.Contribution) int {
		return converted.sourceLines[contribution.RecipeIngredientID]
	}), nil
}

// RenderGroceryLines renders aggregated lines in the evaluation format. keys names a line's
// grocery item and store section keys; sourceLine names a contribution's source line.
func RenderGroceryLines(
	lines []grocery.Line,
	keys func(grocery.Line) (item, section string),
	sourceLine func(grocery.Contribution) int,
) []GroceryLine {
	rendered := make([]GroceryLine, 0, len(lines))
	for _, line := range lines {
		item, section := keys(line)
		output := GroceryLine{
			Item: item, Section: section, Kind: line.QuantityKind,
			Amount: ratString(line.Minimum), Maximum: ratString(line.Maximum), Optional: line.Optional,
		}
		if line.QuantityKind == "presence" || line.QuantityKind == "unspecified" {
			output.Amount, output.Maximum = "", ""
		} else {
			if line.Unit != nil {
				output.Unit = line.Unit.Key
			}
			if line.Package != nil {
				output.Package = &ingest.Package{Type: line.Package.Type, Size: ratString(line.Package.Size)}
				if line.Package.SizeUnit != nil {
					output.Package.Unit = line.Package.SizeUnit.Key
				}
			}
		}
		for _, contribution := range line.Contributions {
			output.SourceLines = append(output.SourceLines, sourceLine(contribution))
		}
		rendered = append(rendered, output)
	}
	return rendered
}

func rational(value *big.Rat) *grocery.Rational {
	if value == nil {
		return nil
	}
	// Shared validation already proved both parts fit 64-bit storage.
	return &grocery.Rational{Numerator: value.Num().Int64(), Denominator: value.Denom().Int64()}
}

func ratString(value *grocery.Rational) string {
	if value == nil {
		return ""
	}
	return big.NewRat(value.Numerator, value.Denominator).RatString()
}
