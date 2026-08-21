// Package grocery deterministically aggregates approved recipe requirements.
package grocery

import (
	"fmt"
	"math/big"
)

// Rational is an exact SQLite-safe fraction.
type Rational struct {
	Numerator   int64
	Denominator int64
}

// Unit describes an exact conversion to its dimension's base unit.
type Unit struct {
	ID                int64
	Key               string
	Dimension         string
	ToBaseNumerator   int64
	ToBaseDenominator int64
}

// Package preserves a source package type and optional exact package size.
type Package struct {
	Type     string
	Size     *Rational
	SizeUnit *Unit
}

// Requirement is one shoppable ingredient from one week-recipe occurrence.
type Requirement struct {
	WeekRecipeID       int64
	RecipeIngredientID int64
	RecipeName         string
	GroceryItemID      int64
	GroceryItemKey     string
	GroceryItemName    string
	ShoppingMode       string
	StoreSectionID     int64
	StoreSectionName   string
	QuantityKind       string
	Minimum            *Rational
	Maximum            *Rational
	Unit               *Unit
	Package            *Package
	Optional           bool
}

// Contribution preserves one requirement behind an aggregated line.
type Contribution struct {
	WeekRecipeID       int64
	RecipeIngredientID int64
	RecipeName         string
	QuantityKind       string
	Minimum            *Rational
	Maximum            *Rational
	Unit               *Unit
	Package            *Package
	Optional           bool
}

// Line is one deterministic generated grocery line.
type Line struct {
	Key              string
	GroceryItemID    int64
	GroceryItemName  string
	StoreSectionID   int64
	StoreSectionName string
	QuantityKind     string
	Minimum          *Rational
	Maximum          *Rational
	Unit             *Unit
	Package          *Package
	Optional         bool
	Contributions    []Contribution
}

// Aggregate combines only universally compatible requirements. Input order controls the
// selected display unit and stable output order; arithmetic itself is exact.
func Aggregate(requirements []Requirement) ([]Line, error) {
	lines := make([]Line, 0, len(requirements))
	byKey := make(map[string]int, len(requirements))
	for _, requirement := range requirements {
		if err := validateRequirement(requirement); err != nil {
			return nil, err
		}
		key := aggregationKey(requirement)
		index, exists := byKey[key]
		if !exists {
			line := Line{
				Key: key, GroceryItemID: requirement.GroceryItemID,
				GroceryItemName:  requirement.GroceryItemName,
				StoreSectionID:   requirement.StoreSectionID,
				StoreSectionName: requirement.StoreSectionName,
				Optional:         true,
			}
			if requirement.ShoppingMode == "presence-only" {
				line.QuantityKind = "presence"
			} else {
				line.QuantityKind = requirement.QuantityKind
				line.Unit = requirement.Unit
				line.Package = requirement.Package
			}
			lines = append(lines, line)
			index = len(lines) - 1
			byKey[key] = index
		}

		line := &lines[index]
		line.Optional = line.Optional && requirement.Optional
		line.Contributions = append(line.Contributions, contribution(requirement))
		if line.QuantityKind == "presence" || requirement.QuantityKind == "unspecified" {
			continue
		}
		minimum, err := convert(requirement.Minimum, requirement.Unit, line.Unit, requirement.Package != nil)
		if err != nil {
			return nil, fmt.Errorf("aggregate %s: %w", requirement.GroceryItemKey, err)
		}
		maximum := minimum
		if requirement.QuantityKind == "range" {
			maximum, err = convert(requirement.Maximum, requirement.Unit, line.Unit, requirement.Package != nil)
			if err != nil {
				return nil, fmt.Errorf("aggregate %s: %w", requirement.GroceryItemKey, err)
			}
		}
		line.Minimum, err = add(line.Minimum, minimum)
		if err != nil {
			return nil, fmt.Errorf("aggregate %s minimum: %w", requirement.GroceryItemKey, err)
		}
		line.Maximum, err = add(line.Maximum, maximum)
		if err != nil {
			return nil, fmt.Errorf("aggregate %s maximum: %w", requirement.GroceryItemKey, err)
		}
		if line.Minimum.Numerator == line.Maximum.Numerator && line.Minimum.Denominator == line.Maximum.Denominator {
			line.QuantityKind = "exact"
		} else {
			line.QuantityKind = "range"
		}
	}
	for index := range lines {
		if lines[index].QuantityKind == "exact" {
			lines[index].Maximum = nil
		}
	}
	return lines, nil
}

