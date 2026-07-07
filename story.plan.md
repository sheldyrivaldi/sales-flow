# story.plan.md — SalesPilot (Layer 2: STORY)

> **Induk:** [epic.plan.md](./epic.plan.md) · **Sumber:** [PRD.md](./PRD.md) v1.3 · [Design.md](./Design.md) v1.3.
> **Layer:** Lapis tengah. Tiap **Story** memecah satu **Epic** jadi unit yang bisa diselesaikan & diuji. Detail langkah mikro ada di [task.plan.md](./task.plan.md).
> **Warisan konvensi:** semua keputusan teknis, stack, struktur repo, koreksi v1.3 (**tanpa `organization_id`**, tanpa self-signup), dan Definition of Done **diwarisi dari epic.plan.md §1** dan tidak diulang di sini.

**Format story:**
> **ST-NN.s — Judul** `[Prioritas · Estimasi S/M/L]`
> *As a [persona], I want [...] so that [...].*
> **AC:** kriteria penerimaan terukur.
> **Teknis:** file/endpoint/komponen kunci.
> **Dep:** story/epic prasyarat.

Estimasi: **S** ≈ ≤0.5 hari · **M** ≈ 0.5–1.5 hari · **L** ≈ 2–4 hari (untuk satu engineer + model).

---

## EP-00 — Fondasi Proyek, Monorepo & DevOps

### ✅ ST-00.1 — Inisialisasi monorepo & struktur folder `[P0 · S]`
*As a developer, I want struktur repo & module Go siap so that semua kode punya tempat yang jelas.*
- **AC:** struktur folder sesuai epic §1 terbentuk; `go mod init` sukses; `.gitignore`, `.editorconfig` ada; placeholder package kompil.
- **Teknis:** `go.mod` (module `salespilot`), folder `apps/`, `internal/*`, `db/migrations`, `deploy/`.
- **Dep:** —

### ✅ ST-00.2 — Echo bootstrap + `/healthz` + config loader `[P0 · M]`
*As a developer, I want API server minimal yang bisa start so that ada fondasi semua handler.*
- **AC:** `go run ./apps/api` start di `:8080`; `GET /healthz` → `200 {"status":"ok"}`; config dibaca dari env (`.env`), gagal-cepat bila env wajib hilang.
- **Teknis:** `apps/api/main.go`, `internal/config/config.go` (struct + loader), `internal/http/router.go` (Echo + middleware logger/recover/CORS).
- **Dep:** ST-00.1.

### ✅ ST-00.3 — Vite + TS app shell kosong `[P0 · S]`
*As a developer, I want frontend skeleton so that UI bisa dikembangkan.*
- **AC:** `npm run dev` di `apps/web` jalan; halaman placeholder render; dev proxy `/api` → `:8080`; `npm run build` & `tsc --noEmit` hijau.
- **Teknis:** `apps/web/` (Vite React-TS), `vite.config.ts` (proxy), `src/main.tsx`, `src/App.tsx`.
- **Dep:** ST-00.1.

### ✅ ST-00.4 — Postgres + GORM + golang-migrate `[P0 · M]`
*As a developer, I want koneksi DB & sistem migrasi so that schema bisa berevolusi terkontrol.*
- **AC:** API connect Postgres saat boot (retry); `make migrate-up/down` (atau script) jalan; migrasi awal `0001_init.sql` (kosong/extension) ter-apply; healthz cek DB.
- **Teknis:** `internal/repository/db.go` (GORM open + pool), `db/migrations/0001_init.up.sql`/`down.sql`, script migrasi.
- **Dep:** ST-00.2.

### ✅ ST-00.5 — docker-compose + env example + README `[P0 · M]`
*As a developer, I want one-command run so that seluruh sistem (incl. Hermes) bisa dinyalakan.*
- **AC:** `docker compose -f deploy/docker-compose.yml up` menyalakan postgres+api+web+hermes-gateway; `.env.example` lengkap; `README.md` punya run guide Docker & native Windows.
- **Teknis:** `deploy/docker-compose.yml`, `deploy/hermes/config.yaml.example`, `deploy/hermes/.env.example`, `.env.example`, `README.md`.
- **Dep:** ST-00.2, ST-00.3, ST-00.4.

### ✅ ST-00.6 — Tooling lint/format/test `[P0 · S]`
*As a developer, I want gate kualitas so that kode konsisten & aman di-refactor.*
- **AC:** `golangci-lint run` & `go test ./...` hijau; `eslint`/`prettier`/`tsc` hijau; satu script agregat `make check` / `npm run check`.
- **Teknis:** `.golangci.yml`, `apps/web/.eslintrc`, `apps/web/.prettierrc`, Makefile/scripts.
- **Dep:** ST-00.1.

---

## EP-01 — Anti-Corruption Layer & Jembatan Hermes

### ST-01.1 — `Client` interface + tipe domain Hermes `[P0 · M]`
*As a developer, I want kontrak sempit ke Hermes so that perubahan API terisolasi.*
- **AC:** interface `Client{ Chat, ChatStream, GenerateJSON, Health }` + tipe `ChatRequest/ChatResponse/Chunk/Capabilities/SessionKey` terdefinisi; tidak ada `import` modul internal Hermes.
- **Teknis:** `internal/hermes/client.go` (interface + tipe), `internal/hermes/doc.go`.
- **Dep:** ST-00.2.

### ST-01.2 — Impl `/v1/chat/completions` (stream & non-stream) `[P0 · L]`
*As the backend, I want memanggil Hermes chat so that fitur chat & orchestrasi bisa jalan.*
- **AC:** `Chat` (non-stream) & `ChatStream` (SSE → channel `Chunk`) bekerja terhadap Hermes nyata; header `Authorization: Bearer`, `X-Hermes-Session-Key`, `X-Hermes-Session-Id` terkirim; error & timeout tertangani.
- **Teknis:** `internal/hermes/chat.go`, SSE parser, context cancel.
- **Dep:** ST-01.1.

