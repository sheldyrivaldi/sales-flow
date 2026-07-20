ALTER TABLE event
    DROP COLUMN IF EXISTS analysis_status,
    DROP COLUMN IF EXISTS analysis_error;
