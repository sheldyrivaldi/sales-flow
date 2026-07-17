DROP TABLE IF EXISTS scoring_config CASCADE;

ALTER TABLE source DROP COLUMN IF EXISTS data_types;
ALTER TABLE source DROP COLUMN IF EXISTS frequency;

ALTER TABLE target_criteria DROP COLUMN IF EXISTS decision_maker_roles;
ALTER TABLE target_criteria DROP COLUMN IF EXISTS onsite_limit_note;
ALTER TABLE target_criteria DROP COLUMN IF EXISTS work_model;
ALTER TABLE target_criteria DROP COLUMN IF EXISTS document_languages;
ALTER TABLE target_criteria DROP COLUMN IF EXISTS buyer_size_note;

ALTER TABLE company_profile DROP COLUMN IF EXISTS portfolio_refs;
