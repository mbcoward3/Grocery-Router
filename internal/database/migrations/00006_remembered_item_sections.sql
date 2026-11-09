-- +goose Up
-- Household memory for manual items the corpus does not know. When someone places
-- "Cottage cheese" in Dairy, the next manual "cottage cheese" goes to Dairy and is
-- offered as a suggestion. Known grocery items keep their reviewed corpus section.
CREATE TABLE remembered_item_sections (
    household_id UUID NOT NULL REFERENCES households (id) ON UPDATE RESTRICT ON DELETE CASCADE,
    name_key TEXT NOT NULL CHECK (name_key <> '' AND name_key = lower(name_key)),
    display_name TEXT NOT NULL CHECK (trim(display_name) <> ''),
    store_section_id BIGINT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT (
        to_char(clock_timestamp() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"')
    ),
    PRIMARY KEY (household_id, name_key),
    FOREIGN KEY (household_id, store_section_id) REFERENCES store_sections (household_id, id)
    ON UPDATE RESTRICT ON DELETE RESTRICT
);

-- +goose Down
DROP TABLE remembered_item_sections;
