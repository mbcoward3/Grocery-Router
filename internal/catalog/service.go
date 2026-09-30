// Package catalog publishes global shared recipe catalog releases and materializes
// verified catalog recipes into tenant-scoped household collections.
package catalog

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/mbcoward3/grocery-router/internal/ingest"
	"github.com/mbcoward3/grocery-router/internal/store"
)

var (
	// ErrConflict means an immutable release or recipe identity already has different content.
	ErrConflict = errors.New("catalog conflict")
	// ErrNotVerified means a reviewable candidate cannot enter a household collection.
	ErrNotVerified = errors.New("catalog recipe is not verified")
)

// Service owns catalog publication and household membership transactions.
type Service struct{ db *sql.DB }

// NewService constructs transactional catalog publication and adoption services.
func NewService(db *sql.DB) *Service { return &Service{db: db} }

// Release is one immutable approved catalog publication batch.
type Release struct {
	ID       string
	Digest   string
	Manifest json.RawMessage
	Recipes  []ReleaseRecipe
}

// ReleaseRecipe combines approved recipe truth, semantic profile, and evidence digests.
type ReleaseRecipe struct {
	Document             ingest.Document
	Status               string
	SourceIdentity       json.RawMessage
	SemanticProfile      json.RawMessage
	DocumentDigest       string
	SourceIdentityDigest string
	SelectedRecipeDigest string
}

// PublishResult reports whether an exact release caused database writes.
type PublishResult struct{ Applied bool }

