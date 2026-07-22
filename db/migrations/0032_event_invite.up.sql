-- Undangan event terjadwal. Sebelumnya "Kirim Undangan" hanya membuka draft
-- mailto di klien; sekarang undangan dikirim DARI server pada waktu terjadwal
-- (hari H, H-1, H-3, H-7 jam 07:00, atau tanggal+jam custom) ke seluruh daftar
-- peserta. Satu event boleh punya banyak jadwal (mis. reminder H-7 dan H-1).
CREATE TABLE event_invite (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id     UUID NOT NULL REFERENCES event(id) ON DELETE CASCADE,
    subject      TEXT NOT NULL,
    body         TEXT NOT NULL,
    -- Pengirim = user yang memicu; dipasang di header From/Reply-To agar
    -- undangan tampak datang dari yang mengundang.
    sender_name  TEXT,
    sender_email TEXT,
    recipients   JSONB NOT NULL DEFAULT '[]',
    scheduled_at TIMESTAMPTZ NOT NULL,
    status       TEXT NOT NULL DEFAULT 'pending'
                 CHECK (status IN ('pending','sent','failed','cancelled')),
    sent_at      TIMESTAMPTZ,
    error        TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Scheduler menyapu (status='pending' AND scheduled_at <= now()) tiap menit.
CREATE INDEX idx_event_invite_due ON event_invite (status, scheduled_at);
CREATE INDEX idx_event_invite_event ON event_invite (event_id);