### ✅ ST-01.3 — Impl `GenerateJSON` (output terstruktur) `[P0 · M]`
*As the backend, I want generasi JSON terstruktur so that scoring/playbook/report andal.*
- **AC:** `GenerateJSON(ctx, prompt, schema, sk)` mengembalikan `json.RawMessage` valid sesuai schema (pakai `/v1/responses` atau chat + JSON mode); retry pada output non-JSON.
- **Teknis:** `internal/hermes/generate.go`, validasi schema.
- **Dep:** ST-01.1.

### ✅ ST-01.4 — Health guard + degrade graceful `[P0 · M]`
*As the system, I want tahu status Hermes so that CRUD tetap jalan saat AI down.*
- **AC:** `Health()` panggil `/health` + `/v1/capabilities`; startup log status & fitur; bila Hermes mati, fitur AI mengembalikan error ramah, CRUD tidak terpengaruh.
- **Teknis:** `internal/hermes/health.go`, startup hook di `main.go`.
- **Dep:** ST-01.1.

### ✅ ST-01.5 — Contract tests + pin versi + config `[P0 · M]`
*As a maintainer, I want jaminan anti-breaking-change so that upgrade Hermes aman.*
- **AC:** `go test ./internal/hermes/... -run Contract` memverifikasi bentuk chat stream, responses, capabilities; versi Hermes di-pin di compose; semua config dari env.
- **Teknis:** `internal/hermes/contract_test.go`, env `HERMES_BASE_URL`/`API_SERVER_KEY`/`WORKSPACE_SESSION_KEY`.
- **Dep:** ST-01.2, ST-01.3, ST-01.4.

### ✅ ST-01.6 — `hermes-bridge` (jembatan Python ke Hermes `AIAgent`) `[P0 · L]`
*As the backend, I want service Python yang membungkus Hermes `AIAgent` & mengekspos HTTP `/v1` so that ACL Go bicara seperti OpenAI tanpa tahu detail Hermes, dan upgrade major Hermes hanya menyentuh bridge.*
- **AC:** `hermes-bridge` (FastAPI) hidup; `GET /health` ok; `POST /v1/chat/completions` (non-stream & SSE MVP blocking-then-send) memetakan `messages`→`conversation_history`+`user_message` ke `AIAgent.run_conversation`; auth Bearer `API_SERVER_KEY`; `GET /v1/capabilities` lapor versi+model; `POST /v1/responses` skeleton untuk `GenerateJSON`; `POST /admin/config` override provider/model/key runtime (default env `OPENAI_API_KEY`/`OPENROUTER_API_KEY`); error provider → JSON ramah, worker tak crash; instance `AIAgent` baru per request (tidak thread-safe).
- **Teknis:** `services/hermes-bridge/` (`pyproject.toml` pin `hermes-agent`, `app/main.py`, `config.py`, `auth.py`, `agent_factory.py`, `routes/{chat,health,responses,admin}.py`, `Dockerfile`); wiring `deploy/docker-compose.yml` (service `hermes-bridge` ganti `hermes-gateway`); `internal/hermes.Configure` (additive ke interface `Client`). Detail: [hermes-bridge.plan.md](./hermes-bridge.plan.md).
- **Dep:** ST-01.1, ST-01.2 (kontrak `/v1` yang ditiru bridge).

---

## EP-02 — Design System & Application Shell

### ✅ ST-02.1 — Tailwind config + design tokens `[P0 · M]`
*As a frontend dev, I want token terpusat so that UI konsisten dengan Design.md.*
- **AC:** palet (§2.1: indigo/violet/emerald/amber/rose/sky + neutral/surface/border), tipografi Inter (§2.2), spacing/radius/shadow (§2.3), score color scale & recommended-action color map ter-set sebagai token Tailwind.
- **Teknis:** `apps/web/tailwind.config.ts`, `src/styles/tokens.css`, import font Inter.
- **Dep:** ST-00.3.

### ✅ ST-02.2 — Komponen dasar (form & atom) `[P0 · L]`
*As a frontend dev, I want komponen input so that form fitur cepat dibuat.*
- **AC:** Button (primary/secondary/ghost/danger; sm/md/lg; ikon; loading), Input/Textarea/Select/DatePicker/Combobox, Chip/Tag input (preset+tambah), Toggle/Switch, Badge/Pill — semua punya state §2.4 (default/hover/active/focus/disabled/loading/error/selected).
- **Teknis:** `apps/web/src/components/ui/*`.
- **Dep:** ST-02.1.

### ✅ ST-02.3 — Komponen dasar (struktur & feedback) `[P0 · L]`
*As a frontend dev, I want komponen layout/feedback so that layar konsisten.*
- **AC:** Card, Table (sortable/sticky/pagination/kebab), Tabs, Breadcrumb, Avatar, Tooltip, Toast, Modal, Drawer (slide-over), Skeleton, Empty state, Confirmation dialog — render + state.
- **Teknis:** `apps/web/src/components/ui/*`.
- **Dep:** ST-02.1.

### ✅ ST-02.4 — Komponen AI & khusus `[P0 · L]`
*As a frontend dev, I want komponen AI so that elemen AI menonjol & konsisten.*
- **AC:** Score ring/gauge (warna ikut skala), Stat card, AI panel/callout (aksen violet + sparkles + "Lihat alasan"), Streaming text, File dropzone (PDF), Stepper, Risk-flag chip (amber/rose ⚠), Kanban primitives.
- **Teknis:** `apps/web/src/components/ui/*`, ikon lucide.
- **Dep:** ST-02.1.

### ✅ ST-02.5 — Application Shell & navigasi `[P0 · M]`
*As a user, I want navigasi konsisten so that semua fitur mudah dijangkau.*
- **AC:** sidebar collapsible (urutan §3.1: Dashboard, Penemuan AI[badge], Tenders, Events, Prospects, Playbooks, Reports, Chat[badge AI], — Otak Agent, Settings), topbar (breadcrumb/judul, search ⌘K placeholder, +New kontekstual, bell, avatar), floating "Tanya AI", routing semua halaman (placeholder dulu).
- **Teknis:** `apps/web/src/layout/AppShell.tsx`, `Sidebar.tsx`, `Topbar.tsx`, `src/routes.tsx`.
- **Dep:** ST-02.2, ST-02.3.

