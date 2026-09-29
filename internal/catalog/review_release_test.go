package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommittedReviewReleaseAuditsAllDigests(t *testing.T) {
	root := filepath.Join("..", "..")
	manifest, _, err := LoadReviewManifest(root, "catalog/releases/initial-review-2026-09-29.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Recipes) != 17 || manifest.Status != "reviewable" {
		t.Fatalf("release = %d %s", len(manifest.Recipes), manifest.Status)
	}
}

func TestReviewReleaseRejectsArtifactDrift(t *testing.T) {
	root := filepath.Join("..", "..")
	manifestPath := filepath.Join(root, "catalog", "releases", "initial-review-2026-09-29.yaml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	temp := t.TempDir()
	candidate := filepath.Join(temp, "candidate.json")
	profile := filepath.Join(temp, "profile.json")
	if err := os.WriteFile(candidate, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(profile, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	text := string(data)
	first := strings.Index(text, "  candidate: ")
	if first < 0 {
		t.Fatal("manifest candidate missing")
	}
	// A digest-only mutation is sufficient to prove audit fails before database work.
	text = strings.Replace(text, "sha256:", "sha256:0", 1)
	bad := filepath.Join(temp, "release.yaml")
	if err := os.WriteFile(bad, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadReviewManifest(root, bad); err == nil {
		t.Fatal("drifted release unexpectedly passed")
	}
}
