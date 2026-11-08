package recipepilot

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"mime"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// Extraction contains literal recipe candidates and page-level evidence.
type Extraction struct {
	Canonical    []CanonicalLink
	JSONLDBlocks []JSONLDBlock
	Recipes      []RecipeCandidate
	Warnings     []string
}

// Extract finds source-provided Schema.org Recipe nodes without interpreting fields.
func Extract(body []byte, finalURL string) (Extraction, error) {
	document, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return Extraction{}, fmt.Errorf("parse HTML: %w", err)
	}
	baseURL, _ := url.Parse(finalURL)
	extraction := Extraction{
		Canonical:    extractCanonicalLinks(document, baseURL),
		JSONLDBlocks: make([]JSONLDBlock, 0),
		Recipes:      make([]RecipeCandidate, 0),
		Warnings:     make([]string, 0),
	}
	scriptIndex := 0
	walkHTML(document, func(node *html.Node) {
		if node.Type != html.ElementNode || !strings.EqualFold(node.Data, "script") || !isJSONLDScript(node) {
			return
		}
		currentIndex := scriptIndex
		scriptIndex++
		raw := []byte(scriptText(node))
		trimmed := bytes.TrimSpace(raw)
		digest := sha256.Sum256(raw)
		block := JSONLDBlock{
			ScriptIndex: currentIndex,
			Bytes:       len(raw),
			SHA256:      hex.EncodeToString(digest[:]),
		}
		if len(trimmed) == 0 {
			extraction.JSONLDBlocks = append(extraction.JSONLDBlocks, block)
			return
		}
		var value json.RawMessage
		if err := json.Unmarshal(trimmed, &value); err != nil {
			block.ParseError = err.Error()
			extraction.Warnings = append(extraction.Warnings, fmt.Sprintf("JSON-LD script %d is malformed: %v", currentIndex, err))
			extraction.JSONLDBlocks = append(extraction.JSONLDBlocks, block)
			return
		}
		extraction.JSONLDBlocks = append(extraction.JSONLDBlocks, block)
		findRecipes(value, "$", currentIndex, append(json.RawMessage(nil), trimmed...), &extraction.Recipes)
	})
	if len(extraction.Recipes) == 0 {
		return extraction, fmt.Errorf("no Schema.org Recipe JSON-LD found")
	}
	return extraction, nil
}

func isJSONLDScript(node *html.Node) bool {
	for _, attribute := range node.Attr {
		if !strings.EqualFold(attribute.Key, "type") {
			continue
		}
		mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(attribute.Val))
		return err == nil && strings.EqualFold(mediaType, "application/ld+json")
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

func extractCanonicalLinks(document *html.Node, base *url.URL) []CanonicalLink {
	links := make([]CanonicalLink, 0)
	walkHTML(document, func(node *html.Node) {
		if node.Type != html.ElementNode || !strings.EqualFold(node.Data, "link") {
			return
		}
		var rel, href string
		for _, attribute := range node.Attr {
			switch {
			case strings.EqualFold(attribute.Key, "rel"):
				rel = attribute.Val
			case strings.EqualFold(attribute.Key, "href"):
				href = attribute.Val
			}
		}
		if href == "" || !containsFold(strings.Fields(rel), "canonical") {
			return
		}
		link := CanonicalLink{Href: href}
		if parsed, err := url.Parse(href); err == nil && base != nil {
			link.Resolved = base.ResolveReference(parsed).String()
		}
		links = append(links, link)
	})
	return links
}

func containsFold(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(value, target) {
			return true
		}
	}
	return false
}

func walkHTML(node *html.Node, visit func(*html.Node)) {
	visit(node)
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		walkHTML(child, visit)
	}
}

func findRecipes(raw json.RawMessage, path string, scriptIndex int, jsonLD json.RawMessage, candidates *[]RecipeCandidate) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return
	}
	switch trimmed[0] {
	case '[':
		var values []json.RawMessage
		if json.Unmarshal(trimmed, &values) != nil {
			return
		}
		for index, value := range values {
			findRecipes(value, path+"["+strconv.Itoa(index)+"]", scriptIndex, jsonLD, candidates)
		}
	case '{':
		var object map[string]json.RawMessage
		if json.Unmarshal(trimmed, &object) != nil {
			return
		}
		if hasRecipeType(object["@type"]) {
			*candidates = append(*candidates, RecipeCandidate{
				ScriptIndex: scriptIndex,
				JSONPath:    path,
				JSONLD:      append(json.RawMessage(nil), jsonLD...),
				Recipe:      sourceRecipe(object, trimmed),
			})
		}
		keys := make([]string, 0, len(object))
		for key := range object {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			findRecipes(object[key], path+"/"+escapeJSONPointer(key), scriptIndex, jsonLD, candidates)
		}
	}
}

func hasRecipeType(raw json.RawMessage) bool {
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
	if strings.EqualFold(value, "Recipe") {
		return true
	}
	parsed, err := url.Parse(value)
	if err != nil || !strings.EqualFold(parsed.Hostname(), "schema.org") {
		return false
	}
	return strings.EqualFold(strings.Trim(parsed.Path, "/"), "Recipe")
}

func sourceRecipe(object map[string]json.RawMessage, raw json.RawMessage) SourceRecipe {
	copyField := func(name string) json.RawMessage {
		if value, ok := object[name]; ok {
			return append(json.RawMessage(nil), value...)
		}
		return nil
	}
	return SourceRecipe{
		Context: copyField("@context"), Type: copyField("@type"), ID: copyField("@id"),
		Name: copyField("name"), Headline: copyField("headline"), Description: copyField("description"),
		URL: copyField("url"), MainEntityOfPage: copyField("mainEntityOfPage"), Image: copyField("image"),
		Author: copyField("author"), Publisher: copyField("publisher"), DatePublished: copyField("datePublished"),
		DateModified: copyField("dateModified"), RecipeYield: copyField("recipeYield"), PrepTime: copyField("prepTime"),
		CookTime: copyField("cookTime"), TotalTime: copyField("totalTime"), RecipeCategory: copyField("recipeCategory"),
		RecipeCuisine: copyField("recipeCuisine"), Keywords: copyField("keywords"),
		RecipeIngredient: copyField("recipeIngredient"), RecipeInstructions: copyField("recipeInstructions"),
		Nutrition: copyField("nutrition"), AggregateRating: copyField("aggregateRating"), Video: copyField("video"),
		Raw: append(json.RawMessage(nil), raw...),
	}
}

func escapeJSONPointer(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "~", "~0"), "/", "~1")
}