### ✅ ST-02.6 — Util format lokal & i18n & a11y `[P0 · M]`
*As a user, I want format Indonesia so that data terbaca alami.*
- **AC:** `formatRupiah` (`Rp 2.500.000.000`, singkat `Rp 2,5 M`/`Rp 300 jt`), `formatTanggal` (`24 Jun 2026`), relative time ("2 jam lalu"), id-ID number; string label terpusat; baseline a11y (focus ring, kontras AA, live-region helper).
- **Teknis:** `apps/web/src/lib/format.ts`, `src/lib/i18n.ts`, `src/lib/a11y.ts`; unit test format.
- **Dep:** ST-02.1.

---

## EP-03 — Auth, RBAC & User Management

### ST-03.1 — Entity user + migrasi + seed admin `[P0 · M]`
*As an admin, I want akun awal so that sistem bisa diakses pertama kali.*
- **AC:** tabel `user(id,email,password_hash,name,role,active,created_at,updated_at)`; role enum `SALES/OPS/MANAGER/ADMIN`; seed 1 admin dari env (`SEED_ADMIN_EMAIL/PASSWORD`); email unik.
- **Teknis:** `db/migrations/000X_users.up.sql`, `internal/domain/user.go`, `internal/repository/user_repo.go`, seed.
- **Dep:** ST-00.4.

### ST-03.2 — Login + JWT + password hash `[P0 · M]`
*As an internal user, I want login so that hanya yang berwenang masuk.*
- **AC:** `POST /api/auth/login` (email+password) → JWT (claims: sub, role, exp); bcrypt verify; refresh sederhana `/api/auth/refresh`; gagal → 401 ramah; **tidak ada endpoint register publik**.
- **Teknis:** `internal/auth/jwt.go`, `internal/auth/password.go`, `internal/http/handlers/auth_handler.go`.
- **Dep:** ST-03.1.

### ST-03.3 — Auth + RBAC middleware `[P0 · M]`
*As the system, I want gate per-role so that permission §3.1 ditegakkan.*
- **AC:** middleware verifikasi JWT → context user; helper `RequireRole(...)` / `RequireCapability(cap)` memetakan matriks §3.1; akses terlarang → 403; diuji per-role.
- **Teknis:** `internal/auth/middleware.go`, `internal/auth/rbac.go` (peta capability→roles), test.
- **Dep:** ST-03.2.

### ST-03.4 — Admin user management `[P0 · M]`
*As an admin, I want kelola akun so that karyawan bisa diberi/dicabut akses.*
- **AC:** `GET/POST /api/users`, `PATCH /api/users/:id` (role/active), `POST /api/users/:id/reset-password` — semua **ADMIN only**; tidak bisa menonaktifkan diri sendiri terakhir.
- **Teknis:** `internal/http/handlers/user_handler.go`, DTO.
- **Dep:** ST-03.3.

### ST-03.5 — FE Login + auth store + guard `[P0 · M]`
*As a user, I want halaman login so that bisa masuk aplikasi.*
- **AC:** Login (Design §4.1): card gradient, email/password(toggle), "Masuk", error inline, loading; **tanpa link Daftar** + catatan "Akun dikelola Admin internal"; auth store (token persist), route guard, UI sembunyikan aksi sesuai role.
- **Teknis:** `apps/web/src/pages/Login.tsx`, `src/store/auth.ts`, `src/lib/api.ts` (interceptor token), `src/components/RequireAuth.tsx`.
- **Dep:** ST-02.5, ST-03.2.

---

## EP-04 — Chat Assistant (Hermes streaming)

### ST-04.1 — Entity conversation/message + migrasi `[P0 · S]`
*As the system, I want simpan chat so that histori bisa dibuka ulang.*
- **AC:** tabel `conversation(id,title,hermes_session_id,session_key,owner_user_id,created_at,updated_at)`, `message(id,conversation_id,role,content,tool_calls jsonb,created_at)`.
- **Teknis:** migrasi, `internal/domain/{conversation,message}.go`, repo.
- **Dep:** ST-00.4.

### ST-04.2 — Create conversation + session key `[P0 · M]`
*As a user, I want mulai percakapan so that chat punya konteks tersimpan.*
- **AC:** `POST /api/conversations` membuat conversation dengan `session_key = WORKSPACE_SESSION_KEY` (stabil) + `hermes_session_id` per conversation; judul auto dari pesan pertama.
- **Teknis:** `internal/http/handlers/chat_handler.go`, service.
- **Dep:** ST-04.1, ST-01.1.

### ST-04.3 — Chat SSE relay `[P0 · L]`
*As a user, I want jawaban streaming so that respons terasa cepat.*
- **AC:** `POST /api/conversations/:id/chat` menerima pesan, forward ke `hermes.ChatStream`, relay SSE ke browser, simpan message user & assistant (incl. `tool_calls`); Stop membatalkan stream; Hermes down → event error ramah.
- **Teknis:** `chat_handler.go` (SSE writer + flush), context cancel.
- **Dep:** ST-04.2, ST-01.2.

### ST-04.4 — History list/get `[P0 · S]`
*As a user, I want lihat percakapan lama so that bisa lanjut konteks.*
- **AC:** `GET /api/conversations` (list, terbaru dulu), `GET /api/conversations/:id` (+messages); hanya milik user (atau sesuai role).
- **Teknis:** handler + repo query.
- **Dep:** ST-04.1.

### ST-04.5 — FE Chat UI `[P0 · L]`
*As a user, I want antarmuka chat so that bisa tanya AI dengan nyaman.*
- **AC:** Design §4.12: list percakapan + search, bubble user (kanan)/assistant (kiri, violet, sparkles, markdown), **tool-call chip** ("🔧 Membaca data tender…" → "✓ N dibaca", expandable), streaming + Stop, input auto-grow Enter-kirim, suggested chips, context chip bila dari detail, footer cue belajar.
- **Teknis:** `apps/web/src/pages/Chat.tsx`, `src/lib/sse.ts` (fetch-stream parser), komponen bubble/toolchip.
- **Dep:** ST-02.4, ST-04.3.

