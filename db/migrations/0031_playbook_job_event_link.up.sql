-- Tautan eksplisit playbook_job -> event. Sebelumnya keterkaitan hanya ditebak
-- dari kecocokan judul ("Playbook Event: {nama}"), yang rapuh: ganti nama event
-- memutus tautan, dan dua event bernama sama saling tumpang tindih.
--
-- Aturan bisnis: SATU event hanya boleh punya SATU playbook yang tertaut. Saat
-- di-generate ulang, playbook lama dilepas (event_id di-NULL-kan) lalu yang baru
-- mengambil tautannya. Indeks unik parsial menjaga invarian itu di level DB.
ALTER TABLE playbook_job
    ADD COLUMN event_id UUID REFERENCES event(id) ON DELETE SET NULL;

CREATE UNIQUE INDEX uq_playbook_job_event
    ON playbook_job (event_id)
    WHERE event_id IS NOT NULL;
