-- +goose Up
-- Root ownership is backfilled explicitly to the one approved household, then made mandatory.
ALTER TABLE store_sections ADD COLUMN household_id UUID REFERENCES households(id);
ALTER TABLE grocery_items ADD COLUMN household_id UUID REFERENCES households(id);
ALTER TABLE recipes ADD COLUMN household_id UUID REFERENCES households(id);
ALTER TABLE weeks ADD COLUMN household_id UUID REFERENCES households(id);
ALTER TABLE week_recipes ADD COLUMN household_id UUID REFERENCES households(id);
ALTER TABLE shopping_lists ADD COLUMN household_id UUID REFERENCES households(id);
ALTER TABLE shopping_lines ADD COLUMN household_id UUID REFERENCES households(id);
ALTER TABLE shopping_line_contributions ADD COLUMN household_id UUID REFERENCES households(id);

UPDATE store_sections SET household_id = 'c0a7a2d8-669b-4e47-91c1-4d9a32f339d5';
UPDATE grocery_items SET household_id = 'c0a7a2d8-669b-4e47-91c1-4d9a32f339d5';
-- The v1 lifecycle trigger intentionally rejects every in-place edit of a verified recipe.
-- Disable only that trigger while adding ownership; recipe content and status remain unchanged.
ALTER TABLE recipes DISABLE TRIGGER enforce_recipe_lifecycle;
UPDATE recipes SET household_id = 'c0a7a2d8-669b-4e47-91c1-4d9a32f339d5';
ALTER TABLE recipes ENABLE TRIGGER enforce_recipe_lifecycle;
UPDATE weeks SET household_id = 'c0a7a2d8-669b-4e47-91c1-4d9a32f339d5';
UPDATE week_recipes SET household_id = 'c0a7a2d8-669b-4e47-91c1-4d9a32f339d5';
UPDATE shopping_lists SET household_id = 'c0a7a2d8-669b-4e47-91c1-4d9a32f339d5';
UPDATE shopping_lines SET household_id = 'c0a7a2d8-669b-4e47-91c1-4d9a32f339d5';
UPDATE shopping_line_contributions SET household_id = 'c0a7a2d8-669b-4e47-91c1-4d9a32f339d5';

ALTER TABLE store_sections ALTER COLUMN household_id SET NOT NULL;
ALTER TABLE grocery_items ALTER COLUMN household_id SET NOT NULL;
ALTER TABLE recipes ALTER COLUMN household_id SET NOT NULL;
ALTER TABLE weeks ALTER COLUMN household_id SET NOT NULL;
ALTER TABLE week_recipes ALTER COLUMN household_id SET NOT NULL;
ALTER TABLE shopping_lists ALTER COLUMN household_id SET NOT NULL;
ALTER TABLE shopping_lines ALTER COLUMN household_id SET NOT NULL;
ALTER TABLE shopping_line_contributions ALTER COLUMN household_id SET NOT NULL;

-- Names and natural keys are household-local.
ALTER TABLE store_sections DROP CONSTRAINT store_sections_key_key;
ALTER TABLE store_sections DROP CONSTRAINT store_sections_name_key;
ALTER TABLE store_sections ADD CONSTRAINT store_sections_household_key_key UNIQUE (household_id, key);
ALTER TABLE store_sections ADD CONSTRAINT store_sections_household_name_key UNIQUE (household_id, name);
ALTER TABLE store_sections ADD CONSTRAINT store_sections_household_id_id_key UNIQUE (household_id, id);

ALTER TABLE grocery_items DROP CONSTRAINT grocery_items_key_key;
ALTER TABLE grocery_items DROP CONSTRAINT grocery_items_name_key;
ALTER TABLE grocery_items DROP CONSTRAINT grocery_items_store_section_id_fkey;
ALTER TABLE grocery_items ADD CONSTRAINT grocery_items_household_key_key UNIQUE (household_id, key);
ALTER TABLE grocery_items ADD CONSTRAINT grocery_items_household_name_key UNIQUE (household_id, name);
ALTER TABLE grocery_items ADD CONSTRAINT grocery_items_household_id_id_key UNIQUE (household_id, id);
ALTER TABLE grocery_items ADD CONSTRAINT grocery_items_household_section_fkey
    FOREIGN KEY (household_id, store_section_id) REFERENCES store_sections(household_id, id)
    ON UPDATE RESTRICT ON DELETE RESTRICT;

ALTER TABLE recipes DROP CONSTRAINT recipes_key_key;
ALTER TABLE recipes DROP CONSTRAINT recipes_name_key;
ALTER TABLE recipes ADD CONSTRAINT recipes_household_key_key UNIQUE (household_id, key);
ALTER TABLE recipes ADD CONSTRAINT recipes_household_name_key UNIQUE (household_id, name);
ALTER TABLE recipes ADD CONSTRAINT recipes_household_id_id_key UNIQUE (household_id, id);

