-- Judul playbook adalah milik user, BUKAN turunan prompt dan bukan karangan
-- AI. Saat user mengisi judul di form create, tandai user_titled agar callback
-- hasil generate tidak pernah menimpanya. Bila dibiarkan kosong, judul boleh
-- diisi AI (perilaku lama tetap jalan untuk baris yang sudah ada).
ALTER TABLE playbook_job
    ADD COLUMN user_titled BOOLEAN NOT NULL DEFAULT false;
