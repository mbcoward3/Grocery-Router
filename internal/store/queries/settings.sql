-- name: GetHouseholdTimezone :one
SELECT timezone FROM households WHERE id = sqlc.arg(household_id);

-- name: SetHouseholdTimezone :execrows
UPDATE households SET timezone = sqlc.arg(timezone), updated_at = clock_timestamp()
WHERE id = sqlc.arg(household_id);
