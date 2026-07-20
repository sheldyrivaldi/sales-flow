ALTER TABLE playbook_job
  DROP COLUMN IF EXISTS attachment_url,
  DROP COLUMN IF EXISTS revisions;
