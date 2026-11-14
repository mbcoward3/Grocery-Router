-- +goose Up
-- Past plans keep what they showed when they were made, so later recipe edits never change
-- History (RECIPE_LIBRARY_SPEC.md §3.4). Contributions and occurrences carry snapshots of the
-- recipe fields they display, and a contribution outlives the ingredient it came from.

ALTER TABLE shopping_line_contributions ADD COLUMN recipe_id BIGINT;
ALTER TABLE shopping_line_contributions ADD COLUMN recipe_name TEXT;
ALTER TABLE shopping_line_contributions ADD COLUMN source_text TEXT;
ALTER TABLE shopping_line_contributions ADD COLUMN preparation TEXT;

-- The backfill only copies values; it does not move a contribution, so the matching check is
-- not needed for it.
ALTER TABLE shopping_line_contributions DISABLE TRIGGER shopping_contributions_require_matching_recipe;
UPDATE shopping_line_contributions AS c
SET
    recipe_id = r.id,
    recipe_name = r.name,
    source_text = ri.source_text,
    preparation = ri.preparation
FROM week_recipes AS wr, recipes AS r, recipe_ingredients AS ri
WHERE
    wr.household_id = c.household_id
    AND wr.id = c.week_recipe_id
    AND r.household_id = wr.household_id
    AND r.id = wr.recipe_id
    AND ri.id = c.recipe_ingredient_id;
ALTER TABLE shopping_line_contributions ENABLE TRIGGER shopping_contributions_require_matching_recipe;

ALTER TABLE shopping_line_contributions ALTER COLUMN recipe_id SET NOT NULL;
ALTER TABLE shopping_line_contributions ALTER COLUMN recipe_name SET NOT NULL;
ALTER TABLE shopping_line_contributions ALTER COLUMN source_text SET NOT NULL;

ALTER TABLE week_recipes ADD COLUMN recipe_name TEXT;
ALTER TABLE week_recipes ADD COLUMN recipe_key TEXT;
ALTER TABLE week_recipes ADD COLUMN image_url TEXT;
ALTER TABLE week_recipes ADD COLUMN yield_text TEXT;
ALTER TABLE week_recipes ADD COLUMN hands_on_min_minutes BIGINT;
ALTER TABLE week_recipes ADD COLUMN hands_on_max_minutes BIGINT;
ALTER TABLE week_recipes ADD COLUMN unattended_min_minutes BIGINT;
ALTER TABLE week_recipes ADD COLUMN unattended_max_minutes BIGINT;

UPDATE week_recipes AS wr
SET
    recipe_name = r.name,
    recipe_key = r.key,
    image_url = r.image_url,
    yield_text = r.yield_text,
    hands_on_min_minutes = r.hands_on_min_minutes,
    hands_on_max_minutes = r.hands_on_max_minutes,
    unattended_min_minutes = r.unattended_min_minutes,
    unattended_max_minutes = r.unattended_max_minutes
FROM recipes AS r
WHERE r.household_id = wr.household_id AND r.id = wr.recipe_id;

ALTER TABLE week_recipes ALTER COLUMN recipe_name SET NOT NULL;
ALTER TABLE week_recipes ALTER COLUMN recipe_key SET NOT NULL;

-- Deleting an ingredient detaches the contributions that came from it instead of being refused.
ALTER TABLE shopping_line_contributions ALTER COLUMN recipe_ingredient_id DROP NOT NULL;
ALTER TABLE shopping_line_contributions
DROP CONSTRAINT shopping_line_contributions_recipe_ingredient_id_fkey;
ALTER TABLE shopping_line_contributions
ADD CONSTRAINT shopping_line_contributions_recipe_ingredient_id_fkey
FOREIGN KEY (recipe_ingredient_id) REFERENCES recipe_ingredients (id)
ON UPDATE RESTRICT ON DELETE SET NULL;

-- The SET NULL above is an UPDATE. It is the one change exempt from the matching check: the row
-- keeps its line and occurrence and only loses its deleted ingredient. An INSERT, or any other
-- UPDATE, is still checked, so a NULL ingredient can never be inserted.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION require_matching_contribution() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE'
        AND NEW.recipe_ingredient_id IS NULL
        AND OLD.recipe_ingredient_id IS NOT NULL
        AND NEW.shopping_line_id = OLD.shopping_line_id
        AND NEW.week_recipe_id = OLD.week_recipe_id THEN
        RETURN NEW;
    END IF;
    IF NOT EXISTS (
        SELECT 1
        FROM shopping_lines sl
        JOIN shopping_lists list ON list.id = sl.shopping_list_id
        JOIN week_recipes wr ON wr.id = NEW.week_recipe_id AND wr.week_id = list.week_id
        JOIN recipe_ingredient_sections ris ON ris.recipe_id = wr.recipe_id
        JOIN recipe_ingredients ri ON ri.section_id = ris.id AND ri.id = NEW.recipe_ingredient_id
        WHERE sl.id = NEW.shopping_line_id AND sl.origin = 'generated'
          AND sl.grocery_item_id = ri.grocery_item_id
    ) THEN
        RAISE EXCEPTION 'contribution must match its line, week occurrence, and recipe ingredient';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION require_matching_contribution() RETURNS TRIGGER AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM shopping_lines sl
        JOIN shopping_lists list ON list.id = sl.shopping_list_id
        JOIN week_recipes wr ON wr.id = NEW.week_recipe_id AND wr.week_id = list.week_id
        JOIN recipe_ingredient_sections ris ON ris.recipe_id = wr.recipe_id
        JOIN recipe_ingredients ri ON ri.section_id = ris.id AND ri.id = NEW.recipe_ingredient_id
        WHERE sl.id = NEW.shopping_line_id AND sl.origin = 'generated'
          AND sl.grocery_item_id = ri.grocery_item_id
    ) THEN
        RAISE EXCEPTION 'contribution must match its line, week occurrence, and recipe ingredient';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- A contribution whose ingredient was deleted cannot exist under the earlier schema; rolling back
-- drops it.
DELETE FROM shopping_line_contributions WHERE recipe_ingredient_id IS NULL;
ALTER TABLE shopping_line_contributions
DROP CONSTRAINT shopping_line_contributions_recipe_ingredient_id_fkey;
ALTER TABLE shopping_line_contributions
ADD CONSTRAINT shopping_line_contributions_recipe_ingredient_id_fkey
FOREIGN KEY (recipe_ingredient_id) REFERENCES recipe_ingredients (id)
ON UPDATE RESTRICT ON DELETE RESTRICT;
ALTER TABLE shopping_line_contributions ALTER COLUMN recipe_ingredient_id SET NOT NULL;

ALTER TABLE week_recipes DROP COLUMN unattended_max_minutes;
ALTER TABLE week_recipes DROP COLUMN unattended_min_minutes;
ALTER TABLE week_recipes DROP COLUMN hands_on_max_minutes;
ALTER TABLE week_recipes DROP COLUMN hands_on_min_minutes;
ALTER TABLE week_recipes DROP COLUMN yield_text;
ALTER TABLE week_recipes DROP COLUMN image_url;
ALTER TABLE week_recipes DROP COLUMN recipe_key;
ALTER TABLE week_recipes DROP COLUMN recipe_name;

ALTER TABLE shopping_line_contributions DROP COLUMN preparation;
ALTER TABLE shopping_line_contributions DROP COLUMN source_text;
ALTER TABLE shopping_line_contributions DROP COLUMN recipe_name;
ALTER TABLE shopping_line_contributions DROP COLUMN recipe_id;
