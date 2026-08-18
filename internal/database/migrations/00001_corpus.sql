-- +goose Up
PRAGMA foreign_keys = ON;

CREATE TABLE store_sections (
    id INTEGER PRIMARY KEY,
    key TEXT NOT NULL UNIQUE CHECK (key <> '' AND key = lower(key)),
    name TEXT NOT NULL UNIQUE CHECK (trim(name) <> '')
) STRICT;

CREATE TABLE units (
    id INTEGER PRIMARY KEY,
    key TEXT NOT NULL UNIQUE CHECK (key <> '' AND key = lower(key)),
    name TEXT NOT NULL CHECK (trim(name) <> ''),
    symbol TEXT NOT NULL CHECK (trim(symbol) <> ''),
    dimension TEXT NOT NULL CHECK (dimension IN ('count', 'mass', 'volume')),
    to_base_numerator INTEGER NOT NULL CHECK (to_base_numerator > 0),
    to_base_denominator INTEGER NOT NULL CHECK (to_base_denominator > 0),
    UNIQUE (dimension, name)
) STRICT;

CREATE TABLE grocery_items (
    id INTEGER PRIMARY KEY,
    key TEXT NOT NULL UNIQUE CHECK (key <> '' AND key = lower(key)),
    name TEXT NOT NULL UNIQUE CHECK (trim(name) <> ''),
    store_section_id INTEGER NOT NULL REFERENCES store_sections(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    shopping_mode TEXT NOT NULL CHECK (shopping_mode IN ('measured', 'counted', 'presence-only')),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
) STRICT;

CREATE TABLE recipes (
    id INTEGER PRIMARY KEY,
    key TEXT NOT NULL UNIQUE CHECK (key <> '' AND key = lower(key)),
    name TEXT NOT NULL UNIQUE CHECK (trim(name) <> ''),
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'reviewable', 'verified')),
    image_url TEXT,
    yield_text TEXT,
    hands_on_min_minutes INTEGER,
    hands_on_max_minutes INTEGER,
    unattended_min_minutes INTEGER,
    unattended_max_minutes INTEGER,
    verified_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    CHECK (image_url IS NULL OR trim(image_url) <> ''),
    CHECK (yield_text IS NULL OR trim(yield_text) <> ''),
    CHECK (
        (hands_on_min_minutes IS NULL AND hands_on_max_minutes IS NULL)
        OR (hands_on_min_minutes >= 0 AND hands_on_max_minutes >= hands_on_min_minutes)
    ),
    CHECK (
        (unattended_min_minutes IS NULL AND unattended_max_minutes IS NULL)
        OR (unattended_min_minutes >= 0 AND unattended_max_minutes >= unattended_min_minutes)
    ),
    CHECK (
        (status = 'verified' AND verified_at IS NOT NULL)
        OR (status <> 'verified' AND verified_at IS NULL)
    )
) STRICT;

