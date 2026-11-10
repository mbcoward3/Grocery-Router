-- +goose Up
-- When a household's week rolls over: the local day (0 = Sunday, as Go's time.Weekday) and time
-- of day, in minutes after midnight, and whether the week closes itself then (automatic) or
-- stays open until someone starts the next one (manual). The defaults are the earlier fixed
-- Sunday 12:00 AM automatic rollover.
ALTER TABLE households
ADD COLUMN week_start_day SMALLINT NOT NULL DEFAULT 0
CHECK (week_start_day BETWEEN 0 AND 6),
ADD COLUMN rollover_minutes SMALLINT NOT NULL DEFAULT 0
CHECK (rollover_minutes BETWEEN 0 AND 1439),
ADD COLUMN rollover_mode TEXT NOT NULL DEFAULT 'automatic'
CHECK (rollover_mode IN ('automatic', 'manual'));

-- A week now starts on the household's configured day, or on any day in manual mode, and two
-- weeks may share a start date (a week closed early and the next one opened the same day).
-- The Sunday check was declared inline without a name in 00001.
-- +goose StatementBegin
DO $$
DECLARE sunday_check TEXT;
BEGIN
    SELECT conname INTO sunday_check FROM pg_constraint
    WHERE conrelid = 'weeks'::regclass AND contype = 'c'
        AND pg_get_constraintdef(oid) ILIKE '%dow%';
    IF sunday_check IS NULL THEN
        RAISE EXCEPTION 'weeks Sunday check not found';
    END IF;
    EXECUTE format('ALTER TABLE weeks DROP CONSTRAINT %I', sunday_check);
END;
$$;
-- +goose StatementEnd
ALTER TABLE weeks DROP CONSTRAINT weeks_household_start_key;
CREATE INDEX weeks_household_start_idx ON weeks (household_id, starts_on);

-- +goose Down
-- Fail rather than silently delete weeks that do not start on a Sunday or share a start date.
DROP INDEX weeks_household_start_idx;
ALTER TABLE weeks ADD CONSTRAINT weeks_household_start_key UNIQUE (household_id, starts_on);
ALTER TABLE weeks ADD CHECK (EXTRACT(DOW FROM to_date(starts_on, 'YYYY-MM-DD')) = 0);
ALTER TABLE households DROP COLUMN rollover_mode;
ALTER TABLE households DROP COLUMN rollover_minutes;
ALTER TABLE households DROP COLUMN week_start_day;
