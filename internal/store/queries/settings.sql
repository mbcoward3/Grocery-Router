-- name: GetHouseholdTimezone :one
SELECT timezone FROM households WHERE id = sqlc.arg(household_id);

-- name: SetHouseholdTimezone :execrows
UPDATE households SET timezone = sqlc.arg(timezone), updated_at = clock_timestamp()
WHERE id = sqlc.arg(household_id);

-- name: GetHouseholdName :one
SELECT name FROM households WHERE id = sqlc.arg(household_id);

-- name: GetHouseholdSettings :one
SELECT timezone, default_recipe_count FROM households WHERE id = sqlc.arg(household_id);

-- name: UpdateHouseholdSettings :one
-- A NULL argument keeps the stored value, so a change to one setting cannot overwrite a
-- concurrent change to the other.
UPDATE households SET
    timezone = coalesce(sqlc.narg(timezone), timezone),
    default_recipe_count = coalesce(sqlc.narg(default_recipe_count), default_recipe_count),
    updated_at = clock_timestamp()
WHERE id = sqlc.arg(household_id)
RETURNING timezone, default_recipe_count;