// PublishRelease validates and transactionally applies an approved catalog release.
func (s *Service) PublishRelease(ctx context.Context, rel Release) (PublishResult, error) {
	if rel.ID == "" || rel.Digest == "" || len(rel.Recipes) == 0 {
		return PublishResult{}, fmt.Errorf("release id, digest, and recipes are required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return PublishResult{}, err
	}
	defer tx.Rollback()
	q := store.New(tx)
	existing, err := q.GetCatalogAppliedRelease(ctx, rel.ID)
	if err == nil {
		if existing.ReleaseDigest == rel.Digest {
			return PublishResult{Applied: false}, nil
		}
		return PublishResult{}, fmt.Errorf("%w: release %s already applied with different digest", ErrConflict, rel.ID)
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return PublishResult{}, err
	}
	manifest := rel.Manifest
	if len(manifest) == 0 {
		manifest = []byte(`{}`)
	}
	if _, err := q.CreateCatalogAppliedRelease(ctx, store.CreateCatalogAppliedReleaseParams{ReleaseID: rel.ID, ReleaseDigest: rel.Digest, Manifest: manifest}); err != nil {
		return PublishResult{}, err
	}
	for _, recipe := range rel.Recipes {
		if recipe.Status != "reviewable" && recipe.Status != "verified" {
			return PublishResult{}, fmt.Errorf("recipe %s has invalid status %q", recipe.Document.Key, recipe.Status)
		}
		doc := recipe.Document
		doc.Status = "verified"
		if err := doc.Validate(); err != nil {
			return PublishResult{}, err
		}
		doc.Status = recipe.Status
		docJSON, err := json.Marshal(doc)
		if err != nil {
			return PublishResult{}, err
		}
		if recipe.DocumentDigest == "" {
			recipe.DocumentDigest = digest(docJSON)
		}
		profile := recipe.SemanticProfile
		if len(profile) == 0 {
			profile = []byte(`{}`)
		}
		if len(recipe.SourceIdentity) == 0 {
			recipe.SourceIdentity = []byte(`{}`)
		}
		if err := s.ensureNoConflicts(ctx, q, recipe); err != nil {
			return PublishResult{}, err
		}
		_, err = q.CreateCatalogRecipe(ctx, store.CreateCatalogRecipeParams{Key: recipe.Document.Key, Name: recipe.Document.Name, Status: recipe.Status, RecipeDocument: docJSON, SourceIdentity: recipe.SourceIdentity, SemanticProfile: profile, SourceUrl: nullString(recipe.Document.Source.URL), SourceAttribution: recipe.Document.Source.Attribution, DocumentDigest: recipe.DocumentDigest, SourceIdentityDigest: recipe.SourceIdentityDigest, SelectedRecipeDigest: recipe.SelectedRecipeDigest, ReleaseID: rel.ID, ReleaseDigest: rel.Digest})
		if err != nil {
			return PublishResult{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return PublishResult{}, err
	}
	return PublishResult{Applied: true}, nil
}

func (s *Service) ensureNoConflicts(ctx context.Context, q *store.Queries, r ReleaseRecipe) error {
	checks := []struct {
		name, value string
		get         func(context.Context, string) (store.CatalogRecipe, error)
	}{
		{"key", r.Document.Key, q.GetCatalogRecipeByKey}, {"source identity", r.SourceIdentityDigest, q.GetCatalogRecipeBySourceIdentityDigest}, {"selected recipe", r.SelectedRecipeDigest, q.GetCatalogRecipeBySelectedRecipeDigest},
	}
	for _, c := range checks {
		if c.value == "" {
			return fmt.Errorf("%s digest/key is required", c.name)
		}
		ex, err := c.get(ctx, c.value)
		if err == nil {
			if ex.DocumentDigest == r.DocumentDigest {
				continue
			}
			return fmt.Errorf("%w: catalog %s %s already exists", ErrConflict, c.name, c.value)
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}
	return nil
}

// AdoptionResult returns the tenant membership and materialized household recipe.
type AdoptionResult struct {
	Membership         store.HouseholdCatalogMembership
	MaterializedRecipe store.Recipe
	Created            bool
}

// AddToHousehold atomically trials or adopts one verified catalog recipe.
func (s *Service) AddToHousehold(ctx context.Context, householdID string, catalogRecipeID int64, state string) (AdoptionResult, error) {
	if state != "trial" && state != "adopted" {
		return AdoptionResult{}, fmt.Errorf("invalid membership state %q", state)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return AdoptionResult{}, err
	}
	defer tx.Rollback()
	q := store.New(tx)
	cat, err := q.GetCatalogRecipe(ctx, catalogRecipeID)
	if err != nil {
		return AdoptionResult{}, err
	}
	if cat.Status != "verified" {
		return AdoptionResult{}, ErrNotVerified
	}
	if existing, err := q.GetHouseholdCatalogMembership(ctx, store.GetHouseholdCatalogMembershipParams{HouseholdID: householdID, CatalogRecipeID: catalogRecipeID}); err == nil {
		if state == "adopted" && existing.State == "trial" {
			existing, err = q.PromoteHouseholdCatalogMembership(ctx, store.PromoteHouseholdCatalogMembershipParams{HouseholdID: householdID, CatalogRecipeID: catalogRecipeID})
			if err != nil {
				return AdoptionResult{}, err
			}
		}
		recipe, err := q.GetRecipe(ctx, store.GetRecipeParams{HouseholdID: householdID, ID: existing.MaterializedRecipeID})
		if err != nil {
			return AdoptionResult{}, err
		}
		if err := tx.Commit(); err != nil {
			return AdoptionResult{}, err
		}
		return AdoptionResult{Membership: existing, MaterializedRecipe: recipe}, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return AdoptionResult{}, err
	}
	var doc ingest.Document
	if err := json.Unmarshal(cat.RecipeDocument, &doc); err != nil {
		return AdoptionResult{}, err
	}
	doc.Status = "verified"
	recipe, err := materializeDocument(ctx, q, householdID, doc)
	if err != nil {
		return AdoptionResult{}, err
	}
	m, err := q.CreateHouseholdCatalogMembership(ctx, store.CreateHouseholdCatalogMembershipParams{HouseholdID: householdID, CatalogRecipeID: catalogRecipeID, MaterializedRecipeID: recipe.ID, State: state, CatalogDocumentDigest: cat.DocumentDigest})
	if err != nil {
		return AdoptionResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return AdoptionResult{}, err
	}
	return AdoptionResult{Membership: m, MaterializedRecipe: recipe, Created: true}, nil
}

func materializeDocument(ctx context.Context, q *store.Queries, householdID string, d ingest.Document) (store.Recipe, error) {
	if err := d.Validate(); err != nil {
		return store.Recipe{}, err
	}
	recipe, err := q.CreateDraftRecipe(ctx, store.CreateDraftRecipeParams{HouseholdID: householdID, Key: d.Key, Name: d.Name, ImageUrl: nullString(d.ImageURL), YieldText: nullString(d.Yield), HandsOnMinMinutes: durationMin(d.HandsOn), HandsOnMaxMinutes: durationMax(d.HandsOn), UnattendedMinMinutes: durationMin(d.Unattended), UnattendedMaxMinutes: durationMax(d.Unattended)})
	if err != nil {
		return store.Recipe{}, err
	}
	if _, err := q.CreateRecipeSource(ctx, store.CreateRecipeSourceParams{RecipeID: recipe.ID, Relationship: d.Source.Relationship, Attribution: d.Source.Attribution, Url: nullString(d.Source.URL), CheckedOn: nullString(d.Source.CheckedOn), IsPrimary: 1, Position: 0}); err != nil {
		return store.Recipe{}, err
	}
	sections := map[string]store.StoreSection{}
	items := map[string]store.GroceryItem{}
	units := map[string]store.Unit{}
	for si, sec := range d.IngredientSections {
		s, err := q.CreateIngredientSection(ctx, store.CreateIngredientSectionParams{RecipeID: recipe.ID, Name: sec.Name, Position: int64(si)})
		if err != nil {
			return store.Recipe{}, err
		}
		for ii, ing := range sec.Ingredients {
			item, err := ensureItem(ctx, q, householdID, ing.GroceryItem, sections, items)
			if err != nil {
				return store.Recipe{}, err
			}
			params, err := ingredientParams(ctx, q, ing, s.ID, item.ID, int64(ii), units)
			if err != nil {
				return store.Recipe{}, err
			}
			if _, err := q.CreateRecipeIngredient(ctx, params); err != nil {
				return store.Recipe{}, err
			}
		}
	}
	for si, sec := range d.InstructionSections {
		s, err := q.CreateInstructionSection(ctx, store.CreateInstructionSectionParams{RecipeID: recipe.ID, Name: sec.Name, Position: int64(si)})
		if err != nil {
			return store.Recipe{}, err
		}
		for pi, step := range sec.Steps {
			if _, err := q.CreateRecipeStep(ctx, store.CreateRecipeStepParams{SectionID: s.ID, Position: int64(pi), Instruction: step}); err != nil {
				return store.Recipe{}, err
			}
		}
	}
	for _, decision := range d.Review {
		f, err := q.CreateReviewFlag(ctx, store.CreateReviewFlagParams{RecipeID: recipe.ID, FieldPath: decision.Field, Kind: decision.Kind, Note: decision.Note})
		if err != nil {
			return store.Recipe{}, err
		}
		if decision.Approved {
			if _, err := q.ApproveReviewFlag(ctx, f.ID); err != nil {
				return store.Recipe{}, err
			}
		}
	}
	if _, err := q.MarkRecipeReviewable(ctx, store.MarkRecipeReviewableParams{HouseholdID: householdID, ID: recipe.ID}); err != nil {
		return store.Recipe{}, err
	}
	return q.VerifyRecipe(ctx, store.VerifyRecipeParams{HouseholdID: householdID, ID: recipe.ID, VerifiedAt: sql.NullString{String: d.ApprovedOn + "T00:00:00Z", Valid: true}})
}

func ensureItem(ctx context.Context, q *store.Queries, householdID string, wanted ingest.GroceryItem, sections map[string]store.StoreSection, items map[string]store.GroceryItem) (store.GroceryItem, error) {
	section, ok := sections[wanted.StoreSection.Key]
	if !ok {
		var err error
		section, err = q.GetStoreSectionByKey(ctx, store.GetStoreSectionByKeyParams{HouseholdID: householdID, Key: wanted.StoreSection.Key})
		if errors.Is(err, sql.ErrNoRows) {
			section, err = q.CreateStoreSection(ctx, store.CreateStoreSectionParams{HouseholdID: householdID, Key: wanted.StoreSection.Key, Name: wanted.StoreSection.Name})
		}
		if err != nil {
			return store.GroceryItem{}, err
		}
		if section.Name != wanted.StoreSection.Name {
			return store.GroceryItem{}, fmt.Errorf("store section %q conflicts", wanted.StoreSection.Key)
		}
		sections[wanted.StoreSection.Key] = section
	}
	item, ok := items[wanted.Key]
	if !ok {
		var err error
		item, err = q.GetGroceryItemByKey(ctx, store.GetGroceryItemByKeyParams{HouseholdID: householdID, Key: wanted.Key})
		if errors.Is(err, sql.ErrNoRows) {
			item, err = q.CreateGroceryItem(ctx, store.CreateGroceryItemParams{HouseholdID: householdID, Key: wanted.Key, Name: wanted.Name, StoreSectionID: section.ID, ShoppingMode: wanted.ShoppingMode})
		}
		if err != nil {
			return store.GroceryItem{}, err
		}
		items[wanted.Key] = item
	}
	if item.Name != wanted.Name || item.StoreSectionID != section.ID || item.ShoppingMode != wanted.ShoppingMode {
		return store.GroceryItem{}, fmt.Errorf("grocery item %q conflicts", wanted.Key)
	}
	return item, nil
}

func ingredientParams(ctx context.Context, q *store.Queries, ing ingest.Ingredient, sectionID, itemID, position int64, units map[string]store.Unit) (store.CreateRecipeIngredientParams, error) {
	p := store.CreateRecipeIngredientParams{SectionID: sectionID, GroceryItemID: sql.NullInt64{Int64: itemID, Valid: true}, Position: position, SourceText: ing.SourceText, QuantityKind: ing.Quantity.Kind, Preparation: nullString(ing.Preparation), IsOptional: boolInt(ing.Optional), IncludeOnGroceryList: boolInt(!ing.NonShopping), DisplayNote: nullString(ing.Note)}
	if ing.Quantity.Kind != "unspecified" {
		minimum, err := parseRat(ing.Quantity.Amount)
		if err != nil {
			return p, err
		}
		p.AmountMinNumerator, p.AmountMinDenominator = ratInts(minimum)
		if ing.Quantity.Kind == "range" {
			maximum, err := parseRat(ing.Quantity.Maximum)
			if err != nil {
				return p, err
			}
			p.AmountMaxNumerator, p.AmountMaxDenominator = ratInts(maximum)
		}
		if ing.Quantity.Unit != "" {
			u, err := ensureUnit(ctx, q, ing.Quantity.Unit, units)
			if err != nil {
				return p, err
			}
			p.UnitID = sql.NullInt64{Int64: u.ID, Valid: true}
		} else {
			p.PackageType = nullString(ing.Quantity.Package.Type)
			if ing.Quantity.Package.Size != "" {
				r, err := parseRat(ing.Quantity.Package.Size)
				if err != nil {
					return p, err
				}
				p.PackageSizeNumerator, p.PackageSizeDenominator = ratInts(r)
				u, err := ensureUnit(ctx, q, ing.Quantity.Package.Unit, units)
				if err != nil {
					return p, err
				}
				p.PackageSizeUnitID = sql.NullInt64{Int64: u.ID, Valid: true}
			}
		}
	}
	return p, nil
}
func ensureUnit(ctx context.Context, q *store.Queries, key string, units map[string]store.Unit) (store.Unit, error) {
	if u, ok := units[key]; ok {
		return u, nil
	}
	u, err := q.GetUnitByKey(ctx, key)
	if err != nil {
		return store.Unit{}, err
	}
	units[key] = u
	return u, nil
}
func parseRat(v string) (*big.Rat, error) {
	parts := strings.Fields(v)
	if len(parts) == 1 {
		r, ok := new(big.Rat).SetString(parts[0])
		if ok {
			return r, nil
		}
	} else if len(parts) == 2 {
		whole, wholeOK := new(big.Rat).SetString(parts[0])
		fraction, fractionOK := new(big.Rat).SetString(parts[1])
		if wholeOK && fractionOK {
			return new(big.Rat).Add(whole, fraction), nil
		}
	}
	return nil, fmt.Errorf("invalid rational %q", v)
}
func ratInts(r *big.Rat) (sql.NullInt64, sql.NullInt64) {
	return sql.NullInt64{Int64: r.Num().Int64(), Valid: true}, sql.NullInt64{Int64: r.Denom().Int64(), Valid: true}
}
func nullString(v string) sql.NullString { return sql.NullString{String: v, Valid: v != ""} }
func durationMin(d *ingest.Duration) sql.NullInt64 {
	if d == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: d.Min, Valid: true}
}
func durationMax(d *ingest.Duration) sql.NullInt64 {
	if d == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: d.Max, Valid: true}
}
func boolInt(v bool) int64 {
	if v {
		return 1
	}
	return 0
}
func digest(b []byte) string { s := sha256.Sum256(b); return "sha256:" + hex.EncodeToString(s[:]) }
