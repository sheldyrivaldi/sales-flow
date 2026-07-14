CREATE TABLE prospect_score (
    id                  UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type         TEXT             NOT NULL CHECK (target_type IN ('tender', 'prospect')),
    target_id           UUID             NOT NULL,
    fit_score           INT              NOT NULL,
    recommended_action  TEXT             NOT NULL,
    confidence          DOUBLE PRECISION,
    reasoning           TEXT,
    evidence            JSONB,
    risk_flags          JSONB,
    model               TEXT,
    created_at          TIMESTAMPTZ      NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ      NOT NULL DEFAULT now()
);

CREATE INDEX ON prospect_score(target_type, target_id, created_at DESC);
