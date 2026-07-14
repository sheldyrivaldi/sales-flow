CREATE TABLE telemetry_event (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    event       TEXT        NOT NULL,
    props       JSONB,
    actor       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON telemetry_event(event, created_at DESC);
