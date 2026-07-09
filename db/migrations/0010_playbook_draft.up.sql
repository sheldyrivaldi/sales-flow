CREATE TABLE playbook_draft (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type TEXT        NOT NULL CHECK (target_type IN ('tender', 'prospect')),
    target_id   UUID        NOT NULL,
    title       TEXT,
    content     JSONB       NOT NULL,
    source      TEXT        NOT NULL DEFAULT 'mcp',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON playbook_draft(target_type, target_id);
