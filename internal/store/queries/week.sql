-- name: CreateWeek :one
INSERT INTO weeks (household_id, starts_on, opened_at)
VALUES (sqlc.arg(household_id), sqlc.arg(starts_on), sqlc.arg(opened_at))
RETURNING *;

-- name: CloseOpenWeek :execrows
-- Closes the household's open plan at now with its last local date, never before its start
-- date. Dates are ISO text, so GREATEST compares them as dates.
UPDATE weeks SET
    closed_at = CAST(sqlc.arg(closed_at) AS TIMESTAMPTZ),
    ends_on = greatest(CAST(sqlc.arg(ends_on) AS TEXT), starts_on)
WHERE household_id = sqlc.arg(household_id) AND closed_at IS NULL;

-- name: ReopenWeek :execrows
-- Makes a past plan the open one again. The caller closes the open plan first.
UPDATE weeks SET closed_at = NULL, ends_on = NULL
WHERE household_id = sqlc.arg(household_id) AND id = sqlc.arg(id) AND closed_at IS NOT NULL;

-- name: GetOpenWeek :one
SELECT * FROM weeks WHERE household_id = sqlc.arg(household_id) AND closed_at IS NULL;

-- name: LockOpenWeek :one
-- Taken before a change to the open week or its list. A concurrent close waits for the change,
-- or, if it closed the week first, this finds no open week once it commits.
SELECT * FROM weeks WHERE household_id = sqlc.arg(household_id) AND closed_at IS NULL
FOR NO KEY UPDATE;

-- name: ListPastWeeks :many
SELECT
    w.*,
    CAST((SELECT count(*) FROM week_recipes wr WHERE wr.week_id = w.id) AS BIGINT) AS recipe_count,
    CAST((SELECT count(*) FROM shopping_lines sl
        JOIN shopping_lists list ON list.id = sl.shopping_list_id
        WHERE list.week_id = w.id AND sl.is_removed = 0) AS BIGINT) AS grocery_count,
    CAST((SELECT count(*) FROM shopping_lines sl
        JOIN shopping_lists list ON list.id = sl.shopping_list_id
        WHERE list.week_id = w.id AND sl.is_removed = 0 AND sl.is_completed = 1) AS BIGINT) AS completed_count
FROM weeks w
WHERE w.household_id = sqlc.arg(household_id) AND w.closed_at IS NOT NULL
ORDER BY w.starts_on DESC, w.id DESC;

-- name: ListRecentPlanRecipeIDs :many
-- The recipes in the household's most recently closed plans, which selection avoids (D052).
SELECT DISTINCT wr.recipe_id FROM week_recipes AS wr
WHERE wr.household_id = sqlc.arg(household_id) AND wr.week_id IN (
    SELECT w.id FROM weeks AS w
    WHERE w.household_id = sqlc.arg(household_id) AND w.closed_at IS NOT NULL
    ORDER BY w.closed_at DESC, w.id DESC
    LIMIT sqlc.arg(plans)
);

-- name: GetPastWeek :one
SELECT * FROM weeks
WHERE household_id = sqlc.arg(household_id) AND id = sqlc.arg(id) AND closed_at IS NOT NULL;

