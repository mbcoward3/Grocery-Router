package catalogcandidate

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"regexp"
	"strings"
)

var keyPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

var knownUnits = map[string]bool{
	"tsp": true, "teaspoon": true, "teaspoons": true,
	"tbsp": true, "tablespoon": true, "tablespoons": true,
	"fl-oz": true, "fluid-ounce": true, "fluid-ounces": true,
	"cup": true, "cups": true, "pt": true, "pint": true, "pints": true,
	"qt": true, "quart": true, "quarts": true, "gal": true, "gallon": true, "gallons": true,
	"ml": true, "milliliter": true, "milliliters": true, "l": true, "liter": true, "liters": true,
	"oz": true, "ounce": true, "ounces": true, "lb": true, "lbs": true, "pound": true, "pounds": true,
	"g": true, "gram": true, "grams": true, "kg": true, "kilogram": true, "kilograms": true,
	"count": true, "clove": true, "cloves": true, "slice": true, "slices": true,
	"pinch": true, "pinches": true, "dash": true, "dashes": true, "bunch": true, "bunches": true,
	"sprig": true, "sprigs": true, "can": true, "cans": true, "jar": true, "jars": true, "package": true, "packages": true,
}

func (c Candidate) Validate(sourceIngredientLines []string) error {
	if c.FormatVersion != FormatVersion {
		return fmt.Errorf("format_version = %d, want %d", c.FormatVersion, FormatVersion)
	}
	if !keyPattern.MatchString(c.CatalogKey) {
		return fmt.Errorf("invalid catalog_key %q", c.CatalogKey)
	}
	if c.State != "draft" && c.State != "reviewable" && c.State != "approved" {
		return fmt.Errorf("invalid state %q", c.State)
	}
	if strings.TrimSpace(c.CandidateID) == "" {
		return fmt.Errorf("candidate_id is required")
	}
	if err := c.Source.validate(); err != nil {
		return err
	}
	if err := validateField("name", c.Name.State, c.Name.Value, true); err != nil {
		return err
	}
	if err := validateField("yield", c.Yield.State, c.Yield.Value, false); err != nil {
		return err
	}
	if err := validateDurationField("hands_on", c.HandsOn); err != nil {
		return err
	}
	if err := validateDurationField("unattended", c.Unattended); err != nil {
		return err
	}
	if c.Issues == nil {
		return fmt.Errorf("issues must be explicit (use [] when none)")
	}
	if c.SourceNotes == nil {
		return fmt.Errorf("source_notes must be explicit (use [] when none)")
	}
	if c.Decisions == nil {
		return fmt.Errorf("decisions must be explicit (use [] when none)")
	}
	if len(c.IngredientSections) == 0 {
		return fmt.Errorf("ingredient_sections are required")
	}
	seenLines := make([]string, 0)
	for si, section := range c.IngredientSections {
		if strings.TrimSpace(section.Name) == "" || len(section.Ingredients) == 0 {
			return fmt.Errorf("ingredient section %d needs name and ingredients", si)
		}
		for ii, ingredient := range section.Ingredients {
			loc := fmt.Sprintf("ingredient_sections[%d].ingredients[%d]", si, ii)
			if ingredient.SourceLineIndex != len(seenLines) {
				return fmt.Errorf("%s source_line_index=%d, want %d", loc, ingredient.SourceLineIndex, len(seenLines))
			}
			if strings.TrimSpace(ingredient.SourceText) == "" {
				return fmt.Errorf("%s source_text is required", loc)
			}
			seenLines = append(seenLines, ingredient.SourceText)
			if err := ingredient.Quantity.validate(); err != nil {
				return fmt.Errorf("%s quantity: %w", loc, err)
			}
			if strings.TrimSpace(ingredient.ItemPhrase) == "" && !ingredient.NonShopping {
				return fmt.Errorf("%s item_phrase is required", loc)
			}
			if err := ingredient.GroceryProposal.validate(); err != nil {
				return fmt.Errorf("%s grocery_proposal: %w", loc, err)
			}
			for issueIndex, issue := range ingredient.Issues {
				if err := issue.validate(); err != nil {
					return fmt.Errorf("%s issues[%d]: %w", loc, issueIndex, err)
				}
			}
		}
	}
	if sourceIngredientLines != nil {
		if len(seenLines) != len(sourceIngredientLines) {
			return fmt.Errorf("ingredient source-line coverage = %d, want %d", len(seenLines), len(sourceIngredientLines))
		}
		for i := range sourceIngredientLines {
			if seenLines[i] != sourceIngredientLines[i] {
				return fmt.Errorf("ingredient source-line %d = %q, want %q", i, seenLines[i], sourceIngredientLines[i])
			}
		}
	}
	if len(c.InstructionSections) == 0 {
		return fmt.Errorf("instruction_sections are required")
	}
	for si, section := range c.InstructionSections {
		if strings.TrimSpace(section.Name) == "" || len(section.Steps) == 0 {
			return fmt.Errorf("instruction section %d needs name and steps", si)
		}
		for stepIndex, step := range section.Steps {
			if strings.TrimSpace(step) == "" {
				return fmt.Errorf("instruction_sections[%d].steps[%d] is empty", si, stepIndex)
			}
		}
	}
	for i, issue := range c.Issues {
		if err := issue.validate(); err != nil {
			return fmt.Errorf("issues[%d]: %w", i, err)
		}
	}
	for i, note := range c.SourceNotes {
		if strings.TrimSpace(note.Reference) == "" || !validState(note.State) {
			return fmt.Errorf("source_notes[%d] needs reference and valid state", i)
		}
	}
	for i, decision := range c.Decisions {
		if strings.TrimSpace(decision.Field) == "" || !validState(decision.State) || strings.TrimSpace(decision.Rationale) == "" {
			return fmt.Errorf("decisions[%d] needs field, state, and rationale", i)
		}
	}
	return nil
}

