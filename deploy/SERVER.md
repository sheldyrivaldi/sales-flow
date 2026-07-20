# Deploy ke Server

Server menjalankan **tiga image milikmu** (api, web, hermes-bridge). Postgres dan
Hermes **sudah ada** di server (eksternal) — tidak dibuat ulang oleh compose.

Berkas yang dipakai:
- `docker-compose.server.yml` — hanya mengelola api, web, hermes-bridge.
- `.env.server` — nilai rahasia + alamat layanan eksternal (kamu buat dari
  `.env.server.example`, TIDAK di-commit).

---

## Jawaban singkat: cukup push code, atau perlu env baru?

**Perlu env baru** — push code saja tidak cukup. Alasannya: server ini beda dari
dev lokal (DB & Hermes eksternal), jadi butuh berkas `.env.server` +
`docker-compose.server.yml` yang baru ini. Kode-nya sendiri sudah siap.

Tiga hal yang WAJIB kamu isi sebelum jalan, sisanya sudah punya default aman:

1. `DATABASE_URL` → Postgres eksternal server.
2. `HERMES_HOME_HOST_PATH` → folder home Hermes existing di server (INI yang
   membuat semua menu AI jalan otomatis tanpa login ulang).
3. `HERMES_TUI_BASE_URL` → dashboard Hermes existing (untuk halaman Setting AI Agent).

Plus beberapa rahasia acak (`JWT_SECRET`, `API_SERVER_KEY`, `CRON_TRIGGER_SECRET`,
`WORKSPACE_SESSION_KEY`, `SALES_MCP_TOKEN`) — sekali buat, simpan.

---

## Langkah

```bash
cd deploy
cp .env.server.example .env.server
# edit .env.server — isi DATABASE_URL, HERMES_HOME_HOST_PATH, HERMES_TUI_BASE_URL,
# dan semua rahasia acak. Buat rahasia dengan: openssl rand -base64 32

docker compose -f docker-compose.server.yml --env-file .env.server up -d --build
```

Update berikutnya (setelah pull code baru):

```bash
git pull
docker compose -f docker-compose.server.yml --env-file .env.server up -d --build
```

---

## Menemukan `HERMES_HOME_HOST_PATH` (langkah paling penting)

Ini folder tempat Hermes existing menyimpan login provider (OAuth). Bridge
menumpang folder ini → engine memakai login yang sudah ada.

- **Hermes existing jalan sebagai proses biasa (bukan Docker):**
  ```bash
  echo $HERMES_HOME        # kalau di-set
  ls -d ~/.hermes          # default
  ```
  Isi `HERMES_HOME_HOST_PATH` dengan path itu.

- **Hermes existing jalan sebagai container Docker:**
  ```bash
  docker inspect <nama-container-hermes> --format '{{json .Mounts}}' | jq
  ```
  Kalau home-nya berupa **named volume**, jangan pakai `HERMES_HOME_HOST_PATH`.
  Sebagai gantinya, di `docker-compose.server.yml` ganti baris mount bridge jadi:
  ```yaml
  volumes:
    - <nama-volume-hermes>:/root/.hermes
  ```
  dan tambahkan di bawah:
  ```yaml
  volumes:
    uploads:
    <nama-volume-hermes>:
      external: true
  ```

> **Catatan versi:** bridge memakai `hermes-agent` (pip). Kalau Hermes existing
> versinya jauh berbeda, format home dir bisa tak seragam. Token login biasanya
> kompatibel; kalau engine tetap gagal auth, jalankan ulang login OAuth lewat
> dashboard sekali lagi.

---

## Yang membuat "auto tanpa login" bekerja — ringkas

- **Menu AI (chat, playbook, analisa)** → lewat **bridge**, diautentikasi
  `API_SERVER_KEY` (Bearer, otomatis). Login provider diambil dari home dir yang
  di-share. **Tidak ada form login.** Ini yang menggerakkan semua menu.
- **Dashboard (halaman Setting AI Agent)** → panel konfigurasi. Kalau Hermes
  existing pakai login form, kamu login manual di iframe saat mau mengatur. Ini
  TIDAK memengaruhi jalannya engine.
- Ingin dashboard juga tanpa login? Jalankan Hermes existing mode `--insecure`
  (token) dengan `HERMES_DASHBOARD_SESSION_TOKEN` yang sama seperti `.env.server`
  — proxy sudah menyuntik token itu. **Hanya aman bila dashboard TIDAK diekspos
  publik** (di balik proxy internal), karena mode token membocorkan token ke
  siapa pun yang bisa menghubunginya langsung.

---

## Cek setelah up

```bash
# tiga container jalan
docker compose -f docker-compose.server.yml ps

# api boot bersih (migrasi + http server)
docker compose -f docker-compose.server.yml logs api | tail -20

# engine sehat (dari dalam jaringan compose)
docker compose -f docker-compose.server.yml exec api wget -qO- http://hermes-bridge:8642/health
```

Lalu buka web di `http://<server>:${WEB_PORT}`, login admin (SEED_ADMIN_*),
dan coba satu fitur AI (mis. generate playbook) untuk memastikan engine +
login provider tersambung.
