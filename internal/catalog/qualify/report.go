package qualify

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// WriteText writes a human-readable qualification report.
func (r Report) WriteText(w io.Writer) error {
	var out strings.Builder
	verdict := "QUALIFIED"
	if !r.Qualified() {
		verdict = "NOT QUALIFIED"
	}
	fmt.Fprintf(&out, "catalog qualification: %s\n\nCriteria (CATALOG_INGEST_SPEC.md §6.3)\n", verdict)
	for _, criterion := range r.Criteria {
		mark := "PASS"
		if !criterion.Passed {
			mark = "FAIL"
		}
		fmt.Fprintf(&out, "  [%s] %s\n", mark, criterion.Name)
		for _, failure := range criterion.Failures {
			fmt.Fprintf(&out, "         - %s\n", failure)
		}
	}

	writeTallies := func(title string, tallies []Tally, sites bool) {
		fmt.Fprintf(&out, "\n%s\n", title)
		table := tabwriter.NewWriter(&out, 0, 0, 2, ' ', 0)
		fmt.Fprintln(table, "  name\tcases\tpublished\tcorrect\tmaterial errors\trejected as expected\trejected\t")
		for _, tally := range tallies {
			name := tally.Name
			if sites && tally.Active {
				name += " (active)"
			}
			fmt.Fprintf(table, "  %s\t%d\t%d\t%d\t%d\t%d\t%d\t\n", name, tally.Cases, tally.Published,
				tally.Correct, tally.MaterialErrors, tally.RejectedAsExpected, tally.Rejected)
		}
		_ = table.Flush()
	}
	writeTallies("By site", r.BySite, true)
	writeTallies("By category", r.ByCategory, false)

	fmt.Fprintf(&out, "\nCases\n")
	for _, result := range r.Results {
		reviewed := ""
		if !result.Case.Expected.Reviewed {
			reviewed = " (unreviewed draft)"
		}
		fmt.Fprintf(&out, "  %s/%s [%s]: %s%s\n", result.Case.Part, result.Case.ID, result.Case.Site(), result.Status, reviewed)
		if !result.Published && len(result.RejectCodes) > 0 {
			fmt.Fprintf(&out, "      rejection codes: %s\n", strings.Join(result.RejectCodes, ", "))
		}
		if len(result.MissingRejectCodes) > 0 {
			fmt.Fprintf(&out, "      expected codes not reported: %s\n", strings.Join(result.MissingRejectCodes, ", "))
		}
		if result.AcceptedReason != "" {
			fmt.Fprintf(&out, "      accepted regression: %s\n", result.AcceptedReason)
		}
		for _, difference := range result.Differences {
			kind := "cosmetic"
			if difference.Material() {
				kind = "MATERIAL"
			}
			fmt.Fprintf(&out, "      %s %s\n", kind, difference)
		}
	}
	_, err := io.WriteString(w, out.String())
	return err
}
