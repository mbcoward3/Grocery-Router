-- +goose Up
-- A household member's save in the recipe editor is the household's approval
-- (RECIPE_LIBRARY_SPEC.md §3.3). verified_by records who saved it; bootstrap recipes keep NULL.
ALTER TABLE recipes
ADD COLUMN verified_by UUID REFERENCES app_users (id) ON UPDATE RESTRICT ON DELETE SET NULL;

-- Deleting a user sets verified_by to NULL on the recipes they approved. That UPDATE is the one
-- change to a verified recipe that does not need it returned to draft first.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION enforce_recipe_lifecycle() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.status <> 'draft' OR NEW.verified_at IS NOT NULL THEN
            RAISE EXCEPTION 'new recipe must start as draft';
        END IF;
        RETURN NEW;
    END IF;

    IF OLD.status = 'verified' AND NEW.status = 'verified' THEN
        IF NEW.verified_by IS NULL AND OLD.verified_by IS NOT NULL
            AND (to_jsonb(NEW) - 'verified_by') = (to_jsonb(OLD) - 'verified_by') THEN
            RETURN NEW;
        END IF;
        RAISE EXCEPTION 'return recipe to draft before editing';
    END IF;

    IF NEW.status = 'verified' AND OLD.status <> 'verified' THEN
        IF OLD.status <> 'reviewable' THEN
            RAISE EXCEPTION 'recipe must be reviewable before verification';
        ELSIF NEW.verified_at IS NULL THEN
            RAISE EXCEPTION 'verified recipe requires verified_at';
        ELSIF (SELECT count(*) FROM recipe_sources WHERE recipe_id = NEW.id AND is_primary = 1) <> 1 THEN
            RAISE EXCEPTION 'verified recipe requires exactly one primary source';
        ELSIF NOT EXISTS (SELECT 1 FROM recipe_ingredient_sections WHERE recipe_id = NEW.id) THEN
            RAISE EXCEPTION 'verified recipe requires an ingredient section';
        ELSIF NOT EXISTS (
            SELECT 1 FROM recipe_ingredients ri
            JOIN recipe_ingredient_sections ris ON ris.id = ri.section_id
            WHERE ris.recipe_id = NEW.id
        ) THEN
            RAISE EXCEPTION 'verified recipe requires ingredients';
        ELSIF EXISTS (
            SELECT 1 FROM recipe_ingredients ri
            JOIN recipe_ingredient_sections ris ON ris.id = ri.section_id
            WHERE ris.recipe_id = NEW.id AND ri.grocery_item_id IS NULL
        ) THEN
            RAISE EXCEPTION 'verified recipe has an unmapped ingredient';
        ELSIF NOT EXISTS (
            SELECT 1 FROM recipe_steps rs
            JOIN recipe_instruction_sections rxs ON rxs.id = rs.section_id
            WHERE rxs.recipe_id = NEW.id
        ) THEN
            RAISE EXCEPTION 'verified recipe requires instructions';
        ELSIF EXISTS (SELECT 1 FROM recipe_review_flags WHERE recipe_id = NEW.id AND approved = 0) THEN
            RAISE EXCEPTION 'verified recipe has unapproved review flags';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION enforce_recipe_lifecycle() RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.status <> 'draft' OR NEW.verified_at IS NOT NULL THEN
            RAISE EXCEPTION 'new recipe must start as draft';
        END IF;
        RETURN NEW;
    END IF;

    IF OLD.status = 'verified' AND NEW.status = 'verified' THEN
        RAISE EXCEPTION 'return recipe to draft before editing';
    END IF;

    IF NEW.status = 'verified' AND OLD.status <> 'verified' THEN
        IF OLD.status <> 'reviewable' THEN
            RAISE EXCEPTION 'recipe must be reviewable before verification';
        ELSIF NEW.verified_at IS NULL THEN
            RAISE EXCEPTION 'verified recipe requires verified_at';
        ELSIF (SELECT count(*) FROM recipe_sources WHERE recipe_id = NEW.id AND is_primary = 1) <> 1 THEN
            RAISE EXCEPTION 'verified recipe requires exactly one primary source';
        ELSIF NOT EXISTS (SELECT 1 FROM recipe_ingredient_sections WHERE recipe_id = NEW.id) THEN
            RAISE EXCEPTION 'verified recipe requires an ingredient section';
        ELSIF NOT EXISTS (
            SELECT 1 FROM recipe_ingredients ri
            JOIN recipe_ingredient_sections ris ON ris.id = ri.section_id
            WHERE ris.recipe_id = NEW.id
        ) THEN
            RAISE EXCEPTION 'verified recipe requires ingredients';
        ELSIF EXISTS (
            SELECT 1 FROM recipe_ingredients ri
            JOIN recipe_ingredient_sections ris ON ris.id = ri.section_id
            WHERE ris.recipe_id = NEW.id AND ri.grocery_item_id IS NULL
        ) THEN
            RAISE EXCEPTION 'verified recipe has an unmapped ingredient';
        ELSIF NOT EXISTS (
            SELECT 1 FROM recipe_steps rs
            JOIN recipe_instruction_sections rxs ON rxs.id = rs.section_id
            WHERE rxs.recipe_id = NEW.id
        ) THEN
            RAISE EXCEPTION 'verified recipe requires instructions';
        ELSIF EXISTS (SELECT 1 FROM recipe_review_flags WHERE recipe_id = NEW.id AND approved = 0) THEN
            RAISE EXCEPTION 'verified recipe has unapproved review flags';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

ALTER TABLE recipes DROP COLUMN verified_by;
