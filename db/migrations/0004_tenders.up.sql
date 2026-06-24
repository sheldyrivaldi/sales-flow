CREATE TABLE tender (
    id                       UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    title                    TEXT          NOT NULL,
    buyer_name               TEXT,
    buyer_country            TEXT,
    buyer_industry           TEXT,
    value_estimate           NUMERIC(18,2) CHECK (value_estimate IS NULL OR value_estimate >= 0),
    currency                 TEXT          NOT NULL DEFAULT 'IDR',
    published_date           TIMESTAMPTZ,
    submission_deadline      TIMESTAMPTZ,
    source_name              TEXT,
    source_url               TEXT,
    service_category         TEXT,
    scope_summary            TEXT,
    eligibility_requirements TEXT,
    technical_requirements   TEXT,
    status                   TEXT          NOT NULL DEFAULT 'IDENTIFIED'
                                           CHECK (status IN ('IDENTIFIED','QUALIFYING','BIDDING','SUBMITTED','WON','LOST')),
    fit_score                INT           CHECK (fit_score IS NULL OR (fit_score BETWEEN 0 AND 100)),
    recommended_action       TEXT          CHECK (recommended_action IS NULL OR
                                                   recommended_action IN ('PURSUE','REVIEW','WATCHLIST','REJECT','NEED_PARTNER')),
    risk_flags               JSONB,
    reasoning_summary        TEXT,
    dedup_key                TEXT,
    origin                   TEXT          NOT NULL DEFAULT 'manual'
                                           CHECK (origin IN ('manual','discovery')),
    created_at               TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE INDEX ON tender(status);
CREATE INDEX ON tender(submission_deadline);
CREATE INDEX ON tender(recommended_action);
CREATE INDEX ON tender(origin);
CREATE UNIQUE INDEX ON tender(dedup_key) WHERE dedup_key IS NOT NULL;