ALTER TABLE weeks DROP CONSTRAINT weeks_starts_on_key;
ALTER TABLE weeks ADD CONSTRAINT weeks_household_start_key UNIQUE (household_id, starts_on);
ALTER TABLE weeks ADD CONSTRAINT weeks_household_id_id_key UNIQUE (household_id, id);

-- Every relationship between household aggregates proves equal tenancy in the database.
ALTER TABLE week_recipes DROP CONSTRAINT week_recipes_week_id_fkey;
ALTER TABLE week_recipes DROP CONSTRAINT week_recipes_recipe_id_fkey;
ALTER TABLE week_recipes ADD CONSTRAINT week_recipes_household_id_id_key UNIQUE (household_id, id);
ALTER TABLE week_recipes ADD CONSTRAINT week_recipes_household_week_fkey
    FOREIGN KEY (household_id, week_id) REFERENCES weeks(household_id, id)
    ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE week_recipes ADD CONSTRAINT week_recipes_household_recipe_fkey
    FOREIGN KEY (household_id, recipe_id) REFERENCES recipes(household_id, id)
    ON UPDATE RESTRICT ON DELETE RESTRICT;

ALTER TABLE shopping_lists DROP CONSTRAINT shopping_lists_week_id_fkey;
ALTER TABLE shopping_lists ADD CONSTRAINT shopping_lists_household_id_id_key UNIQUE (household_id, id);
ALTER TABLE shopping_lists ADD CONSTRAINT shopping_lists_household_week_fkey
    FOREIGN KEY (household_id, week_id) REFERENCES weeks(household_id, id)
    ON UPDATE RESTRICT ON DELETE CASCADE;

ALTER TABLE shopping_lines DROP CONSTRAINT shopping_lines_shopping_list_id_fkey;
ALTER TABLE shopping_lines DROP CONSTRAINT shopping_lines_grocery_item_id_fkey;
ALTER TABLE shopping_lines DROP CONSTRAINT shopping_lines_store_section_id_fkey;
ALTER TABLE shopping_lines ADD CONSTRAINT shopping_lines_household_id_id_key UNIQUE (household_id, id);
ALTER TABLE shopping_lines ADD CONSTRAINT shopping_lines_household_list_fkey
    FOREIGN KEY (household_id, shopping_list_id) REFERENCES shopping_lists(household_id, id)
    ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE shopping_lines ADD CONSTRAINT shopping_lines_household_item_fkey
    FOREIGN KEY (household_id, grocery_item_id) REFERENCES grocery_items(household_id, id)
    ON UPDATE RESTRICT ON DELETE RESTRICT;
ALTER TABLE shopping_lines ADD CONSTRAINT shopping_lines_household_section_fkey
    FOREIGN KEY (household_id, store_section_id) REFERENCES store_sections(household_id, id)
    ON UPDATE RESTRICT ON DELETE RESTRICT;

ALTER TABLE shopping_line_contributions DROP CONSTRAINT shopping_line_contributions_shopping_line_id_fkey;
ALTER TABLE shopping_line_contributions DROP CONSTRAINT shopping_line_contributions_week_recipe_id_fkey;
ALTER TABLE shopping_line_contributions ADD CONSTRAINT contributions_household_line_fkey
    FOREIGN KEY (household_id, shopping_line_id) REFERENCES shopping_lines(household_id, id)
    ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE shopping_line_contributions ADD CONSTRAINT contributions_household_occurrence_fkey
    FOREIGN KEY (household_id, week_recipe_id) REFERENCES week_recipes(household_id, id)
    ON UPDATE RESTRICT ON DELETE CASCADE;

-- +goose Down
ALTER TABLE shopping_line_contributions DROP CONSTRAINT contributions_household_occurrence_fkey;
ALTER TABLE shopping_line_contributions DROP CONSTRAINT contributions_household_line_fkey;
ALTER TABLE shopping_line_contributions ADD CONSTRAINT shopping_line_contributions_week_recipe_id_fkey
    FOREIGN KEY (week_recipe_id) REFERENCES week_recipes(id) ON UPDATE RESTRICT ON DELETE CASCADE;
ALTER TABLE shopping_line_contributions ADD CONSTRAINT shopping_line_contributions_shopping_line_id_fkey
    FOREIGN KEY (shopping_line_id) REFERENCES shopping_lines(id) ON UPDATE RESTRICT ON DELETE CASCADE;

