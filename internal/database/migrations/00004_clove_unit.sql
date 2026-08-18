-- +goose Up
INSERT INTO units (key, name, symbol, dimension, to_base_numerator, to_base_denominator)
VALUES ('clove', 'clove', 'cloves', 'count', 1, 1);

-- +goose Down
DELETE FROM units WHERE key = 'clove';
