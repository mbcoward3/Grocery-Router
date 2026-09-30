package recipeintel

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mbcoward3/grocery-router/internal/ingest"
	"github.com/mbcoward3/grocery-router/internal/recipepilot"
)

// Output format and implementation identifiers make local evidence replayable.
const (
	OutputVersion  = 1
	Implementation = "grocery-router-jev-assessment/v1"
)

// Runner assesses all loaded recipe candidates and publishes an immutable local run.
type Runner struct {
	Evaluator Evaluator
	Now       func() time.Time
}

// RunConfig identifies all local inputs and the ignored runtime-output root.
type RunConfig struct {
	CatalogPath string
	SourcePath  string
	CorpusPath  string
	OutputRoot  string
}

// RunReport identifies the published run and its summary.
type RunReport struct {
	Directory string
	Index     RunIndex
}

// RunIndex summarizes provenance, usage, and every candidate artifact.
type RunIndex struct {
	FormatVersion  int             `json:"format_version"`
	Implementation string          `json:"implementation"`
	StartedAt      time.Time       `json:"started_at"`
	CatalogPath    string          `json:"catalog_path"`
	CatalogSHA256  string          `json:"catalog_sha256"`
	CatalogVersion int             `json:"catalog_version"`
	RequestedModel string          `json:"requested_model"`
	SourceRun      string          `json:"source_run,omitempty"`
	CorpusPath     string          `json:"corpus_path,omitempty"`
	Candidates     int             `json:"candidates"`
	Succeeded      int             `json:"succeeded"`
	Failed         int             `json:"failed"`
	InputTokens    int64           `json:"input_tokens"`
	OutputTokens   int64           `json:"output_tokens"`
	Artifacts      []ArtifactIndex `json:"artifacts"`
}

// ArtifactIndex records one candidate's output file and disposition.
type ArtifactIndex struct {
	CandidateID string `json:"candidate_id"`
	Name        string `json:"name"`
	File        string `json:"file"`
	Status      string `json:"status"`
	Error       string `json:"error,omitempty"`
}

// AssessmentArtifact preserves one candidate's states, questions, and raw answers.
type AssessmentArtifact struct {
	FormatVersion  int                    `json:"format_version"`
	Implementation string                 `json:"implementation"`
	Candidate      CandidateMetadata      `json:"candidate"`
	CatalogSHA256  string                 `json:"catalog_sha256"`
	CatalogVersion int                    `json:"catalog_version"`
	RequestedModel string                 `json:"requested_model"`
	Projections    []ProjectionAssessment `json:"projections"`
	Error          string                 `json:"error,omitempty"`
}

// CandidateMetadata identifies the exact candidate represented by an artifact.
type CandidateMetadata struct {
	ID         string `json:"id"`
	SourceKind string `json:"source_kind"`
	SourceID   string `json:"source_id"`
	Candidate  int    `json:"candidate_index,omitempty"`
	Name       string `json:"name"`
}

