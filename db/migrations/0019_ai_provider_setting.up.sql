CREATE TABLE ai_provider_setting (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    provider          TEXT        NOT NULL CHECK (provider IN ('openai','openrouter')),
    model             TEXT        NOT NULL,
    base_url          TEXT,
    api_key_encrypted TEXT        NOT NULL,
    enabled_toolsets  JSONB,
    is_active         BOOLEAN     NOT NULL DEFAULT false,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Only one config may be active at a time — the row Go re-pushes to the
-- bridge via hermes.Configure (single-workspace deployment, no multi-tenant).
CREATE UNIQUE INDEX ON ai_provider_setting (is_active) WHERE is_active = true;
