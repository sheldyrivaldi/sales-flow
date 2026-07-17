CREATE TABLE hermes_tui_session (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES "user"(id),
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ended_at   TIMESTAMPTZ,
    remote_ip  TEXT
);

-- Audit metadata only (who/when/duration/IP) — never terminal content, by
-- design (see plan §Audit logging).
CREATE INDEX ON hermes_tui_session (user_id);
