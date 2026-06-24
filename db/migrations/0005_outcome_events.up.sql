CREATE TABLE outcome_event (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type TEXT        NOT NULL CHECK (target_type IN ('tender', 'prospect')),
    target_id   UUID        NOT NULL,
    result      TEXT        NOT NULL CHECK (result IN ('WON', 'LOST')),
    notes       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON outcome_event(target_type, target_id);
