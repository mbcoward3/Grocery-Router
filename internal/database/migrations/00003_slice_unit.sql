-- +goose Up
INSERT INTO units (key, name, symbol, dimension, to_base_numerator, to_base_denominator)
VALUES ('slice', 'slice', 'slices', 'count', 1, 1);

-- +goose Down
DELETE FROM units WHERE key = 'slice';
