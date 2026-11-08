package qualify

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mbcoward3/grocery-router/internal/ingest"
	"gopkg.in/yaml.v3"
)

// CandidateFormatVersion is the candidate file format version.
const CandidateFormatVersion = 1

// candidateExtension is the file extension of one candidate in a candidate set directory.
const candidateExtension = ".yaml"

// Candidate is one candidate file: the pipeline's result for one case's source node. It is
// published only if it has a document, no rejections, and the document passes
// ingest.CheckPublishable.
type Candidate struct {
	FormatVersion int                     `yaml:"format_version"`
	Document      *ingest.CatalogDocument `yaml:"document,omitempty"`
	// Rejections are recipe-level reasons from checks the publishability check does not
	// evaluate yet (language, alternative coherence, missing notes, instruction discrepancy).
	Rejections []Rejection `yaml:"rejections,omitempty"`
}

// Rejection is one recipe-level rejection reason.
type Rejection struct {
	Code   string `yaml:"code"`
	Detail string `yaml:"detail,omitempty"`
}

// CandidateSet maps case IDs to candidates.
type CandidateSet map[string]Candidate

// ReadCandidateSet reads a directory of candidate files named <case-id>.yaml. Any other
// entry is an error.
func ReadCandidateSet(dir string) (CandidateSet, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read candidate set: %w", err)
	}
	set := make(CandidateSet)
	for _, entry := range entries {
		id, ok := strings.CutSuffix(entry.Name(), candidateExtension)
		if entry.IsDir() || !ok || !caseIDPattern.MatchString(id) {
			return nil, fmt.Errorf("candidate set %s: %q is not a <case-id>%s file", dir, entry.Name(), candidateExtension)
		}
		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		candidate, err := ParseCandidate(content)
		if err != nil {
			return nil, fmt.Errorf("candidate %s: %w", id, err)
		}
		set[id] = candidate
	}
	return set, nil
}

// ParseCandidate strictly decodes one candidate file. The document is decoded strictly but
// not validated here: an invalid document is a rejection, scored by the harness.
func ParseCandidate(content []byte) (Candidate, error) {
	var candidate Candidate
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)
	if err := decoder.Decode(&candidate); err != nil {
		return Candidate{}, fmt.Errorf("decode candidate: %w", err)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return Candidate{}, errors.New("candidate must contain exactly one YAML document")
	}
	if candidate.FormatVersion != CandidateFormatVersion {
		return Candidate{}, fmt.Errorf("format_version = %d, want %d", candidate.FormatVersion, CandidateFormatVersion)
	}
	if candidate.Document == nil && len(candidate.Rejections) == 0 {
		return Candidate{}, errors.New("candidate needs a document or at least one rejection")
	}
	for index, rejection := range candidate.Rejections {
		if !codePattern.MatchString(rejection.Code) {
			return Candidate{}, fmt.Errorf("rejections[%d] code %q is not a snake_case code", index, rejection.Code)
		}
	}
	return candidate, nil
}

// Evaluate applies the publishability check. It reports whether the candidate publishes and,
// if not, every rejection code: recipe-level rejections, publishability issue codes, and the
// reasons of unresolved or unsupported requirements, sorted and unique.
func (c Candidate) Evaluate() (published bool, codes []string) {
	seen := make(map[string]bool)
	addCode := func(code string) {
		if !seen[code] {
			seen[code] = true
			codes = append(codes, code)
		}
	}
	for _, rejection := range c.Rejections {
		addCode(rejection.Code)
	}
	if c.Document != nil {
		for _, issue := range ingest.CheckPublishable(*c.Document) {
			addCode(string(issue.Code))
		}
		for _, section := range c.Document.IngredientSections {
			for _, requirement := range section.Requirements {
				if !requirement.State.Publishable() && requirement.Reason != "" {
					addCode(requirement.Reason)
				}
			}
		}
	} else {
		addCode("no_document")
	}
	sort.Strings(codes)
	return len(codes) == 0, codes
}

// AcceptedRegressionsFormatVersion is the accepted-regressions manifest format version.
const AcceptedRegressionsFormatVersion = 1

// AcceptedRegressions is the manifest of must-pass or regression cases allowed to reject
// under this release, each with a reason (§6.3).
type AcceptedRegressions struct {
	FormatVersion int                  `yaml:"format_version"`
	Accepted      []AcceptedRegression `yaml:"accepted"`
}

// AcceptedRegression is one accepted regression.
type AcceptedRegression struct {
	Case   string `yaml:"case"`
	Reason string `yaml:"reason"`
}

// ReadAcceptedRegressions strictly reads an accepted-regressions manifest file.
func ReadAcceptedRegressions(path string) (AcceptedRegressions, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return AcceptedRegressions{}, fmt.Errorf("read accepted regressions: %w", err)
	}
	var manifest AcceptedRegressions
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)
	if err := decoder.Decode(&manifest); err != nil {
		return AcceptedRegressions{}, fmt.Errorf("decode accepted regressions: %w", err)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return AcceptedRegressions{}, errors.New("accepted regressions must contain exactly one YAML document")
	}
	if manifest.FormatVersion != AcceptedRegressionsFormatVersion {
		return AcceptedRegressions{}, fmt.Errorf("accepted regressions format_version = %d, want %d", manifest.FormatVersion, AcceptedRegressionsFormatVersion)
	}
	seen := make(map[string]bool)
	for index, accepted := range manifest.Accepted {
		if !caseIDPattern.MatchString(accepted.Case) || strings.TrimSpace(accepted.Reason) == "" {
			return AcceptedRegressions{}, fmt.Errorf("accepted[%d] needs a case ID and a reason", index)
		}
		if seen[accepted.Case] {
			return AcceptedRegressions{}, fmt.Errorf("accepted[%d] repeats case %q", index, accepted.Case)
		}
		seen[accepted.Case] = true
	}
	return manifest, nil
}
