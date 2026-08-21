-- name: ListGeneratedShoppingLineStates :many
SELECT id, aggregation_key, is_removed, is_completed, override_text
FROM shopping_lines
WHERE shopping_list_id = ? AND origin = 'generated';

-- name: DeleteGeneratedShoppingLines :exec
DELETE FROM shopping_lines WHERE shopping_list_id = ? AND origin = 'generated';

-- name: CreateGeneratedShoppingLine :one
INSERT INTO shopping_lines (
    shopping_list_id, grocery_item_id, store_section_id, aggregation_key,
    origin, display_name, quantity_kind,
    amount_min_numerator, amount_min_denominator,
    amount_max_numerator, amount_max_denominator, unit_id,
    package_type, package_size_numerator, package_size_denominator, package_size_unit_id,
    is_optional, is_removed, is_completed, display_position, override_text
) VALUES (?, ?, ?, ?, 'generated', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: CreateShoppingLineContribution :one
INSERT INTO shopping_line_contributions (
    shopping_line_id, week_recipe_id, recipe_ingredient_id, quantity_kind,
    amount_min_numerator, amount_min_denominator,
    amount_max_numerator, amount_max_denominator, unit_id,
    package_type, package_size_numerator, package_size_denominator, package_size_unit_id,
    is_optional
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListShoppingLines :many
SELECT sl.*, ss.name AS store_section_name, u.key AS unit_key, psu.key AS package_size_unit_key
FROM shopping_lines sl
JOIN store_sections ss ON ss.id = sl.store_section_id
LEFT JOIN units u ON u.id = sl.unit_id
LEFT JOIN units psu ON psu.id = sl.package_size_unit_id
WHERE sl.shopping_list_id = ?
ORDER BY ss.name, sl.display_position, sl.display_name, sl.id;

-- name: ListShoppingLineContributions :many
SELECT c.*, r.name AS recipe_name, ri.source_text, ri.preparation
FROM shopping_line_contributions c
JOIN week_recipes wr ON wr.id = c.week_recipe_id
JOIN recipes r ON r.id = wr.recipe_id
JOIN recipe_ingredients ri ON ri.id = c.recipe_ingredient_id
WHERE c.shopping_line_id = ?
ORDER BY wr.position, c.id;

-- name: GetOtherStoreSection :one
SELECT * FROM store_sections WHERE key = 'other';

-- name: NextManualShoppingLinePosition :one
SELECT CAST(coalesce(max(display_position) + 1, 0) AS INTEGER)
FROM shopping_lines WHERE shopping_list_id = ? AND origin = 'manual';

-- name: CreateManualShoppingLine :one
INSERT INTO shopping_lines (
    shopping_list_id, store_section_id, origin, display_name, quantity_kind,
    display_position
) VALUES (?, ?, 'manual', ?, 'unspecified', ?)
RETURNING *;

-- name: SetShoppingLineRemoved :execrows
UPDATE shopping_lines
SET is_removed = sqlc.arg(removed), updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE shopping_lines.id = sqlc.arg(line_id) AND shopping_list_id = (
    SELECT list.id FROM shopping_lists list
    JOIN weeks w ON w.id = list.week_id
    WHERE w.starts_on = sqlc.arg(starts_on)
);

-- name: SetShoppingLineCompleted :execrows
UPDATE shopping_lines
SET is_completed = sqlc.arg(completed), updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE shopping_lines.id = sqlc.arg(line_id) AND shopping_list_id = (
    SELECT list.id FROM shopping_lists list
    JOIN weeks w ON w.id = list.week_id
    WHERE w.starts_on = sqlc.arg(starts_on)
);

-- name: SetShoppingLineOverride :execrows
UPDATE shopping_lines
SET override_text = sqlc.arg(override_text), updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE shopping_lines.id = sqlc.arg(line_id) AND origin = 'generated' AND shopping_list_id = (
    SELECT list.id FROM shopping_lists list
    JOIN weeks w ON w.id = list.week_id
    WHERE w.starts_on = sqlc.arg(starts_on)
);
