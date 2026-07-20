-- Analisa berjalan ASINKRON seperti generate playbook: app menitipkan tugas
-- lalu Hermes melapor balik. Status dipakai UI untuk mengunci event selama
-- analisa berjalan (tidak bisa diedit, ditambah lampiran, atau dianalisa
-- ulang) sehingga hasilnya tidak dihitung dari data yang berubah di tengah.
ALTER TABLE event
    ADD COLUMN analysis_status TEXT NOT NULL DEFAULT 'idle'
        CHECK (analysis_status IN ('idle','running','success','failed')),
    ADD COLUMN analysis_error  TEXT;
