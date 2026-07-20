-- 0024: lampiran pada pesan chat — file yang diunggah user (atau dikembalikan
-- AI) disimpan di disk dan direferensikan via URL agar bisa dibuka/diunduh
-- dari riwayat chat.
ALTER TABLE message
  ADD COLUMN attachment_url  TEXT,
  ADD COLUMN attachment_name TEXT,
  ADD COLUMN attachment_mime TEXT;
