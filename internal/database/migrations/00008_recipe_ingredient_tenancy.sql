-- +goose Up
ALTER TABLE recipe_ingredient_sections ADD COLUMN household_id UUID;
ALTER TABLE recipe_ingredients ADD COLUMN household_id UUID;

-- Verified recipes reject content updates; ownership backfill leaves their content unchanged.
ALTER TABLE recipe_ingredient_sections DISABLE TRIGGER protect_verified_ingredient_sections;
UPDATE recipe_ingredient_sections ris
SET household_id = r.household_id
FROM recipes r WHERE r.id = ris.recipe_id;
ALTER TABLE recipe_ingredient_sections ENABLE TRIGGER protect_verified_ingredient_sections;

ALTER TABLE recipe_ingredients DISABLE TRIGGER protect_verified_ingredients;
UPDATE recipe_ingredients ri
SET household_id = ris.household_id
FROM recipe_ingredient_sections ris WHERE ris.id = ri.section_id;
ALTER TABLE recipe_ingredients ENABLE TRIGGER protect_verified_ingredients;

ALTER TABLE recipe_ingredient_sections ALTER COLUMN household_id SET NOT NULL;
ALTER TABLE recipe_ingredients ALTER COLUMN household_id SET NOT NULL;

ALTER TABLE recipe_ingredient_sections DROP CONSTRAINT recipe_ingredient_sections_recipe_id_fkey;
ALTER TABLE recipe_ingredient_sections ADD CONSTRAINT recipe_ingredient_sections_household_id_id_key UNIQUE (household_id, id);
ALTER TABLE recipe_ingredient_sections ADD CONSTRAINT recipe_ingredient_sections_household_recipe_fkey
    FOREIGN KEY (household_id, recipe_id) REFERENCES recipes(household_id, id)
    ON UPDATE RESTRICT ON DELETE CASCADE;

ALTER TABLE recipe_ingredients DROP CONSTRAINT recipe_ingredients_section_id_fkey;
ALTER TABLE recipe_ingredients DROP CONSTRAINT recipe_ingredients_grocery_item_id_fkey;
ALTER TABLE recipe_ingredients ADD CONSTRAINT recipe_ingredients_household_section_fkey
    FOREIGN KEY (household_id, section_id) REFERENCES recipe_ingredient_sections(household_id, id)
    ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE recipe_ingredients ADD CONSTRAINT recipe_ingredients_household_item_fkey
    FOREIGN KEY (household_id, grocery_item_id) REFERENCES grocery_items(household_id, id)
    ON UPDATE RESTRICT ON DELETE RESTRICT;

-- +goose Down
ALTER TABLE recipe_ingredients DROP CONSTRAINT recipe_ingredients_household_item_fkey;
ALTER TABLE recipe_ingredients DROP CONSTRAINT recipe_ingredients_household_section_fkey;
ALTER TABLE recipe_ingredients ADD CONSTRAINT recipe_ingredients_section_id_fkey
    FOREIGN KEY (section_id) REFERENCES recipe_ingredient_sections(id) ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE recipe_ingredients ADD CONSTRAINT recipe_ingredients_grocery_item_id_fkey
    FOREIGN KEY (grocery_item_id) REFERENCES grocery_items(id) ON UPDATE RESTRICT ON DELETE RESTRICT;

ALTER TABLE recipe_ingredient_sections DROP CONSTRAINT recipe_ingredient_sections_household_recipe_fkey;
ALTER TABLE recipe_ingredient_sections DROP CONSTRAINT recipe_ingredient_sections_household_id_id_key;
ALTER TABLE recipe_ingredient_sections ADD CONSTRAINT recipe_ingredient_sections_recipe_id_fkey
    FOREIGN KEY (recipe_id) REFERENCES recipes(id) ON UPDATE RESTRICT ON DELETE CASCADE;

ALTER TABLE recipe_ingredients DROP COLUMN household_id;
ALTER TABLE recipe_ingredient_sections DROP COLUMN household_id;
