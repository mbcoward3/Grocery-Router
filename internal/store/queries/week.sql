-- name: CreateWeek :one
INSERT INTO weeks (starts_on) VALUES ($1)
RETURNING *;

-- name: GetWeekByStart :one
SELECT * FROM weeks WHERE starts_on = $1;

-- name: TouchWeek :exec
UPDATE weeks SET updated_at = to_char(clock_timestamp() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') WHERE id = $1;

-- name: CreateWeekRecipe :one
INSERT INTO week_recipes (week_id, recipe_id, position) VALUES ($1, $2, $3)
RETURNING *;

-- name: GetWeekRecipe :one
SELECT * FROM week_recipes WHERE id = $1;

-- name: ListWeekRecipes :many
SELECT
    wr.*,
    r.key AS recipe_key,
    r.name AS recipe_name,
    r.image_url,
    r.yield_text,
    r.hands_on_min_minutes,
    r.hands_on_max_minutes,
    r.unattended_min_minutes,
    r.unattended_max_minutes
FROM week_recipes wr
JOIN recipes r ON r.id = wr.recipe_id
WHERE wr.week_id = $1
ORDER BY wr.position, wr.id;

-- name: ListWeekRecipeIDs :many
SELECT recipe_id FROM week_recipes WHERE week_id = $1;

-- name: NextWeekRecipePosition :one
SELECT CAST(coalesce(max(position) + 1, 0) AS BIGINT) FROM week_recipes WHERE week_id = $1;

-- name: UpdateWeekRecipeRecipe :one
UPDATE week_recipes SET recipe_id = $1 WHERE id = $2
RETURNING *;

-- name: DeleteWeekRecipe :execrows
DELETE FROM week_recipes WHERE id = $1;

-- name: DeleteWeekRecipes :exec
DELETE FROM week_recipes WHERE week_id = $1;

-- name: CreateShoppingList :one
INSERT INTO shopping_lists (week_id) VALUES ($1)
RETURNING *;

-- name: GetShoppingListByWeek :one
SELECT * FROM shopping_lists WHERE week_id = $1;

-- name: ListWeekIngredientRequirements :many
SELECT
    wr.id AS week_recipe_id,
    wr.position AS week_recipe_position,
    r.id AS recipe_id,
    r.name AS recipe_name,
    ri.id AS recipe_ingredient_id,
    ri.position AS ingredient_position,
    ri.quantity_kind,
    ri.amount_min_numerator,
    ri.amount_min_denominator,
    ri.amount_max_numerator,
    ri.amount_max_denominator,
    ri.unit_id,
    ri.package_type,
    ri.package_size_numerator,
    ri.package_size_denominator,
    ri.package_size_unit_id,
    ri.preparation,
    ri.is_optional,
    gi.id AS grocery_item_id,
    gi.key AS grocery_item_key,
    gi.name AS grocery_item_name,
    gi.shopping_mode,
    ss.id AS store_section_id,
    ss.name AS store_section_name,
    u.key AS unit_key,
    u.dimension AS unit_dimension,
    u.to_base_numerator AS unit_to_base_numerator,
    u.to_base_denominator AS unit_to_base_denominator,
    psu.key AS package_size_unit_key
FROM week_recipes wr
JOIN recipes r ON r.id = wr.recipe_id
JOIN recipe_ingredient_sections ris ON ris.recipe_id = r.id
JOIN recipe_ingredients ri ON ri.section_id = ris.id
JOIN grocery_items gi ON gi.id = ri.grocery_item_id
JOIN store_sections ss ON ss.id = gi.store_section_id
LEFT JOIN units u ON u.id = ri.unit_id
LEFT JOIN units psu ON psu.id = ri.package_size_unit_id
WHERE wr.week_id = $1 AND ri.include_on_grocery_list = 1
ORDER BY wr.position, wr.id, ris.position, ri.position;
