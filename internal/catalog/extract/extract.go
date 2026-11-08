// Package extract finds Schema.org Recipe nodes in stored evidence
// (CATALOG_INGEST_SPEC.md §3, §7.5). It reads only application/ld+json
// scripts, preserves each Recipe node's complete raw JSON, and never needs
// network access, so it can replay over stored evidence.
package extract

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"mime"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/net/html"

	"github.com/mbcoward3/grocery-router/internal/catalog/evidence"
	"github.com/mbcoward3/grocery-router/internal/catalog/safefetch"
)

// Version identifies the extraction rules; stored with every result.
const Version = "jsonld-recipe/1"

// NormalizationVersion identifies the content-digest normalization
// (ingest.source_nodes.normalization_version).
const NormalizationVersion = 1

// Outcome is the page-level extraction result.
type Outcome string

// Outcomes.
const (
	OutcomeRecipes  Outcome = "recipes"
	OutcomeNoRecipe Outcome = "no_recipe"
)

// CanonicalSource says where Result.CanonicalURL came from.
type CanonicalSource string

// Canonical sources.
const (
	CanonicalFromLink     CanonicalSource = "canonical_link"
	CanonicalFromFinalURL CanonicalSource = "final_url"
)

// Block is one JSON-LD script, kept verbatim.
type Block struct {
	Index      int
	Raw        string
	SHA256     string
	ParseError string
}

// Node is one Schema.org Recipe node.
type Node struct {
	// BlockIndex and Location say where the node was found; they are not
	// part of its identity. Location is a JSON Pointer into the block ("" for
	// the block's top-level object).
	BlockIndex int
	Location   string
	// ID is the node's "@id" string, or empty.
	ID string
	// ContentDigest is the SHA-256 of the node's normalized JSON (sorted
	// keys, no insignificant whitespace, numbers kept literally).
	ContentDigest string
	// Raw is the complete node exactly as the source wrote it.
	Raw json.RawMessage
}

// Identity is the source node identity: content digest plus @id.
func (node Node) Identity() string { return node.ContentDigest + "|" + node.ID }

// Result is the extraction of one page.
type Result struct {
	Version         string
	Outcome         Outcome
	CanonicalURL    string
	CanonicalSource CanonicalSource
	// CanonicalLinks are the raw rel=canonical hrefs found, in order.
	CanonicalLinks []string
	Blocks         []Block
	Nodes          []Node
	Warnings       []string
}

// FromEvidence decompresses stored evidence and extracts from it.
func FromEvidence(stored evidence.Evidence, siteHosts []string) (Result, error) {
	body, err := stored.Body(safefetch.PageProfile.MaxBodyBytes)
	if err != nil {
		return Result{}, fmt.Errorf("evidence %d: %w", stored.ID, err)
	}
	return Extract(body, stored.FinalURL, siteHosts)
}

