CREATE TABLE prospect (
    id            UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT          NOT NULL,
    company       TEXT,
    contact_info  TEXT,
    source_type   TEXT          NOT NULL DEFAULT 'manual'
                                CHECK (source_type IN ('manual','event','tender')),
    source_id     UUID,
    stage         TEXT          NOT NULL DEFAULT 'NEW'
                                CHECK (stage IN ('NEW','QUALIFIED','ENGAGED','PROPOSAL','WON','LOST')),
    est_value     NUMERIC(18,2) CHECK (est_value IS NULL OR est_value >= 0),
    owner_user_id UUID          REFERENCES "user"(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE INDEX ON prospect(stage);
CREATE INDEX ON prospect(owner_user_id);
CREATE UNIQUE INDEX ON prospect(source_type, source_id) WHERE source_id IS NOT NULL;
