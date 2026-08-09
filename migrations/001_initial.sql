CREATE TABLE IF NOT EXISTS weekly_plans (
    week_start DATE PRIMARY KEY,
    document TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS shopping_ticks (
    week_start DATE NOT NULL REFERENCES weekly_plans (week_start) ON DELETE CASCADE,
    item_key TEXT NOT NULL,
    checked BOOLEAN NOT NULL DEFAULT false,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (week_start, item_key)
);

CREATE TABLE IF NOT EXISTS plan_events (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    week_start DATE NOT NULL,
    event JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS plan_events_week_start_idx
    ON plan_events (week_start, created_at);