CREATE TABLE recipe_sources (
    id INTEGER PRIMARY KEY,
    recipe_id INTEGER NOT NULL REFERENCES recipes(id) ON UPDATE RESTRICT ON DELETE CASCADE,
    relationship TEXT NOT NULL CHECK (relationship IN ('source', 'adapted-from')),
    attribution TEXT NOT NULL CHECK (trim(attribution) <> ''),
    url TEXT CHECK (url IS NULL OR trim(url) <> ''),
    checked_on TEXT CHECK (checked_on IS NULL OR checked_on GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'),
    is_primary INTEGER NOT NULL DEFAULT 0 CHECK (is_primary IN (0, 1)),
    position INTEGER NOT NULL CHECK (position >= 0),
    UNIQUE (recipe_id, position)
) STRICT;

CREATE UNIQUE INDEX one_primary_source_per_recipe
    ON recipe_sources(recipe_id)
    WHERE is_primary = 1;

CREATE TABLE recipe_ingredient_sections (
    id INTEGER PRIMARY KEY,
    recipe_id INTEGER NOT NULL REFERENCES recipes(id) ON UPDATE RESTRICT ON DELETE CASCADE,
    name TEXT NOT NULL CHECK (trim(name) <> ''),
    position INTEGER NOT NULL CHECK (position >= 0),
    UNIQUE (recipe_id, position)
) STRICT;

CREATE TABLE recipe_ingredients (
    id INTEGER PRIMARY KEY,
    section_id INTEGER NOT NULL REFERENCES recipe_ingredient_sections(id) ON UPDATE RESTRICT ON DELETE CASCADE,
    grocery_item_id INTEGER REFERENCES grocery_items(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    position INTEGER NOT NULL CHECK (position >= 0),
    source_text TEXT NOT NULL CHECK (trim(source_text) <> ''),
    quantity_kind TEXT NOT NULL CHECK (quantity_kind IN ('exact', 'range', 'unspecified')),
    amount_min_numerator INTEGER,
    amount_min_denominator INTEGER,
    amount_max_numerator INTEGER,
    amount_max_denominator INTEGER,
    unit_id INTEGER REFERENCES units(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    package_type TEXT CHECK (package_type IS NULL OR trim(package_type) <> ''),
    package_size_numerator INTEGER,
    package_size_denominator INTEGER,
    package_size_unit_id INTEGER REFERENCES units(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    preparation TEXT CHECK (preparation IS NULL OR trim(preparation) <> ''),
    is_optional INTEGER NOT NULL DEFAULT 0 CHECK (is_optional IN (0, 1)),
    display_note TEXT CHECK (display_note IS NULL OR trim(display_note) <> ''),
    UNIQUE (section_id, position),
    CHECK (
        (quantity_kind = 'unspecified'
            AND amount_min_numerator IS NULL
            AND amount_min_denominator IS NULL
            AND amount_max_numerator IS NULL
            AND amount_max_denominator IS NULL
            AND unit_id IS NULL
            AND package_type IS NULL
            AND package_size_numerator IS NULL
            AND package_size_denominator IS NULL
            AND package_size_unit_id IS NULL)
        OR
        (quantity_kind = 'exact'
            AND amount_min_numerator IS NOT NULL
            AND amount_min_numerator >= 0
            AND amount_min_denominator IS NOT NULL
            AND amount_min_denominator > 0
            AND amount_max_numerator IS NULL
            AND amount_max_denominator IS NULL
            AND ((unit_id IS NOT NULL AND package_type IS NULL)
                OR (unit_id IS NULL AND package_type IS NOT NULL)))
        OR
        (quantity_kind = 'range'
            AND amount_min_numerator IS NOT NULL
            AND amount_min_numerator >= 0
            AND amount_min_denominator IS NOT NULL
            AND amount_min_denominator > 0
            AND amount_max_numerator IS NOT NULL
            AND amount_max_numerator >= 0
            AND amount_max_denominator IS NOT NULL
            AND amount_max_denominator > 0
            AND (amount_min_numerator * amount_max_denominator)
                <= (amount_max_numerator * amount_min_denominator)
            AND ((unit_id IS NOT NULL AND package_type IS NULL)
                OR (unit_id IS NULL AND package_type IS NOT NULL)))
    ),
    CHECK (
        (package_type IS NULL
            AND package_size_numerator IS NULL
            AND package_size_denominator IS NULL
            AND package_size_unit_id IS NULL)
        OR
        (package_type IS NOT NULL
            AND ((package_size_numerator IS NULL
                    AND package_size_denominator IS NULL
                    AND package_size_unit_id IS NULL)
                OR (package_size_numerator IS NOT NULL
                    AND package_size_numerator > 0
                    AND package_size_denominator IS NOT NULL
                    AND package_size_denominator > 0
                    AND package_size_unit_id IS NOT NULL)))
    )
) STRICT;

CREATE TABLE recipe_instruction_sections (
    id INTEGER PRIMARY KEY,
    recipe_id INTEGER NOT NULL REFERENCES recipes(id) ON UPDATE RESTRICT ON DELETE CASCADE,
    name TEXT NOT NULL CHECK (trim(name) <> ''),
    position INTEGER NOT NULL CHECK (position >= 0),
    UNIQUE (recipe_id, position)
) STRICT;

CREATE TABLE recipe_steps (
    id INTEGER PRIMARY KEY,
    section_id INTEGER NOT NULL REFERENCES recipe_instruction_sections(id) ON UPDATE RESTRICT ON DELETE CASCADE,
    position INTEGER NOT NULL CHECK (position >= 0),
    instruction TEXT NOT NULL CHECK (trim(instruction) <> ''),
    UNIQUE (section_id, position)
) STRICT;

CREATE TABLE recipe_review_flags (
    id INTEGER PRIMARY KEY,
    recipe_id INTEGER NOT NULL REFERENCES recipes(id) ON UPDATE RESTRICT ON DELETE CASCADE,
    field_path TEXT NOT NULL CHECK (trim(field_path) <> ''),
    kind TEXT NOT NULL CHECK (kind IN ('backfilled', 'rewritten', 'conflict-resolved')),
    note TEXT NOT NULL CHECK (trim(note) <> ''),
    approved INTEGER NOT NULL DEFAULT 0 CHECK (approved IN (0, 1)),
    UNIQUE (recipe_id, field_path, kind)
) STRICT;

-- Every recipe enters through the draft/review lifecycle; no insert may bypass it.
-- +goose StatementBegin
CREATE TRIGGER recipes_start_as_draft
BEFORE INSERT ON recipes
WHEN NEW.status <> 'draft' OR NEW.verified_at IS NOT NULL
BEGIN SELECT RAISE(ABORT, 'new recipe must start as draft'); END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER protect_verified_recipe_update
BEFORE UPDATE ON recipes
WHEN OLD.status = 'verified' AND NEW.status = 'verified'
BEGIN SELECT RAISE(ABORT, 'return recipe to draft before editing'); END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER verify_recipe_before_status_update
BEFORE UPDATE OF status ON recipes
WHEN NEW.status = 'verified' AND OLD.status <> 'verified'
BEGIN
    SELECT CASE WHEN OLD.status <> 'reviewable'
        THEN RAISE(ABORT, 'recipe must be reviewable before verification') END;
    SELECT CASE WHEN NEW.verified_at IS NULL
        THEN RAISE(ABORT, 'verified recipe requires verified_at') END;
    SELECT CASE WHEN (SELECT count(*) FROM recipe_sources
                      WHERE recipe_id = NEW.id AND is_primary = 1) <> 1
        THEN RAISE(ABORT, 'verified recipe requires exactly one primary source') END;
    SELECT CASE WHEN NOT EXISTS (SELECT 1 FROM recipe_ingredient_sections
                                 WHERE recipe_id = NEW.id)
        THEN RAISE(ABORT, 'verified recipe requires an ingredient section') END;
    SELECT CASE WHEN NOT EXISTS (
            SELECT 1
            FROM recipe_ingredients ri
            JOIN recipe_ingredient_sections ris ON ris.id = ri.section_id
            WHERE ris.recipe_id = NEW.id)
        THEN RAISE(ABORT, 'verified recipe requires ingredients') END;
    SELECT CASE WHEN EXISTS (
            SELECT 1
            FROM recipe_ingredients ri
            JOIN recipe_ingredient_sections ris ON ris.id = ri.section_id
            WHERE ris.recipe_id = NEW.id AND ri.grocery_item_id IS NULL)
        THEN RAISE(ABORT, 'verified recipe has an unmapped ingredient') END;
    SELECT CASE WHEN NOT EXISTS (
            SELECT 1
            FROM recipe_steps rs
            JOIN recipe_instruction_sections rxs ON rxs.id = rs.section_id
            WHERE rxs.recipe_id = NEW.id)
        THEN RAISE(ABORT, 'verified recipe requires instructions') END;
    SELECT CASE WHEN EXISTS (SELECT 1 FROM recipe_review_flags
                             WHERE recipe_id = NEW.id AND approved = 0)
        THEN RAISE(ABORT, 'verified recipe has unapproved review flags') END;
END;
-- +goose StatementEnd

-- Verified recipes are immutable corpus truth until deliberately returned to draft.
-- +goose StatementBegin
CREATE TRIGGER protect_verified_recipe_sources_insert
BEFORE INSERT ON recipe_sources
WHEN (SELECT status FROM recipes WHERE id = NEW.recipe_id) = 'verified'
BEGIN SELECT RAISE(ABORT, 'return recipe to draft before editing sources'); END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER protect_verified_recipe_sources_update
BEFORE UPDATE ON recipe_sources
WHEN (SELECT status FROM recipes WHERE id = OLD.recipe_id) = 'verified'
BEGIN SELECT RAISE(ABORT, 'return recipe to draft before editing sources'); END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER protect_verified_recipe_sources_delete
BEFORE DELETE ON recipe_sources
WHEN (SELECT status FROM recipes WHERE id = OLD.recipe_id) = 'verified'
BEGIN SELECT RAISE(ABORT, 'return recipe to draft before editing sources'); END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER protect_verified_ingredient_sections_insert
BEFORE INSERT ON recipe_ingredient_sections
WHEN (SELECT status FROM recipes WHERE id = NEW.recipe_id) = 'verified'
BEGIN SELECT RAISE(ABORT, 'return recipe to draft before editing ingredients'); END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER protect_verified_ingredient_sections_update
BEFORE UPDATE ON recipe_ingredient_sections
WHEN (SELECT status FROM recipes WHERE id = OLD.recipe_id) = 'verified'
BEGIN SELECT RAISE(ABORT, 'return recipe to draft before editing ingredients'); END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER protect_verified_ingredient_sections_delete
BEFORE DELETE ON recipe_ingredient_sections
WHEN (SELECT status FROM recipes WHERE id = OLD.recipe_id) = 'verified'
BEGIN SELECT RAISE(ABORT, 'return recipe to draft before editing ingredients'); END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER protect_verified_ingredients_insert
BEFORE INSERT ON recipe_ingredients
WHEN (SELECT r.status FROM recipes r
      JOIN recipe_ingredient_sections s ON s.recipe_id = r.id
      WHERE s.id = NEW.section_id) = 'verified'
BEGIN SELECT RAISE(ABORT, 'return recipe to draft before editing ingredients'); END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER protect_verified_ingredients_update
BEFORE UPDATE ON recipe_ingredients
WHEN (SELECT r.status FROM recipes r
      JOIN recipe_ingredient_sections s ON s.recipe_id = r.id
      WHERE s.id = OLD.section_id) = 'verified'
BEGIN SELECT RAISE(ABORT, 'return recipe to draft before editing ingredients'); END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER protect_verified_ingredients_delete
BEFORE DELETE ON recipe_ingredients
WHEN (SELECT r.status FROM recipes r
      JOIN recipe_ingredient_sections s ON s.recipe_id = r.id
      WHERE s.id = OLD.section_id) = 'verified'
BEGIN SELECT RAISE(ABORT, 'return recipe to draft before editing ingredients'); END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER protect_verified_instruction_sections_insert
BEFORE INSERT ON recipe_instruction_sections
WHEN (SELECT status FROM recipes WHERE id = NEW.recipe_id) = 'verified'
BEGIN SELECT RAISE(ABORT, 'return recipe to draft before editing instructions'); END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER protect_verified_instruction_sections_update
BEFORE UPDATE ON recipe_instruction_sections
WHEN (SELECT status FROM recipes WHERE id = OLD.recipe_id) = 'verified'
BEGIN SELECT RAISE(ABORT, 'return recipe to draft before editing instructions'); END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER protect_verified_instruction_sections_delete
BEFORE DELETE ON recipe_instruction_sections
WHEN (SELECT status FROM recipes WHERE id = OLD.recipe_id) = 'verified'
BEGIN SELECT RAISE(ABORT, 'return recipe to draft before editing instructions'); END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER protect_verified_steps_insert
BEFORE INSERT ON recipe_steps
WHEN (SELECT r.status FROM recipes r
      JOIN recipe_instruction_sections s ON s.recipe_id = r.id
      WHERE s.id = NEW.section_id) = 'verified'
BEGIN SELECT RAISE(ABORT, 'return recipe to draft before editing instructions'); END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER protect_verified_steps_update
BEFORE UPDATE ON recipe_steps
WHEN (SELECT r.status FROM recipes r
      JOIN recipe_instruction_sections s ON s.recipe_id = r.id
      WHERE s.id = OLD.section_id) = 'verified'
BEGIN SELECT RAISE(ABORT, 'return recipe to draft before editing instructions'); END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER protect_verified_steps_delete
BEFORE DELETE ON recipe_steps
WHEN (SELECT r.status FROM recipes r
      JOIN recipe_instruction_sections s ON s.recipe_id = r.id
      WHERE s.id = OLD.section_id) = 'verified'
BEGIN SELECT RAISE(ABORT, 'return recipe to draft before editing instructions'); END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER protect_verified_review_flags_insert
BEFORE INSERT ON recipe_review_flags
WHEN (SELECT status FROM recipes WHERE id = NEW.recipe_id) = 'verified'
BEGIN SELECT RAISE(ABORT, 'return recipe to draft before adding review flags'); END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER protect_verified_review_flags_update
BEFORE UPDATE ON recipe_review_flags
WHEN (SELECT status FROM recipes WHERE id = OLD.recipe_id) = 'verified'
BEGIN SELECT RAISE(ABORT, 'return recipe to draft before editing review flags'); END;
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER protect_verified_review_flags_delete
BEFORE DELETE ON recipe_review_flags
WHEN (SELECT status FROM recipes WHERE id = OLD.recipe_id) = 'verified'
BEGIN SELECT RAISE(ABORT, 'return recipe to draft before editing review flags'); END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS protect_verified_review_flags_delete;
DROP TRIGGER IF EXISTS protect_verified_review_flags_update;
DROP TRIGGER IF EXISTS protect_verified_review_flags_insert;
DROP TRIGGER IF EXISTS protect_verified_steps_delete;
DROP TRIGGER IF EXISTS protect_verified_steps_update;
DROP TRIGGER IF EXISTS protect_verified_steps_insert;
DROP TRIGGER IF EXISTS protect_verified_instruction_sections_delete;
DROP TRIGGER IF EXISTS protect_verified_instruction_sections_update;
DROP TRIGGER IF EXISTS protect_verified_instruction_sections_insert;
DROP TRIGGER IF EXISTS protect_verified_ingredients_delete;
DROP TRIGGER IF EXISTS protect_verified_ingredients_update;
DROP TRIGGER IF EXISTS protect_verified_ingredients_insert;
DROP TRIGGER IF EXISTS protect_verified_ingredient_sections_delete;
DROP TRIGGER IF EXISTS protect_verified_ingredient_sections_update;
DROP TRIGGER IF EXISTS protect_verified_ingredient_sections_insert;
DROP TRIGGER IF EXISTS protect_verified_recipe_sources_delete;
DROP TRIGGER IF EXISTS protect_verified_recipe_sources_update;
DROP TRIGGER IF EXISTS protect_verified_recipe_sources_insert;
DROP TRIGGER IF EXISTS verify_recipe_before_status_update;
DROP TRIGGER IF EXISTS protect_verified_recipe_update;
DROP TRIGGER IF EXISTS recipes_start_as_draft;
DROP TABLE IF EXISTS recipe_review_flags;
DROP TABLE IF EXISTS recipe_steps;
DROP TABLE IF EXISTS recipe_instruction_sections;
DROP TABLE IF EXISTS recipe_ingredients;
DROP TABLE IF EXISTS recipe_ingredient_sections;
DROP INDEX IF EXISTS one_primary_source_per_recipe;
DROP TABLE IF EXISTS recipe_sources;
DROP TABLE IF EXISTS recipes;
DROP TABLE IF EXISTS grocery_items;
DROP TABLE IF EXISTS units;
DROP TABLE IF EXISTS store_sections;
