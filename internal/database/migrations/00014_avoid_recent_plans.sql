-- +goose Up
-- How many of the household's most recent past plans Generate and random swap avoid repeating
-- (D052). 0 keeps plain random selection over every verified recipe.
ALTER TABLE households ADD COLUMN avoid_recent_plans SMALLINT NOT NULL DEFAULT 0
CHECK (avoid_recent_plans BETWEEN 0 AND 8);

-- +goose Down
ALTER TABLE households DROP COLUMN avoid_recent_plans;
