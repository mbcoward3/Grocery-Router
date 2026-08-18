-- +goose Up
INSERT INTO units (key, name, symbol, dimension, to_base_numerator, to_base_denominator)
VALUES ('leaf', 'leaf', 'leaves', 'count', 1, 1);

-- +goose Down
DELETE FROM units WHERE key = 'leaf';
