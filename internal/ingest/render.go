package ingest

import (
	"fmt"
	"sort"
	"strings"
)

// RenderBody produces the human-readable half of an approved bootstrap recipe. The YAML
// front matter remains the ingestion input; exact body comparison prevents the readable view
// from silently drifting away from it.
func RenderBody(document Document) string {
	var out strings.Builder
	fmt.Fprintf(&out, "# %s\n\n", document.Name)
	out.WriteString("> Approved bootstrap recipe. YAML front matter is ingested into SQLite; the sections below\n")
	out.WriteString("> are the checked human-readable view.\n\n")

	out.WriteString("## Recipe details\n\n")
	if document.Source.URL != "" {
		fmt.Fprintf(&out, "- Source: [%s](%s) (`%s`)\n", document.Source.Attribution, document.Source.URL, document.Source.Relationship)
	} else {
		fmt.Fprintf(&out, "- Source: %s (`%s`)\n", document.Source.Attribution, document.Source.Relationship)
	}
	if document.Source.CheckedOn != "" {
		fmt.Fprintf(&out, "- Source checked: %s\n", document.Source.CheckedOn)
	}
	fmt.Fprintf(&out, "- Yield: %s\n", valueOrUnknown(document.Yield))
	fmt.Fprintf(&out, "- Hands-on: %s\n", formatDuration(document.HandsOn))
	fmt.Fprintf(&out, "- Unattended: %s\n", formatDuration(document.Unattended))

	out.WriteString("\n## Ingredients\n")
	for _, section := range document.IngredientSections {
		fmt.Fprintf(&out, "\n### %s\n\n", section.Name)
		for _, ingredient := range section.Ingredients {
			fmt.Fprintf(&out, "- %s\n", ingredient.SourceText)
			if ingredient.NonShopping {
				fmt.Fprintf(&out, "  - Recipe only: %s — not added to the grocery list\n", formatShoppingRequirement(ingredient))
			} else {
				fmt.Fprintf(&out, "  - Shopping: %s — %s\n", formatShoppingRequirement(ingredient), ingredient.GroceryItem.StoreSection.Name)
			}
			if ingredient.Preparation != "" {
				fmt.Fprintf(&out, "  - Preparation: %s\n", ingredient.Preparation)
			}
			if ingredient.Optional {
				out.WriteString("  - Optional: yes\n")
			}
			if ingredient.Note != "" {
				fmt.Fprintf(&out, "  - Note: %s\n", ingredient.Note)
			}
		}
	}

	out.WriteString("\n## Instructions\n")
	for _, section := range document.InstructionSections {
		fmt.Fprintf(&out, "\n### %s\n\n", section.Name)
		for index, step := range section.Steps {
			fmt.Fprintf(&out, "%d. %s\n", index+1, step)
		}
	}

	out.WriteString("\n## One-batch grocery preview\n")
	bySection := make(map[string][]Ingredient)
	for _, section := range document.IngredientSections {
		for _, ingredient := range section.Ingredients {
			if ingredient.NonShopping {
				continue
			}
			name := ingredient.GroceryItem.StoreSection.Name
			bySection[name] = append(bySection[name], ingredient)
		}
	}
	sectionNames := make([]string, 0, len(bySection))
	for section := range bySection {
		sectionNames = append(sectionNames, section)
	}
	sort.Strings(sectionNames)
	for _, section := range sectionNames {
		fmt.Fprintf(&out, "\n### %s\n\n", section)
		for _, ingredient := range bySection[section] {
			fmt.Fprintf(&out, "- %s\n", formatShoppingRequirement(ingredient))
		}
	}

	if len(document.Review) > 0 {
		out.WriteString("\n## Approved true-up decisions\n\n")
		for _, decision := range document.Review {
			fmt.Fprintf(&out, "- `%s` — **%s:** %s\n", decision.Field, decision.Kind, decision.Note)
		}
	}
	return strings.TrimSpace(out.String()) + "\n"
}

func formatDuration(duration *Duration) string {
	if duration == nil {
		return "unknown"
	}
	if duration.Min == duration.Max {
		return fmt.Sprintf("%d minutes", duration.Min)
	}
	return fmt.Sprintf("%d–%d minutes", duration.Min, duration.Max)
}

func valueOrUnknown(value string) string {
	if value == "" {
		return "unknown"
	}
	return value
}

func formatShoppingRequirement(ingredient Ingredient) string {
	quantity := formatQuantity(ingredient.Quantity)
	line := strings.TrimSpace(quantity + " " + ingredient.GroceryItem.Name)
	if ingredient.Optional {
		line += " — optional"
	}
	if ingredient.Note != "" {
		line += " — `" + ingredient.Note + "`"
	}
	return line
}

func formatQuantity(quantity Quantity) string {
	if quantity.Kind == "unspecified" {
		return ""
	}
	amount := quantity.Amount
	if quantity.Kind == "range" {
		amount += "–" + quantity.Maximum
	}
	if quantity.Package != nil {
		packageName := quantity.Package.Type
		if amount != "1" {
			packageName = simplePlural(packageName)
		}
		if quantity.Package.Size == "" {
			return amount + " " + packageName
		}
		if quantity.Package.Unit == "each" {
			return fmt.Sprintf("%s × %s-count %s", amount, quantity.Package.Size, packageName)
		}
		return fmt.Sprintf("%s × %s %s %s", amount, quantity.Package.Size, displayUnit(quantity.Package.Unit, quantity.Package.Size), packageName)
	}
	unit := displayUnit(quantity.Unit, amount)
	if unit == "" {
		return amount
	}
	return amount + " " + unit
}

func displayUnit(key, amount string) string {
	switch key {
	case "each":
		return ""
	case "slice":
		if amount == "1" {
			return "slice"
		}
		return "slices"
	default:
		return key
	}
}

func simplePlural(value string) string {
	if strings.HasSuffix(value, "s") {
		return value
	}
	return value + "s"
}
