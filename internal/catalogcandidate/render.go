package catalogcandidate

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// RenderMarkdown produces the deterministic human review representation.
func RenderMarkdown(c Candidate) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "# %s\n\n", c.Name.Value)
	fmt.Fprintf(&b, "- Candidate: `%s`\n- Catalog key: `%s`\n- State: `%s`\n- Source: `%s` <%s>\n- Selected JSON-LD: artifact `%s`, recipe %d, script %d, path `%s`\n\n", c.CandidateID, c.CatalogKey, c.State, c.Source.ManifestID, c.Source.ManifestURL, c.Source.ArtifactFile, c.Source.Selected.CandidateIndex, c.Source.Selected.ScriptIndex, c.Source.Selected.JSONPath)
	blockers := blockerLines(c)
	fmt.Fprintf(&b, "## Blockers\n\n")
	if len(blockers) == 0 {
		b.WriteString("No blockers recorded.\n\n")
	} else {
		for _, blocker := range blockers {
			fmt.Fprintf(&b, "- **BLOCKER:** %s\n", blocker)
		}
		b.WriteByte('\n')
	}
	fmt.Fprintf(&b, "## Summary\n\n- Yield: %s\n- Hands on: %s\n- Unattended: %s\n\n", display(c.Yield.Value), displayDuration(c.HandsOn.Value), displayDuration(c.Unattended.Value))
	b.WriteString("## Ingredients\n\n")
	for _, section := range c.IngredientSections {
		fmt.Fprintf(&b, "### %s\n\n", section.Name)
		for _, ingredient := range section.Ingredients {
			flags := make([]string, 0)
			if ingredient.Optional {
				flags = append(flags, "optional")
			}
			if ingredient.NonShopping {
				flags = append(flags, "non-shopping")
			}
			flagText := ""
			if len(flags) > 0 {
				flagText = " _( " + strings.Join(flags, ", ") + " )_"
			}
			fmt.Fprintf(&b, "- [%d] %s%s\n  - Proposal: %s; quantity: %s; grocery: %s %s\n", ingredient.SourceLineIndex, ingredient.SourceText, flagText, display(ingredient.ItemPhrase), formatQuantity(ingredient.Quantity), ingredient.GroceryProposal.State, ingredient.GroceryProposal.Key)
		}
		b.WriteByte('\n')
	}
	b.WriteString("## Instructions\n\n")
	for _, section := range c.InstructionSections {
		fmt.Fprintf(&b, "### %s\n\n", section.Name)
		for i, step := range section.Steps {
			fmt.Fprintf(&b, "%d. %s\n", i+1, step)
		}
		b.WriteByte('\n')
	}
	b.WriteString("## Issues\n\n")
	if len(c.Issues) == 0 {
		b.WriteString("No recipe-level issues recorded.\n")
	} else {
		for _, issue := range c.Issues {
			fmt.Fprintf(&b, "- %s `%s`: %s\n", strings.ToUpper(issue.Severity), issue.Field, issue.Message)
		}
	}
	return b.Bytes()
}

func blockerLines(c Candidate) []string {
	out := make([]string, 0)
	for _, issue := range c.Issues {
		if issue.Severity == "blocker" {
			out = append(out, issue.Field+": "+issue.Message)
		}
	}
	for _, note := range c.SourceNotes {
		if note.State == "unresolved" || note.State == "ambiguous" {
			out = append(out, "source note "+note.Reference+" is "+note.State)
		}
	}
	for _, section := range c.IngredientSections {
		for _, ingredient := range section.Ingredients {
			if ingredient.GroceryProposal.State == "unresolved" || ingredient.GroceryProposal.State == "ambiguous" {
				out = append(out, fmt.Sprintf("ingredient %d grocery mapping is %s", ingredient.SourceLineIndex, ingredient.GroceryProposal.State))
			}
			for _, issue := range ingredient.Issues {
				if issue.Severity == "blocker" {
					out = append(out, fmt.Sprintf("ingredient %d %s: %s", ingredient.SourceLineIndex, issue.Field, issue.Message))
				}
			}
		}
	}
	sort.Strings(out)
	return out
}
func display(s string) string {
	if strings.TrimSpace(s) == "" {
		return "(unresolved)"
	}
	return s
}
func displayDuration(d *Duration) string {
	if d == nil {
		return "(unresolved)"
	}
	if d.Min == d.Max {
		return fmt.Sprintf("%d min", d.Min)
	}
	return fmt.Sprintf("%d-%d min", d.Min, d.Max)
}
func formatQuantity(q Quantity) string {
	if q.Kind == "unspecified" {
		return "unspecified"
	}
	if q.Kind == "range" {
		return q.Amount + "-" + q.Maximum + " " + q.Unit
	}
	if q.Package != nil {
		return q.Amount + " " + q.Package.Type
	}
	return q.Amount + " " + q.Unit
}
