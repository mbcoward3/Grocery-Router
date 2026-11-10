-- +goose Up
-- The household's IANA time zone decides when its week begins. Existing households keep UTC,
-- the zone the server used before, until an operator sets theirs.
ALTER TABLE households ADD COLUMN timezone TEXT NOT NULL DEFAULT 'UTC' CHECK (trim(timezone) <> '');

-- +goose Down
ALTER TABLE households DROP COLUMN timezone;
