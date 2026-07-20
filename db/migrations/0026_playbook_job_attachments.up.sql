-- 0026: lampiran & riwayat revisi pada playbook_job.
-- attachment_url: file konteks saat pembuatan job (bisa dibuka seperti chat).
-- revisions: [{"instruction","attachment_name","attachment_url","at"}] — riwayat
-- prompt revisi beserta lampirannya.
ALTER TABLE playbook_job
  ADD COLUMN attachment_url TEXT,
  ADD COLUMN revisions JSONB NOT NULL DEFAULT '[]';