func DecodeCandidate(data []byte) (Candidate, error) {
	var c Candidate
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return Candidate{}, err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return Candidate{}, fmt.Errorf("trailing JSON")
	}
	return c, nil
}

func (s SourceIdentity) validate() error {
	for name, value := range map[string]string{
		"pilot_implementation": s.PilotImplementation, "manifest_id": s.ManifestID, "manifest_url": s.ManifestURL,
		"run_index_sha256": s.RunIndexSHA256, "artifact_file": s.ArtifactFile, "artifact_sha256": s.ArtifactSHA256,
		"requested_url": s.RequestedURL, "final_url": s.FinalURL, "response_body_sha256": s.ResponseBodySHA256,
		"json_path": s.Selected.JSONPath, "json_ld_block_sha256": s.Selected.JSONLDBlockSHA256,
		"recipe_raw_sha256": s.Selected.RecipeRawSHA256, "recipe_raw_canonical_sha256": s.Selected.RecipeRawCanonical,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("source %s is required", name)
		}
	}
	if s.PilotFormatVersion != 1 {
		return fmt.Errorf("source pilot_format_version = %d, want 1", s.PilotFormatVersion)
	}
	if s.Selected.CandidateIndex < 0 || s.Selected.ScriptIndex < 0 {
		return fmt.Errorf("source selected indexes must be non-negative")
	}
	return nil
}

func validateField(name, state, value string, required bool) error {
	if !validState(state) {
		return fmt.Errorf("%s has invalid state %q", name, state)
	}
	if required && strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", name)
	}
	return nil
}
func validateDurationField(name string, field Field[*Duration]) error {
	if !validState(field.State) {
		return fmt.Errorf("%s has invalid state %q", name, field.State)
	}
	if field.Value != nil && (field.Value.Min < 0 || field.Value.Max < field.Value.Min) {
		return fmt.Errorf("%s range is invalid", name)
	}
	return nil
}
func validState(state string) bool {
	switch state {
	case "resolved", "ambiguous", "unresolved", "not_applicable", "approved":
		return true
	}
	return false
}

func (q Quantity) validate() error {
	switch q.Kind {
	case "exact", "range":
	default:
		if q.Kind == "unspecified" {
			if q.Amount != "" || q.Maximum != "" || q.Unit != "" || q.Package != nil {
				return fmt.Errorf("unspecified quantity cannot carry amount, maximum, unit, or package")
			}
			return nil
		}
		return fmt.Errorf("invalid kind %q", q.Kind)
	}
	amount, ok := new(big.Rat).SetString(q.Amount)
	if !ok || amount.Sign() <= 0 {
		return fmt.Errorf("invalid amount %q", q.Amount)
	}
	if q.Kind == "range" {
		maximum, ok := new(big.Rat).SetString(q.Maximum)
		if !ok || maximum.Sign() <= 0 || amount.Cmp(maximum) > 0 {
			return fmt.Errorf("invalid maximum %q", q.Maximum)
		}
	} else if q.Maximum != "" {
		return fmt.Errorf("exact quantity cannot carry maximum")
	}
	if (q.Unit == "") == (q.Package == nil) {
		return fmt.Errorf("numeric quantity needs exactly one of unit or package")
	}
	if q.Unit != "" && !knownUnits[q.Unit] {
		return fmt.Errorf("unknown unit %q", q.Unit)
	}
	if q.Package != nil {
		if strings.TrimSpace(q.Package.Type) == "" {
			return fmt.Errorf("package type is required")
		}
		if q.Package.Unit != "" && !knownUnits[q.Package.Unit] {
			return fmt.Errorf("unknown package unit %q", q.Package.Unit)
		}
		if q.Package.Count != "" {
			if _, ok := new(big.Rat).SetString(q.Package.Count); !ok {
				return fmt.Errorf("invalid package count %q", q.Package.Count)
			}
		}
		if q.Package.Size != "" {
			if _, ok := new(big.Rat).SetString(q.Package.Size); !ok {
				return fmt.Errorf("invalid package size %q", q.Package.Size)
			}
		}
	}
	return nil
}

func (g GroceryProposal) validate() error {
	switch g.State {
	case "match", "create", "ambiguous", "unresolved":
	default:
		return fmt.Errorf("invalid state %q", g.State)
	}
	if (g.State == "match" || g.State == "create") && (!keyPattern.MatchString(g.Key) || strings.TrimSpace(g.Name) == "" || strings.TrimSpace(g.StoreSection) == "" || strings.TrimSpace(g.ShoppingMode) == "") {
		return fmt.Errorf("resolved proposal needs key, name, store_section, and shopping_mode")
	}
	return nil
}
func (i Issue) validate() error {
	if i.Severity != "blocker" && i.Severity != "warning" && i.Severity != "info" {
		return fmt.Errorf("invalid severity %q", i.Severity)
	}
	if strings.TrimSpace(i.Field) == "" || strings.TrimSpace(i.Message) == "" {
		return fmt.Errorf("field and message are required")
	}
	return nil
}
