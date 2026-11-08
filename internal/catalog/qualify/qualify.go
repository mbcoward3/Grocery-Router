package qualify

import (
	"fmt"
	"slices"
	"sort"

	"github.com/mbcoward3/grocery-router/internal/catalog/registry"
)

// Spec §6.3 thresholds.
const (
	// HoldoutMinimumPublished is how many holdout recipes must publish (of ~20).
	HoldoutMinimumPublished = 15
	// SiteMinimumPublished is how many evaluation recipes each active site must publish.
	SiteMinimumPublished = 3
)

// Status is how one case scored.
type Status string

// Case statuses.
const (
	// StatusCorrect means the case expected publication and published with no material error.
	StatusCorrect Status = "correct"
	// StatusMaterialErrors means the case published with at least one material error,
	// including an expected rejection that published.
	StatusMaterialErrors Status = "material_errors"
	// StatusRejectedAsExpected means the case expected rejection and was rejected.
	StatusRejectedAsExpected Status = "rejected_as_expected"
	// StatusRejected means the case expected publication and was rejected.
	StatusRejected Status = "rejected"
	// StatusAcceptedRegression is StatusRejected for a case listed as an accepted regression.
	StatusAcceptedRegression Status = "accepted_regression"
	// StatusNoCandidate means the candidate set has no file for the case; it counts as rejected.
	StatusNoCandidate Status = "no_candidate"
)

// CaseResult is one scored case.
type CaseResult struct {
	Case        Case
	Status      Status
	Published   bool
	RejectCodes []string
	// MissingRejectCodes are expected rejection codes the rejection did not report. They are
	// reported for review; §6.3 only requires that the case is rejected.
	MissingRejectCodes []string
	Differences        []Difference
	AcceptedReason     string
}

// MaterialErrors returns the material differences.
func (r CaseResult) MaterialErrors() []Difference {
	var material []Difference
	for _, difference := range r.Differences {
		if difference.Material() {
			material = append(material, difference)
		}
	}
	return material
}

// Correct reports whether the case published with no material error.
func (r CaseResult) Correct() bool { return r.Status == StatusCorrect }

// Criterion is one §6.3 qualification criterion.
type Criterion struct {
	Name     string
	Passed   bool
	Failures []string
}

// Tally counts results for one site or category.
type Tally struct {
	Name               string
	Active             bool // sites only
	Cases              int
	Published          int
	Correct            int
	MaterialErrors     int
	RejectedAsExpected int
	Rejected           int
}

// Report is the result of one qualification run.
type Report struct {
	Results    []CaseResult
	Criteria   []Criterion
	BySite     []Tally
	ByCategory []Tally
}

// Qualified reports whether every criterion passed.
func (r Report) Qualified() bool {
	for _, criterion := range r.Criteria {
		if !criterion.Passed {
			return false
		}
	}
	return len(r.Criteria) > 0
}

// Input is everything one qualification run scores.
type Input struct {
	EvalSet EvalSet
	// Holdout is the operator's holdout; nil or empty fails qualification.
	Holdout    []Case
	Candidates CandidateSet
	Accepted   AcceptedRegressions
	Sites      registry.Registry
}