func validateRequirement(requirement Requirement) error {
	if requirement.GroceryItemID == 0 || requirement.GroceryItemKey == "" || requirement.GroceryItemName == "" {
		return fmt.Errorf("requirement has no canonical grocery item")
	}
	if requirement.ShoppingMode != "measured" && requirement.ShoppingMode != "counted" && requirement.ShoppingMode != "presence-only" {
		return fmt.Errorf("grocery item %s has invalid shopping mode %q", requirement.GroceryItemKey, requirement.ShoppingMode)
	}
	if requirement.ShoppingMode == "presence-only" {
		return nil
	}
	if requirement.QuantityKind == "unspecified" {
		return nil
	}
	if requirement.QuantityKind != "exact" && requirement.QuantityKind != "range" {
		return fmt.Errorf("grocery item %s has invalid quantity kind %q", requirement.GroceryItemKey, requirement.QuantityKind)
	}
	if requirement.Minimum == nil || (requirement.QuantityKind == "range" && requirement.Maximum == nil) {
		return fmt.Errorf("grocery item %s has incomplete numeric quantity", requirement.GroceryItemKey)
	}
	if requirement.Package == nil && requirement.Unit == nil {
		return fmt.Errorf("grocery item %s has no unit or package", requirement.GroceryItemKey)
	}
	return nil
}

func aggregationKey(requirement Requirement) string {
	prefix := fmt.Sprintf("item:%d:", requirement.GroceryItemID)
	if requirement.ShoppingMode == "presence-only" {
		return prefix + "presence"
	}
	if requirement.QuantityKind == "unspecified" {
		return fmt.Sprintf("%sunspecified:%d:%d", prefix, requirement.WeekRecipeID, requirement.RecipeIngredientID)
	}
	if requirement.Package != nil {
		size := "unsized"
		if requirement.Package.Size != nil {
			size = fmt.Sprintf("%d/%d:%s", requirement.Package.Size.Numerator,
				requirement.Package.Size.Denominator, requirement.Package.SizeUnit.Key)
		}
		return fmt.Sprintf("%spackage:%s:%s", prefix, requirement.Package.Type, size)
	}
	if requirement.Unit.Dimension == "mass" || requirement.Unit.Dimension == "volume" {
		return prefix + "dimension:" + requirement.Unit.Dimension
	}
	return prefix + "unit:" + requirement.Unit.Key
}

func contribution(requirement Requirement) Contribution {
	return Contribution{
		WeekRecipeID: requirement.WeekRecipeID, RecipeIngredientID: requirement.RecipeIngredientID,
		RecipeName: requirement.RecipeName, QuantityKind: requirement.QuantityKind,
		Minimum: requirement.Minimum, Maximum: requirement.Maximum, Unit: requirement.Unit,
		Package: requirement.Package, Optional: requirement.Optional,
	}
}

func convert(value *Rational, from, to *Unit, packaged bool) (*Rational, error) {
	if value == nil {
		return nil, fmt.Errorf("missing quantity")
	}
	result := rat(value)
	if !packaged {
		if from == nil || to == nil || from.Dimension != to.Dimension {
			return nil, fmt.Errorf("incompatible units")
		}
		result.Mul(result, big.NewRat(from.ToBaseNumerator, from.ToBaseDenominator))
		result.Quo(result, big.NewRat(to.ToBaseNumerator, to.ToBaseDenominator))
	}
	return fromRat(result)
}

func add(left, right *Rational) (*Rational, error) {
	if left == nil {
		return &Rational{Numerator: right.Numerator, Denominator: right.Denominator}, nil
	}
	result := new(big.Rat).Add(rat(left), rat(right))
	return fromRat(result)
}

func rat(value *Rational) *big.Rat {
	return new(big.Rat).SetFrac(big.NewInt(value.Numerator), big.NewInt(value.Denominator))
}

func fromRat(value *big.Rat) (*Rational, error) {
	if !value.Num().IsInt64() || !value.Denom().IsInt64() {
		return nil, fmt.Errorf("quantity %s exceeds SQLite integer range", value.RatString())
	}
	return &Rational{Numerator: value.Num().Int64(), Denominator: value.Denom().Int64()}, nil
}
