DROP INDEX IF EXISTS uq_playbook_job_event;
ALTER TABLE playbook_job DROP COLUMN IF EXISTS event_id;
