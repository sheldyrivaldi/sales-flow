-- Peserta yang diundang ke event TIDAK perlu punya akun di aplikasi ini —
-- cukup alamat email lepas, jadi disimpan sebagai array teks, bukan relasi
-- ke tabel user. Lampiran menyusul pola yang sama (rundown, undangan, denah
-- booth) dengan metadata {name,url,mime,size} per berkas.
ALTER TABLE event
    ADD COLUMN participant_emails JSONB NOT NULL DEFAULT '[]',
    ADD COLUMN attachments        JSONB NOT NULL DEFAULT '[]';
