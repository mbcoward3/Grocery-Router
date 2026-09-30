package catalogcandidate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// BuildOptions identifies local evidence, untrusted proposals, and artifact output.
type BuildOptions struct {
	SourceRunDir string
	ProposalDir  string
	OutputDir    string
	Agent        AgentProvenance
	Now          func() time.Time
}

// BatchReport returns the validated candidates and review index location.
type BatchReport struct {
	Candidates []Candidate
	IndexPath  string
}

// ReviewIndex summarizes a deterministic candidate batch.
type ReviewIndex struct {
	FormatVersion  int               `json:"format_version"`
	Implementation string            `json:"implementation"`
	GeneratedAt    time.Time         `json:"generated_at"`
	SourceRunDir   string            `json:"source_run_dir"`
	ProposalDir    string            `json:"proposal_dir"`
	Candidates     []ReviewIndexItem `json:"candidates"`
}

// ReviewIndexItem points to one candidate and rendered review.
type ReviewIndexItem struct {
	CandidateID string `json:"candidate_id"`
	CatalogKey  string `json:"catalog_key"`
	JSONPath    string `json:"json_path"`
	SourceID    string `json:"source_id"`
	JSONFile    string `json:"json_file"`
	Markdown    string `json:"markdown"`
	SHA256      string `json:"sha256"`
	Blockers    int    `json:"blockers"`
}

