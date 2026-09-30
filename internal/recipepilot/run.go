package recipepilot

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Runner executes a complete manifest and publishes a new evidence directory.
type Runner struct {
	Fetcher Fetcher
	Now     func() time.Time
}

// Run validates before fetching, attempts every source, and atomically publishes output.
func (runner Runner) Run(ctx context.Context, manifestPath, outputRoot string) (RunReport, error) {
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return RunReport{}, fmt.Errorf("read manifest: %w", err)
	}
	manifest, digest, err := ReadManifest(manifestData)
	if err != nil {
		return RunReport{}, err
	}
	if runner.Fetcher == nil {
		return RunReport{}, fmt.Errorf("pilot fetcher is required")
	}
	startedAt := runner.now().UTC()
	if err := os.MkdirAll(outputRoot, 0o755); err != nil {
		return RunReport{}, fmt.Errorf("create output root: %w", err)
	}
	temporary, err := os.MkdirTemp(outputRoot, ".run-")
	if err != nil {
		return RunReport{}, fmt.Errorf("create temporary run directory: %w", err)
	}
	completed := false
	defer func() {
		if !completed {
			_ = os.RemoveAll(temporary)
		}
	}()

	index := RunIndex{
		FormatVersion: OutputVersion, Implementation: Implementation, StartedAt: startedAt,
		ManifestPath: manifestPath, ManifestSHA256: digest, Sources: len(manifest.Sources),
		Artifacts: make([]ArtifactIndex, 0, len(manifest.Sources)),
	}
	for _, source := range manifest.Sources {
		artifact := SourceArtifact{FormatVersion: OutputVersion, Source: source}
		result, fetchErr := runner.Fetcher.Fetch(ctx, source.URL)
		if result.Metadata.RequestedURL == "" {
			result.Metadata.RequestedURL = source.URL
		}
		artifact.Fetch = &result.Metadata
		var sourceErr error
		if fetchErr != nil {
			sourceErr = fetchErr
		} else {
			extraction, extractErr := Extract(result.Body, result.Metadata.FinalURL)
			artifact.Canonical = extraction.Canonical
			artifact.JSONLDBlocks = extraction.JSONLDBlocks
			artifact.Recipes = extraction.Recipes
			artifact.Warnings = extraction.Warnings
			sourceErr = extractErr
		}
		status := "succeeded"
		if sourceErr != nil {
			status = "failed"
			artifact.Error = sourceErr.Error()
			index.Failed++
		} else {
			index.Succeeded++
		}
		fileName := source.ID + ".json"
		if err := writeJSON(filepath.Join(temporary, fileName), artifact); err != nil {
			return RunReport{}, err
		}
		index.Artifacts = append(index.Artifacts, ArtifactIndex{
			SourceID: source.ID, File: fileName, Status: status, Error: artifact.Error,
		})
	}
	if err := writeJSON(filepath.Join(temporary, "index.json"), index); err != nil {
		return RunReport{}, err
	}
	runName := startedAt.Format("20060102T150405.000000000Z")
	finalDirectory := filepath.Join(outputRoot, runName)
	if err := os.Rename(temporary, finalDirectory); err != nil {
		return RunReport{}, fmt.Errorf("publish run directory: %w", err)
	}
	completed = true
	report := RunReport{Directory: finalDirectory, Index: index}
	if index.Failed > 0 {
		return report, fmt.Errorf("recipe source pilot completed with %d of %d sources failed", index.Failed, index.Sources)
	}
	return report, nil
}

func (runner Runner) now() time.Time {
	if runner.Now != nil {
		return runner.Now()
	}
	return time.Now()
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
