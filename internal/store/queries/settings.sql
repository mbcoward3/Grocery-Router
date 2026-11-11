-- name: GetHouseholdTimezone :one
SELECT timezone FROM households WHERE id = sqlc.arg(household_id);

-- name: GetHouseholdName :one
SELECT name FROM households WHERE id = sqlc.arg(household_id);

-- name: GetHouseholdSettings :one
SELECT timezone, default_recipe_count, avoid_recent_plans
FROM households WHERE id = sqlc.arg(household_id);

-- name: LockHouseholdPlans :one
-- Taken before a plan is opened, closed, or reopened, so those changes run one at a time per
-- household and a household never has two open plans. Changes inside the open plan lock only
-- the plan row (LockOpenWeek), so they never wait for this.
SELECT timezone FROM households WHERE id = sqlc.arg(household_id)
FOR NO KEY UPDATE;

-- name: UpdateHouseholdSettings :one
-- A NULL argument keeps the stored value, so a change to one setting cannot overwrite a
-- concurrent change to another.
UPDATE households SET
    timezone = coalesce(sqlc.narg(timezone), timezone),
    default_recipe_count = coalesce(sqlc.narg(default_recipe_count), default_recipe_count),
    avoid_recent_plans = coalesce(sqlc.narg(avoid_recent_plans), avoid_recent_plans),
    updated_at = clock_timestamp()
WHERE id = sqlc.arg(household_id)
RETURNING timezone, default_recipe_count, avoid_recent_plans;

-- name: GetAvoidRecentPlans :one
SELECT avoid_recent_plans FROM households WHERE id = sqlc.arg(household_id);
