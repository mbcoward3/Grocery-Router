-- name: ListGeneratedShoppingLineStates :many
SELECT id, aggregation_key, is_removed, is_completed, override_text
FROM shopping_lines
WHERE household_id = sqlc.arg(household_id) AND shopping_list_id = sqlc.arg(shopping_list_id)
    AND origin = 'generated';

-- name: DeleteGeneratedShoppingLines :exec
DELETE FROM shopping_lines
WHERE household_id = sqlc.arg(household_id) AND shopping_list_id = sqlc.arg(shopping_list_id)
    AND origin = 'generated';

-- name: CreateGeneratedShoppingLine :one
INSERT INTO shopping_lines (
    household_id, shopping_list_id, grocery_item_id, store_section_id, aggregation_key,
    origin, display_name, quantity_kind,
    amount_min_numerator, amount_min_denominator,
    amount_max_numerator, amount_max_denominator, unit_id,
    package_type, package_size_numerator, package_size_denominator, package_size_unit_id,
    is_optional, is_removed, is_completed, display_position, override_text
) VALUES (sqlc.arg(household_id), $1, $2, $3, $4, 'generated', $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
RETURNING *;

-- name: CreateShoppingLineContribution :one
INSERT INTO shopping_line_contributions (
    household_id, shopping_line_id, week_recipe_id, recipe_ingredient_id, quantity_kind,
    amount_min_numerator, amount_min_denominator,
    amount_max_numerator, amount_max_denominator, unit_id,
    package_type, package_size_numerator, package_size_denominator, package_size_unit_id,
    is_optional
) VALUES (sqlc.arg(household_id), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
RETURNING *;

-- name: ListShoppingLines :many
SELECT sl.*, ss.name AS store_section_name, u.key AS unit_key, u.symbol AS unit_symbol,
    psu.key AS package_size_unit_key, psu.symbol AS package_size_unit_symbol
FROM shopping_lines sl
JOIN store_sections ss ON ss.household_id = sl.household_id AND ss.id = sl.store_section_id
LEFT JOIN grocery_items gi ON gi.household_id = sl.household_id AND gi.id = sl.grocery_item_id
LEFT JOIN units u ON u.id = sl.unit_id
LEFT JOIN units psu ON psu.id = sl.package_size_unit_id
WHERE sl.household_id = sqlc.arg(household_id)
    AND sl.shopping_list_id = sqlc.arg(shopping_list_id)
ORDER BY ss.display_position, ss.name, coalesce(gi.name, sl.display_name), sl.origin, sl.display_position, sl.id;

-- name: ListShoppingLineRecipeNames :many
SELECT
    c.shopping_line_id,
    r.id AS recipe_id,
    r.name AS recipe_name
FROM shopping_line_contributions AS c
INNER JOIN shopping_lines AS sl
    ON sl.household_id = c.household_id AND sl.id = c.shopping_line_id
INNER JOIN week_recipes AS wr
    ON wr.household_id = c.household_id AND wr.id = c.week_recipe_id
INNER JOIN recipes AS r
    ON r.household_id = wr.household_id AND r.id = wr.recipe_id
WHERE
    c.household_id = sqlc.arg(household_id)
    AND sl.shopping_list_id = sqlc.arg(shopping_list_id)
GROUP BY c.shopping_line_id, r.id, r.name
ORDER BY c.shopping_line_id, min(wr.position), r.name, r.id;

-- name: ListShoppingLineContributions :many
SELECT c.*, r.id AS recipe_id, r.name AS recipe_name, ri.source_text, ri.preparation,
    u.symbol AS unit_symbol, psu.symbol AS package_size_unit_symbol
FROM shopping_line_contributions c
JOIN week_recipes wr ON wr.household_id = c.household_id AND wr.id = c.week_recipe_id
JOIN recipes r ON r.household_id = wr.household_id AND r.id = wr.recipe_id
JOIN recipe_ingredients ri ON ri.id = c.recipe_ingredient_id
LEFT JOIN units u ON u.id = c.unit_id
LEFT JOIN units psu ON psu.id = c.package_size_unit_id
WHERE c.household_id = sqlc.arg(household_id)
    AND c.shopping_line_id = sqlc.arg(shopping_line_id)
ORDER BY wr.position, c.id;

-- name: GetOtherStoreSection :one
SELECT * FROM store_sections WHERE household_id = $1 AND key = 'other';

-- name: NextManualShoppingLinePosition :one
SELECT CAST(coalesce(max(display_position) + 1, 0) AS BIGINT)
FROM shopping_lines
WHERE household_id = sqlc.arg(household_id) AND shopping_list_id = sqlc.arg(shopping_list_id)
    AND origin = 'manual';

-- name: CreateManualShoppingLine :one
INSERT INTO shopping_lines (
    household_id, shopping_list_id, store_section_id, grocery_item_id, origin, display_name, quantity_kind,
    display_position
) VALUES (sqlc.arg(household_id), $1, $2, sqlc.narg(grocery_item_id), 'manual', $3, 'unspecified', $4)
RETURNING *;

-- name: SetShoppingLineRemoved :execrows
UPDATE shopping_lines
SET is_removed = sqlc.arg(removed), updated_at = to_char(clock_timestamp() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"')
WHERE shopping_lines.household_id = sqlc.arg(household_id)
    AND shopping_lines.id = sqlc.arg(line_id) AND shopping_list_id = (
        SELECT list.id FROM shopping_lists list
        JOIN weeks w ON w.household_id = list.household_id AND w.id = list.week_id
        WHERE w.household_id = sqlc.arg(household_id) AND w.starts_on = sqlc.arg(starts_on)
    );

-- name: SetShoppingLineCompleted :execrows
UPDATE shopping_lines
SET is_completed = sqlc.arg(completed), updated_at = to_char(clock_timestamp() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"')
WHERE shopping_lines.household_id = sqlc.arg(household_id)
    AND shopping_lines.id = sqlc.arg(line_id) AND shopping_list_id = (
        SELECT list.id FROM shopping_lists list
        JOIN weeks w ON w.household_id = list.household_id AND w.id = list.week_id
        WHERE w.household_id = sqlc.arg(household_id) AND w.starts_on = sqlc.arg(starts_on)
    );

-- name: SetShoppingLineOverride :execrows
UPDATE shopping_lines
SET override_text = sqlc.arg(override_text), updated_at = to_char(clock_timestamp() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"')
WHERE shopping_lines.household_id = sqlc.arg(household_id)
    AND shopping_lines.id = sqlc.arg(line_id) AND origin = 'generated' AND shopping_list_id = (
        SELECT list.id FROM shopping_lists list
        JOIN weeks w ON w.household_id = list.household_id AND w.id = list.week_id
        WHERE w.household_id = sqlc.arg(household_id) AND w.starts_on = sqlc.arg(starts_on)
    );

-- name: SetManualLineSection :one
UPDATE shopping_lines SET store_section_id = sqlc.arg(store_section_id)
WHERE shopping_lines.household_id = sqlc.arg(household_id) AND shopping_lines.id = sqlc.arg(line_id) AND origin = 'manual'
    AND shopping_list_id = (
        SELECT list.id FROM shopping_lists list
        JOIN weeks w ON w.household_id = list.household_id AND w.id = list.week_id
        WHERE w.household_id = sqlc.arg(household_id) AND w.starts_on = sqlc.arg(starts_on)
    )
RETURNING display_name, grocery_item_id;

-- name: ListRememberedItemSections :many
SELECT ris.name_key, ris.display_name, ris.store_section_id, ss.name AS store_section_name
FROM remembered_item_sections ris
JOIN store_sections ss ON ss.household_id = ris.household_id AND ss.id = ris.store_section_id
WHERE ris.household_id = $1
ORDER BY ris.display_name;

-- name: GetRememberedItemSection :one
SELECT store_section_id FROM remembered_item_sections
WHERE household_id = $1 AND name_key = $2;

-- name: RememberItemSection :exec
INSERT INTO remembered_item_sections (household_id, name_key, display_name, store_section_id)
VALUES ($1, $2, $3, $4)
ON CONFLICT (household_id, name_key) DO UPDATE
SET display_name = excluded.display_name, store_section_id = excluded.store_section_id,
    updated_at = to_char(clock_timestamp() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"');

-- name: ForgetItemSection :exec
DELETE FROM remembered_item_sections WHERE household_id = $1 AND name_key = $2;

-- name: LockSectionOrder :one
SELECT id FROM households WHERE id = $1 FOR UPDATE;

-- name: SetSectionPosition :exec
UPDATE store_sections SET display_position = sqlc.arg(display_position)
WHERE household_id = sqlc.arg(household_id) AND id = sqlc.arg(id);

-- name: ResetSectionPositions :exec
UPDATE store_sections SET display_position = default_section_position(key) WHERE household_id = $1;

-- name: CreateShoppingLineQuantity :exec
INSERT INTO shopping_line_quantities (
    household_id, shopping_line_id, position, quantity_kind,
    amount_min_numerator, amount_min_denominator,
    amount_max_numerator, amount_max_denominator, unit_id,
    package_type, package_size_numerator, package_size_denominator, package_size_unit_id
) VALUES (sqlc.arg(household_id), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);

-- name: ListShoppingLineQuantities :many
SELECT
    q.shopping_line_id,
    q.position,
    q.quantity_kind,
    q.amount_min_numerator,
    q.amount_min_denominator,
    q.amount_max_numerator,
    q.amount_max_denominator,
    q.package_type,
    q.package_size_numerator,
    q.package_size_denominator,
    u.key AS unit_key,
    u.symbol AS unit_symbol,
    psu.key AS package_size_unit_key,
    psu.symbol AS package_size_unit_symbol
FROM shopping_line_quantities AS q
INNER JOIN shopping_lines AS sl
    ON sl.household_id = q.household_id AND sl.id = q.shopping_line_id
LEFT JOIN units AS u ON q.unit_id = u.id
LEFT JOIN units AS psu ON q.package_size_unit_id = psu.id
WHERE
    q.household_id = sqlc.arg(household_id)
    AND sl.shopping_list_id = sqlc.arg(shopping_list_id)
ORDER BY q.shopping_line_id, q.position;

-- name: ListShoppingLinesWithUnspecified :many
SELECT DISTINCT c.shopping_line_id
FROM shopping_line_contributions AS c
INNER JOIN shopping_lines AS sl
    ON sl.household_id = c.household_id AND sl.id = c.shopping_line_id
WHERE
    c.household_id = sqlc.arg(household_id)
    AND sl.shopping_list_id = sqlc.arg(shopping_list_id)
    AND c.quantity_kind = 'unspecified'
ORDER BY c.shopping_line_id;

-- name: SetShoppingListAggregationVersion :execrows
UPDATE shopping_lists
SET aggregation_version = sqlc.arg(version)
WHERE
    household_id = sqlc.arg(household_id)
    AND id = sqlc.arg(id)
    AND aggregation_version < sqlc.arg(version);