### ST-04.6 — FE Floating "Tanya AI" + degrade `[P0 · M]`
*As a user, I want chat dari mana saja so that konteks selalu siap.*
- **AC:** floating button → slide-over chat dari layar mana pun; passing context chip dari detail tender/prospect; error banner "Agent tidak tersedia" saat Hermes down (CRUD tetap jalan).
- **Teknis:** `src/components/AskAIDrawer.tsx`, integrasi shell.
- **Dep:** ST-04.5.

---

## EP-05 — Tender Management

### ✅ ST-05.1 — Entity tender + migrasi + validasi `[P0 · M]`
*As the system, I want model tender lengkap so that data discovery & manual tertampung.*
- **AC:** tabel `tender` dengan semua field §10 (incl. `service_category`, `scope_summary`, `eligibility_requirements`, `technical_requirements`, `fit_score`, `recommended_action` enum, `risk_flags jsonb`, `reasoning_summary`, `dedup_key`, `origin` enum manual/discovery, `status` enum); validasi value ≥ 0, enum.
- **Teknis:** migrasi `0004_tenders`, `internal/domain/tender.go`, `repository/tender_repo.go`, unit test enum.
- **Dep:** ST-00.4. ✓ **DONE**

### ✅ ST-05.2 — CRUD endpoints + filter/pagination `[P0 · M]`
*As a user, I want kelola tender so that pipeline terjaga.*
- **AC:** `GET/POST /api/tenders`, `GET/PUT/DELETE /api/tenders/:id`; buat dengan min `title`; filter status/buyer/deadline/recommended_action/origin + search; pagination; hapus → 204.
- **Teknis:** `internal/http/handlers/tender_handler.go`, `dto/tender.go`, `service/tender_service.go`, query builder.
- **Dep:** ST-05.1, ST-03.3. ✓ **DONE**

### ✅ ST-05.3 — Status transition + outcome `[P0 · M]`
*As a user, I want ubah status & tandai hasil so that progres & pembelajaran tercatat.*
- **AC:** `PATCH /api/tenders/:id/status` (validasi transisi `IDENTIFIED→QUALIFYING→BIDDING→SUBMITTED→WON/LOST`); `POST /api/tenders/:id/outcome` (WON/LOST + notes) membuat `outcome_event` & emit hook learning (no-op). Tabel `outcome_event` dibuat sekarang (TK-16.1.1 superseded).
- **Teknis:** `service/tender_service.go`, `domain/outcome.go`, `repository/outcome_repo.go`, migrasi `0005_outcome_events`.
- **Dep:** ST-05.2. ✓ **DONE**

### ✅ ST-05.4 — Promote dari discovery `[P0 · S]` — DONE
*As a user, I want promote tender temuan so that masuk pipeline aktif.*
- **AC:** tender `origin=discovery` bisa di-"promote" (set status `QUALIFYING`/keluar inbox) tanpa kehilangan field AI; endpoint/aksi tersedia (dipakai EP-12).
- **Teknis:** field/flag inbox vs pipeline; service `Promote`.
- **Dep:** ST-05.2.

### ✅ ST-05.5 — FE Tender List `[P0 · M]` — DONE
*As a user, I want daftar tender so that cepat menilai prioritas.*
- **AC:** Design §4.4: filter (status/buyer/deadline/rekomendasi/origin/search), kolom Judul/Buyer/Nilai/Deadline(badge)/Status(pill)/Fit Score(mini ring)/Rekomendasi(badge)/Origin(✨)/kebab; empty/loading.
- **Teknis:** `apps/web/src/pages/tenders/TenderList.tsx`, query hooks.
- **Dep:** ST-02.3, ST-05.2.

### ✅ ST-05.6 — FE Tender Detail `[P0 · L]` — DONE
*As a user, I want detail tender so that semua info & analisa di satu tempat.*
- **AC:** Design §4.5: ringkasan (buyer/negara/industri/nilai/deadline badge/sumber link/scope/syarat/origin), panel Analisa AI (score ring + recommended_action badge + reasoning + evidence per dimensi + risk chips + "Dibuat AI • waktu" + Analisa ulang — placeholder hingga EP-10), tabs Ringkasan/Analisa AI/Playbook/Timeline, tombol Edit/Analisa/Playbook/WON-LOST.
- **Teknis:** `src/pages/tenders/TenderDetail.tsx`, tab components.
- **Dep:** ST-02.4, ST-05.2.

### ✅ ST-05.7 — FE Tender Form drawer `[P0 · M]` — DONE
*As a user, I want buat/edit tender so that data manual mudah masuk.*
- **AC:** Design §4.6 drawer: Judul*, Buyer, Negara, Industri, Nilai+currency, Deadline, Sumber/URL, Service category, Status, Scope; "Simpan & Analisa AI"; validasi inline.
- **Teknis:** `src/pages/tenders/TenderFormDrawer.tsx`.
- **Dep:** ST-02.2, ST-05.2.

---

## EP-06 — Event Management

### ST-06.1 — Entity event + migrasi + validasi `[P0 · S]`
- *As the system, I want model event.* **AC:** `event(name*,type*,date,location,organizer,notes,status)` + enum type/status.
- **Teknis:** migrasi, domain, repo. **Dep:** ST-00.4.

### ST-06.2 — CRUD endpoints `[P0 · S]`
- *As a user, I want kelola event.* **AC:** `GET/POST /api/events`, `GET/PUT/DELETE /api/events/:id`; min `name`+`type`.
- **Teknis:** `event_handler.go`. **Dep:** ST-06.1, ST-03.3.

### ST-06.3 — Konversi event → prospect `[P0 · M]`
- *As a user, I want konversi event jadi prospek so that peluang dari event terkelola.* **AC:** `POST /api/events/:id/convert` membuat `prospect` (`source_type=event`, `source_id`), kembalikan prospect baru.
- **Teknis:** service konversi. **Dep:** ST-06.2, ST-07.1.

### ST-06.4 — FE Event List/Detail/Form `[P0 · M]`
- *As a user, I want UI event.* **AC:** Design §4.7: list (Nama/Tipe pill/Tanggal/Lokasi/Organizer/Status/aksi), detail + "Kontak/Prospek dari event" + "+ Konversi ke Prospek", form (Nama*/Tipe/Tanggal/Lokasi/Organizer/Catatan).
- **Teknis:** `src/pages/events/*`. **Dep:** ST-02.3, ST-06.2.

---

