-- +goose Up
INSERT INTO units (key, name, symbol, dimension, to_base_numerator, to_base_denominator)
VALUES
    ('bunch', 'bunch', 'bunches', 'count', 1, 1),
    ('sprig', 'sprig', 'sprigs', 'count', 1, 1);

-- +goose Down
DELETE FROM units WHERE key IN ('bunch', 'sprig');
