-- +goose Up
ALTER TABLE recipe_ingredients
ADD COLUMN include_on_grocery_list INTEGER NOT NULL DEFAULT 1
CHECK (include_on_grocery_list IN (0, 1));

-- +goose Down
ALTER TABLE recipe_ingredients DROP COLUMN include_on_grocery_list;
