ALTER TABLE feedback_form_submission
  DROP COLUMN IF EXISTS respondent_division;

ALTER TABLE feedback_form
  DROP COLUMN IF EXISTS created_by_name,
  DROP COLUMN IF EXISTS created_by,
  DROP COLUMN IF EXISTS language;
