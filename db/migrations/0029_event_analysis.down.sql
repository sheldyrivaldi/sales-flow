ALTER TABLE event
    DROP COLUMN IF EXISTS analysis,
    DROP COLUMN IF EXISTS analyzed_at;