ALTER TABLE shopping_lines DROP CONSTRAINT shopping_lines_household_section_fkey;
ALTER TABLE shopping_lines DROP CONSTRAINT shopping_lines_household_item_fkey;
ALTER TABLE shopping_lines DROP CONSTRAINT shopping_lines_household_list_fkey;
ALTER TABLE shopping_lines DROP CONSTRAINT shopping_lines_household_id_id_key;
ALTER TABLE shopping_lines ADD CONSTRAINT shopping_lines_store_section_id_fkey FOREIGN KEY (store_section_id) REFERENCES store_sections(id) ON UPDATE RESTRICT ON DELETE RESTRICT;
ALTER TABLE shopping_lines ADD CONSTRAINT shopping_lines_grocery_item_id_fkey FOREIGN KEY (grocery_item_id) REFERENCES grocery_items(id) ON UPDATE RESTRICT ON DELETE RESTRICT;
ALTER TABLE shopping_lines ADD CONSTRAINT shopping_lines_shopping_list_id_fkey FOREIGN KEY (shopping_list_id) REFERENCES shopping_lists(id) ON UPDATE RESTRICT ON DELETE CASCADE;

ALTER TABLE shopping_lists DROP CONSTRAINT shopping_lists_household_week_fkey;
ALTER TABLE shopping_lists DROP CONSTRAINT shopping_lists_household_id_id_key;
ALTER TABLE shopping_lists ADD CONSTRAINT shopping_lists_week_id_fkey FOREIGN KEY (week_id) REFERENCES weeks(id) ON UPDATE RESTRICT ON DELETE CASCADE;

ALTER TABLE week_recipes DROP CONSTRAINT week_recipes_household_recipe_fkey;
ALTER TABLE week_recipes DROP CONSTRAINT week_recipes_household_week_fkey;
ALTER TABLE week_recipes DROP CONSTRAINT week_recipes_household_id_id_key;
ALTER TABLE week_recipes ADD CONSTRAINT week_recipes_recipe_id_fkey FOREIGN KEY (recipe_id) REFERENCES recipes(id) ON UPDATE RESTRICT ON DELETE RESTRICT;
ALTER TABLE week_recipes ADD CONSTRAINT week_recipes_week_id_fkey FOREIGN KEY (week_id) REFERENCES weeks(id) ON UPDATE RESTRICT ON DELETE CASCADE;

ALTER TABLE weeks DROP CONSTRAINT weeks_household_id_id_key;
ALTER TABLE weeks DROP CONSTRAINT weeks_household_start_key;
ALTER TABLE weeks ADD CONSTRAINT weeks_starts_on_key UNIQUE (starts_on);
ALTER TABLE recipes DROP CONSTRAINT recipes_household_id_id_key;
ALTER TABLE recipes DROP CONSTRAINT recipes_household_name_key;
ALTER TABLE recipes DROP CONSTRAINT recipes_household_key_key;
ALTER TABLE recipes ADD CONSTRAINT recipes_name_key UNIQUE (name);
ALTER TABLE recipes ADD CONSTRAINT recipes_key_key UNIQUE (key);
ALTER TABLE grocery_items DROP CONSTRAINT grocery_items_household_section_fkey;
ALTER TABLE grocery_items DROP CONSTRAINT grocery_items_household_id_id_key;
ALTER TABLE grocery_items DROP CONSTRAINT grocery_items_household_name_key;
ALTER TABLE grocery_items DROP CONSTRAINT grocery_items_household_key_key;
ALTER TABLE grocery_items ADD CONSTRAINT grocery_items_store_section_id_fkey FOREIGN KEY (store_section_id) REFERENCES store_sections(id) ON UPDATE RESTRICT ON DELETE RESTRICT;
ALTER TABLE grocery_items ADD CONSTRAINT grocery_items_name_key UNIQUE (name);
ALTER TABLE grocery_items ADD CONSTRAINT grocery_items_key_key UNIQUE (key);
ALTER TABLE store_sections DROP CONSTRAINT store_sections_household_id_id_key;
ALTER TABLE store_sections DROP CONSTRAINT store_sections_household_name_key;
ALTER TABLE store_sections DROP CONSTRAINT store_sections_household_key_key;
ALTER TABLE store_sections ADD CONSTRAINT store_sections_name_key UNIQUE (name);
ALTER TABLE store_sections ADD CONSTRAINT store_sections_key_key UNIQUE (key);

ALTER TABLE shopping_line_contributions DROP COLUMN household_id;
ALTER TABLE shopping_lines DROP COLUMN household_id;
ALTER TABLE shopping_lists DROP COLUMN household_id;
ALTER TABLE week_recipes DROP COLUMN household_id;
ALTER TABLE weeks DROP COLUMN household_id;
ALTER TABLE recipes DROP COLUMN household_id;
ALTER TABLE grocery_items DROP COLUMN household_id;
ALTER TABLE store_sections DROP COLUMN household_id;