// ProjectionAssessment records one focused request and its response or error.
type ProjectionAssessment struct {
	Name        string                 `json:"name"`
	InputSHA256 string                 `json:"input_sha256"`
	State       ProjectedState         `json:"state"`
	Questions   map[string]APIQuestion `json:"questions"`
	Response    *EvaluationResponse    `json:"response,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// Run validates all inputs before evaluating candidates and atomically publishes output.
func (runner Runner) Run(ctx context.Context, config RunConfig) (RunReport, error) {
	if runner.Evaluator == nil {
		return RunReport{}, fmt.Errorf("recipe intelligence evaluator is required")
	}
	catalogData, err := os.ReadFile(config.CatalogPath)
	if err != nil {
		return RunReport{}, fmt.Errorf("read recipe intelligence catalog: %w", err)
	}
	catalog, catalogDigest, err := ReadCatalog(catalogData)
	if err != nil {
		return RunReport{}, err
	}
	grouped, err := catalog.QuestionsByProjection()
	if err != nil {
		return RunReport{}, err
	}
	candidates, sourceRun, err := loadCandidates(config.SourcePath, config.CorpusPath)
	if err != nil {
		return RunReport{}, err
	}
	if len(candidates) == 0 {
		return RunReport{}, fmt.Errorf("recipe intelligence run has no candidates")
	}
	startedAt := runner.now().UTC()
	if err := os.MkdirAll(config.OutputRoot, 0o755); err != nil {
		return RunReport{}, fmt.Errorf("create recipe intelligence output root: %w", err)
	}
	temporary, err := os.MkdirTemp(config.OutputRoot, ".run-")
	if err != nil {
		return RunReport{}, fmt.Errorf("create recipe intelligence temporary directory: %w", err)
	}
	completed := false
	defer func() {
		if !completed {
			_ = os.RemoveAll(temporary)
		}
	}()

	index := RunIndex{
		FormatVersion: OutputVersion, Implementation: Implementation, StartedAt: startedAt,
		CatalogPath: config.CatalogPath, CatalogSHA256: catalogDigest, CatalogVersion: catalog.Version,
		RequestedModel: catalog.Model, SourceRun: sourceRun, CorpusPath: config.CorpusPath,
		Candidates: len(candidates), Artifacts: make([]ArtifactIndex, 0, len(candidates)),
	}
	projectionNames := SortedProjectionNames(grouped)
	for _, candidate := range candidates {
		artifact := AssessmentArtifact{
			FormatVersion: OutputVersion, Implementation: Implementation,
			Candidate:     CandidateMetadata{ID: candidate.ID, SourceKind: candidate.SourceKind, SourceID: candidate.SourceID, Candidate: candidate.Candidate, Name: candidate.Name},
			CatalogSHA256: catalogDigest, CatalogVersion: catalog.Version, RequestedModel: catalog.Model,
			Projections: make([]ProjectionAssessment, 0, len(projectionNames)),
		}
		for _, projectionName := range projectionNames {
			state, projectionErr := candidate.Project(projectionName)
			assessment := ProjectionAssessment{Name: projectionName, State: state, Questions: grouped[projectionName]}
			if projectionErr == nil {
				canonical, encodeErr := marshalCanonical(state)
				if encodeErr != nil {
					projectionErr = encodeErr
				} else {
					assessment.InputSHA256 = sha256Hex(canonical)
				}
			}
			if projectionErr == nil {
				response, evaluateErr := runner.Evaluator.Evaluate(ctx, EvaluationRequest{State: state, Model: catalog.Model, Questions: grouped[projectionName]})
				if evaluateErr != nil {
					projectionErr = evaluateErr
				} else {
					assessment.Response = &response
					index.InputTokens += response.Usage.InputTokens
					index.OutputTokens += response.Usage.OutputTokens
				}
			}
			if projectionErr != nil {
				assessment.Error = projectionErr.Error()
				artifact.Error = fmt.Sprintf("projection %s: %v", projectionName, projectionErr)
			}
			artifact.Projections = append(artifact.Projections, assessment)
			if projectionErr != nil {
				break
			}
		}
		status := "succeeded"
		if artifact.Error != "" {
			status = "failed"
			index.Failed++
		} else {
			index.Succeeded++
		}
		fileName := safeFileName(candidate.ID) + ".json"
		if err := writeJSON(filepath.Join(temporary, fileName), artifact); err != nil {
			return RunReport{}, err
		}
		index.Artifacts = append(index.Artifacts, ArtifactIndex{CandidateID: candidate.ID, Name: candidate.Name, File: fileName, Status: status, Error: artifact.Error})
	}
	if err := writeJSON(filepath.Join(temporary, "index.json"), index); err != nil {
		return RunReport{}, err
	}
	if err := writeReviewReport(filepath.Join(temporary, "review.md"), index, temporary); err != nil {
		return RunReport{}, err
	}
	finalDirectory := filepath.Join(config.OutputRoot, startedAt.Format("20060102T150405.000000000Z"))
	if err := os.Rename(temporary, finalDirectory); err != nil {
		return RunReport{}, fmt.Errorf("publish recipe intelligence run: %w", err)
	}
	completed = true
	report := RunReport{Directory: finalDirectory, Index: index}
	if index.Failed > 0 {
		return report, fmt.Errorf("recipe intelligence run completed with %d of %d candidates failed", index.Failed, index.Candidates)
	}
	return report, nil
}

func loadCandidates(sourcePath, corpusPath string) ([]Candidate, string, error) {
	var candidates []Candidate
	resolvedSource := ""
	if sourcePath != "" {
		var err error
		resolvedSource, err = resolveSourceRun(sourcePath)
		if err != nil {
			return nil, "", err
		}
		indexData, err := os.ReadFile(filepath.Join(resolvedSource, "index.json"))
		if err != nil {
			return nil, "", fmt.Errorf("read source run index: %w", err)
		}
		var index recipepilot.RunIndex
		if err := json.Unmarshal(indexData, &index); err != nil {
			return nil, "", fmt.Errorf("decode source run index: %w", err)
		}
		for _, entry := range index.Artifacts {
			if entry.Status != "succeeded" {
				continue
			}
			data, err := os.ReadFile(filepath.Join(resolvedSource, entry.File))
			if err != nil {
				return nil, "", fmt.Errorf("read source artifact %q: %w", entry.File, err)
			}
			var artifact recipepilot.SourceArtifact
			if err := json.Unmarshal(data, &artifact); err != nil {
				return nil, "", fmt.Errorf("decode source artifact %q: %w", entry.File, err)
			}
			sourceCandidates, err := CandidatesFromSourceArtifact(artifact)
			if err != nil {
				return nil, "", err
			}
			candidates = append(candidates, sourceCandidates...)
		}
	}
	if corpusPath != "" {
		documents, err := ingest.ReadDirectory(corpusPath)
		if err != nil {
			return nil, "", fmt.Errorf("read approved corpus for intelligence: %w", err)
		}
		for _, document := range documents {
			candidates = append(candidates, CandidateFromDocument(document))
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
	return candidates, resolvedSource, nil
}

func resolveSourceRun(path string) (string, error) {
	if _, err := os.Stat(filepath.Join(path, "index.json")); err == nil {
		return path, nil
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("read source pilot output root: %w", err)
	}
	var runs []string
	for _, entry := range entries {
		if entry.IsDir() {
			candidate := filepath.Join(path, entry.Name())
			if _, err := os.Stat(filepath.Join(candidate, "index.json")); err == nil {
				runs = append(runs, candidate)
			}
		}
	}
	if len(runs) == 0 {
		return "", fmt.Errorf("source pilot output %q contains no runs", path)
	}
	sort.Strings(runs)
	return runs[len(runs)-1], nil
}

func writeReviewReport(path string, index RunIndex, directory string) error {
	var report strings.Builder
	fmt.Fprintf(&report, "# Recipe intelligence review\n\n")
	fmt.Fprintf(&report, "- Started: %s\n- Requested model: `%s`\n- Catalog: `%s`\n- Candidates: %d succeeded, %d failed\n- Tokens: %d input, %d output\n\n", index.StartedAt.Format(time.RFC3339), index.RequestedModel, index.CatalogSHA256, index.Succeeded, index.Failed, index.InputTokens, index.OutputTokens)
	for _, entry := range index.Artifacts {
		fmt.Fprintf(&report, "## %s\n\n- Candidate: `%s`\n- Status: **%s**\n", entry.Name, entry.CandidateID, entry.Status)
		if entry.Error != "" {
			fmt.Fprintf(&report, "- Error: %s\n\n", entry.Error)
			continue
		}
		data, err := os.ReadFile(filepath.Join(directory, entry.File))
		if err != nil {
			return err
		}
		var artifact AssessmentArtifact
		if err := json.Unmarshal(data, &artifact); err != nil {
			return err
		}
		report.WriteString("\n| Question | Answer | Confidence |\n|---|---:|---:|\n")
		for _, projection := range artifact.Projections {
			if projection.Response == nil {
				continue
			}
			keys := make([]string, 0, len(projection.Response.Answers))
			for key := range projection.Response.Answers {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				var answer Answer
				if err := json.Unmarshal(projection.Response.Answers[key], &answer); err != nil {
					return err
				}
				value, confidence := displayAnswer(answer)
				fmt.Fprintf(&report, "| `%s` | %s | %s |\n", key, value, confidence)
			}
		}
		report.WriteString("\n")
	}
	if err := os.WriteFile(path, []byte(report.String()), 0o644); err != nil {
		return fmt.Errorf("write review report: %w", err)
	}
	return nil
}

func displayAnswer(answer Answer) (string, string) {
	switch answer.Type {
	case "noul":
		return fmt.Sprintf("%.3f", *answer.Noul), "—"
	case "choice":
		return "`" + answer.Choice + "`", fmt.Sprintf("%.3f", *answer.Confidence)
	case "score":
		return fmt.Sprintf("%.3f", *answer.Score), fmt.Sprintf("%.3f", *answer.Confidence)
	default:
		return "unknown", "—"
	}
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", filepath.Base(path), err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}
	return nil
}

func safeFileName(value string) string {
	value = strings.ReplaceAll(value, string(filepath.Separator), "-")
	return strings.ReplaceAll(value, "..", "-")
}

func sha256Hex(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func (runner Runner) now() time.Time {
	if runner.Now != nil {
		return runner.Now()
	}
	return time.Now()
}