// Extract parses an HTML body fetched from finalURL. Malformed JSON-LD is
// reported in Blocks and Warnings, never fatal; a page without Recipe nodes
// yields OutcomeNoRecipe.
func Extract(body []byte, finalURL string, siteHosts []string) (Result, error) {
	document, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return Result{}, fmt.Errorf("parse HTML: %w", err)
	}
	result := Result{Version: Version, Outcome: OutcomeNoRecipe}
	base, err := url.Parse(finalURL)
	if err != nil {
		return Result{}, fmt.Errorf("parse final URL: %w", err)
	}
	var canonicalCandidates []string
	seen := map[string]bool{}
	walk(document, func(node *html.Node) {
		if node.Type != html.ElementNode {
			return
		}
		switch strings.ToLower(node.Data) {
		case "link":
			if href, ok := canonicalHref(node); ok {
				result.CanonicalLinks = append(result.CanonicalLinks, href)
				canonicalCandidates = append(canonicalCandidates, resolveCanonical(base, href, siteHosts))
			}
		case "script":
			if !isJSONLD(node) {
				return
			}
			raw := scriptText(node)
			digest := sha256.Sum256([]byte(raw))
			block := Block{Index: len(result.Blocks), Raw: raw, SHA256: hex.EncodeToString(digest[:])}
			trimmed := bytes.TrimSpace([]byte(raw))
			if !json.Valid(trimmed) {
				var probe any
				parseErr := json.Unmarshal(trimmed, &probe)
				if parseErr == nil {
					parseErr = fmt.Errorf("invalid JSON")
				}
				block.ParseError = parseErr.Error()
				result.Warnings = append(result.Warnings, fmt.Sprintf("JSON-LD block %d is malformed: %v", block.Index, parseErr))
				result.Blocks = append(result.Blocks, block)
				return
			}
			result.Blocks = append(result.Blocks, block)
			for _, found := range findRecipes(trimmed) {
				recipe, err := newNode(block.Index, found.location, found.raw)
				if err != nil {
					result.Warnings = append(result.Warnings, fmt.Sprintf("JSON-LD block %d %s: %v", block.Index, found.location, err))
					continue
				}
				if seen[recipe.Identity()] {
					result.Warnings = append(result.Warnings, fmt.Sprintf("JSON-LD block %d %s repeats an earlier Recipe node", block.Index, found.location))
					continue
				}
				seen[recipe.Identity()] = true
				result.Nodes = append(result.Nodes, recipe)
			}
		}
	})
	result.CanonicalURL, result.CanonicalSource = chooseCanonical(canonicalCandidates, finalURL)
	if len(result.Nodes) > 0 {
		result.Outcome = OutcomeRecipes
	}
	return result, nil
}

type located struct {
	location string
	raw      json.RawMessage
}

// findRecipes returns Recipe nodes at the top level, in a top-level array,
// or in an "@graph" array of either.
func findRecipes(document []byte) []located {
	var found []located
	visitObject := func(location string, raw json.RawMessage, allowGraph bool) {
		var object map[string]json.RawMessage
		if json.Unmarshal(raw, &object) != nil {
			return
		}
		if isRecipe(object["@type"]) {
			found = append(found, located{location: location, raw: bytes.TrimSpace(raw)})
		}
		if !allowGraph {
			return
		}
		var graph []json.RawMessage
		if json.Unmarshal(object["@graph"], &graph) != nil {
			return
		}
		for index, item := range graph {
			var nested map[string]json.RawMessage
			if json.Unmarshal(item, &nested) == nil && isRecipe(nested["@type"]) {
				found = append(found, located{location: location + "/@graph/" + strconv.Itoa(index), raw: bytes.TrimSpace(item)})
			}
		}
	}
	trimmed := bytes.TrimSpace(document)
	switch {
	case len(trimmed) > 0 && trimmed[0] == '[':
		var items []json.RawMessage
		if json.Unmarshal(trimmed, &items) == nil {
			for index, item := range items {
				visitObject("/"+strconv.Itoa(index), item, true)
			}
		}
	case len(trimmed) > 0 && trimmed[0] == '{':
		visitObject("", trimmed, true)
	}
	return found
}

func isRecipe(raw json.RawMessage) bool {
	var single string
	if json.Unmarshal(raw, &single) == nil {
		return isRecipeType(single)
	}
	var multiple []string
	if json.Unmarshal(raw, &multiple) != nil {
		return false
	}
	for _, value := range multiple {
		if isRecipeType(value) {
			return true
		}
	}
	return false
}

func isRecipeType(value string) bool {
	value = strings.TrimSpace(value)
	if strings.EqualFold(value, "Recipe") || strings.EqualFold(value, "schema:Recipe") {
		return true
	}
	parsed, err := url.Parse(value)
	if err != nil || !strings.EqualFold(strings.TrimPrefix(parsed.Hostname(), "www."), "schema.org") {
		return false
	}
	return strings.EqualFold(strings.Trim(parsed.Path, "/"), "Recipe")
}

func newNode(blockIndex int, location string, raw json.RawMessage) (Node, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return Node{}, fmt.Errorf("decode Recipe node: %w", err)
	}
	normalized, err := json.Marshal(value)
	if err != nil {
		return Node{}, fmt.Errorf("normalize Recipe node: %w", err)
	}
	digest := sha256.Sum256(normalized)
	node := Node{
		BlockIndex: blockIndex, Location: location,
		ContentDigest: hex.EncodeToString(digest[:]),
		Raw:           append(json.RawMessage(nil), raw...),
	}
	if object, ok := value.(map[string]any); ok {
		if id, ok := object["@id"].(string); ok {
			node.ID = id
		}
	}
	return node, nil
}

