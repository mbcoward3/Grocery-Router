// Package recipepilot fetches explicitly manifested public recipe pages and
// preserves source-provided Schema.org Recipe JSON-LD as inspectable evidence.
package recipepilot

import (
	"encoding/json"
	"time"
)

// Pilot format versions and implementation identity are written into evidence artifacts.
const (
	ManifestVersion = 1
	OutputVersion   = 1
	Implementation  = "grocery-router-go-jsonld/v1"
)

// Manifest is the strict checked-in list of sources to fetch.
type Manifest struct {
	Version int              `yaml:"version"`
	Sources []ManifestSource `yaml:"sources"`
}

// ManifestSource identifies one explicitly allowed source page.
type ManifestSource struct {
	ID  string `yaml:"id" json:"id"`
	URL string `yaml:"url" json:"url"`
}

// Redirect records one HTTP redirect without discarding either URL.
type Redirect struct {
	Status int    `json:"status"`
	From   string `json:"from"`
	To     string `json:"to"`
}

// FetchMetadata describes the request and accepted response.
type FetchMetadata struct {
	RequestedURL string     `json:"requested_url"`
	FinalURL     string     `json:"final_url"`
	Redirects    []Redirect `json:"redirects"`
	FetchedAt    time.Time  `json:"fetched_at"`
	Status       int        `json:"status"`
	MediaType    string     `json:"media_type"`
	ETag         string     `json:"etag,omitempty"`
	LastModified string     `json:"last_modified,omitempty"`
	DecodedBytes int64      `json:"decoded_bytes"`
	BodySHA256   string     `json:"body_sha256"`
}

// FetchResult contains decoded response bytes and their provenance.
type FetchResult struct {
	Metadata FetchMetadata
	Body     []byte
}

// CanonicalLink preserves a page's canonical href and its resolved form.
type CanonicalLink struct {
	Href     string `json:"href"`
	Resolved string `json:"resolved,omitempty"`
}

// JSONLDBlock identifies and accounts for one JSON-LD script.
type JSONLDBlock struct {
	ScriptIndex int    `json:"script_index"`
	Bytes       int    `json:"bytes"`
	SHA256      string `json:"sha256"`
	ParseError  string `json:"parse_error,omitempty"`
}

// SourceRecipe keeps common Schema.org fields in their source-provided JSON
// shape. Raw contains the complete Recipe object, including unknown fields.
type SourceRecipe struct {
	Context            json.RawMessage `json:"@context,omitempty"`
	Type               json.RawMessage `json:"@type,omitempty"`
	ID                 json.RawMessage `json:"@id,omitempty"`
	Name               json.RawMessage `json:"name,omitempty"`
	Headline           json.RawMessage `json:"headline,omitempty"`
	Description        json.RawMessage `json:"description,omitempty"`
	URL                json.RawMessage `json:"url,omitempty"`
	MainEntityOfPage   json.RawMessage `json:"mainEntityOfPage,omitempty"`
	Image              json.RawMessage `json:"image,omitempty"`
	Author             json.RawMessage `json:"author,omitempty"`
	Publisher          json.RawMessage `json:"publisher,omitempty"`
	DatePublished      json.RawMessage `json:"datePublished,omitempty"`
	DateModified       json.RawMessage `json:"dateModified,omitempty"`
	RecipeYield        json.RawMessage `json:"recipeYield,omitempty"`
	PrepTime           json.RawMessage `json:"prepTime,omitempty"`
	CookTime           json.RawMessage `json:"cookTime,omitempty"`
	TotalTime          json.RawMessage `json:"totalTime,omitempty"`
	RecipeCategory     json.RawMessage `json:"recipeCategory,omitempty"`
	RecipeCuisine      json.RawMessage `json:"recipeCuisine,omitempty"`
	Keywords           json.RawMessage `json:"keywords,omitempty"`
	RecipeIngredient   json.RawMessage `json:"recipeIngredient,omitempty"`
	RecipeInstructions json.RawMessage `json:"recipeInstructions,omitempty"`
	Nutrition          json.RawMessage `json:"nutrition,omitempty"`
	AggregateRating    json.RawMessage `json:"aggregateRating,omitempty"`
	Video              json.RawMessage `json:"video,omitempty"`
	Raw                json.RawMessage `json:"raw"`
}

// RecipeCandidate ties a source recipe to its containing JSON-LD and location.
type RecipeCandidate struct {
	ScriptIndex int             `json:"script_index"`
	JSONPath    string          `json:"json_path"`
	JSONLD      json.RawMessage `json:"json_ld"`
	Recipe      SourceRecipe    `json:"recipe"`
}

// SourceArtifact is the inspectable output for one manifest source.
type SourceArtifact struct {
	FormatVersion int               `json:"format_version"`
	Source        ManifestSource    `json:"source"`
	Fetch         *FetchMetadata    `json:"fetch,omitempty"`
	Canonical     []CanonicalLink   `json:"canonical_links,omitempty"`
	JSONLDBlocks  []JSONLDBlock     `json:"json_ld_blocks,omitempty"`
	Recipes       []RecipeCandidate `json:"recipes,omitempty"`
	Warnings      []string          `json:"warnings,omitempty"`
	Error         string            `json:"error,omitempty"`
}

// ArtifactIndex summarizes one source in a run index.
type ArtifactIndex struct {
	SourceID string `json:"source_id"`
	File     string `json:"file"`
	Status   string `json:"status"`
	Error    string `json:"error,omitempty"`
}

// RunIndex summarizes a complete manifest run.
type RunIndex struct {
	FormatVersion  int             `json:"format_version"`
	Implementation string          `json:"implementation"`
	StartedAt      time.Time       `json:"started_at"`
	ManifestPath   string          `json:"manifest_path"`
	ManifestSHA256 string          `json:"manifest_sha256"`
	Sources        int             `json:"sources"`
	Succeeded      int             `json:"succeeded"`
	Failed         int             `json:"failed"`
	Artifacts      []ArtifactIndex `json:"artifacts"`
}

// RunReport identifies the published output and its index.
type RunReport struct {
	Directory string
	Index     RunIndex
}
