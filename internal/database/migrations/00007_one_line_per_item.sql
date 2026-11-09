-- +goose Up
-- One checklist row per grocery item. Requirements that cannot be added exactly (cloves and
-- tablespoons, a jar and cups) stay separate quantities on the same row: the row's own columns
-- hold its first quantity and this table holds the second and later ones, in order.
CREATE TABLE shopping_line_quantities (
    household_id UUID NOT NULL,
    shopping_line_id BIGINT NOT NULL,
    position BIGINT NOT NULL CHECK (position >= 1),
    quantity_kind TEXT NOT NULL CHECK (quantity_kind IN ('exact', 'range')),
    amount_min_numerator BIGINT NOT NULL CHECK (amount_min_numerator >= 0),
    amount_min_denominator BIGINT NOT NULL CHECK (amount_min_denominator > 0),
    amount_max_numerator BIGINT,
    amount_max_denominator BIGINT,
    unit_id BIGINT REFERENCES units (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    package_type TEXT,
    package_size_numerator BIGINT,
    package_size_denominator BIGINT,
    package_size_unit_id BIGINT REFERENCES units (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    PRIMARY KEY (shopping_line_id, position),
    FOREIGN KEY (household_id, shopping_line_id) REFERENCES shopping_lines (household_id, id)
    ON UPDATE RESTRICT ON DELETE CASCADE,
    CHECK ((unit_id IS NOT NULL AND package_type IS NULL) OR (unit_id IS NULL AND package_type IS NOT NULL)),
    CHECK (
        (quantity_kind = 'exact' AND amount_max_numerator IS NULL AND amount_max_denominator IS NULL)
        OR (
            quantity_kind = 'range'
            AND amount_max_numerator IS NOT NULL AND amount_max_numerator >= 0
            AND amount_max_denominator IS NOT NULL AND amount_max_denominator > 0
            AND (amount_min_numerator * amount_max_denominator)
            <= (amount_max_numerator * amount_min_denominator)
        )
    ),
    CHECK (
        (
            package_type IS NULL AND package_size_numerator IS NULL
            AND package_size_denominator IS NULL AND package_size_unit_id IS NULL
        )
        OR (
            package_type IS NOT NULL
            AND (
                (
                    package_size_numerator IS NULL AND package_size_denominator IS NULL
                    AND package_size_unit_id IS NULL
                )
                OR (
                    package_size_numerator IS NOT NULL AND package_size_numerator > 0
                    AND package_size_denominator IS NOT NULL AND package_size_denominator > 0
                    AND package_size_unit_id IS NOT NULL
                )
            )
        )
    )
);

-- Lists built before this migration have one row per compatible quantity. The current week is
-- rebuilt in the new shape the next time it is read; past weeks stay as they were shown.
ALTER TABLE shopping_lists ADD COLUMN aggregation_version BIGINT NOT NULL DEFAULT 1
CHECK (aggregation_version >= 1);
ALTER TABLE shopping_lists ALTER COLUMN aggregation_version SET DEFAULT 2;

-- +goose Down
ALTER TABLE shopping_lists DROP COLUMN aggregation_version;
DROP TABLE shopping_line_quantities;