## EP-07 — Prospect Management & Pipeline

### ✅ ST-07.1 — Entity prospect + migrasi + validasi `[P0 · M]`
- *As the system, I want model prospect.* **AC:** `prospect(name*,company,contact_info,source_type,source_id,stage*,est_value,owner_user_id)`; stage enum `NEW/QUALIFIED/ENGAGED/PROPOSAL/WON/LOST`.
- **Teknis:** migrasi, domain, repo. **Dep:** ST-00.4. ✓ **DONE**

### ✅ ST-07.2 — CRUD + stage patch `[P0 · M]`
- *As a user, I want kelola prospek & pindah stage.* **AC:** `GET/POST /api/prospects`, `GET/PUT/DELETE`, `PATCH /api/prospects/:id/stage`; WON/LOST emit outcome hook (EP-16).
- **Teknis:** `prospect_handler.go`. **Dep:** ST-07.1, ST-03.3. ✓ **DONE**

### ✅ ST-07.3 — FE Kanban board `[P0 · L]`
- *As a user, I want board pipeline so that lihat & geser prospek cepat.* **AC:** Design §4.8: kolom + header (nama+jumlah+total nilai), kartu (nama+company, badge sumber, score ring, est value, owner), drag-drop optimistic + rollback saat gagal, toggle Board↔Table, filter owner/sumber/min skor.
- **Teknis:** `src/pages/prospects/ProspectBoard.tsx` (dnd-kit), query invalidation. **Dep:** ST-02.4, ST-07.2. ✓ **DONE**

### ✅ ST-07.4 — FE Prospect detail drawer `[P0 · M]`
- *As a user, I want detail prospek.* **AC:** Design §4.9: header (nama+company+stage+score ring+owner), section Info (sumber→link tender/event), Analisa AI, Playbook, Timeline, aksi cepat (ubah stage, WON/LOST, "Tanya AI tentang prospek ini" → context chip ke chat).
- **Teknis:** `src/pages/prospects/ProspectDrawer.tsx`. **Dep:** ST-02.3, ST-07.2. ✓ **DONE**

---

## EP-08 — Knowledge / Company Profile (Otak Agent)

### ST-08.1 — Entities profil + migrasi + versioning `[P0 · L]`
- *As the system, I want simpan "otak" agent ter-versi.* **AC:** tabel `company_profile`, `target_criteria`, `nogo_rule`, `source`, `keyword_set` (field §10); mekanisme versi (mis. `version` + snapshot/`is_current`); satu profil aktif (single-org).
- **Teknis:** migrasi, domain, repo. **Dep:** ST-00.4.

### ST-08.2 — Profile read/write + defaults/preset `[P0 · L]`
- *As Ops, I want isi/ubah profil dengan default so that cepat aktif.* **AC:** `GET /api/profile` (versi terbaru), `PUT /api/profile` (buat versi baru); default (value_min Rp 1.000.000.000, deadline_min_days 7, countries=[Indonesia]); profil dipakai discovery/scoring; **RBAC: OPS/MANAGER/ADMIN** (SALES read-only).
- **Teknis:** `profile_handler.go`, preset seed. **Dep:** ST-08.1, ST-03.3.

### ST-08.3 — Source management `[P0 · M]`
- *As Ops, I want kelola sumber crawling.* **AC:** CRUD `source` (`name*,url*,country,access[publik/login/manual],legal_note,enabled,priority`); preset Indonesia (SPSE/Inaproc LKPP, eProc PLN, Pertamina, Telkom/SMILE, PaDi UMKM) bisa diaktifkan 1-klik; validasi URL; sumber Login/Manual ditandai (kepatuhan §9).
- **Teknis:** `source_handler.go`, preset. **Dep:** ST-08.2.

### ST-08.4 — Keyword set + auto-generate `[P0 · M]`
- *As Ops, I want keyword otomatis dari kapabilitas.* **AC:** `keyword_set` CRUD; keyword di-generate dari `service_categories` (bisa edit); `negative_keywords` preset; `language`.
- **Teknis:** `keyword_handler.go`, generator. **Dep:** ST-08.2.

### ST-08.5 — FE Onboarding lean `[P0 · L]`
- *As a new user, I want setup cepat < 2 menit so that discovery bisa mulai.* **AC:** Design §4.2: dua jalur (Upload PDF / Isi manual), stepper ringan, "Lewati atur nanti", akhir "Aktifkan Agent" → memicu discovery pertama → arahkan ke Penemuan AI; skip → banner di Dashboard.
- **Teknis:** `src/pages/onboarding/Onboarding.tsx` (jalur PDF placeholder hingga EP-13). **Dep:** ST-02.4, ST-08.2.

### ST-08.6 — FE Otak Agent edit (6 kartu) `[P0 · L]`
- *As Ops, I want halaman edit profil lean.* **AC:** Design §4.13: 6 kartu (Profil/Kapabilitas/Target/No-Go/Sumber&Keyword/Scoring advanced collapsed), chip preset + toggle + slider bobot, Simpan sticky, badge "diperbarui {waktu}", sub-tab Sumber (tabel + badge akses), tooltip per field, field hasil PDF ditandai "diisi AI ✨".
- **Teknis:** `src/pages/profile/OtakAgent.tsx` + sub-komponen kartu. **Dep:** ST-08.2, ST-08.3, ST-08.4.

---

## EP-09 — MCP Server & Sales Data Tools

### ST-09.1 — MCP server bootstrap + register `[P0 · M]`
- *As the system, I want server MCP so that Hermes bisa baca data.* **AC:** HTTP MCP (mark3labs/mcp-go) di `/mcp`, Bearer `${SALES_MCP_TOKEN}`; terdaftar di `deploy/hermes/config.yaml` (`mcp_servers.sales`, `supports_parallel_tool_calls:true`).
- **Teknis:** `internal/mcp/server.go`, config Hermes. **Dep:** ST-00.2, ST-01.x.

### ST-09.2 — Read tools `[P0 · L]`
- *As the agent, I want baca data sales.* **AC:** tools `list_tenders`, `get_tender`, `search_tenders`, `list_events`, `get_event`, `list_prospects`, `get_prospect`, `get_pipeline_summary`, `get_revenue_summary`, `get_company_profile` — schema input/output stabil, baca dari repo.
- **Teknis:** `internal/mcp/tools_read.go`. **Dep:** ST-09.1, entity EP-05/06/07/08.

