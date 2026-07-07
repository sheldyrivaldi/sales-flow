CREATE TABLE conversation (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id     UUID        NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    title             TEXT        NOT NULL DEFAULT '',
    session_key       TEXT        NOT NULL,
    hermes_session_id TEXT        NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON conversation(owner_user_id);

CREATE TABLE message (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID        NOT NULL REFERENCES conversation(id) ON DELETE CASCADE,
    role            TEXT        NOT NULL CHECK (role IN ('user', 'assistant', 'system', 'tool')),
    content         TEXT        NOT NULL DEFAULT '',
    tool_calls      JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON message(conversation_id);
