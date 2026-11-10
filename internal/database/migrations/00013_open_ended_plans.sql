-- +goose Up
-- A plan (stored as a week) stays open until someone starts a new one or reopens a past one;
-- nothing closes it by the calendar (D051). The rollover settings and the automatic close time
-- go. An open week whose close time had already passed simply stays open.
ALTER TABLE weeks DROP COLUMN auto_close_at;
ALTER TABLE households
DROP COLUMN week_start_day,
DROP COLUMN rollover_minutes,
DROP COLUMN rollover_mode;

-- +goose Down
ALTER TABLE households
ADD COLUMN week_start_day SMALLINT NOT NULL DEFAULT 0
CHECK (week_start_day BETWEEN 0 AND 6),
ADD COLUMN rollover_minutes SMALLINT NOT NULL DEFAULT 0
CHECK (rollover_minutes BETWEEN 0 AND 1439),
ADD COLUMN rollover_mode TEXT NOT NULL DEFAULT 'automatic'
CHECK (rollover_mode IN ('automatic', 'manual'));
-- Restored weeks have no close time, so they close only by hand.
ALTER TABLE weeks ADD COLUMN auto_close_at TIMESTAMPTZ;
