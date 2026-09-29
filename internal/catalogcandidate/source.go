package catalogcandidate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/mbcoward3/grocery-router/internal/recipepilot"
)

type SourceNode struct {
	ArtifactFile string
	Artifact     recipepilot.SourceArtifact
	Index        int
	Recipe       recipepilot.RecipeCandidate
}

func ReadSourceRun(dir string) ([]byte, recipepilot.RunIndex, []SourceNode, error) {
	indexPath := filepath.Join(dir, "index.json")
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, recipepilot.RunIndex{}, nil, fmt.Errorf("read source run index: %w", err)
	}
	var index recipepilot.RunIndex
	if err := json.Unmarshal(indexData, &index); err != nil {
		return nil, recipepilot.RunIndex{}, nil, fmt.Errorf("decode source run index: %w", err)
	}
	if index.FormatVersion != recipepilot.OutputVersion {
		return nil, recipepilot.RunIndex{}, nil, fmt.Errorf("source run format_version = %d, want %d", index.FormatVersion, recipepilot.OutputVersion)
	}
	nodes := make([]SourceNode, 0)
	for _, entry := range index.Artifacts {
		if entry.Status != "succeeded" {
			continue
		}
		artifactPath := filepath.Join(dir, entry.File)
		artifactData, err := os.ReadFile(artifactPath)
		if err != nil {
			return nil, recipepilot.RunIndex{}, nil, fmt.Errorf("read artifact %s: %w", entry.File, err)
		}
		var artifact recipepilot.SourceArtifact
		if err := json.Unmarshal(artifactData, &artifact); err != nil {
			return nil, recipepilot.RunIndex{}, nil, fmt.Errorf("decode artifact %s: %w", entry.File, err)
		}
		if artifact.FormatVersion != recipepilot.OutputVersion {
			return nil, recipepilot.RunIndex{}, nil, fmt.Errorf("artifact %s format_version = %d, want %d", entry.File, artifact.FormatVersion, recipepilot.OutputVersion)
		}
		if artifact.Source.ID != entry.SourceID {
			return nil, recipepilot.RunIndex{}, nil, fmt.Errorf("artifact %s source id mismatch", entry.File)
		}
		for i, recipe := range artifact.Recipes {
			nodes = append(nodes, SourceNode{ArtifactFile: entry.File, Artifact: artifact, Index: i, Recipe: recipe})
		}
	}
	if len(nodes) > MaxBatchSize {
		return nil, recipepilot.RunIndex{}, nil, fmt.Errorf("source run has %d selected recipes, maximum is %d", len(nodes), MaxBatchSize)
	}
	return indexData, index, nodes, nil
}

func BuildSourceIdentity(indexData []byte, index recipepilot.RunIndex, node SourceNode, artifactBytes []byte) (SourceIdentity, []string, error) {
	if node.Artifact.Fetch == nil {
		return SourceIdentity{}, nil, fmt.Errorf("artifact %s has no fetch metadata", node.ArtifactFile)
	}
	var blockSHA string
	for _, block := range node.Artifact.JSONLDBlocks {
		if block.ScriptIndex == node.Recipe.ScriptIndex {
			blockSHA = block.SHA256
			break
		}
	}
	if blockSHA == "" {
		return SourceIdentity{}, nil, fmt.Errorf("artifact %s recipe script %d has no JSON-LD block", node.ArtifactFile, node.Recipe.ScriptIndex)
	}
	rawDigest := sha256Hex(node.Recipe.Recipe.Raw)
	canonical, err := canonicalJSON(node.Recipe.Recipe.Raw)
	if err != nil {
		return SourceIdentity{}, nil, fmt.Errorf("canonicalize recipe raw: %w", err)
	}
	lines, err := recipeIngredientLines(node.Recipe.Recipe.RecipeIngredient)
	if err != nil {
		return SourceIdentity{}, nil, err
	}
	return SourceIdentity{
		PilotImplementation: index.Implementation, PilotFormatVersion: index.FormatVersion,
		ManifestID: node.Artifact.Source.ID, ManifestURL: node.Artifact.Source.URL,
		RunIndexSHA256: sha256Hex(indexData), ArtifactFile: node.ArtifactFile, ArtifactSHA256: sha256Hex(artifactBytes),
		RequestedURL: node.Artifact.Fetch.RequestedURL, FinalURL: node.Artifact.Fetch.FinalURL,
		FetchedAt: node.Artifact.Fetch.FetchedAt, ResponseBodySHA256: node.Artifact.Fetch.BodySHA256,
		Selected: Selection{CandidateIndex: node.Index, ScriptIndex: node.Recipe.ScriptIndex, JSONPath: node.Recipe.JSONPath, JSONLDBlockSHA256: blockSHA, RecipeRawSHA256: rawDigest, RecipeRawCanonical: sha256Hex(canonical)},
	}, lines, nil
}

func recipeIngredientLines(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("source recipeIngredient is required")
	}
	var lines []string
	if err := json.Unmarshal(raw, &lines); err == nil {
		return lines, nil
	}
	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		return []string{single}, nil
	}
	return nil, fmt.Errorf("source recipeIngredient must be a string array")
}

func canonicalJSON(raw json.RawMessage) ([]byte, error) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return json.Marshal(value)
}

func sha256Hex(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

func sortedJSONFiles(dir string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	return matches, nil
}
