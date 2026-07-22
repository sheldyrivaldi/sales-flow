-- 0034: metadata tambahan untuk Feedback Client.
--  - language: bahasa form (id/en), menentai bahasa pertanyaan & label default.
--  - created_by / created_by_name: pembuat form (ditampilkan di daftar).
--  - respondent_division: divisi pengisi (opsional, sejajar nama & email).
-- Kolom min_label/max_label rating disimpan di dalam questions JSONB (tak perlu
-- kolom baru).

ALTER TABLE feedback_form
  ADD COLUMN IF NOT EXISTS language TEXT NOT NULL DEFAULT 'id'
    CHECK (language IN ('id','en')),
  ADD COLUMN IF NOT EXISTS created_by UUID REFERENCES "user"(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS created_by_name TEXT;

ALTER TABLE feedback_form_submission
  ADD COLUMN IF NOT EXISTS respondent_division TEXT;
