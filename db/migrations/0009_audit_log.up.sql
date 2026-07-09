CREATE TABLE audit_log (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    actor       TEXT        NOT NULL,
    action      TEXT        NOT NULL,
    target_type TEXT,
    target_id   UUID,
    payload     JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON audit_log(action);
CREATE INDEX ON audit_log(target_type, target_id);