-- name: TouchWeek :exec
UPDATE weeks SET updated_at = to_char(clock_timestamp() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"')
WHERE household_id = sqlc.arg(household_id) AND id = sqlc.arg(id) AND closed_at IS NULL;

-- name: CreateWeekRecipe :one
-- Snapshots the recipe fields History shows, so later recipe edits never change a past plan.
INSERT INTO week_recipes (
    household_id, week_id, recipe_id, position,
    recipe_name, recipe_key, image_url, yield_text,
    hands_on_min_minutes, hands_on_max_minutes, unattended_min_minutes, unattended_max_minutes
)
SELECT
    r.household_id, sqlc.arg(week_id), r.id, sqlc.arg(position),
    r.name, r.key, r.image_url, r.yield_text,
    r.hands_on_min_minutes, r.hands_on_max_minutes, r.unattended_min_minutes, r.unattended_max_minutes
FROM recipes AS r
WHERE r.household_id = sqlc.arg(household_id) AND r.id = sqlc.arg(recipe_id)
RETURNING *;

-- name: GetWeekRecipe :one
SELECT * FROM week_recipes WHERE household_id = sqlc.arg(household_id) AND id = sqlc.arg(id);

-- name: ListWeekRecipes :many
-- The open plan shows live recipe values; History reads ListPastWeekRecipes instead.
SELECT
    wr.id,
    wr.week_id,
    wr.recipe_id,
    wr.position,
    wr.created_at,
    wr.household_id,
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
WHERE wr.household_id = sqlc.arg(household_id) AND wr.week_id = sqlc.arg(week_id)
ORDER BY wr.position, wr.id;

-- name: ListPastWeekRecipes :many
-- A plan in History shows the recipe fields snapshotted on its occurrences, never live rows.
SELECT
    wr.id,
    wr.week_id,
    wr.recipe_id,
    wr.position,
    wr.created_at,
    wr.household_id,
    wr.recipe_key,
    wr.recipe_name,
    wr.image_url,
    wr.yield_text,
    wr.hands_on_min_minutes,
    wr.hands_on_max_minutes,
    wr.unattended_min_minutes,
    wr.unattended_max_minutes
FROM week_recipes AS wr
WHERE wr.household_id = sqlc.arg(household_id) AND wr.week_id = sqlc.arg(week_id)
ORDER BY wr.position, wr.id;

-- name: ListWeekRecipeIDs :many
SELECT recipe_id FROM week_recipes
WHERE household_id = sqlc.arg(household_id) AND week_id = sqlc.arg(week_id);

-- name: NextWeekRecipePosition :one
SELECT CAST(coalesce(max(position) + 1, 0) AS BIGINT) FROM week_recipes
WHERE household_id = sqlc.arg(household_id) AND week_id = sqlc.arg(week_id);

-- name: UpdateWeekRecipeRecipe :one
-- A swap replaces the occurrence's snapshot with the new recipe's fields.
UPDATE week_recipes SET
    recipe_id = r.id,
    recipe_name = r.name,
    recipe_key = r.key,
    image_url = r.image_url,
    yield_text = r.yield_text,
    hands_on_min_minutes = r.hands_on_min_minutes,
    hands_on_max_minutes = r.hands_on_max_minutes,
    unattended_min_minutes = r.unattended_min_minutes,
    unattended_max_minutes = r.unattended_max_minutes
FROM recipes AS r
WHERE week_recipes.household_id = sqlc.arg(household_id) AND week_recipes.id = sqlc.arg(id)
    AND r.household_id = sqlc.arg(household_id) AND r.id = sqlc.arg(recipe_id)
RETURNING week_recipes.*;

-- name: DeleteWeekRecipe :execrows
DELETE FROM week_recipes WHERE household_id = sqlc.arg(household_id) AND id = sqlc.arg(id);

-- name: DeleteWeekRecipes :exec
DELETE FROM week_recipes
WHERE household_id = sqlc.arg(household_id) AND week_id = sqlc.arg(week_id);

-- name: CreateShoppingList :one
INSERT INTO shopping_lists (household_id, week_id) VALUES (sqlc.arg(household_id), sqlc.arg(week_id))
RETURNING *;

-- name: GetShoppingListByWeek :one
SELECT * FROM shopping_lists
WHERE household_id = sqlc.arg(household_id) AND week_id = sqlc.arg(week_id);

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
    ri.source_text,
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
LEFT JOIN household_item_sections his ON his.household_id = gi.household_id AND his.grocery_item_id = gi.id
JOIN store_sections ss ON ss.household_id = gi.household_id AND ss.id = coalesce(his.store_section_id, gi.store_section_id)
LEFT JOIN units u ON u.id = ri.unit_id
LEFT JOIN units psu ON psu.id = ri.package_size_unit_id
WHERE wr.household_id = sqlc.arg(household_id) AND wr.week_id = sqlc.arg(week_id)
    AND ri.include_on_grocery_list = 1
ORDER BY wr.position, wr.id, ris.position, ri.position;

-- name: RefreshWeekRecipeSnapshots :execrows
-- After a recipe edit, the open plan's occurrences of it show the edited fields. Callers pass
-- only the open plan's week; past plans keep their snapshots.
UPDATE week_recipes SET
    recipe_name = r.name,
    recipe_key = r.key,
    image_url = r.image_url,
    yield_text = r.yield_text,
    hands_on_min_minutes = r.hands_on_min_minutes,
    hands_on_max_minutes = r.hands_on_max_minutes,
    unattended_min_minutes = r.unattended_min_minutes,
    unattended_max_minutes = r.unattended_max_minutes
FROM recipes AS r
WHERE week_recipes.household_id = sqlc.arg(household_id) AND week_recipes.week_id = sqlc.arg(week_id)
    AND week_recipes.recipe_id = sqlc.arg(recipe_id)
    AND r.household_id = sqlc.arg(household_id) AND r.id = sqlc.arg(recipe_id);
