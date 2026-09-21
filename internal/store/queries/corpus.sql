-- name: CreateDraftRecipe :one
INSERT INTO recipes (
    key, name, image_url, yield_text,
    hands_on_min_minutes, hands_on_max_minutes,
    unattended_min_minutes, unattended_max_minutes
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetRecipe :one
SELECT * FROM recipes WHERE id = $1;

-- name: GetVerifiedRecipe :one
SELECT * FROM recipes WHERE id = $1 AND status = 'verified';

-- name: GetRecipeByKey :one
SELECT * FROM recipes WHERE key = $1;

-- name: ListVerifiedRecipes :many
SELECT * FROM recipes WHERE status = 'verified' ORDER BY name;

-- name: MarkRecipeReviewable :one
UPDATE recipes
SET status = 'reviewable', verified_at = NULL,
    updated_at = to_char(clock_timestamp() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"')
WHERE id = $1 AND status = 'draft'
RETURNING *;

-- name: ReturnRecipeToDraft :one
UPDATE recipes
SET status = 'draft', verified_at = NULL,
    updated_at = to_char(clock_timestamp() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"')
WHERE id = $1 AND status IN ('reviewable', 'verified')
RETURNING *;

-- name: VerifyRecipe :one
UPDATE recipes
SET status = 'verified', verified_at = $1,
    updated_at = to_char(clock_timestamp() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"')
WHERE id = $2 AND status = 'reviewable'
RETURNING *;

-- name: CreateRecipeSource :one
INSERT INTO recipe_sources (
    recipe_id, relationship, attribution, url, checked_on, is_primary, position
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListRecipeSources :many
SELECT * FROM recipe_sources WHERE recipe_id = $1 ORDER BY position;

-- name: CreateStoreSection :one
INSERT INTO store_sections (key, name) VALUES ($1, $2)
RETURNING *;

-- name: GetStoreSectionByKey :one
SELECT * FROM store_sections WHERE key = $1;

-- name: ListStoreSections :many
SELECT * FROM store_sections ORDER BY name;

-- name: CreateUnit :one
INSERT INTO units (
    key, name, symbol, dimension, to_base_numerator, to_base_denominator
) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetUnitByKey :one
SELECT * FROM units WHERE key = $1;

-- name: ListUnits :many
SELECT * FROM units ORDER BY dimension, name;

-- name: CreateGroceryItem :one
INSERT INTO grocery_items (key, name, store_section_id, shopping_mode)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetGroceryItemByKey :one
SELECT * FROM grocery_items WHERE key = $1;

-- name: ListGroceryItems :many
SELECT gi.*, ss.name AS store_section_name
FROM grocery_items gi
JOIN store_sections ss ON ss.id = gi.store_section_id
ORDER BY gi.name;

-- name: CreateIngredientSection :one
INSERT INTO recipe_ingredient_sections (recipe_id, name, position)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListIngredientSections :many
SELECT * FROM recipe_ingredient_sections WHERE recipe_id = $1 ORDER BY position;

-- name: CreateRecipeIngredient :one
INSERT INTO recipe_ingredients (
    section_id, grocery_item_id, position, source_text, quantity_kind,
    amount_min_numerator, amount_min_denominator,
    amount_max_numerator, amount_max_denominator,
    unit_id, package_type,
    package_size_numerator, package_size_denominator, package_size_unit_id,
    preparation, is_optional, include_on_grocery_list, display_note
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
RETURNING *;

-- name: ListRecipeIngredients :many
SELECT
    ri.*,
    ris.recipe_id,
    ris.name AS section_name,
    ris.position AS section_position,
    gi.key AS grocery_item_key,
    gi.name AS grocery_item_name,
    gi.shopping_mode,
    ss.name AS store_section_name,
    u.key AS unit_key,
    u.symbol AS unit_symbol,
    psu.key AS package_size_unit_key,
    psu.symbol AS package_size_unit_symbol
FROM recipe_ingredients ri
JOIN recipe_ingredient_sections ris ON ris.id = ri.section_id
LEFT JOIN grocery_items gi ON gi.id = ri.grocery_item_id
LEFT JOIN store_sections ss ON ss.id = gi.store_section_id
LEFT JOIN units u ON u.id = ri.unit_id
LEFT JOIN units psu ON psu.id = ri.package_size_unit_id
WHERE ris.recipe_id = $1
ORDER BY ris.position, ri.position;

-- name: CreateInstructionSection :one
INSERT INTO recipe_instruction_sections (recipe_id, name, position)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListInstructionSections :many
SELECT * FROM recipe_instruction_sections WHERE recipe_id = $1 ORDER BY position;

-- name: CreateRecipeStep :one
INSERT INTO recipe_steps (section_id, position, instruction)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListRecipeSteps :many
SELECT rs.*, ris.recipe_id, ris.name AS section_name, ris.position AS section_position
FROM recipe_steps rs
JOIN recipe_instruction_sections ris ON ris.id = rs.section_id
WHERE ris.recipe_id = $1
ORDER BY ris.position, rs.position;

-- name: CreateReviewFlag :one
INSERT INTO recipe_review_flags (recipe_id, field_path, kind, note)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ApproveReviewFlag :one
UPDATE recipe_review_flags SET approved = 1
WHERE id = $1
RETURNING *;

-- name: ListRecipeReviewFlags :many
SELECT * FROM recipe_review_flags WHERE recipe_id = $1 ORDER BY field_path, kind;

-- name: CountUnapprovedReviewFlags :one
SELECT count(*) FROM recipe_review_flags WHERE recipe_id = $1 AND approved = 0;
