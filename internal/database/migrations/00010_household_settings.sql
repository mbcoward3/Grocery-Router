-- +goose Up
-- The number of recipes the Week screen offers when a household generates a week. Members edit
-- it, and the household time zone, on the Settings screen.
ALTER TABLE households ADD COLUMN default_recipe_count INTEGER NOT NULL DEFAULT 5
CHECK (default_recipe_count BETWEEN 1 AND 21);

-- +goose Down
ALTER TABLE households DROP COLUMN default_recipe_count;
