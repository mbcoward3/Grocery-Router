-- +goose Up
-- A week is open until it is closed; the open week is the household's current week and closed
-- weeks are History. auto_close_at is when an open week closes itself (NULL: only by hand).
ALTER TABLE weeks
ADD COLUMN opened_at TIMESTAMPTZ,
ADD COLUMN auto_close_at TIMESTAMPTZ,
ADD COLUMN closed_at TIMESTAMPTZ;

-- Existing weeks opened at local midnight of their Sunday in the owning household's zone and
-- close at local midnight seven calendar days later, so daylight-saving weeks keep local
-- midnight. A timestamp without zone AT TIME ZONE reads it as that zone's wall-clock time.
UPDATE weeks AS w SET
    opened_at = CAST(CAST(w.starts_on AS DATE) AS TIMESTAMP) AT TIME ZONE h.timezone,
    auto_close_at = CAST(CAST(w.starts_on AS DATE) + 7 AS TIMESTAMP) AT TIME ZONE h.timezone
FROM households AS h
WHERE h.id = w.household_id;

UPDATE weeks SET closed_at = auto_close_at
WHERE auto_close_at <= now();

-- A week generated for a later Sunday before a time-zone change can leave two weeks unexpired.
-- Only the newest stays open; an older one closes now, as if it had been ended by hand.
UPDATE weeks AS w SET closed_at = greatest(w.opened_at, now())
WHERE
    w.closed_at IS NULL
    AND EXISTS (
        SELECT 1 FROM weeks AS newer
        WHERE
            newer.household_id = w.household_id
            AND newer.closed_at IS NULL
            AND newer.starts_on > w.starts_on
    );

ALTER TABLE weeks ALTER COLUMN opened_at SET NOT NULL;
ALTER TABLE weeks ADD CONSTRAINT weeks_closed_after_opened
CHECK (closed_at IS NULL OR closed_at >= opened_at);
CREATE UNIQUE INDEX weeks_one_open_per_household ON weeks (household_id) WHERE closed_at IS NULL;

-- +goose Down
DROP INDEX weeks_one_open_per_household;
ALTER TABLE weeks DROP CONSTRAINT weeks_closed_after_opened;
ALTER TABLE weeks DROP COLUMN closed_at;
ALTER TABLE weeks DROP COLUMN auto_close_at;
ALTER TABLE weeks DROP COLUMN opened_at;
