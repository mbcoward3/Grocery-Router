package catalogcandidate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

type Proposal struct {
	Key                 string                      `json:"key"`
	Name                string                      `json:"name"`
	ImageURL            string                      `json:"image_url"`
	Yield               string                      `json:"yield"`
	HandsOn             *Duration                   `json:"hands_on"`
	Unattended          *Duration                   `json:"unattended"`
	IngredientSections  []ProposalIngredientSection `json:"ingredient_sections"`
	InstructionSections []InstructionSection        `json:"instruction_sections"`
	Issues              []Issue                     `json:"issues"`
	SourceNotes         []SourceNote                `json:"source_notes"`
	Decisions           []Decision                  `json:"decisions"`
	Agent               AgentProvenance             `json:"agent,omitempty"`
}

type ProposalIngredientSection struct {
	Name        string               `json:"name"`
	Ingredients []ProposalIngredient `json:"ingredients"`
}

type ProposalIngredient struct {
	SourceText      string          `json:"source_text"`
	SourceLine      string          `json:"source_line"`
	SourceLineIndex *int            `json:"source_line_index"`
	Quantity        Quantity        `json:"quantity"`
	ItemPhrase      string          `json:"item_phrase"`
	Preparation     string          `json:"preparation"`
	Optional        bool            `json:"optional"`
	NonShopping     bool            `json:"non_shopping"`
	Alternatives    []string        `json:"alternatives"`
	GroceryProposal GroceryProposal `json:"grocery_proposal"`
	Issues          []Issue         `json:"issues"`
}

func DecodeProposal(r io.Reader) (Proposal, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return Proposal{}, fmt.Errorf("read proposal: %w", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return Proposal{}, fmt.Errorf("decode proposal: %w", err)
	}
	for _, required := range []string{"key", "name", "image_url", "yield", "hands_on", "unattended", "ingredient_sections", "instruction_sections", "issues", "source_notes", "decisions"} {
		if _, ok := fields[required]; !ok {
			return Proposal{}, fmt.Errorf("decode proposal: missing required field %q", required)
		}
	}
	var p Proposal
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return Proposal{}, fmt.Errorf("decode proposal: %w", err)
	}
	if dec.Decode(&struct{}{}) != io.EOF {
		return Proposal{}, fmt.Errorf("decode proposal: trailing JSON")
	}
	return p, nil
}

func proposalFromBytes(data []byte) (Proposal, error) { return DecodeProposal(bytes.NewReader(data)) }

func (p Proposal) toCandidate(source SourceIdentity, agent AgentProvenance) Candidate {
	if p.Agent.Kind != "" || p.Agent.Provider != "" || p.Agent.Model != "" {
		agent = p.Agent
	}
	if agent.Kind == "" {
		agent.Kind = "pi-proposed"
	}
	if agent.RunAt.IsZero() {
		agent.RunAt = time.Unix(0, 0).UTC()
	}
	sections := make([]IngredientSection, 0, len(p.IngredientSections))
	line := 0
	for _, s := range p.IngredientSections {
		out := IngredientSection{Name: s.Name, Ingredients: make([]Ingredient, 0, len(s.Ingredients))}
		for _, ing := range s.Ingredients {
			idx := line
			if ing.SourceLineIndex != nil {
				idx = *ing.SourceLineIndex
			}
			text := ing.SourceText
			if text == "" {
				text = ing.SourceLine
			}
			out.Ingredients = append(out.Ingredients, Ingredient{
				SourceLineIndex: idx, SourceText: text, Quantity: ing.Quantity, ItemPhrase: ing.ItemPhrase,
				Preparation: ing.Preparation, Optional: ing.Optional, NonShopping: ing.NonShopping,
				Alternatives: ing.Alternatives, GroceryProposal: ing.GroceryProposal, Issues: ing.Issues,
			})
			line++
		}
		sections = append(sections, out)
	}
	return Candidate{
		FormatVersion: FormatVersion, CandidateID: source.ManifestID + "#" + source.Selected.JSONPath,
		CatalogKey: p.Key, State: "reviewable", Source: source, Agent: agent,
		Name:               Field[string]{State: "resolved", Value: p.Name, Provenance: "pi-proposed"},
		ImageURL:           Field[string]{State: fieldState(p.ImageURL), Value: p.ImageURL, Provenance: "pi-proposed"},
		Yield:              Field[string]{State: fieldState(p.Yield), Value: p.Yield, Provenance: "pi-proposed"},
		HandsOn:            Field[*Duration]{State: pointerFieldState(p.HandsOn), Value: p.HandsOn, Provenance: "pi-proposed"},
		Unattended:         Field[*Duration]{State: pointerFieldState(p.Unattended), Value: p.Unattended, Provenance: "pi-proposed"},
		IngredientSections: sections, InstructionSections: p.InstructionSections,
		Issues: p.Issues, SourceNotes: p.SourceNotes, Decisions: p.Decisions,
	}
}

func fieldState(v string) string {
	if v == "" {
		return "unresolved"
	}
	return "resolved"
}
func pointerFieldState[T any](v *T) string {
	if v == nil {
		return "unresolved"
	}
	return "resolved"
}
