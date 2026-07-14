CREATE TABLE discovery_run (
    id               UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    started_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    finished_at      TIMESTAMPTZ,
    source_ids       JSONB         NOT NULL DEFAULT '[]',
    status           TEXT          NOT NULL DEFAULT 'pending'
                                    CHECK (status IN ('pending','running','success','failed')),
    found_count      INT           NOT NULL DEFAULT 0,
    summary          TEXT,
    correlation_key  TEXT,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX ON discovery_run (correlation_key) WHERE correlation_key IS NOT NULL;
CREATE INDEX ON discovery_run (status);
