-- +goose Up
-- Exact conversion ratios. Volume is based on US teaspoons; mass is based on ounces.
INSERT INTO units (key, name, symbol, dimension, to_base_numerator, to_base_denominator) VALUES
    ('each', 'each', 'each', 'count', 1, 1),
    ('tsp', 'teaspoon', 'tsp', 'volume', 1, 1),
    ('tbsp', 'tablespoon', 'tbsp', 'volume', 3, 1),
    ('floz', 'fluid ounce', 'fl oz', 'volume', 6, 1),
    ('cup', 'cup', 'cup', 'volume', 48, 1),
    ('pint', 'pint', 'pt', 'volume', 96, 1),
    ('quart', 'quart', 'qt', 'volume', 192, 1),
    ('gallon', 'gallon', 'gal', 'volume', 768, 1),
    ('ml', 'milliliter', 'mL', 'volume', 32000000, 157725491),
    ('l', 'liter', 'L', 'volume', 32000000000, 157725491),
    ('oz', 'ounce', 'oz', 'mass', 1, 1),
    ('lb', 'pound', 'lb', 'mass', 16, 1),
    ('g', 'gram', 'g', 'mass', 1600000, 45359237),
    ('kg', 'kilogram', 'kg', 'mass', 1600000000, 45359237);

-- +goose Down
DELETE FROM units WHERE key IN (
    'each', 'tsp', 'tbsp', 'floz', 'cup', 'pint', 'quart', 'gallon',
    'ml', 'l', 'oz', 'lb', 'g', 'kg'
);
