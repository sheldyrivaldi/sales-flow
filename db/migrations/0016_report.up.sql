CREATE TABLE report (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    report_type  TEXT        NOT NULL CHECK (report_type IN ('daily_digest', 'weekly_pipeline', 'per_opportunity')),
    title        TEXT        NOT NULL,
    period_start TIMESTAMPTZ NOT NULL,
    period_end   TIMESTAMPTZ NOT NULL,
    content      TEXT        NOT NULL,
    model        TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON report(report_type, created_at DESC);
