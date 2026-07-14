CREATE TABLE playbook (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type TEXT        NOT NULL CHECK (target_type IN ('tender', 'prospect')),
    target_id   UUID        NOT NULL,
    version     INT         NOT NULL,
    content     JSONB       NOT NULL,
    model       TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (target_type, target_id, version)
);

CREATE INDEX ON playbook(target_type, target_id, version DESC);
