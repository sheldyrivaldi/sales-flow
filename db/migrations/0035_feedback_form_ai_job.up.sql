-- 0035: dukung saran AI ASINKRON pada Feedback Client. Generate kuesioner
-- lewat AI bisa lama (banyak lampiran/konteks) — bila diminta menunggu di
-- request HTTP yang sama, pengguna kehilangan progres saat pindah halaman.
-- Job disimpan di baris form itu sendiri (ai_job, sementara secara bisnis —
-- dibuang begitu user memilih/menambahkan atau membatalkan), memakai model
-- titip-tugas yang sama dengan playbook/analisa event (lihat
-- internal/service/feedback_form_service.go).

ALTER TABLE feedback_form
  DROP CONSTRAINT IF EXISTS feedback_form_status_check;
ALTER TABLE feedback_form
  ADD CONSTRAINT feedback_form_status_check
  CHECK (status IN ('draft','published','closed','processing_ai','need_clarification'));

ALTER TABLE feedback_form
  ADD COLUMN IF NOT EXISTS ai_job JSONB;
