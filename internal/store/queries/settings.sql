-- name: GetHouseholdTimezone :one
SELECT timezone FROM households WHERE id = sqlc.arg(household_id);

-- name: GetHouseholdName :one
SELECT name FROM households WHERE id = sqlc.arg(household_id);

-- name: GetHouseholdSettings :one
SELECT timezone, default_recipe_count, week_start_day, rollover_minutes, rollover_mode
FROM households WHERE id = sqlc.arg(household_id);

-- name: LockHouseholdSettings :one
-- Read before a change in the same transaction, so the change is compared with the values it
-- replaces even when another member saves at the same moment.
SELECT timezone, default_recipe_count, week_start_day, rollover_minutes, rollover_mode
FROM households WHERE id = sqlc.arg(household_id)
FOR UPDATE;

-- name: GetWeekRules :one
-- Read when a week is opened. FOR SHARE waits for a settings change in progress, and holds off a
-- new one until the week exists, so that change then moves the new week's close time.
SELECT timezone, week_start_day, rollover_minutes, rollover_mode
FROM households WHERE id = sqlc.arg(household_id)
FOR SHARE;

-- name: UpdateHouseholdSettings :one
-- A NULL argument keeps the stored value, so a change to one setting cannot overwrite a
-- concurrent change to another.
UPDATE households SET
    timezone = coalesce(sqlc.narg(timezone), timezone),
    default_recipe_count = coalesce(sqlc.narg(default_recipe_count), default_recipe_count),
    week_start_day = coalesce(sqlc.narg(week_start_day), week_start_day),
    rollover_minutes = coalesce(sqlc.narg(rollover_minutes), rollover_minutes),
    rollover_mode = coalesce(sqlc.narg(rollover_mode), rollover_mode),
    updated_at = clock_timestamp()
WHERE id = sqlc.arg(household_id)
RETURNING timezone, default_recipe_count, week_start_day, rollover_minutes, rollover_mode;
