-- name: GetCatalogAppliedRelease :one
SELECT * FROM catalog_applied_releases WHERE release_id = $1;

-- name: CreateCatalogAppliedRelease :one
INSERT INTO catalog_applied_releases (release_id, release_digest, manifest)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetCatalogRecipeByKey :one
SELECT * FROM catalog_recipes WHERE key = $1;

-- name: GetCatalogRecipeBySourceIdentityDigest :one
SELECT * FROM catalog_recipes WHERE source_identity_digest = $1;

-- name: GetCatalogRecipeBySelectedRecipeDigest :one
SELECT * FROM catalog_recipes WHERE selected_recipe_digest = $1;

-- name: GetCatalogRecipe :one
SELECT * FROM catalog_recipes WHERE id = $1;

-- name: ListVerifiedCatalogRecipes :many
SELECT * FROM catalog_recipes WHERE status = 'verified' ORDER BY name;

-- name: ListFamilyRecipes :many
SELECT r.*, hcm.state AS catalog_state
FROM recipes r
LEFT JOIN household_catalog_memberships hcm
    ON hcm.household_id = r.household_id AND hcm.materialized_recipe_id = r.id
WHERE r.household_id = sqlc.arg(household_id) AND r.status = 'verified'
ORDER BY r.name;

-- name: ListCatalogRecipes :many
SELECT cr.*, hcm.state AS membership_state, hcm.materialized_recipe_id
FROM catalog_recipes cr
LEFT JOIN household_catalog_memberships hcm
    ON hcm.catalog_recipe_id = cr.id AND hcm.household_id = sqlc.arg(household_id)
ORDER BY cr.name;

-- name: CreateCatalogRecipe :one
INSERT INTO catalog_recipes (
    key, name, status, recipe_document, source_identity, semantic_profile,
    source_url, source_attribution, document_digest, source_identity_digest,
    selected_recipe_digest, release_id, release_digest
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
RETURNING *;

-- name: GetHouseholdCatalogMembership :one
SELECT * FROM household_catalog_memberships
WHERE household_id = sqlc.arg(household_id) AND catalog_recipe_id = sqlc.arg(catalog_recipe_id);

-- name: CreateHouseholdCatalogMembership :one
INSERT INTO household_catalog_memberships (
    household_id, catalog_recipe_id, materialized_recipe_id, state, catalog_document_digest
) VALUES (sqlc.arg(household_id), $1, $2, $3, $4)
RETURNING *;

-- name: ListHouseholdCatalogMemberships :many
SELECT * FROM household_catalog_memberships
WHERE household_id = $1
ORDER BY created_at;

-- name: PromoteHouseholdCatalogMembership :one
UPDATE household_catalog_memberships
SET state = 'adopted', updated_at = clock_timestamp()
WHERE household_id = sqlc.arg(household_id)
  AND catalog_recipe_id = sqlc.arg(catalog_recipe_id)
  AND state = 'trial'
RETURNING *;