func canonicalHref(node *html.Node) (string, bool) {
	var rel, href string
	hasHref := false
	for _, attribute := range node.Attr {
		switch strings.ToLower(attribute.Key) {
		case "rel":
			rel = attribute.Val
		case "href":
			href, hasHref = strings.TrimSpace(attribute.Val), true
		}
	}
	if !hasHref {
		return "", false
	}
	for _, token := range strings.Fields(rel) {
		if strings.EqualFold(token, "canonical") {
			return href, true
		}
	}
	return "", false
}

// resolveCanonical returns the absolute canonical URL if it is a valid HTTPS
// URL on one of the site's hosts, or "" otherwise.
func resolveCanonical(base *url.URL, href string, siteHosts []string) string {
	reference, err := url.Parse(href)
	if err != nil {
		return ""
	}
	resolved := base.ResolveReference(reference)
	resolved.Fragment, resolved.RawFragment = "", ""
	checked, err := safefetch.ValidateURL(resolved.String())
	if err != nil || !safefetch.HostAllowed(checked, siteHosts) {
		return ""
	}
	return checked.String()
}

// chooseCanonical accepts the canonical link only when every canonical link
// resolves to the same valid on-site URL; otherwise it uses the final URL.
func chooseCanonical(candidates []string, finalURL string) (string, CanonicalSource) {
	if len(candidates) == 0 {
		return finalURL, CanonicalFromFinalURL
	}
	for _, candidate := range candidates {
		if candidate == "" || candidate != candidates[0] {
			return finalURL, CanonicalFromFinalURL
		}
	}
	return candidates[0], CanonicalFromLink
}

func isJSONLD(node *html.Node) bool {
	for _, attribute := range node.Attr {
		if strings.EqualFold(attribute.Key, "type") {
			mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(attribute.Val))
			return err == nil && strings.EqualFold(mediaType, "application/ld+json")
		}
	}
	return false
}

func scriptText(node *html.Node) string {
	var builder strings.Builder
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.TextNode {
			builder.WriteString(child.Data)
		}
	}
	return builder.String()
}

func walk(node *html.Node, visit func(*html.Node)) {
	visit(node)
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		walk(child, visit)
	}
}

// Store persists an extraction for one evidence row
// (ingest.evidence_jsonld_blocks and ingest.source_nodes).
type Store interface {
	SaveExtraction(ctx context.Context, evidenceID int64, result Result) error
}

// Replay extracts stored evidence and saves the result, without network access.
func Replay(ctx context.Context, evidenceStore evidence.Store, store Store, evidenceID int64, siteHosts []string) (Result, error) {
	stored, err := evidenceStore.Get(ctx, evidenceID)
	if err != nil {
		return Result{}, fmt.Errorf("load evidence %d: %w", evidenceID, err)
	}
	result, err := FromEvidence(stored, siteHosts)
	if err != nil {
		return Result{}, err
	}
	if err := store.SaveExtraction(ctx, evidenceID, result); err != nil {
		return Result{}, fmt.Errorf("save extraction for evidence %d: %w", evidenceID, err)
	}
	return result, nil
}

// MemoryStore is an in-memory Store keeping every saved extraction.
type MemoryStore struct {
	mu      sync.Mutex
	results map[int64][]Result
}

// NewMemoryStore returns an empty MemoryStore.
func NewMemoryStore() *MemoryStore { return &MemoryStore{results: map[int64][]Result{}} }

// SaveExtraction appends result for evidenceID.
func (store *MemoryStore) SaveExtraction(_ context.Context, evidenceID int64, result Result) error {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.results[evidenceID] = append(store.results[evidenceID], result)
	return nil
}

// Results returns every extraction saved for evidenceID.
func (store *MemoryStore) Results(evidenceID int64) []Result {
	store.mu.Lock()
	defer store.mu.Unlock()
	return append([]Result(nil), store.results[evidenceID]...)
}
