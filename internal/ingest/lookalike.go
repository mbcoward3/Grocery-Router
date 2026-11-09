package ingest

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

// ItemReviews classifies each pair of grocery items whose names look alike ("Salt" and "Kosher
// Salt"). A new look-alike pair must be classified before the corpus audits clean, so a
// near-duplicate item cannot slip in quietly with a new recipe. The catalog freeze applies the
// same classifications automatically (CATALOG_INGEST_SPEC §6.1).
type ItemReviews struct {
	Pairs []ItemReview `yaml:"pairs"`
}

// ItemReview is one look-alike pair and its classification.
type ItemReview struct {
	Items []string `yaml:"items,flow"`
	// Decision is alias (the same product; to become one item), group (different products a
	// household may choose to combine on its list), distinct (never combined), or undecided.
	Decision string `yaml:"decision"`
	// Group names the substitution group for a group decision.
	Group string `yaml:"group,omitempty"`
	Note  string `yaml:"note,omitempty"`
}

// LookAlike is a pair of item keys whose names share every word of the shorter name.
type LookAlike struct {
	Items [2]string
	Names [2]string
}

// ParseItemReviews reads the reviewed look-alike pairs, rejecting unknown fields and decisions.
func ParseItemReviews(reader io.Reader) (ItemReviews, error) {
	decoder := yaml.NewDecoder(reader)
	decoder.KnownFields(true)
	var reviews ItemReviews
	if err := decoder.Decode(&reviews); err != nil {
		return ItemReviews{}, fmt.Errorf("parse item reviews: %w", err)
	}
	seen := make(map[[2]string]bool, len(reviews.Pairs))
	for _, review := range reviews.Pairs {
		if len(review.Items) != 2 || review.Items[0] >= review.Items[1] {
			return ItemReviews{}, fmt.Errorf("item review %v must name two item keys in sorted order", review.Items)
		}
		switch review.Decision {
		case "alias", "distinct", "undecided":
			if review.Group != "" {
				return ItemReviews{}, fmt.Errorf("item review %v names a group without a group decision", review.Items)
			}
		case "group":
			if review.Group == "" {
				return ItemReviews{}, fmt.Errorf("item review %v needs a group name", review.Items)
			}
		default:
			return ItemReviews{}, fmt.Errorf("item review %v has invalid decision %q", review.Items, review.Decision)
		}
		key := [2]string{review.Items[0], review.Items[1]}
		if seen[key] {
			return ItemReviews{}, fmt.Errorf("item review %v appears twice", review.Items)
		}
		seen[key] = true
	}
	return reviews, nil
}

// FindLookAlikes lists every pair of distinct grocery items in the corpus whose normalized name
// words contain the other's: "Salt" and "Kosher Salt", "Tomato" and "Tomatoes". It
// over-reports on purpose (Garlic and Garlic Powder); a reviewed decision settles each pair.
func FindLookAlikes(documents []Document) []LookAlike {
	names := make(map[string]string)
	for _, document := range documents {
		for _, section := range document.IngredientSections {
			for _, ingredient := range section.Ingredients {
				names[ingredient.GroceryItem.Key] = ingredient.GroceryItem.Name
			}
		}
	}
	keys := make([]string, 0, len(names))
	for key := range names {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	words := make(map[string][]string, len(keys))
	for _, key := range keys {
		words[key] = nameWords(names[key])
	}
	var pairs []LookAlike
	for i, left := range keys {
		for _, right := range keys[i+1:] {
			if containsAll(words[left], words[right]) || containsAll(words[right], words[left]) {
				pairs = append(pairs, LookAlike{Items: [2]string{left, right}, Names: [2]string{names[left], names[right]}})
			}
		}
	}
	return pairs
}

// CheckLookAlikes fails for any look-alike pair without a recorded decision and returns the
// pairs still marked undecided.
func CheckLookAlikes(documents []Document, reviews ItemReviews) ([]ItemReview, error) {
	decided := make(map[[2]string]ItemReview, len(reviews.Pairs))
	for _, review := range reviews.Pairs {
		decided[[2]string{review.Items[0], review.Items[1]}] = review
	}
	var missing []string
	var undecided []ItemReview
	for _, pair := range FindLookAlikes(documents) {
		review, ok := decided[pair.Items]
		if !ok {
			missing = append(missing, fmt.Sprintf("%s (%s) / %s (%s)", pair.Names[0], pair.Items[0], pair.Names[1], pair.Items[1]))
			continue
		}
		if review.Decision == "undecided" {
			undecided = append(undecided, review)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("grocery items with look-alike names need a classification in corpus/item-reviews.yaml:\n  %s", strings.Join(missing, "\n  "))
	}
	return undecided, nil
}

func nameWords(name string) []string {
	fields := strings.FieldsFunc(strings.ToLower(name), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	for index, word := range fields {
		// Plurals compare as one word: tomatoes/tomato, carrots/carrot.
		switch {
		case len(word) > 4 && strings.HasSuffix(word, "oes"):
			word = strings.TrimSuffix(word, "es")
		case len(word) > 3 && strings.HasSuffix(word, "s") && !strings.HasSuffix(word, "ss"):
			word = strings.TrimSuffix(word, "s")
		}
		fields[index] = word
	}
	return fields
}

func containsAll(set, subset []string) bool {
	for _, word := range subset {
		if !slices.Contains(set, word) {
			return false
		}
	}
	return true
}