### ST-09.3 — Write tools (gated) `[P0 · M]`
- *As the agent, I want usul aksi terbatas.* **AC:** `update_prospect_stage`, `save_playbook_draft` hanya yang di-whitelist (`tools.include`); aksi tercatat audit; human-in-the-loop (tidak final tanpa konfirmasi).
- **Teknis:** `internal/mcp/tools_write.go`, whitelist. **Dep:** ST-09.1, ST-07.2.

### ST-09.4 — Contract test MCP `[P0 · M]`
- *As a maintainer, I want jaminan round-trip MCP.* **AC:** test menembak Hermes nyata → chat memicu `list_tenders` → hasil dipakai; schema tool tidak berubah (aditif saja).
- **Teknis:** `internal/mcp/contract_test.go`. **Dep:** ST-09.2.

---

## EP-10 — AI Scoring & Recommendation

### ST-10.1 — Scoring service + prompt builder + schema `[P0 · L]`
- *As the backend, I want skor terstruktur.* **AC:** service merakit prompt (tender/prospect + Company Profile + rubrik §8 8-dimensi) → `GenerateJSON` schema `{fit_score 0–100, recommended_action, confidence, reasoning, evidence[], risk_flags[]}`; sesi pakai workspace session key.
- **Teknis:** `internal/ai/scoring.go`, prompt template, schema. **Dep:** ST-01.3, ST-08.2.

### ST-10.2 — Threshold & no-go rule `[P0 · M]`
- *As the system, I want rekomendasi konsisten.* **AC:** map skor→action (80–100 Pursue, 65–79 Review, 50–64 Watchlist, <50 Reject); no-go rule → Need Partner / Auto No-Go sesuai §8; deterministik.
- **Teknis:** `internal/ai/recommend.go`, evaluator no-go. **Dep:** ST-10.1, ST-08.2.

### ST-10.3 — Persist + endpoints + re-score `[P0 · M]`
- *As a user, I want simpan & ulang analisa.* **AC:** `POST /api/tenders/:id/score`, `POST /api/prospects/:id/score`; simpan `prospect_score` (target_type/target_id/fit_score/confidence/reasoning/evidence jsonb/risk_flags jsonb/model) + update skor pada tender; "Analisa ulang" idempotent; gagal AI → pesan ramah, data utuh; audit.
- **Teknis:** `internal/ai` + `prospect_score` repo + handler. **Dep:** ST-10.1, ST-05.1, ST-07.1.

### ST-10.4 — FE skor & rekomendasi `[P0 · M]`
- *As a user, I want lihat skor & alasan.* **AC:** score ring berwarna (skala §2.1), badge recommended_action, evidence per dimensi (✓/⚠), risk chips, "Dibuat AI • {confidence} • {waktu}" + "Lihat alasan", tombol "Analisa ulang" dengan streaming/loading; render di Tender Detail (ST-05.6) & Prospect drawer (ST-07.4).
- **Teknis:** `src/components/AiScorePanel.tsx`. **Dep:** ST-02.4, ST-10.3.

---

## EP-11 — Dashboard

### ST-11.1 — Dashboard summary endpoint `[P1 · M]`
- *As a manager, I want ringkasan terukur.* **AC:** `GET /api/dashboard/summary` → pipeline per stage, estimasi revenue (sum est_value per stage/total), prospek/tender prioritas (skor tinggi), penemuan AI hari ini (count); agregasi SQL efisien.
- **Teknis:** `dashboard_handler.go`, query agregat + indeks. **Dep:** ST-05/07/10; discovery count degrade bila EP-12 belum ada.

### ST-11.2 — FE Dashboard `[P1 · M]`
- *As a user, I want dashboard.* **AC:** stat cards (pipeline/revenue), penemuan AI hari ini, prioritas (score ring), AI insight callout, banner "Lengkapi Otak Agent" bila profil kosong; empty/loading.
- **Teknis:** `src/pages/Dashboard.tsx`. **Dep:** ST-02.4, ST-11.1.

---

## EP-12 — Tender Discovery via Hermes & Discovery Inbox

### ST-12.1 — Entity discovery_run + migrasi `[P1 · S]`
- *As the system, I want catat run discovery.* **AC:** `discovery_run(started_at,source_ids[],status,found_count,summary,finished_at,correlation_key)`.
- **Teknis:** migrasi, domain, repo. **Dep:** ST-00.4.

### ST-12.2 — Discovery orchestrator + compliance `[P1 · L]`
- *As Ops, I want agent menemukan tender legal.* **AC:** orchestrator pakai Company Profile (sumber enabled/keyword/target/no-go) → Hermes crawling/browser tools → ekstrak field tender → scoring (EP-10) → simpan `origin=discovery`; **§9: tidak bypass CAPTCHA/login/paywall**; sumber Login/Manual hanya ditandai (tidak dibobol); prioritas API/RSS/portal resmi.
- **Teknis:** `internal/ai/discovery.go`, integrasi `hermes` + `mcp`. **Dep:** ST-08.x, ST-10.x, ST-01.x.

### ST-12.3 — Dedup + idempotency `[P1 · M]`
- *As the system, I want hindari duplikat.* **AC:** `dedup_key = hash(buyer+title+deadline)`; tender sama dari beberapa sumber digabung ("ditemukan di N sumber"); run idempotent (correlation/idempotency key).
- **Teknis:** dedup di service + unique index. **Dep:** ST-12.2, ST-05.1.

### ST-12.4 — Run endpoints + async + rate limit `[P1 · M]`
- *As Ops, I want jalankan & pantau discovery.* **AC:** `POST /api/discovery/run` (async, kembalikan run id), `GET /api/discovery/runs`, `GET /api/discovery/inbox` (tender origin=discovery belum direview); rate limit + backoff per sumber; status run live.
- **Teknis:** `discovery_handler.go`, worker/goroutine + queue. **Dep:** ST-12.2.