// BuildBatch validates source coverage and writes deterministic review artifacts.
func BuildBatch(opts BuildOptions) (BatchReport, error) {
	if opts.SourceRunDir == "" || opts.ProposalDir == "" || opts.OutputDir == "" {
		return BatchReport{}, fmt.Errorf("source run, proposal, and output directories are required")
	}
	indexData, index, nodes, err := ReadSourceRun(opts.SourceRunDir)
	if err != nil {
		return BatchReport{}, err
	}
	proposalFiles, err := sortedJSONFiles(opts.ProposalDir)
	if err != nil {
		return BatchReport{}, fmt.Errorf("list proposals: %w", err)
	}
	if len(proposalFiles) > MaxBatchSize {
		return BatchReport{}, fmt.Errorf("proposal batch has %d candidates, maximum is %d", len(proposalFiles), MaxBatchSize)
	}
	if len(proposalFiles) != len(nodes) {
		return BatchReport{}, fmt.Errorf("proposal count %d does not match selected source recipes %d", len(proposalFiles), len(nodes))
	}
	proposalsByName := make(map[string]string, len(proposalFiles))
	for _, proposalFile := range proposalFiles {
		proposalsByName[filepath.Base(proposalFile)] = proposalFile
	}
	if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil {
		return BatchReport{}, fmt.Errorf("create output dir: %w", err)
	}
	generatedAt := time.Now().UTC()
	if opts.Now != nil {
		generatedAt = opts.Now().UTC()
	}
	review := ReviewIndex{FormatVersion: FormatVersion, Implementation: Implementation, GeneratedAt: generatedAt, SourceRunDir: opts.SourceRunDir, ProposalDir: opts.ProposalDir, Candidates: make([]ReviewIndexItem, 0, len(nodes))}
	seenKeys := make(map[string]bool)
	outCandidates := make([]Candidate, 0, len(nodes))
	for _, node := range nodes {
		proposalName := fmt.Sprintf("%s--%d.json", node.Artifact.Source.ID, node.Index)
		proposalFile, ok := proposalsByName[proposalName]
		if !ok && len(nodes) == 1 && len(proposalFiles) == 1 {
			// A single-candidate fixture or handoff is unambiguous even when its file is renamed.
			proposalFile, ok = proposalFiles[0], true
		}
		if !ok {
			return BatchReport{}, fmt.Errorf("selected source recipe %s has no proposal %s", node.Artifact.Source.ID, proposalName)
		}
		proposalData, err := os.ReadFile(proposalFile)
		if err != nil {
			return BatchReport{}, fmt.Errorf("read proposal %s: %w", proposalName, err)
		}
		proposal, err := proposalFromBytes(proposalData)
		if err != nil {
			return BatchReport{}, fmt.Errorf("%s: %w", proposalName, err)
		}
		artifactData, err := os.ReadFile(filepath.Join(opts.SourceRunDir, node.ArtifactFile))
		if err != nil {
			return BatchReport{}, err
		}
		source, lines, err := BuildSourceIdentity(indexData, index, node, artifactData)
		if err != nil {
			return BatchReport{}, err
		}
		agent := opts.Agent
		agent.OutputSHA256 = sha256Hex(proposalData)
		candidate := proposal.toCandidate(source, agent)
		if seenKeys[candidate.CatalogKey] {
			return BatchReport{}, fmt.Errorf("duplicate catalog key %q", candidate.CatalogKey)
		}
		seenKeys[candidate.CatalogKey] = true
		if err := candidate.Validate(lines); err != nil {
			return BatchReport{}, fmt.Errorf("candidate %s: %w", candidate.CatalogKey, err)
		}
		candidateJSON, err := json.MarshalIndent(candidate, "", "  ")
		if err != nil {
			return BatchReport{}, err
		}
		candidateJSON = append(candidateJSON, '\n')
		base := candidate.CatalogKey
		jsonName := base + ".candidate.json"
		mdName := base + ".review.md"
		if err := os.WriteFile(filepath.Join(opts.OutputDir, jsonName), candidateJSON, 0o644); err != nil {
			return BatchReport{}, fmt.Errorf("write candidate: %w", err)
		}
		if err := os.WriteFile(filepath.Join(opts.OutputDir, mdName), RenderMarkdown(candidate), 0o644); err != nil {
			return BatchReport{}, fmt.Errorf("write review markdown: %w", err)
		}
		review.Candidates = append(review.Candidates, ReviewIndexItem{CandidateID: candidate.CandidateID, CatalogKey: candidate.CatalogKey, JSONPath: candidate.Source.Selected.JSONPath, SourceID: candidate.Source.ManifestID, JSONFile: jsonName, Markdown: mdName, SHA256: sha256Hex(candidateJSON), Blockers: len(blockerLines(candidate))})
		outCandidates = append(outCandidates, candidate)
	}
	sort.Slice(review.Candidates, func(i, j int) bool { return review.Candidates[i].CatalogKey < review.Candidates[j].CatalogKey })
	indexJSON, err := json.MarshalIndent(review, "", "  ")
	if err != nil {
		return BatchReport{}, err
	}
	indexJSON = append(indexJSON, '\n')
	indexPath := filepath.Join(opts.OutputDir, "review-index.json")
	if err := os.WriteFile(indexPath, indexJSON, 0o644); err != nil {
		return BatchReport{}, fmt.Errorf("write review index: %w", err)
	}
	md := renderReviewIndexMarkdown(review)
	if err := os.WriteFile(filepath.Join(opts.OutputDir, "review-index.md"), md, 0o644); err != nil {
		return BatchReport{}, fmt.Errorf("write review index markdown: %w", err)
	}
	return BatchReport{Candidates: outCandidates, IndexPath: indexPath}, nil
}

func renderReviewIndexMarkdown(index ReviewIndex) []byte {
	var b strings.Builder
	b.WriteString("# Catalog candidate review index\n\n")
	fmt.Fprintf(&b, "Generated: %s\n\n", index.GeneratedAt.Format(time.RFC3339))
	for _, item := range index.Candidates {
		marker := ""
		if item.Blockers > 0 {
			marker = fmt.Sprintf(" — **%d blocker(s)**", item.Blockers)
		}
		fmt.Fprintf(&b, "- `%s` from `%s`: [%s](%s)%s\n", item.CatalogKey, item.SourceID, item.Markdown, item.Markdown, marker)
	}
	return []byte(b.String())
}