// Qualify scores a candidate set against the evaluation set and holdout and evaluates every
// §6.3 criterion. It returns an error only for inconsistent input (a candidate or accepted
// regression naming no case, or a holdout case ID that collides with the evaluation set).
func Qualify(input Input) (Report, error) {
	cases := append(slices.Clone(input.EvalSet.Cases), input.Holdout...)
	if err := checkUniqueIDs(cases); err != nil {
		return Report{}, err
	}
	byID := make(map[string]Case, len(cases))
	for _, c := range cases {
		byID[c.ID] = c
	}
	var unknown []string
	for id := range input.Candidates {
		if _, ok := byID[id]; !ok {
			unknown = append(unknown, id)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return Report{}, fmt.Errorf("candidates name no evaluation or holdout case: %v", unknown)
	}
	accepted := make(map[string]string)
	for _, entry := range input.Accepted.Accepted {
		c, ok := byID[entry.Case]
		if !ok || (c.Part != PartMustPass && c.Part != PartRegression) {
			return Report{}, fmt.Errorf("accepted regression %q is not a must-pass or regression case", entry.Case)
		}
		accepted[entry.Case] = entry.Reason
	}

	var report Report
	for _, c := range cases {
		candidate, ok := input.Candidates[c.ID]
		report.Results = append(report.Results, score(c, candidate, ok, accepted[c.ID]))
	}
	report.Criteria = criteria(report.Results, input)
	report.BySite = siteTallies(report.Results, input.Sites)
	report.ByCategory = categoryTallies(report.Results)
	return report, nil
}

func score(c Case, candidate Candidate, present bool, acceptedReason string) CaseResult {
	result := CaseResult{Case: c}
	if !present {
		result.RejectCodes = []string{string(StatusNoCandidate)}
	} else {
		result.Published, result.RejectCodes = candidate.Evaluate()
	}
	expectPublish := c.Expected.Outcome == OutcomePublish
	switch {
	case result.Published && expectPublish:
		result.Differences = Compare(c.Expected, *candidate.Document)
		result.Status = StatusCorrect
		if len(result.MaterialErrors()) > 0 {
			result.Status = StatusMaterialErrors
		}
	case result.Published:
		result.Differences = []Difference{{Kind: UnexpectedPublishing, Detail: fmt.Sprintf("expected rejection %v", c.Expected.Reject.Codes)}}
		result.Status = StatusMaterialErrors
	case !expectPublish:
		result.Status = StatusRejectedAsExpected
		for _, code := range c.Expected.Reject.Codes {
			if !slices.Contains(result.RejectCodes, code) {
				result.MissingRejectCodes = append(result.MissingRejectCodes, code)
			}
		}
	case acceptedReason != "":
		result.Status = StatusAcceptedRegression
		result.AcceptedReason = acceptedReason
	case !present:
		result.Status = StatusNoCandidate
	default:
		result.Status = StatusRejected
	}
	return result
}

func criteria(results []CaseResult, input Input) []Criterion {
	nonEmpty := Criterion{Name: "candidate set is not empty", Passed: len(input.Candidates) > 0}
	if !nonEmpty.Passed {
		nonEmpty.Failures = []string{"no candidates were supplied"}
	}
	reviewed := Criterion{Name: "every evaluation and holdout case has been reviewed by the household", Passed: true}
	mustPass := Criterion{Name: "every must-pass recipe publishes with zero material errors", Passed: true}
	expectedReject := Criterion{Name: "every expected-reject recipe is rejected", Passed: true}
	regression := Criterion{Name: "every regression recipe publishes correctly or is rejected as allowed", Passed: true}
	holdout := Criterion{Name: fmt.Sprintf("at least %d holdout recipes publish, with zero material errors", HoldoutMinimumPublished), Passed: true}
	fail := func(criterion *Criterion, format string, args ...any) {
		criterion.Passed = false
		criterion.Failures = append(criterion.Failures, fmt.Sprintf(format, args...))
	}
	describe := func(result CaseResult) string {
		switch result.Status {
		case StatusMaterialErrors:
			return fmt.Sprintf("%s: %d material error(s)", result.Case.ID, len(result.MaterialErrors()))
		default:
			return fmt.Sprintf("%s: %s %v", result.Case.ID, result.Status, result.RejectCodes)
		}
	}

	holdoutCases, holdoutPublished := 0, 0
	for _, result := range results {
		if !result.Case.Expected.Reviewed {
			fail(&reviewed, "%s/%s is an unreviewed draft", result.Case.Part, result.Case.ID)
		}
		switch result.Case.Part {
		case PartMustPass:
			if result.Status != StatusCorrect && result.Status != StatusAcceptedRegression {
				fail(&mustPass, "%s", describe(result))
			}
		case PartExpectedReject:
			if result.Published {
				fail(&expectedReject, "%s: published", result.Case.ID)
			}
		case PartRegression:
			switch result.Status {
			case StatusCorrect, StatusRejectedAsExpected, StatusAcceptedRegression:
			default:
				// A regression recipe got here by publishing with a defect, so it previously
				// published: rejecting it now needs an accepted-regression entry.
				fail(&regression, "%s", describe(result))
			}
		case PartHoldout:
			holdoutCases++
			if result.Published {
				holdoutPublished++
				if result.Status == StatusMaterialErrors {
					fail(&holdout, "%s", describe(result))
				}
			}
		}
	}
	switch {
	case holdoutCases == 0:
		fail(&holdout, "no holdout was supplied (--holdout DIR); an empty holdout fails")
	case holdoutPublished == 0:
		fail(&holdout, "no holdout recipe published; an empty holdout result fails")
	case holdoutPublished < HoldoutMinimumPublished:
		fail(&holdout, "%d of %d holdout recipes published, need at least %d", holdoutPublished, holdoutCases, HoldoutMinimumPublished)
	}

	sites := Criterion{Name: fmt.Sprintf("each active site has at least %d published evaluation recipes", SiteMinimumPublished), Passed: true}
	correctBySite := make(map[string]int)
	for _, result := range results {
		if result.Correct() {
			correctBySite[result.Case.Site()]++
		}
	}
	for _, site := range input.Sites.Sites {
		if site.Active && correctBySite[site.ID] < SiteMinimumPublished {
			fail(&sites, "%s: %d published correctly", site.ID, correctBySite[site.ID])
		}
	}
	return []Criterion{nonEmpty, reviewed, mustPass, expectedReject, regression, holdout, sites}
}

func (t *Tally) count(result CaseResult) {
	t.Cases++
	if result.Published {
		t.Published++
	}
	switch result.Status {
	case StatusCorrect:
		t.Correct++
	case StatusMaterialErrors:
		t.MaterialErrors++
	case StatusRejectedAsExpected:
		t.RejectedAsExpected++
	default:
		t.Rejected++
	}
}

func siteTallies(results []CaseResult, sites registry.Registry) []Tally {
	var tallies []Tally
	for _, site := range sites.Sites {
		tally := Tally{Name: site.ID, Active: site.Active}
		for _, result := range results {
			if result.Case.Site() == site.ID {
				tally.count(result)
			}
		}
		if tally.Cases > 0 || site.Active {
			tallies = append(tallies, tally)
		}
	}
	return tallies
}

func categoryTallies(results []CaseResult) []Tally {
	tallies := make([]Tally, len(Categories))
	for index, category := range Categories {
		tallies[index].Name = string(category)
		for _, result := range results {
			if slices.Contains(result.Case.CaseCategories(), category) {
				tallies[index].count(result)
			}
		}
	}
	return tallies
}
