-- 0025: playbook_job — riwayat generate playbook custom yang berjalan async.
-- Setiap trigger generate langsung membuat baris (status in_progress) lalu
-- di-update oleh worker background menjadi success/failed; refine memakai
-- status updating. content menyimpan PlaybookContent hasil akhir.
CREATE TABLE playbook_job (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  title TEXT NOT NULL,
  prompt TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'in_progress'
    CHECK (status IN ('in_progress','updating','success','failed')),
  content JSONB,
  error_message TEXT,
  attachment_name TEXT,
  source TEXT NOT NULL DEFAULT 'custom',   -- 'custom' | 'event'
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_playbook_job_created ON playbook_job (created_at DESC);
