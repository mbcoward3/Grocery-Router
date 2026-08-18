package trueup

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	inventoryHeader = []string{
		"inventory_key", "pdf_name", "pdf_pages", "linked_url", "supporting_evidence",
		"proposed_recipe_name", "source_relationship", "status", "verified_on", "disposition_note",
	}
	keyPattern   = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	pagesPattern = regexp.MustCompile(`^[1-9][0-9]*(?:-[1-9][0-9]*)?$`)
	datePattern  = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
)

var allowedStatuses = map[string]bool{
	"inventoried":       true,
	"source-found":      true,
	"drafted":           true,
	"reviewable":        true,
	"changes-requested": true,
	"verified":          true,
	"excluded":          true,
}

// InventoryRow is one PDF-controlled corpus member in the true-up ledger.
type InventoryRow struct {
	Key                string
	PDFName            string
	PDFPages           string
	LinkedURL          string
	SupportingEvidence []string
	ProposedRecipeName string
	SourceRelationship string
	Status             string
	VerifiedOn         string
	DispositionNote    string
}

// ReadInventory parses and validates the true-up ledger. root is the repository root used to
// verify supporting-evidence paths.
func ReadInventory(root string, input io.Reader) ([]InventoryRow, error) {
	reader := csv.NewReader(input)
	reader.FieldsPerRecord = len(inventoryHeader)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read inventory CSV: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("inventory is empty")
	}
	for i, want := range inventoryHeader {
		if records[0][i] != want {
			return nil, fmt.Errorf("inventory column %d is %q, want %q", i+1, records[0][i], want)
		}
	}

	seen := make(map[string]int, len(records)-1)
	rows := make([]InventoryRow, 0, len(records)-1)
	for index, record := range records[1:] {
		line := index + 2
		row := InventoryRow{
			Key:                strings.TrimSpace(record[0]),
			PDFName:            strings.TrimSpace(record[1]),
			PDFPages:           strings.TrimSpace(record[2]),
			LinkedURL:          strings.TrimSpace(record[3]),
			ProposedRecipeName: strings.TrimSpace(record[5]),
			SourceRelationship: strings.TrimSpace(record[6]),
			Status:             strings.TrimSpace(record[7]),
			VerifiedOn:         strings.TrimSpace(record[8]),
			DispositionNote:    strings.TrimSpace(record[9]),
		}
		for _, evidence := range strings.Split(record[4], ";") {
			if evidence = strings.TrimSpace(evidence); evidence != "" {
				row.SupportingEvidence = append(row.SupportingEvidence, evidence)
			}
		}

		if !keyPattern.MatchString(row.Key) {
			return nil, fmt.Errorf("line %d: invalid inventory key %q", line, row.Key)
		}
		if previous, ok := seen[row.Key]; ok {
			return nil, fmt.Errorf("line %d: duplicate inventory key %q (first on line %d)", line, row.Key, previous)
		}
		seen[row.Key] = line
		if row.PDFName == "" || row.ProposedRecipeName == "" {
			return nil, fmt.Errorf("line %d (%s): PDF and proposed names are required", line, row.Key)
		}
		if !pagesPattern.MatchString(row.PDFPages) {
			return nil, fmt.Errorf("line %d (%s): invalid PDF pages %q", line, row.Key, row.PDFPages)
		}
		if row.LinkedURL != "" {
			parsed, parseErr := url.ParseRequestURI(row.LinkedURL)
			if parseErr != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
				return nil, fmt.Errorf("line %d (%s): invalid linked URL %q", line, row.Key, row.LinkedURL)
			}
		}
		if row.SourceRelationship != "source" && row.SourceRelationship != "adapted-from" {
			return nil, fmt.Errorf("line %d (%s): invalid source relationship %q", line, row.Key, row.SourceRelationship)
		}
		if !allowedStatuses[row.Status] {
			return nil, fmt.Errorf("line %d (%s): invalid status %q", line, row.Key, row.Status)
		}
		if row.Status == "verified" {
			if !datePattern.MatchString(row.VerifiedOn) {
				return nil, fmt.Errorf("line %d (%s): verified recipe needs verified_on", line, row.Key)
			}
		} else if row.VerifiedOn != "" {
			return nil, fmt.Errorf("line %d (%s): non-verified recipe cannot have verified_on", line, row.Key)
		}
		if len(row.SupportingEvidence) == 0 {
			return nil, fmt.Errorf("line %d (%s): supporting evidence is required", line, row.Key)
		}
		for _, evidence := range row.SupportingEvidence {
			if filepath.IsAbs(evidence) || strings.Contains(evidence, "..") {
				return nil, fmt.Errorf("line %d (%s): unsafe evidence path %q", line, row.Key, evidence)
			}
			if _, statErr := os.Stat(filepath.Join(root, filepath.FromSlash(evidence))); statErr != nil {
				return nil, fmt.Errorf("line %d (%s): evidence %q: %w", line, row.Key, evidence, statErr)
			}
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("inventory has no recipes")
	}
	return rows, nil
}