### ST-12.5 — Scheduling `[P1 · M]`
- *As Ops, I want jadwal discovery.* **AC:** `crawl_frequency` (Harian/2–3x/Mingguan) memicu run terjadwal (cron Hermes atau scheduler internal); menghormati rate limit; bisa dimatikan.
- **Teknis:** scheduler + config. **Dep:** ST-12.4, ST-08.3.

### ST-12.6 — FE Penemuan AI inbox `[P1 · L]`
- *As a user, I want tinjau temuan.* **AC:** Design §4.3: header (Jalankan pencarian + status "terakhir Xj • N baru"), filter (rekomendasi/sumber/negara/min skor/deadline), kartu (score ring + recommended_action badge + judul + buyer + sumber + deadline badge + nilai + risk chips + alasan 1 baris + aksi Tinjau/Pursue/Watchlist/Tolak), dedup tampil, progress crawl, empty states (profil kosong / tidak ada hasil).
- **Teknis:** `src/pages/discovery/DiscoveryInbox.tsx`. **Dep:** ST-02.4, ST-12.4.

### ST-12.7 — Promote & Tolak (learning) `[P1 · M]`
- *As a user, I want aksi cepat dari inbox.* **AC:** Pursue → promote ke pipeline tender (ST-05.4); Watchlist → tandai; Tolak → minta alasan singkat → simpan untuk pembelajaran (EP-16); aksi optimistic.
- **Teknis:** wiring ke ST-05.4 + outcome/reject store. **Dep:** ST-12.6, ST-05.4.

---

## EP-13 — PDF Ingest untuk Company Profile

### ST-13.1 — Upload endpoint + storage ref `[P1 · M]`
- *As Ops, I want upload PDF profil.* **AC:** `POST /api/profile/ingest` (multipart PDF) menyimpan file + ref ke `source_doc_refs`; batas ukuran/tipe; aman.
- **Teknis:** `profile_handler.go` (upload), storage lokal/volume. **Dep:** ST-08.2.

### ST-13.2 — Extraction pipeline via Hermes `[P1 · L]`
- *As the system, I want ekstrak field dari PDF.* **AC:** kirim isi PDF ke Hermes → `GenerateJSON` field profil (company_name/one_liner/service_categories/tech_stack/target/no-go/keyword); kembalikan draft (tidak auto-simpan); gagal baca → error "coba isi manual".
- **Teknis:** `internal/ai/profile_extract.go`, parser PDF→teks. **Dep:** ST-13.1, ST-01.3.

### ST-13.3 — FE dropzone + review `[P1 · M]`
- *As Ops, I want review hasil ekstraksi.* **AC:** dropzone (Onboarding §4.2 & Otak Agent §4.13), progress "AI membaca dokumen…" (live-region), field hasil ditandai chip "diisi AI ✨", user edit & konfirmasi sebelum simpan (buat versi profil baru).
- **Teknis:** integrasi ke `Onboarding.tsx` & `OtakAgent.tsx`. **Dep:** ST-08.5/8.6, ST-13.2.

---

## EP-14 — Playbook Generator

### ST-14.1 — Entity playbook + migrasi (immutable versi) `[P1 · S]`
- *As the system, I want simpan playbook ter-versi.* **AC:** `playbook(target_type*,target_id*,title*,content jsonb*,version*,model,created_at)`; versi immutable (baru = version+1).
- **Teknis:** migrasi, domain, repo. **Dep:** ST-00.4.

### ST-14.2 — Playbook service + sections schema `[P1 · L]`
- *As the backend, I want generate playbook terstruktur.* **AC:** prompt (konteks peluang + playbook menang sebelumnya dari memory) → `GenerateJSON` sections: Ringkasan Peluang, Value Proposition, Stakeholders, Strategi & Langkah (checklist), Timeline, Risiko & Mitigasi, Next Actions.
- **Teknis:** `internal/ai/playbook.go`. **Dep:** ST-01.3, ST-10.x.

### ST-14.3 — Endpoints generate + versi + compare `[P1 · M]`
- *As a user, I want buat & bandingkan versi.* **AC:** `POST /api/tenders/:id/playbook`, `POST /api/prospects/:id/playbook`, `GET .../playbooks` (list versi), `GET /api/playbooks/:id`; compare dua versi.
- **Teknis:** `playbook_handler.go`. **Dep:** ST-14.2.

### ST-14.4 — FE Playbook viewer `[P1 · L]`
- *As a user, I want lihat playbook.* **AC:** Design §4.10: list (target/judul/versi/tanggal), viewer terstruktur per section, generate streaming, footer (versi/model/waktu, "Generate versi baru", "Bandingkan"), salin/export markdown.
- **Teknis:** `src/pages/playbooks/*`. **Dep:** ST-02.4, ST-14.3.

---

## EP-15 — Report Generator

### ST-15.1 — Entity report + migrasi `[P1 · S]`
- *As the system, I want simpan laporan.* **AC:** `report(type*,period_start,period_end,content*,generated_by,created_at)`; validasi `period_start ≤ period_end`.
- **Teknis:** migrasi, domain, repo. **Dep:** ST-00.4.

### ST-15.2 — Report service (3 tipe) `[P1 · L]`
- *As the backend, I want generate laporan.* **AC:** agregasi pipeline/aktivitas → Hermes rangkum markdown untuk **Daily Opportunity Digest**, **Weekly Pipeline**, **Per-peluang**; < 2 menit.
- **Teknis:** `internal/ai/report.go`, agregator. **Dep:** ST-01.x, ST-05/07/11.

### ST-15.3 — Endpoints generate/list/get/delete `[P1 · M]`
- *As a user, I want kelola laporan.* **AC:** `POST /api/reports` (type+period), `GET /api/reports`, `GET /api/reports/:id`, `DELETE /api/reports/:id`.
- **Teknis:** `report_handler.go`. **Dep:** ST-15.2.

### ST-15.4 — FE Reports `[P1 · M]`
- *As a user, I want UI laporan.* **AC:** Design §4.11: list (tipe/periode/tanggal), generate modal (tipe+periode) streaming, viewer terstruktur (ringkasan + tabel pipeline + prospek prioritas + insight AI), export markdown/salin/hapus, empty state.
- **Teknis:** `src/pages/reports/*`. **Dep:** ST-02.4, ST-15.3.

