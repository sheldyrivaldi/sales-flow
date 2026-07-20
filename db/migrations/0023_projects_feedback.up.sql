-- 0023: Dokumen pendukung tender di Company Profile + modul Proyek Berjalan
-- (ongoing) + modul Pasca-Proyek (feedback client via link publik).

ALTER TABLE company_profile
  ADD COLUMN support_documents JSONB NOT NULL DEFAULT '[]';

-- Proyek berjalan: hasil menang tender/prospek (atau input manual) yang
-- sedang dikerjakan dan dipantau kesehatannya oleh tim sales.
CREATE TABLE project (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  client_name TEXT,
  contract_value NUMERIC,
  currency TEXT NOT NULL DEFAULT 'IDR',
  start_date DATE,
  end_date DATE,
  status TEXT NOT NULL DEFAULT 'ON_TRACK'
    CHECK (status IN ('ON_TRACK','AT_RISK','DELAYED','COMPLETED')),
  progress INT NOT NULL DEFAULT 0 CHECK (progress BETWEEN 0 AND 100),
  description TEXT,
  -- milestones: [{"title": "...", "due_date": "YYYY-MM-DD", "done": false}]
  milestones JSONB NOT NULL DEFAULT '[]',
  -- activities (catatan check-in): [{"date": "...", "note": "..."}]
  activities JSONB NOT NULL DEFAULT '[]',
  source_type TEXT,
  source_id UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Permintaan feedback pasca-proyek — token unik menjadi link publik yang
-- dibagikan ke client (tanpa login).
CREATE TABLE feedback_request (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  token TEXT NOT NULL UNIQUE,
  project_name TEXT NOT NULL,
  client_name TEXT,
  project_id UUID REFERENCES project(id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Satu respon per permintaan (UNIQUE request_id) — client mengisi sekali.
CREATE TABLE feedback_response (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  request_id UUID NOT NULL UNIQUE REFERENCES feedback_request(id) ON DELETE CASCADE,
  overall_rating INT NOT NULL CHECK (overall_rating BETWEEN 1 AND 5),
  quality_rating INT CHECK (quality_rating BETWEEN 1 AND 5),
  communication_rating INT CHECK (communication_rating BETWEEN 1 AND 5),
  timeliness_rating INT CHECK (timeliness_rating BETWEEN 1 AND 5),
  nps INT CHECK (nps BETWEEN 0 AND 10),
  comment TEXT,
  respondent_name TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
