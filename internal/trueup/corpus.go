package trueup

import (
	"fmt"

	"github.com/mbcoward3/grocery-router/internal/ingest"
)

// MatchApprovedCorpus proves the PDF ledger and committed bootstrap corpus agree exactly:
// every verified ledger row has one approved document and no unverified row has one.
func MatchApprovedCorpus(inventory []InventoryRow, documents []ingest.Document) error {
	rows := make(map[string]InventoryRow, len(inventory))
	for _, row := range inventory {
		rows[row.Key] = row
	}
	documentsByKey := make(map[string]ingest.Document, len(documents))
	for _, document := range documents {
		row, ok := rows[document.Key]
		if !ok {
			return fmt.Errorf("approved recipe %q is not a PDF inventory member", document.Key)
		}
		if row.Status != "verified" {
			return fmt.Errorf("approved recipe %q has ledger status %q, want verified", document.Key, row.Status)
		}
		if row.VerifiedOn != document.ApprovedOn {
			return fmt.Errorf("approved recipe %q date %q does not match ledger %q", document.Key, document.ApprovedOn, row.VerifiedOn)
		}
		documentsByKey[document.Key] = document
	}
	for _, row := range inventory {
		_, hasDocument := documentsByKey[row.Key]
		if row.Status == "verified" && !hasDocument {
			return fmt.Errorf("verified inventory recipe %q has no approved Markdown document", row.Key)
		}
		if row.Status != "verified" && hasDocument {
			return fmt.Errorf("unverified inventory recipe %q unexpectedly has an approved document", row.Key)
		}
	}
	return nil
}