---

## EP-16 — Continuous Learning & Outcome Feedback

### ST-16.1 — Entity outcome_event + migrasi `[P1 · S]`
- *As the system, I want catat hasil.* **AC:** `outcome_event(target_type*,target_id*,result* WON/LOST,notes,created_at)`; dipakai metrik win rate.
- **Teknis:** migrasi, domain, repo. **Dep:** ST-00.4.

### ST-16.2 — Outcome hook → memory Hermes `[P1 · M]`
- *As the system, I want agent belajar.* **AC:** saat WON/LOST (EP-05/07) & Tolak discovery (EP-12), kirim catatan ringkas ke memory Hermes (session-key workspace) via `internal/hermes`; non-blocking.
- **Teknis:** `internal/ai/learning.go`, hook di handler outcome/reject. **Dep:** ST-01.x, ST-05.3, ST-07.2, ST-12.7.

### ST-16.3 — Reset memory (admin) `[P1 · S]`
- *As an admin, I want reset memory.* **AC:** `POST /api/admin/hermes/reset-memory` (ADMIN only) membersihkan memory workspace; konfirmasi.
- **Teknis:** handler + `hermes` call. **Dep:** ST-03.3, ST-01.x.

### ST-16.4 — FE cue learning + verifikasi `[P1 · S]`
- *As a user, I want tahu AI belajar.* **AC:** cue "Asisten belajar dari aktivitas & hasil kamu" di Chat & modal WON/LOST/Tolak ("AI akan belajar dari ini"); manual verify memory persist lintas restart Hermes.
- **Teknis:** microcopy + integrasi. **Dep:** ST-04.5, ST-16.2.

---

## EP-17 — Telemetry, Observability, Audit & NFR Hardening

### ST-17.1 — Telemetry events `[P1 · M]`
- *As a PM, I want metrik §2 terukur.* **AC:** event `chat_opened`, review pursue, durasi generate report/scoring, `outcome_event` WON/total — terkirim & bisa diquery in-app.
- **Teknis:** `internal/telemetry/*`, emit di titik relevan. **Dep:** fitur terkait.

### ST-17.2 — Structured logging + audit trail `[P1 · M]`
- *As a maintainer, I want audit.* **AC:** structured logging; audit trail crawling (sumber/waktu/akses/data) + tiap output AI (reasoning/evidence/model/waktu).
- **Teknis:** logger + tabel/log audit. **Dep:** EP-10, EP-12.

### ST-17.3 — Performance hardening + health panel `[P1 · M]`
- *As a user, I want responsif.* **AC:** CRUD < 300ms p95 (indeks); chat < 2s; scoring < 15s; discovery async; health-check Hermes terpapar.
- **Teknis:** indeks DB, profiling, health endpoint. **Dep:** fitur terkait.

### ST-17.4 — Contract tests final + verifikasi NFR `[P1 · M]`
- *As a maintainer, I want gate rilis.* **AC:** suite contract (`/v1` + MCP) hijau; checklist NFR §11 terverifikasi; skenario verifikasi epic §4 lulus.
- **Teknis:** test suite + checklist. **Dep:** EP-01, EP-09.

---

## EP-18 — Settings, Admin & Hermes Ops

### ST-18.1 — Hermes status/test endpoint `[P0 · S]`
- *As an admin, I want status integrasi.* **AC:** `GET /api/settings/hermes` → status koneksi + versi (`/v1/capabilities`) + memory aktif; `POST /api/settings/hermes/test` test koneksi.
- **Teknis:** handler + `hermes.Health`. **Dep:** ST-01.4.

### ST-18.2 — FE Settings tabs `[P0/P1 · M]`
- *As a user, I want halaman Settings.* **AC:** Design §4.14: tab Profil user · Workspace · Users (admin: tambah/nonaktif/role) · AI/Hermes (status "Connected • vX", memory "Pembelajaran aktif", reset memory[admin], "Test koneksi"); gagal → badge merah + petunjuk.
- **Teknis:** `src/pages/settings/*`. **Dep:** ST-02.3, ST-03.4, ST-18.1.

### ST-18.3 — Reset memory wiring `[P1 · S]`
- *As an admin, I want reset memory dari UI.* **AC:** tombol reset (ADMIN) → konfirmasi → panggil ST-16.3 → toast.
- **Teknis:** integrasi. **Dep:** ST-18.2, ST-16.3.

### ST-18.4 — AI Provider Config dari UI (OpenAI / OpenRouter) `[P0 · L]`
*As an admin, I want mengatur provider, model, dan API key AI dari UI (seperti TUI Hermes) so that bisa ganti/menyesuaikan engine tanpa redeploy.*
- **AC:** entity `ai_provider_setting` (provider ∈ {openai, openrouter}, model, base_url opsional, api_key terenkripsi, enabled_toolsets, is_active); `GET /api/settings/ai` (key **masked**, tak pernah plaintext), `PUT /api/settings/ai` (RBAC ADMIN; simpan DB + push `hermes.Configure` ke bridge), `POST /api/settings/ai/test` (uji koneksi provider via bridge); saat boot Go re-push config aktif ke bridge (rehydrate); FE tab "AI Provider": pilih provider, model (preset per provider + custom), base_url, API key (write-only), toolsets, Simpan + Test; ganti provider → Test hijau → chat berikutnya pakai provider baru tanpa restart.
- **Teknis:** migrasi `00xx_ai_setting`, `domain/ai_setting.go`, `repository/ai_setting_repo.go`, `internal/auth/crypto.go` (AES-GCM, `CONFIG_ENC_KEY`), `handlers/ai_settings_handler.go`, rehydrate di `apps/api/main.go`, `src/pages/settings/AiProvider.tsx`. Bergantung bridge `/admin/config` & `internal/hermes.Configure` (EP-01).
- **Dep:** ST-01.6, ST-03.4 (RBAC admin), ST-18.2 (Settings shell).

---

## Status
- [x] **Layer 1 (EPIC)** — epic.plan.md.
- [x] **Layer 2 (STORY)** — file ini (≈90 story untuk 19 epic).
- [x] **Layer 3 (TASK)** — task.plan.md (task mikro siap eksekusi + §A pola reusable).
