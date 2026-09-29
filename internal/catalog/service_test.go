package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/mbcoward3/grocery-router/internal/ingest"
	"github.com/mbcoward3/grocery-router/internal/store"
	"github.com/mbcoward3/grocery-router/internal/testdatabase"
)

const cowardHouseholdID = "c0a7a2d8-669b-4e47-91c1-4d9a32f339d5"

func TestPublishReleaseIdempotentAndConflict(t *testing.T) {
	ctx := context.Background()
	db := testdatabase.Open(t)
	svc := NewService(db)
	rel := testRelease("rel-one", "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "banana-bread", "verified")
	got, err := svc.PublishRelease(ctx, rel)
	if err != nil || !got.Applied {
		t.Fatalf("publish = %#v, %v", got, err)
	}
	got, err = svc.PublishRelease(ctx, rel)
	if err != nil || got.Applied {
		t.Fatalf("idempotent republish = %#v, %v", got, err)
	}
	conflict := rel
	conflict.Digest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if _, err := svc.PublishRelease(ctx, conflict); !errors.Is(err, ErrConflict) {
		t.Fatalf("conflict error = %v", err)
	}
}

func TestPublishReleaseRollsBackOnRecipeConflict(t *testing.T) {
	ctx := context.Background()
	db := testdatabase.Open(t)
	svc := NewService(db)
	first := testRelease("rel-one", "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "banana-bread", "verified")
	if _, err := svc.PublishRelease(ctx, first); err != nil {
		t.Fatal(err)
	}
	second := testRelease("rel-two", "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "banana-bread", "verified")
	second.Recipes[0].Document.Name = "Different Banana Bread"
	second.Recipes[0].DocumentDigest = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	if _, err := svc.PublishRelease(ctx, second); !errors.Is(err, ErrConflict) {
		t.Fatalf("publish conflict = %v", err)
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT count(*) FROM catalog_applied_releases WHERE release_id = 'rel-two'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("conflicting release was not rolled back")
	}
}

func TestAdoptionRejectsReviewableAndMaterializesVerifiedTransactionally(t *testing.T) {
	ctx := context.Background()
	db := testdatabase.Open(t)
	svc := NewService(db)
	reviewable := testRelease("rel-review", "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", "review-bread", "reviewable")
	if _, err := svc.PublishRelease(ctx, reviewable); err != nil {
		t.Fatal(err)
	}
	catReview, err := store.New(db).GetCatalogRecipeByKey(ctx, "review-bread")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddToHousehold(ctx, cowardHouseholdID, catReview.ID, "trial"); !errors.Is(err, ErrNotVerified) {
		t.Fatalf("reviewable adoption error = %v", err)
	}

	verified := testRelease("rel-verified", "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", "verified-bread", "verified")
	if _, err := svc.PublishRelease(ctx, verified); err != nil {
		t.Fatal(err)
	}
	catVerified, err := store.New(db).GetCatalogRecipeByKey(ctx, "verified-bread")
	if err != nil {
		t.Fatal(err)
	}
	adopted, err := svc.AddToHousehold(ctx, cowardHouseholdID, catVerified.ID, "trial")
	if err != nil {
		t.Fatalf("adopt verified: %v", err)
	}
	if !adopted.Created || adopted.Membership.State != "trial" || adopted.MaterializedRecipe.Status != "verified" {
		t.Fatalf("bad adoption: %#v", adopted)
	}
	if adopted.Membership.CatalogDocumentDigest != catVerified.DocumentDigest {
		t.Fatalf("membership digest mismatch")
	}
	promoted, err := svc.AddToHousehold(ctx, cowardHouseholdID, catVerified.ID, "adopted")
	if err != nil {
		t.Fatalf("promote: %v", err)
	}
	if promoted.Created || promoted.Membership.State != "adopted" || promoted.MaterializedRecipe.ID != adopted.MaterializedRecipe.ID {
		t.Fatalf("bad promotion: %#v", promoted)
	}
}

func TestMembershipConstraintsRejectCrossHouseholdAndDigestMismatch(t *testing.T) {
	ctx := context.Background()
	db := testdatabase.Open(t)
	svc := NewService(db)
	otherHousehold := "11111111-1111-4111-8111-111111111111"
	if _, err := db.ExecContext(ctx, `INSERT INTO households (id, name) VALUES ($1, 'Other')`, otherHousehold); err != nil {
		t.Fatal(err)
	}
	rel := testRelease("rel-cross", "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "cross-bread", "verified")
	if _, err := svc.PublishRelease(ctx, rel); err != nil {
		t.Fatal(err)
	}
	cat, err := store.New(db).GetCatalogRecipeByKey(ctx, "cross-bread")
	if err != nil {
		t.Fatal(err)
	}
	adopted, err := svc.AddToHousehold(ctx, cowardHouseholdID, cat.ID, "trial")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `INSERT INTO household_catalog_memberships (household_id, catalog_recipe_id, materialized_recipe_id, state, catalog_document_digest) VALUES ($1, $2, $3, 'trial', $4)`, otherHousehold, cat.ID, adopted.MaterializedRecipe.ID, cat.DocumentDigest)
	if err == nil {
		t.Fatalf("cross-household materialized recipe link succeeded")
	}
	_, err = db.ExecContext(ctx, `INSERT INTO household_catalog_memberships (household_id, catalog_recipe_id, materialized_recipe_id, state, catalog_document_digest) VALUES ($1, $2, $3, 'trial', $4)`, otherHousehold, cat.ID, adopted.MaterializedRecipe.ID, "sha256:0000000000000000000000000000000000000000000000000000000000000000")
	if err == nil {
		t.Fatalf("digest-mismatched membership succeeded")
	}
}

func testRelease(id, digestValue, key, status string) Release {
	doc := testDocument(key)
	docJSON, _ := json.Marshal(doc)
	return Release{ID: id, Digest: digestValue, Manifest: json.RawMessage(`{"format_version":1}`), Recipes: []ReleaseRecipe{{Document: doc, Status: status, SourceIdentity: json.RawMessage(`{"source":"test"}`), SemanticProfile: json.RawMessage(`{"accepted":[]}`), DocumentDigest: digest(docJSON), SourceIdentityDigest: digest([]byte(id + key + "source")), SelectedRecipeDigest: digest([]byte(id + key + "selected"))}}}
}

func testDocument(key string) ingest.Document {
	return ingest.Document{FormatVersion: 1, Key: key, Name: "Test " + key, Status: "verified", ApprovedOn: "2026-01-02", Source: ingest.Source{Relationship: "source", Attribution: "Test Kitchen", URL: "https://example.com/" + key, CheckedOn: "2026-01-01"}, IngredientSections: []ingest.IngredientSection{{Name: "Main", Ingredients: []ingest.Ingredient{{SourceText: "1 cup flour", GroceryItem: ingest.GroceryItem{Key: "flour", Name: "Flour", StoreSection: ingest.StoreSection{Key: "baking", Name: "Baking"}, ShoppingMode: "measured"}, Quantity: ingest.Quantity{Kind: "exact", Amount: "1", Unit: "cup"}}}}}, InstructionSections: []ingest.InstructionSection{{Name: "Steps", Steps: []string{"Mix."}}}}
}
