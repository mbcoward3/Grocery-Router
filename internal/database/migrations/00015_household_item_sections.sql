-- +goose Up
-- Where a household's store keeps a known grocery item, when that differs from the reviewed
-- corpus section (D053). Generated lists, manual adds, and suggestions use it instead.
CREATE TABLE household_item_sections (
    household_id UUID NOT NULL REFERENCES households (id) ON UPDATE RESTRICT ON DELETE CASCADE,
    grocery_item_id BIGINT NOT NULL,
    store_section_id BIGINT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT (
        to_char(clock_timestamp() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"')
    ),
    PRIMARY KEY (household_id, grocery_item_id),
    FOREIGN KEY (household_id, grocery_item_id) REFERENCES grocery_items (household_id, id)
    ON UPDATE RESTRICT ON DELETE CASCADE,
    FOREIGN KEY (household_id, store_section_id) REFERENCES store_sections (household_id, id)
    ON UPDATE RESTRICT ON DELETE RESTRICT
);

-- +goose Down
DROP TABLE household_item_sections;
