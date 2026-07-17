ALTER TABLE company_profile ADD COLUMN products JSONB NOT NULL DEFAULT '[]';
ALTER TABLE company_profile ADD COLUMN vision TEXT;
ALTER TABLE company_profile ADD COLUMN mission TEXT;
