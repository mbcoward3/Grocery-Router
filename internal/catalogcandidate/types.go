// Package catalogcandidate builds local, reviewable shared-catalog recipe candidates
// from recipepilot evidence and untrusted agent proposal JSON. It performs no
// network, model, or database work.
package catalogcandidate

import "time"

const (
	FormatVersion  = 1
	MaxBatchSize   = 25
	Implementation = "grocery-router-catalog-candidate/v1"
)

type Candidate struct {
	FormatVersion       int                  `json:"format_version"`
	CandidateID         string               `json:"candidate_id"`
	CatalogKey          string               `json:"catalog_key"`
	State               string               `json:"state"`
	Source              SourceIdentity       `json:"source"`
	Agent               AgentProvenance      `json:"agent"`
	Name                Field[string]        `json:"name"`
	ImageURL            Field[string]        `json:"image_url,omitempty"`
	Yield               Field[string]        `json:"yield"`
	HandsOn             Field[*Duration]     `json:"hands_on"`
	Unattended          Field[*Duration]     `json:"unattended"`
	IngredientSections  []IngredientSection  `json:"ingredient_sections"`
	InstructionSections []InstructionSection `json:"instruction_sections"`
	Issues              []Issue              `json:"issues"`
	SourceNotes         []SourceNote         `json:"source_notes"`
	Decisions           []Decision           `json:"decisions"`
}

type SourceIdentity struct {
	PilotImplementation string    `json:"pilot_implementation"`
	PilotFormatVersion  int       `json:"pilot_format_version"`
	ManifestID          string    `json:"manifest_id"`
	ManifestURL         string    `json:"manifest_url"`
	RunIndexSHA256      string    `json:"run_index_sha256"`
	ArtifactFile        string    `json:"artifact_file"`
	ArtifactSHA256      string    `json:"artifact_sha256"`
	RequestedURL        string    `json:"requested_url"`
	FinalURL            string    `json:"final_url"`
	FetchedAt           time.Time `json:"fetched_at"`
	ResponseBodySHA256  string    `json:"response_body_sha256"`
	Selected            Selection `json:"selected"`
}

type Selection struct {
	CandidateIndex     int    `json:"candidate_index"`
	ScriptIndex        int    `json:"script_index"`
	JSONPath           string `json:"json_path"`
	JSONLDBlockSHA256  string `json:"json_ld_block_sha256"`
	RecipeRawSHA256    string `json:"recipe_raw_sha256"`
	RecipeRawCanonical string `json:"recipe_raw_canonical_sha256"`
}

type AgentProvenance struct {
	Kind         string    `json:"kind"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	PromptSHA256 string    `json:"prompt_sha256"`
	InputSHA256  string    `json:"input_sha256"`
	OutputSHA256 string    `json:"output_sha256"`
	RunAt        time.Time `json:"run_at"`
}

type Field[T any] struct {
	State      string `json:"state"`
	Value      T      `json:"value,omitempty"`
	Source     string `json:"source,omitempty"`
	Provenance string `json:"provenance"`
}

type Duration struct {
	Min int64 `json:"min"`
	Max int64 `json:"max"`
}

type IngredientSection struct {
	Name        string       `json:"name"`
	Ingredients []Ingredient `json:"ingredients"`
}

type Ingredient struct {
	SourceLineIndex int             `json:"source_line_index"`
	SourceText      string          `json:"source_text"`
	Quantity        Quantity        `json:"quantity"`
	ItemPhrase      string          `json:"item_phrase"`
	Preparation     string          `json:"preparation,omitempty"`
	Optional        bool            `json:"optional,omitempty"`
	NonShopping     bool            `json:"non_shopping,omitempty"`
	Alternatives    []string        `json:"alternatives,omitempty"`
	GroceryProposal GroceryProposal `json:"grocery_proposal"`
	Issues          []Issue         `json:"issues,omitempty"`
}

type Quantity struct {
	Kind        string   `json:"kind"`
	Amount      string   `json:"amount,omitempty"`
	Maximum     string   `json:"maximum,omitempty"`
	Unit        string   `json:"unit,omitempty"`
	Equivalents []string `json:"equivalents,omitempty"`
	Package     *Package `json:"package,omitempty"`
}

type Package struct {
	Count string `json:"count,omitempty"`
	Type  string `json:"type"`
	Size  string `json:"size,omitempty"`
	Unit  string `json:"unit,omitempty"`
}

type GroceryProposal struct {
	State        string `json:"state"`
	Key          string `json:"key,omitempty"`
	Name         string `json:"name,omitempty"`
	StoreSection string `json:"store_section,omitempty"`
	ShoppingMode string `json:"shopping_mode,omitempty"`
	Rationale    string `json:"rationale,omitempty"`
}

type InstructionSection struct {
	Name  string   `json:"name"`
	Steps []string `json:"steps"`
}

type Issue struct {
	Severity string `json:"severity"`
	Field    string `json:"field"`
	Message  string `json:"message"`
}

type SourceNote struct {
	Reference string `json:"reference"`
	Text      string `json:"text,omitempty"`
	State     string `json:"state"`
}

type Decision struct {
	Field     string `json:"field"`
	State     string `json:"state"`
	Rationale string `json:"rationale"`
}
