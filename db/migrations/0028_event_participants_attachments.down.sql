ALTER TABLE event
    DROP COLUMN IF EXISTS participant_emails,
    DROP COLUMN IF EXISTS attachments;
