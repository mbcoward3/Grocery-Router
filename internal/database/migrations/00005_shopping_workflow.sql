-- +goose Up
ALTER TABLE store_sections ADD COLUMN display_position BIGINT NOT NULL DEFAULT -1;

-- +goose StatementBegin
CREATE FUNCTION default_section_position(section_key TEXT) RETURNS BIGINT
LANGUAGE sql IMMUTABLE AS $$
    SELECT CASE section_key
        WHEN 'produce' THEN 100 WHEN 'bakery' THEN 200 WHEN 'meat' THEN 300
        WHEN 'seafood' THEN 400 WHEN 'dairy' THEN 500 WHEN 'pantry' THEN 600
        WHEN 'baking' THEN 700 WHEN 'spices' THEN 800 WHEN 'condiments' THEN 900
        WHEN 'wine-beer' THEN 1000 WHEN 'wine-and-beer' THEN 1000
        WHEN 'frozen' THEN 1100 WHEN 'other' THEN 1300
        WHEN 'non-shopping' THEN 1400 ELSE 1200 END;
$$;
-- +goose StatementEnd

UPDATE store_sections SET display_position = default_section_position(key);

-- +goose StatementBegin
CREATE FUNCTION position_new_store_section() RETURNS TRIGGER LANGUAGE plpgsql AS $$
DECLARE
    target_position BIGINT;
BEGIN
    PERFORM id FROM households WHERE id = NEW.household_id FOR UPDATE;
    -- Bootstrap explicitly supplies the default position. Ordinary new sections append
    -- before Other, including households that have never customized their order.
    IF NEW.display_position >= 0 THEN
        RETURN NEW;
    END IF;
    SELECT display_position INTO target_position FROM store_sections
    WHERE household_id = NEW.household_id AND key = 'other';
    IF target_position IS NULL THEN
        SELECT coalesce(max(display_position) + 1, 0) INTO target_position
        FROM store_sections WHERE household_id = NEW.household_id;
    END IF;
    UPDATE store_sections SET display_position = display_position + 1
    WHERE household_id = NEW.household_id AND display_position >= target_position;
    NEW.display_position := target_position;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd
CREATE TRIGGER position_new_store_section BEFORE INSERT ON store_sections
FOR EACH ROW EXECUTE FUNCTION position_new_store_section();

-- Replace the original anonymous origin check without depending on PostgreSQL's generated name.
-- +goose StatementBegin
DO $$
DECLARE origin_check TEXT;
BEGIN
    SELECT conname INTO origin_check FROM pg_constraint
    WHERE conrelid = 'shopping_lines'::regclass AND contype = 'c'
        AND pg_get_constraintdef(oid) LIKE '%origin%generated%grocery_item_id%';
    EXECUTE format('ALTER TABLE shopping_lines DROP CONSTRAINT %I', origin_check);
END;
$$;
-- +goose StatementEnd
ALTER TABLE shopping_lines ADD CONSTRAINT shopping_lines_manual_identity_check
CHECK ((origin = 'generated' AND grocery_item_id IS NOT NULL AND aggregation_key IS NOT NULL)
    OR (origin = 'manual' AND aggregation_key IS NULL));

-- +goose Down
-- Fail rather than silently erase recognized manual-item identities on rollback.
ALTER TABLE shopping_lines DROP CONSTRAINT shopping_lines_manual_identity_check;
ALTER TABLE shopping_lines ADD CHECK (
    (origin = 'generated' AND grocery_item_id IS NOT NULL AND aggregation_key IS NOT NULL)
    OR (origin = 'manual' AND grocery_item_id IS NULL AND aggregation_key IS NULL)
);
DROP TRIGGER position_new_store_section ON store_sections;
DROP FUNCTION position_new_store_section();
DROP FUNCTION default_section_position(TEXT);
ALTER TABLE store_sections DROP COLUMN display_position;
