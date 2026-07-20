ALTER TABLE message
  DROP COLUMN IF EXISTS attachment_url,
  DROP COLUMN IF EXISTS attachment_name,
  DROP COLUMN IF EXISTS attachment_mime;
