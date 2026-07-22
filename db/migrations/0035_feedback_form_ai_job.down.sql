ALTER TABLE feedback_form
  DROP COLUMN IF EXISTS ai_job;
ALTER TABLE feedback_form
  DROP CONSTRAINT IF EXISTS feedback_form_status_check;
ALTER TABLE feedback_form
  ADD CONSTRAINT feedback_form_status_check
  CHECK (status IN ('draft','published','closed'));
