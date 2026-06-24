CREATE TABLE event (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT        NOT NULL,
    type       TEXT        NOT NULL CHECK (type IN ('EXPO','CONFERENCE','SEMINAR','WORKSHOP','NETWORKING','OTHER')),
    event_date TIMESTAMPTZ,
    location   TEXT,
    organizer  TEXT,
    notes      TEXT,
    status     TEXT        NOT NULL DEFAULT 'PLANNED' CHECK (status IN ('PLANNED','ATTENDED','CANCELLED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON event(status);
CREATE INDEX ON event(type);
CREATE INDEX ON event(event_date);
