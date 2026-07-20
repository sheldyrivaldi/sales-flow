-- Analisa AI adalah nilai utama menu Event, jadi hasilnya WAJIB bertahan:
-- sebelumnya ia hanya nilai balik mutation dan hilang begitu halaman dimuat
-- ulang, sehingga riset yang memakan menit-menit terbuang percuma.
ALTER TABLE event
    ADD COLUMN analysis    JSONB,
    ADD COLUMN analyzed_at TIMESTAMPTZ;
