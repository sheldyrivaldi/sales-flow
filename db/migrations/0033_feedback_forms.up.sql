-- 0033: Feedback Client dinamis (form builder ala Google Form) — menggantikan
-- form kaku berskema-tetap 0023 (feedback_request/feedback_response, kini
-- dorman untuk kompatibilitas link lama). Pertanyaan & jawaban disimpan JSONB
-- mengikuti pola project.milestones/activities.

CREATE TABLE feedback_form (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title TEXT NOT NULL,
  description TEXT,
  -- slug = bagian akhir link publik /form/:slug (unik, custom, aman-URL).
  slug TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL DEFAULT 'draft'
    CHECK (status IN ('draft','published','closed')),
  collect_email BOOLEAN NOT NULL DEFAULT true,
  -- questions: [{"id","type":"rating|text|choice|nps","label","description",
  --             "required":bool,"scale":int,"options":["..."],"multiple":bool}]
  questions JSONB NOT NULL DEFAULT '[]',
  project_id UUID REFERENCES project(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Satu baris per pengisian (banyak respon per form, beda dari 0023 yang
-- membatasi satu respon per link).
CREATE TABLE feedback_form_submission (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  form_id UUID NOT NULL REFERENCES feedback_form(id) ON DELETE CASCADE,
  respondent_email TEXT,
  respondent_name TEXT,
  -- answers: [{"question_id","text","rating":int,"choice":["..."]}]
  answers JSONB NOT NULL DEFAULT '[]',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_feedback_form_submission_form ON feedback_form_submission(form_id);
