CREATE TABLE company_profile (
    id                 UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    company_name       TEXT          NOT NULL,
    one_liner          TEXT,
    service_categories JSONB         NOT NULL DEFAULT '[]',
    tech_stack         JSONB         NOT NULL DEFAULT '[]',
    source_doc_refs    JSONB         NOT NULL DEFAULT '[]',
    version            INT           NOT NULL DEFAULT 1,
    is_current         BOOLEAN       NOT NULL DEFAULT true,
    created_at         TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX ON company_profile (is_current) WHERE is_current;

CREATE TABLE target_criteria (
    id                 UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id         UUID          NOT NULL REFERENCES company_profile(id) ON DELETE CASCADE,
    countries          JSONB         NOT NULL DEFAULT '[]',
    industries         JSONB         NOT NULL DEFAULT '[]',
    value_min          NUMERIC(18,2) CHECK (value_min IS NULL OR value_min >= 0),
    value_ideal        NUMERIC(18,2) CHECK (value_ideal IS NULL OR value_ideal >= 0),
    value_max          NUMERIC(18,2) CHECK (value_max IS NULL OR value_max >= 0),
    currency           TEXT          NOT NULL DEFAULT 'IDR',
    deadline_min_days  INT           CHECK (deadline_min_days IS NULL OR deadline_min_days >= 0),
    procurement_types  JSONB         NOT NULL DEFAULT '[]',
    created_at         TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX ON target_criteria (profile_id);

CREATE TABLE nogo_rule (
    id                 UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id         UUID          NOT NULL REFERENCES company_profile(id) ON DELETE CASCADE,
    preset_flags       JSONB         NOT NULL DEFAULT '[]',
    custom             JSONB         NOT NULL DEFAULT '[]',
    created_at         TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX ON nogo_rule (profile_id);

CREATE TABLE keyword_set (
    id                 UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id         UUID          NOT NULL REFERENCES company_profile(id) ON DELETE CASCADE,
    category           TEXT,
    keywords           JSONB         NOT NULL DEFAULT '[]',
    negative_keywords  JSONB         NOT NULL DEFAULT '[]',
    language           TEXT          NOT NULL DEFAULT 'id',
    created_at         TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE INDEX ON keyword_set (profile_id);

CREATE TABLE source (
    id                 UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    name               TEXT          NOT NULL,
    url                TEXT          NOT NULL,
    country            TEXT,
    access             TEXT          NOT NULL DEFAULT 'publik'
                                     CHECK (access IN ('publik','login','manual')),
    legal_note         TEXT,
    enabled            BOOLEAN       NOT NULL DEFAULT false,
    priority           INT           NOT NULL DEFAULT 0,
    preset_key         TEXT,
    created_at         TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE INDEX ON source (enabled);
CREATE UNIQUE INDEX ON source (preset_key) WHERE preset_key IS NOT NULL;
