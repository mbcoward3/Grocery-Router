// Package recipeintel evaluates compact recipe projections with a versioned
// catalog of typed Jev questions. It is operator-invoked pilot tooling only.
package recipeintel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// CatalogVersion is the only assessment-catalog format accepted by this implementation.
const CatalogVersion = 1

var questionKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*|\.<family>\.region)*$`)

// Catalog is a versioned collection of typed recipe-assessment questions.
type Catalog struct {
	Version   int        `yaml:"version" json:"version"`
	Model     string     `yaml:"model" json:"model"`
	Questions []Question `yaml:"questions" json:"questions"`
}

// Question defines one independently versioned semantic judgment.
type Question struct {
	Key             string    `yaml:"key" json:"key"`
	Version         int       `yaml:"version" json:"version"`
	Projection      string    `yaml:"projection" json:"projection"`
	Materialization string    `yaml:"materialization" json:"materialization"`
	Type            string    `yaml:"type" json:"type"`
	Enabled         *bool     `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	Instructions    string    `yaml:"instructions" json:"instructions"`
	Criteria        yaml.Node `yaml:"criteria" json:"-"`
}

// APIQuestion is the TypeSafe wire representation of a catalog question.
type APIQuestion struct {
	Type         string `json:"type"`
	Instructions string `json:"instructions"`
	Criteria     any    `json:"criteria,omitempty"`
}

// ReadCatalog strictly decodes, validates, and digests catalog YAML.
func ReadCatalog(data []byte) (Catalog, string, error) {
	var catalog Catalog
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&catalog); err != nil {
		return Catalog{}, "", fmt.Errorf("decode recipe intelligence catalog: %w", err)
	}
	if err := catalog.Validate(); err != nil {
		return Catalog{}, "", err
	}
	return catalog, sha256Hex(data), nil
}

// Validate enforces catalog shape and question invariants before any API call.
func (catalog Catalog) Validate() error {
	if catalog.Version != CatalogVersion {
		return fmt.Errorf("recipe intelligence catalog version = %d, want %d", catalog.Version, CatalogVersion)
	}
	if strings.TrimSpace(catalog.Model) == "" {
		return fmt.Errorf("recipe intelligence catalog model is required")
	}
	if len(catalog.Questions) == 0 {
		return fmt.Errorf("recipe intelligence catalog has no questions")
	}
	seen := make(map[string]struct{}, len(catalog.Questions))
	for index, question := range catalog.Questions {
		if !questionKeyPattern.MatchString(question.Key) {
			return fmt.Errorf("question %d has invalid key %q", index, question.Key)
		}
		if _, ok := seen[question.Key]; ok {
			return fmt.Errorf("duplicate question key %q", question.Key)
		}
		seen[question.Key] = struct{}{}
		if question.Version < 1 {
			return fmt.Errorf("question %q needs a positive version", question.Key)
		}
		switch question.Projection {
		case "identity", "ingredients", "method", "full_recipe":
		default:
			return fmt.Errorf("question %q has invalid projection %q", question.Key, question.Projection)
		}
		if question.Materialization != "facet" && question.Materialization != "signal" && question.Materialization != "diagnostic" {
			return fmt.Errorf("question %q has invalid materialization %q", question.Key, question.Materialization)
		}
		if strings.TrimSpace(question.Instructions) == "" {
			return fmt.Errorf("question %q has empty instructions", question.Key)
		}
		if _, err := question.APIQuestion(); err != nil {
			return fmt.Errorf("question %q: %w", question.Key, err)
		}
	}
	return nil
}

// IsEnabled reports whether the question participates in an assessment run.
func (question Question) IsEnabled() bool {
	return question.Enabled == nil || *question.Enabled
}

// APIQuestion validates and converts a catalog question to its wire form.
func (question Question) APIQuestion() (APIQuestion, error) {
	result := APIQuestion{Type: question.Type, Instructions: question.Instructions}
	switch question.Type {
	case "noul", "choice":
		var criteria map[string]string
		if err := question.Criteria.Decode(&criteria); err != nil {
			return APIQuestion{}, fmt.Errorf("decode %s criteria: %w", question.Type, err)
		}
		minimum := 2
		if question.Type == "noul" {
			if len(criteria) != 0 && (criteria["true"] == "" || criteria["false"] == "") {
				return APIQuestion{}, fmt.Errorf("noul criteria must define non-empty true and false values")
			}
		} else if len(criteria) < minimum {
			return APIQuestion{}, fmt.Errorf("choice criteria need at least two options")
		}
		result.Criteria = criteria
	case "score":
		var criteria []string
		if err := question.Criteria.Decode(&criteria); err != nil {
			return APIQuestion{}, fmt.Errorf("decode score criteria: %w", err)
		}
		if len(criteria) < 2 || len(criteria) > 10 {
			return APIQuestion{}, fmt.Errorf("score criteria need between 2 and 10 levels")
		}
		for _, level := range criteria {
			if strings.TrimSpace(level) == "" {
				return APIQuestion{}, fmt.Errorf("score criteria contain an empty level")
			}
		}
		result.Criteria = criteria
	default:
		return APIQuestion{}, fmt.Errorf("invalid type %q", question.Type)
	}
	return result, nil
}

// QuestionsByProjection groups enabled questions for focused TypeSafe requests.
func (catalog Catalog) QuestionsByProjection() (map[string]map[string]APIQuestion, error) {
	result := make(map[string]map[string]APIQuestion)
	for _, question := range catalog.Questions {
		if !question.IsEnabled() {
			continue
		}
		apiQuestion, err := question.APIQuestion()
		if err != nil {
			return nil, fmt.Errorf("question %q: %w", question.Key, err)
		}
		if result[question.Projection] == nil {
			result[question.Projection] = make(map[string]APIQuestion)
		}
		result[question.Projection][question.Key] = apiQuestion
	}
	return result, nil
}

// SortedProjectionNames returns projection names in deterministic evaluation order.
func SortedProjectionNames(grouped map[string]map[string]APIQuestion) []string {
	order := map[string]int{"identity": 0, "ingredients": 1, "method": 2, "full_recipe": 3}
	names := make([]string, 0, len(grouped))
	for name := range grouped {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool { return order[names[i]] < order[names[j]] })
	return names
}

func marshalCanonical(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode canonical JSON: %w", err)
	}
	return data, nil
}
