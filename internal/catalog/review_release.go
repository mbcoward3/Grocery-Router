package catalog

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mbcoward3/grocery-router/internal/catalogcandidate"
	"github.com/mbcoward3/grocery-router/internal/store"
	"gopkg.in/yaml.v3"
)

// ReviewManifest is an immutable, local-evidence-free batch suitable for development review.
type ReviewManifest struct {
	FormatVersion int                    `yaml:"format_version"`
	ReleaseID     string                 `yaml:"release_id"`
	Status        string                 `yaml:"status"`
	Recipes       []ReviewManifestRecipe `yaml:"recipes"`
}

// ReviewManifestRecipe references one committed candidate and compact profile by digest.
type ReviewManifestRecipe struct {
	Key                  string `yaml:"key"`
	Status               string `yaml:"status"`
	Candidate            string `yaml:"candidate"`
	CandidateSHA256      string `yaml:"candidate_sha256"`
	Profile              string `yaml:"profile"`
	ProfileSHA256        string `yaml:"profile_sha256"`
	SourceIdentitySHA256 string `yaml:"source_identity_sha256"`
	SelectedRecipeSHA256 string `yaml:"selected_recipe_sha256"`
}

// LoadReviewManifest strictly loads and validates a review release and all referenced files.
func LoadReviewManifest(root, path string) (ReviewManifest, []byte, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		return ReviewManifest{}, nil, fmt.Errorf("read catalog release: %w", err)
	}
	var manifest ReviewManifest
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&manifest); err != nil {
		return ReviewManifest{}, nil, fmt.Errorf("decode catalog release: %w", err)
	}
	if manifest.FormatVersion != 1 || manifest.ReleaseID == "" || manifest.Status != "reviewable" || len(manifest.Recipes) == 0 || len(manifest.Recipes) > catalogcandidate.MaxBatchSize {
		return ReviewManifest{}, nil, fmt.Errorf("invalid review release header or recipe count")
	}
	seen := map[string]bool{}
	for _, entry := range manifest.Recipes {
		if entry.Status != "reviewable" || entry.Key == "" || seen[entry.Key] {
			return ReviewManifest{}, nil, fmt.Errorf("invalid or duplicate review recipe %q", entry.Key)
		}
		seen[entry.Key] = true
		for file, wanted := range map[string]string{entry.Candidate: entry.CandidateSHA256, entry.Profile: entry.ProfileSHA256} {
			contents, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
			if err != nil {
				return ReviewManifest{}, nil, fmt.Errorf("read %s: %w", file, err)
			}
			if got := prefixedDigest(contents); got != wanted {
				return ReviewManifest{}, nil, fmt.Errorf("%s digest = %s, want %s", file, got, wanted)
			}
		}
	}
	return manifest, data, nil
}

// PublishReviewManifest transactionally exposes non-adoptable candidates for authenticated development review.
func (s *Service) PublishReviewManifest(ctx context.Context, root string, manifest ReviewManifest, manifestBytes []byte) (PublishResult, error) {
	releaseDigest := prefixedDigest(manifestBytes)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return PublishResult{}, err
	}
	defer tx.Rollback()
	q := store.New(tx)
	existing, err := q.GetCatalogAppliedRelease(ctx, manifest.ReleaseID)
	if err == nil {
		if existing.ReleaseDigest == releaseDigest {
			return PublishResult{Applied: false}, nil
		}
		return PublishResult{}, fmt.Errorf("%w: release %s already has another digest", ErrConflict, manifest.ReleaseID)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return PublishResult{}, err
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return PublishResult{}, err
	}
	if _, err := q.CreateCatalogAppliedRelease(ctx, store.CreateCatalogAppliedReleaseParams{ReleaseID: manifest.ReleaseID, ReleaseDigest: releaseDigest, Manifest: manifestJSON}); err != nil {
		return PublishResult{}, err
	}
	for _, entry := range manifest.Recipes {
		candidateBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(entry.Candidate)))
		if err != nil {
			return PublishResult{}, err
		}
		candidate, err := catalogcandidate.DecodeCandidate(candidateBytes)
		if err != nil {
			return PublishResult{}, fmt.Errorf("decode %s: %w", entry.Candidate, err)
		}
		if err := candidate.Validate(nil); err != nil {
			return PublishResult{}, fmt.Errorf("validate %s: %w", entry.Candidate, err)
		}
		if candidate.CatalogKey != entry.Key || candidate.State == "approved" {
			return PublishResult{}, fmt.Errorf("candidate %s does not match review entry", entry.Key)
		}
		profile, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(entry.Profile)))
		if err != nil {
			return PublishResult{}, err
		}
		sourceJSON, err := json.Marshal(candidate.Source)
		if err != nil {
			return PublishResult{}, err
		}
		if got := prefixedDigestCanonical(sourceJSON); got != entry.SourceIdentitySHA256 {
			return PublishResult{}, fmt.Errorf("candidate %s source identity digest mismatch", entry.Key)
		}
		if _, err := q.CreateCatalogRecipe(ctx, store.CreateCatalogRecipeParams{
			Key: entry.Key, Name: candidate.Name.Value, Status: "reviewable", RecipeDocument: candidateBytes,
			SourceIdentity: sourceJSON, SemanticProfile: profile, SourceUrl: nullString(candidate.Source.FinalURL),
			SourceAttribution: candidate.Name.Value + " source", DocumentDigest: entry.CandidateSHA256,
			SourceIdentityDigest: entry.SourceIdentitySHA256, SelectedRecipeDigest: entry.SelectedRecipeSHA256,
			ReleaseID: manifest.ReleaseID, ReleaseDigest: releaseDigest,
		}); err != nil {
			return PublishResult{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return PublishResult{}, err
	}
	return PublishResult{Applied: true}, nil
}

func prefixedDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func prefixedDigestCanonical(data []byte) string {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return prefixedDigest(data)
	}
	canonical, _ := json.Marshal(value)
	return prefixedDigest(canonical)
}
