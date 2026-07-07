# epic.plan.md — SalesPilot (Layer 1: EPIC)

> **Sumber kebenaran:** [PRD.md](./PRD.md) v1.3 · [Design.md](./Design.md) v1.3 · Plan teknis: `C:\Users\sheld\.claude\plans\saya-ingin-membuat-tools-effervescent-kahan.md`.
> **Bahasa:** prosa Bahasa Indonesia; istilah teknis, path, kode, dan nama field English.
> **Layer:** Ini **layer paling atas** dari rencana 3-lapis: **EPIC → STORY → TASK**. File ini hanya garis besar (layaknya Epic di Jira). Detail per-story ada di [story.plan.md](./story.plan.md); langkah mikro yang dieksekusi ada di [task.plan.md](./task.plan.md).

---

## 0. Cara membaca dokumen ini

**Filosofi 3-lapis.** Pekerjaan dipecah jadi 3 lapis supaya bisa dieksekusi model yang lebih hemat (mis. Sonnet 4.6) namun hasilnya **konsisten & berkualitas** setara dikerjakan langsung oleh model besar:
- **EPIC (file ini):** *apa & kenapa* — potongan besar bernilai bisnis, urutan, dependency, definition of done.
- **STORY:** *untuk siapa & kapan dianggap selesai* — user story + acceptance criteria + catatan teknis.
- **TASK:** *bagaimana persis* — langkah mikro, path file, signature fungsi, perintah, dan cara verifikasi. Tidak ada keputusan arsitektur yang tersisa di lapis ini.

**Skema ID (dipakai konsisten di 3 file):**
- Epic: `EP-NN` (mis. `EP-05`).
- Story: `ST-NN.s` (mis. `ST-05.2` = story ke-2 di `EP-05`).
- Task: `TK-NN.s.t` (mis. `TK-05.2.3`).

Traceability: tiap Story menyebut Epic induknya; tiap Task menyebut Story induknya. Tiap Epic memetakan ke **PRD §/E** dan **Milestone (M0–M6)**.

---

## 1. Keputusan teknis yang mengikat semua epic (Conventions)

> Lapis Story & Task **mewarisi** ini tanpa mengulang. Bila ada konflik dengan plan teknis lama, **PRD v1.3 menang**.

**Stack**
- **Backend:** Go 1.22+, **Echo v4**, **GORM** + **golang-migrate**, **mark3labs/mcp-go** (HTTP MCP server), `net/http` SSE untuk chat passthrough.
- **Frontend:** React 18 + **Vite** + **TypeScript**, **TanStack Query** (server state), **Zustand** (client state ringan), **React Router**, **Tailwind CSS** (token dari Design.md §2), **lucide-react** (ikon), **dnd-kit** (kanban drag-drop).
- **DB:** PostgreSQL 16.
- **AI "otak":** **Hermes Agent** (Nous Research) = agentic engine yang diakses via **Python library** (`AIAgent`), **bukan** image gateway OpenAI-compatible. Dibungkus service **`hermes-bridge`** (FastAPI) yang mengekspos HTTP `/v1` tiruan + header terdokumentasi, sehingga ACL Go (`internal/hermes`) tetap coupling **hanya** ke kontrak `/v1`. Header sesi: `X-Hermes-Session-Id`, `X-Hermes-Session-Key`. Auth `Authorization: Bearer ${API_SERVER_KEY}`. **Double isolation** (ACL Go + bridge Python) menahan upgrade major Hermes. Detail: [hermes-bridge.plan.md](./hermes-bridge.plan.md).
- **Runtime:** `docker-compose` (postgres + api + web + **hermes-bridge**). Fallback native Windows didukung.

**Koreksi wajib vs plan teknis lama (akibat PRD v1.3):**
- ❌ **TIDAK ada `organization_id`** di entity manapun dan **tidak ada tabel `organization`**. Aplikasi **single-org murni** (internal satu perusahaan). Multi-tenant ditunda hingga benar-benar perlu.
- ✅ **Tanpa self-signup.** Akun dibuat oleh **Admin**. Login internal saja (JWT).
- ✅ **Session key Hermes** = konstanta workspace tunggal (mis. `WORKSPACE_SESSION_KEY` dari env) supaya memory terakumulasi lintas sesi untuk satu perusahaan.

**Struktur repo** (root: `c:\Users\sheld\Documents\Sheldy\sales_pilot`)
```
sales_pilot/
├── apps/
│   ├── api/                 # main.go — bootstrap Echo
│   └── web/                 # React + Vite
├── internal/
│   ├── config/              # load env, config struct
│   ├── auth/                # JWT, middleware, RBAC
│   ├── domain/              # entities + port interfaces (tanpa infra)
│   ├── repository/          # GORM, per-entity
│   ├── http/{handlers,dto,router.go}
│   ├── hermes/              # ★ ANTI-CORRUPTION LAYER (Client iface + /v1 impl)
│   ├── ai/                  # orchestration: scoring, playbook, report, discovery
│   └── mcp/                 # ★ HTTP MCP server (tools data sales)
├── db/migrations/           # golang-migrate (0001_init.sql, ...)
├── deploy/{docker-compose.yml,hermes/{config.yaml.example,.env.example}}
├── go.mod · .env.example · README.md
└── PRD.md · Design.md · epic.plan.md · story.plan.md · task.plan.md
```

**Arsitektur backend:** `handler → service → repository`. `domain` mendefinisikan entity + port interface; tidak boleh `import` infra. Semua AI non-blocking terhadap CRUD (gagal AI ⇒ pesan ramah, data utuh).

**Konvensi REST:** prefix `/api`; resource plural; status & error JSON konsisten `{ "error": { "code", "message" } }`; pagination `?page=&page_size=`; filter via query param.

**Definition of Done (berlaku semua epic):**
1. Acceptance criteria tiap story terpenuhi & bisa didemokan end-to-end.
2. Unit test untuk service/logic non-trivial; contract test untuk batas Hermes/MCP.
3. UI sesuai Design.md (token, state empty/loading/error, Bahasa Indonesia, format Rupiah/tanggal lokal).
4. `go vet` + `golangci-lint` bersih; `tsc --noEmit` + `eslint` bersih.
5. Telemetry event (bila relevan) terpasang; audit log untuk aksi AI/crawling.

---

## 2. Peta Epic & urutan eksekusi

| Epic | Judul | Prioritas | Milestone | PRD ref |
|---|---|:--:|:--:|---|
| **EP-00** | Fondasi Proyek, Monorepo & DevOps | P0 | M0 | §5, §13 |
| **EP-01** | Anti-Corruption Layer & Jembatan Hermes | P0 | M0 | §5, §11, §12 |
| **EP-02** | Design System & Application Shell | P0 | M0–M1 | Design §2–§3 |
| **EP-03** | Auth, RBAC & User Management | P0 | M1 | E10, §3.1 |
| **EP-04** | Chat Assistant (Hermes streaming) | P0 | M0/M3 | E7 |
| **EP-05** | Tender Management | P0 | M1 | E1 |
| **EP-06** | Event Management | P0 | M1 | E2 |
| **EP-07** | Prospect Management & Pipeline (Kanban) | P0 | M1 | E3 |
| **EP-08** | Knowledge / Company Profile (Otak Agent) | P0 | M2 | E11, §6 |
| **EP-09** | MCP Server & Sales Data Tools | P0 | M2–M3 | §5, §8 |
| **EP-10** | AI Scoring & Recommendation | P0 | M3 | E4, §8 |
| **EP-11** | Dashboard | P1 | M3 | E8 |
| **EP-12** | Tender Discovery via Hermes & Discovery Inbox | P1 | M4 | E12, §8, §9 |
| **EP-13** | PDF Ingest untuk Company Profile | P1 | M4 | E11 (PDF) |
| **EP-14** | Playbook Generator | P1 | M5 | E5 |
| **EP-15** | Report Generator | P1 | M5 | E6 |
| **EP-16** | Continuous Learning & Outcome Feedback | P1 | M6 | E9, §8.6 |
| **EP-17** | Telemetry, Observability, Audit & NFR Hardening | P1 | M6 | §11, §9 |
| **EP-18** | Settings, Admin & Hermes Ops | P0/P1 | M1/M6 | E10, Design §4.14 |

**Diagram dependency (garis besar):**
```
EP-00 ─┬─▶ EP-01 ─▶ EP-04 (chat) ─────────────┐
       ├─▶ EP-02 ─▶ EP-03 ─┬─▶ EP-05 ─┐        │
       └────────────────────┼─▶ EP-06 ─┼─▶ EP-07│
                            └─▶ EP-08 ──┘        │
EP-05/06/07/08 ─▶ EP-09 (MCP) ─▶ EP-10 (scoring) ─▶ EP-11 (dashboard)
EP-08 + EP-01 ─▶ EP-12 (discovery) ; EP-08 ─▶ EP-13 (PDF)
EP-10 ─▶ EP-14 (playbook) ; EP-05/07/11 ─▶ EP-15 (report)
EP-10/12/14 ─▶ EP-16 (learning) ; semua ─▶ EP-17 (telemetry) ; EP-01/03 ─▶ EP-18 (settings)
```

**Jalur kritis MVP minimal demoable:** EP-00 → EP-01 → EP-02 → EP-03 → (EP-05/06/07) → EP-08 → EP-09 → EP-10 → EP-04. Sisanya memperkaya.

---

## 3. Detail Epic

> Format tiap epic: **Tujuan · Scope (in/out) · Deliverables · Dependency · Definition of Done (epic) · Risiko.**

### ✅ EP-00 — Fondasi Proyek, Monorepo & DevOps `[P0 · M0]`
- **Tujuan:** kerangka repo yang bisa dijalankan: Go API "hello", React app kosong, Postgres, docker-compose, config env, tooling lint/test, README run guide.
- **Scope in:** struktur folder (§1), `go.mod`, Echo bootstrap + `/healthz`, Vite+TS app shell kosong, koneksi DB + migrasi awal kosong, `docker-compose.yml` (postgres+api+web+hermes-gateway), `.env.example`, lining/format, CI lokal (script test).
- **Scope out:** fitur bisnis apa pun.
- **Deliverables:** repo bisa `docker compose up` → API `/healthz` 200, web terbuka, DB hidup, migrasi jalan.
- **Dependency:** —
- **DoD:** `docker compose up` sukses; `go test ./...` & `npm run build` hijau; README menjelaskan cara run (Docker & native Windows).
- **Risiko:** Hermes image belum tersedia → sediakan build context ke install lokal Hermes + dokumentasikan fallback native.

### ✅ EP-01 — Anti-Corruption Layer & Jembatan Hermes `[P0 · M0]`
- **Tujuan:** satu-satunya titik kontak ke Hermes (`internal/hermes`), tahan breaking change. **Double isolation:** ACL Go + service `hermes-bridge` (Python) yang membungkus `AIAgent`.
- **Scope in:** `Client` interface (`Chat`, `ChatStream`, `GenerateJSON`, `Health`, `Configure`); impl `/v1/chat/completions` (SSE), `/v1/responses`, `/v1/capabilities`, `/health`; injeksi header sesi; startup health-guard + degrade graceful; **`hermes-bridge` (FastAPI)** yang membungkus Hermes `AIAgent` & mengekspos `/v1` tiruan (streaming MVP = blocking-then-send); endpoint `/admin/config` agar provider/model/key bisa di-override runtime (dipakai EP-18); pin versi lib `hermes-agent`; skeleton **contract tests**. Detail bridge: [hermes-bridge.plan.md](./hermes-bridge.plan.md).
- **Scope out:** logika scoring/playbook (pakai layer ini, tapi ada di EP-10/14); UI pengaturan provider (di EP-18).
- **Deliverables:** paket `internal/hermes` + service `services/hermes-bridge` + config env (`HERMES_BASE_URL`, `API_SERVER_KEY`, `WORKSPACE_SESSION_KEY`, `HERMES_MODEL`, `OPENAI_API_KEY`/`OPENROUTER_API_KEY`); contract test `-run Contract`.
- **Dependency:** EP-00.
- **DoD:** dari Go bisa memanggil Hermes nyata via bridge (chat stream + GenerateJSON); health-guard bekerja saat bridge/Hermes mati (degrade, tidak crash); contract test hijau.
- **Risiko:** API Hermes `AIAgent` bergeser → semua perubahan terisolasi di `hermes-bridge`; ACL Go tetap; pin versi + contract test.

### ✅ EP-02 — Design System & Application Shell `[P0 · M0–M1]`
- **Tujuan:** fondasi UI sesuai Design.md: token warna/tipografi/spacing, library komponen dasar, layout global (sidebar+topbar), util format lokal & i18n Bahasa Indonesia.
- **Scope in:** Tailwind config dari token Design §2.1–§2.3; komponen library §2.4 (Button, Input, Select, Chip/Tag, Badge/Pill, Card, Table, Tabs, Drawer, Modal, Toast, Skeleton, Empty state, Score ring, Stat card, AI panel/callout, Streaming text, File dropzone, Toggle, Stepper, Risk-flag chip); Application Shell §3 (sidebar collapsible, topbar, floating "Tanya AI"); state interaksi §2.4; util `formatRupiah`, `formatTanggal`, relative time (id-ID); fondasi aksesibilitas §8.
- **Scope out:** halaman fitur (memakai shell ini).
- **Deliverables:** Storybook/katalog komponen sederhana atau halaman `/dev/components`; util format teruji.
- **Dependency:** EP-00.
- **DoD:** semua komponen dasar render dengan state (default/hover/focus/disabled/loading/error); shell navigasi berfungsi; format Rupiah & tanggal sesuai §11.
- **Risiko:** drift dari Design.md → patuhi token & matriks komponen→layar (Design §12).

### EP-03 — Auth, RBAC & User Management `[P0 · M1]`
- **Tujuan:** login internal (JWT) + role `SALES/OPS/MANAGER/ADMIN` + admin kelola user (tanpa self-signup).
- **Scope in:** entity `user`; login `/api/auth/login`; JWT issue/verify; middleware auth + RBAC sesuai matriks §3.1; admin CRUD user (`/api/users`, hanya ADMIN); seed admin awal; FE Login (Design §4.1, tanpa link daftar) + guard route + store auth.
- **Scope out:** SSO eksternal (catat sebagai opsi P2 di NOTES).
- **Deliverables:** alur login → token → akses terproteksi; admin bisa tambah/nonaktifkan user & set role.
- **Dependency:** EP-00, EP-02.
- **DoD:** matriks permission §3.1 ditegakkan di BE (uji per-role) & FE (sembunyikan aksi terlarang); password hash aman (bcrypt); token expiry + refresh sederhana.
- **Risiko:** kebocoran permission → uji RBAC per endpoint.

### EP-04 — Chat Assistant (Hermes streaming) `[P0 · M0/M3]`
- **Tujuan:** chat streaming ke Hermes, baca data via MCP, histori tersimpan, degrade graceful. (Passthrough minimal sudah bisa di M0; fitur penuh setelah MCP EP-09.)
- **Scope in:** entity `conversation`/`message`; `POST /api/conversations`, `POST /api/conversations/:id/chat` (SSE relay), list/get history; FE Chat (Design §4.12): bubble user/assistant violet, tool-call chip expandable, streaming + Stop, suggested chips, context chip, floating slide-over; error banner "Agent tidak tersedia".
- **Scope out:** tool MCP itu sendiri (EP-09).
- **Deliverables:** chat end-to-end dengan indikator tool-call saat agent membaca data.
- **Dependency:** EP-01 (wajib), EP-02, EP-03; fungsionalitas tool butuh EP-09.
- **DoD:** stream tampil < 2s; histori tersimpan & bisa dibuka ulang; saat Hermes down, CRUD tetap jalan & chat tampilkan banner.
- **Risiko:** SSE relay rumit → reuse `internal/hermes.ChatStream`.

### ✅ EP-05 — Tender Management `[P0 · M1]` — **DONE (ST-05.1–05.7 backend+frontend)**
- **Tujuan:** CRUD tender + status pipeline + hook ke Analisa AI/Playbook + promote dari discovery.
- **Scope in:** entity `tender` (field lengkap §10 termasuk `recommended_action`, `risk_flags`, `origin`, `dedup_key`); CRUD `/api/tenders`; transisi status `IDENTIFIED→QUALIFYING→BIDDING→SUBMITTED→WON/LOST`; `POST /api/tenders/:id/outcome` (WON/LOST + catatan); FE List (Design §4.4: filter status/buyer/deadline/rekomendasi/origin, kolom skor+badge), Detail (§4.5: ringkasan + tab Analisa AI/Playbook/Timeline + recommended_action badge + risk chips + origin link), Form drawer (§4.6).
- **Scope out:** generate skor/playbook (EP-10/14) & discovery (EP-12) — tombol placeholder sudah ada (AiScorePanel, Playbook tab, Analisa button).
- **Deliverables:** CRUD penuh + transisi status tervalidasi + outcome → `outcome_event` + learning hook (no-op). **Catatan:** `outcome_event` dibuat di EP-05 (bukan EP-16); TK-16.1.1 superseded; migrasi EP-06+ bergeser +1.
- **Dependency:** EP-02, EP-03.
- **DoD:** buat dengan min `title`; status & WON/LOST tersimpan; validasi §10 (value ≥ 0, enum status); `go test ./... hijau`; `go vet ./... hijau`.
- **Risiko:** field discovery & manual bercampur → bedakan `origin`.

### EP-06 — Event Management `[P0 · M1]`
- **Tujuan:** CRUD event + konversi event → prospek.
- **Scope in:** entity `event`; CRUD `/api/events`; aksi "+ Konversi ke Prospek" (`source_type=event`); FE List & Detail (Design §4.7) + form.
- **Scope out:** —
- **Deliverables:** CRUD event + konversi membuat prospect tertaut.
- **Dependency:** EP-02, EP-03; konversi butuh EP-07.
- **DoD:** event tersimpan dengan min `name`+`type`; konversi membuat `prospect` dengan `source_type=event`, `source_id=event.id`.
- **Risiko:** —

### ✅ EP-07 — Prospect Management & Pipeline (Kanban) `[P0 · M1]` — **DONE (ST-07.1–07.4 backend+frontend)**
- **Tujuan:** kelola prospek dalam board kanban + detail.
- **Scope in:** entity `prospect`; CRUD `/api/prospects`; `PATCH /api/prospects/:id/stage`; FE Kanban (Design §4.8: kolom `NEW→QUALIFIED→ENGAGED→PROPOSAL→WON/LOST`, header jumlah+total nilai, drag-drop optimistic+rollback, toggle Board↔Table, filter), Detail drawer (§4.9: info+sumber link, Analisa AI, Playbook, Timeline, aksi cepat).
- **Scope out:** skor/playbook (EP-10/14).
- **Deliverables:** board fungsional, drag-drop tersimpan, detail drawer.
- **Dependency:** EP-02, EP-03.
- **DoD:** pindah kartu menyimpan stage (optimistic + rollback saat gagal); konversi dari event/tender membuat tautan sumber.
- **Risiko:** state board kompleks → pakai dnd-kit + TanStack Query invalidation.

### EP-08 — Knowledge / Company Profile (Otak Agent) `[P0 · M2]`
- **Tujuan:** "otak" agent diisi lean (6 kartu) + versioning; jadi sumber kebenaran discovery/scoring.
- **Scope in:** entity `company_profile`, `target_criteria`, `nogo_rule`, `source`, `keyword_set` (§10); endpoint baca/tulis + versioning; preset Indonesia (sumber & keyword) + default; FE Onboarding lean (Design §4.2: dua jalur, target < 2 menit) + halaman edit 6 kartu (§4.13) + sub-tab Sumber (badge akses Publik/Login/Manual) + Scoring advanced collapsed (slider bobot).
- **Scope out:** PDF ingest (EP-13) & crawling (EP-12) & MCP expose (EP-09).
- **Deliverables:** profil tersimpan & versi-an dengan preset & default; UI lean.
- **Dependency:** EP-02, EP-03.
- **DoD:** quick start bisa selesai dengan minimal input (kapabilitas + 1 negara + nilai min) lalu "Aktifkan Agent"; profil ter-versi; badge "diperbarui {waktu}".
- **Risiko:** form membludak → tegakkan prinsip lean (mayoritas opsional, chip preset, default).

### EP-09 — MCP Server & Sales Data Tools `[P0 · M2–M3]`
- **Tujuan:** expose data sales ke Hermes via HTTP MCP (read aman + write gated), termasuk `get_company_profile`.
- **Scope in:** `internal/mcp` server di `/mcp` (Bearer `${SALES_MCP_TOKEN}`); read tools: `list_tenders`, `get_tender`, `search_tenders`, `list_events`, `get_event`, `list_prospects`, `get_prospect`, `get_pipeline_summary`, `get_revenue_summary`, `get_company_profile`; write tools (gated/whitelist): `update_prospect_stage`, `save_playbook_draft`; config Hermes `mcp_servers.sales`.
- **Scope out:** logika scoring (EP-10).
- **Deliverables:** agent (via chat EP-04) bisa membaca data nyata; tool-call terlihat di UI.
- **Dependency:** EP-01, dan entity dari EP-05/06/07/08.
- **DoD:** round-trip MCP terbukti (chat "tender prioritas?" memanggil `list_tenders`); write tools hanya yang di-whitelist; contract test MCP hijau.
- **Risiko:** write tool berbahaya → whitelist + human-in-the-loop.

### EP-10 — AI Scoring & Recommendation `[P0 · M3]`
- **Tujuan:** fit score 0–100 + recommended_action + confidence + reasoning + evidence per dimensi + risk_flags, pakai rubrik §8.
- **Scope in:** `internal/ai` scoring service (rakit prompt: tender/prospect + Company Profile + rubrik) → `GenerateJSON` schema `{fit_score, recommended_action, confidence, reasoning, evidence[], risk_flags[]}`; simpan `prospect_score`/skor tender; ambang rekomendasi + no-go rule → Need Partner/Auto No-Go; endpoint `POST /api/tenders/:id/score` & `/api/prospects/:id/score`; "Analisa ulang"; FE: score ring berwarna (skala §2.1), badge recommended_action, evidence per dimensi, "Dibuat AI • confidence • waktu".
- **Scope out:** discovery scoring massal (EP-12 memakai service ini).
- **Deliverables:** skor tersimpan & tampil di Tender/Prospect detail; gagal AI → pesan ramah, data utuh.
- **Dependency:** EP-08 (profil), EP-09 (opsional konteks), EP-01.
- **DoD:** output sesuai schema & rubrik; ambang & no-go diterapkan; idempotent re-score; audit (model/waktu/evidence).
- **Risiko:** halusinasi → wajib reasoning+evidence+confidence + tanda "Dibuat AI".

### EP-11 — Dashboard `[P1 · M3]`
- **Tujuan:** ringkasan: penemuan AI hari ini, pipeline per stage, estimasi revenue, prospek/tender prioritas.
- **Scope in:** `GET /api/dashboard/summary` (agregasi pipeline + revenue + prioritas + discovery hari ini); FE Dashboard (stat card, score ring, AI insight callout, banner "Lengkapi Otak Agent" bila profil kosong).
- **Scope out:** —
- **Deliverables:** dashboard dengan data nyata + empty states.
- **Dependency:** EP-05/07/10; discovery count dari EP-12 (opsional, degrade bila belum ada).
- **DoD:** metrik akurat terhadap DB; performa < 300ms p95 (query teragregasi).
- **Risiko:** query berat → indeks + agregasi SQL.

### EP-12 — Tender Discovery via Hermes (Crawling) & Discovery Inbox `[P1 · M4]`
- **Tujuan:** agent menemukan tender dari sumber yang disetujui → Inbox Penemuan AI; dedup; terjadwal; patuh hukum.
- **Scope in:** entity `discovery_run`; orchestrator discovery (`internal/ai`): pakai Company Profile (sumber/keyword/target/no-go) → Hermes crawling/browser tools → ekstrak field → scoring (EP-10) → simpan tender `origin=discovery`; **deduplikasi** via `dedup_key`; **kepatuhan §9** (tidak bypass CAPTCHA/login/paywall; sumber Login/Manual ditandai; rate limit + backoff); idempotency key; endpoint `POST /api/discovery/run`, `GET /api/discovery/runs`, list inbox; penjadwalan (cron Hermes / scheduler internal); FE Penemuan AI (Design §4.3): kartu skor+rekomendasi+risk chips, aksi Tinjau/Pursue/Watchlist/Tolak(+alasan), progress crawl, empty states.
- **Scope out:** —
- **Deliverables:** discovery run menghasilkan tender baru di inbox; Pursue mem-promote ke pipeline.
- **Dependency:** EP-08, EP-10, EP-01; audit dari EP-17.
- **DoD:** dedup bekerja; sumber Login/Manual tidak dibobol (ditandai); run async (background) + idempotent; Tolak menyimpan alasan (untuk learning EP-16).
- **Risiko:** legal/blokir sumber → hormati TOS, prioritas API/RSS/portal resmi, audit trail.

### EP-13 — PDF Ingest untuk Company Profile `[P1 · M4]`
- **Tujuan:** upload PDF (capability deck/company profile/RFI) → ekstrak → isi draft field profil untuk direview.
- **Scope in:** upload endpoint + storage ref (`source_doc_refs`); pipeline ekstraksi via Hermes (baca dokumen → field profil terstruktur); FE: dropzone (Onboarding §4.2 & Otak Agent §4.13), progress "AI membaca dokumen…" (streaming/live-region), field hasil ditandai chip "diisi AI ✨", review & konfirmasi sebelum simpan.
- **Scope out:** OCR berat / dokumen non-PDF (P2).
- **Deliverables:** alur PDF → draft profil → konfirmasi → tersimpan.
- **Dependency:** EP-08, EP-01.
- **DoD:** PDF gagal baca → "coba isi manual"; field hasil ekstraksi bisa diedit sebelum disimpan; tidak menimpa profil tanpa konfirmasi.
- **Risiko:** ekstraksi tidak akurat → selalu review manusia; tandai field AI.

### EP-14 — Playbook Generator `[P1 · M5]`
- **Tujuan:** playbook terstruktur per peluang, versi immutable.
- **Scope in:** entity `playbook`; `internal/ai` playbook service (konteks peluang + playbook menang sebelumnya dari memory) → sections terstruktur; endpoint `POST /api/tenders/:id/playbook` & `/api/prospects/:id/playbook`; FE Playbooks (Design §4.10): viewer terstruktur (Ringkasan/Value Prop/Stakeholders/Strategi checklist/Timeline/Risiko/Next Actions), generate streaming, versi baru + bandingkan, salin/export markdown.
- **Scope out:** export PDF (P2).
- **Deliverables:** generate playbook streaming + simpan versi immutable + export markdown.
- **Dependency:** EP-10, EP-01, EP-09 (write tool `save_playbook_draft` opsional).
- **DoD:** versi immutable (generate baru = versi+1, lama tetap); konten terstruktur (jsonb) ter-render rapi.
- **Risiko:** —

### EP-15 — Report Generator `[P1 · M5]`
- **Tujuan:** laporan otomatis: Daily Opportunity Digest, Weekly Pipeline, Per-peluang.
- **Scope in:** entity `report`; `internal/ai` report service (agregasi pipeline/aktivitas → markdown via Hermes); endpoint `POST /api/reports` (type+period); FE Reports (Design §4.11): list, generate modal (tipe+periode) streaming, viewer terstruktur, export markdown/salin/hapus.
- **Scope out:** export PDF (P2).
- **Deliverables:** 3 tipe laporan ter-generate < 2 menit + tersimpan + export markdown.
- **Dependency:** EP-05/07/11, EP-01.
- **DoD:** `period_start ≤ period_end`; laporan berisi ringkasan + tabel pipeline + prospek prioritas + insight AI.
- **Risiko:** —

### EP-16 — Continuous Learning & Outcome Feedback `[P1 · M6]`
- **Tujuan:** WON/LOST + alasan reject → memory Hermes (session-key workspace) → memperkaya discovery/scoring/playbook berikutnya.
- **Scope in:** entity `outcome_event`; hook pada WON/LOST (EP-05/07) & Tolak discovery (EP-12) → kirim catatan ke memory Hermes via `internal/hermes`; reset memory (admin); cue UI "Asisten belajar dari aktivitas & hasil kamu".
- **Scope out:** —
- **Deliverables:** loop feedback aktif; bukti memory bertahan lintas restart Hermes.
- **Dependency:** EP-01, EP-05/07/12.
- **DoD:** outcome tercatat & terkirim ke memory; reset memory (admin) bekerja; chat berikutnya mempertimbangkan konteks.
- **Risiko:** —

### EP-17 — Telemetry, Observability, Audit & NFR Hardening `[P1 · M6]`
- **Tujuan:** penuhi NFR §11 + audit §9 + metrik §2.
- **Scope in:** telemetry event (`chat_opened`, review pursue, durasi report/scoring, outcome) → metrik §2; structured logging; **audit trail** sumber/waktu/akses/data + reasoning/evidence/model tiap output AI; health-check Hermes panel; performa (indeks, p95); contract tests final.
- **Scope out:** dashboard analytics lanjutan (P2).
- **Deliverables:** event telemetry terkirim & terukur; audit log lengkap; target NFR terverifikasi.
- **Dependency:** seluruh epic fungsional.
- **DoD:** metrik §2 terukur in-app; audit trail crawling & AI tersimpan; target performa §11 terpenuhi.
- **Risiko:** —

### EP-18 — Settings, Admin & Hermes Ops `[P0/P1 · M1/M6]`
- **Tujuan:** Settings: profil user, workspace, users (admin), AI/Hermes (status `/v1/capabilities`, memory, reset memory, test koneksi) **+ konfigurasi provider/model/API-key AI dari UI** (fleksibel seperti TUI Hermes).
- **Scope in:** FE Settings (Design §4.14) tab; endpoint status Hermes; reset memory (admin); test koneksi; manajemen user UI (memakai EP-03); **AI Provider Config**: pilih provider (**OpenAI / OpenRouter**), model (preset+custom), base_url opsional, API key (tersimpan terenkripsi, write-only di UI), enabled toolsets — disimpan di DB (source of truth) lalu di-push ke `hermes-bridge` via `internal/hermes.Configure`.
- **Scope out:** billing/multi-tenant (out-of-scope PRD).
- **Deliverables:** halaman Settings lengkap + status integrasi Hermes + pengaturan provider AI yang persisten & bisa diubah tanpa redeploy.
- **Dependency:** EP-01 (bridge `/admin/config` + `Configure`), EP-03, EP-16 (reset memory).
- **DoD:** status "Connected • vX" dari `/v1/capabilities`; gagal → badge merah + petunjuk; reset memory hanya admin; **ganti provider/model/key dari UI → Test koneksi hijau → chat berikutnya pakai provider baru tanpa restart**.
- **Risiko:** kebocoran API key → simpan terenkripsi (AES-GCM, `CONFIG_ENC_KEY`), jangan pernah kirim balik ke UI dalam bentuk plaintext.

---

## 4. Verifikasi tingkat Epic (acuan dari plan teknis §Verifikasi)
1. `docker compose up` → semua service hidup (EP-00).
2. Hermes `/v1/models` 200 + `/health` ok (EP-01).
3. Login → buat tender → tersimpan (EP-03/05).
4. Chat "tender prioritas minggu ini?" → terlihat tool-call `list_tenders` → jawaban ranking (EP-04/09).
5. Score tender/prospect → `prospect_score`/skor tersimpan (EP-10).
6. Isi Otak Agent → Aktifkan Agent → discovery run → tender baru di Inbox (EP-08/12).
7. Generate playbook & report → tersimpan & tampil (EP-14/15).
8. Tandai WON → restart Hermes → chat tetap ingat konteks (EP-16).
9. `go test ./internal/hermes/... -run Contract` hijau (EP-01/09/17).

---

## 5. Status & langkah berikut
- [x] **Layer 1 (EPIC)** — file ini.
- [x] **Layer 2 (STORY)** — `story.plan.md`: pecah tiap epic jadi user story + acceptance criteria + catatan teknis.
- [x] **Layer 3 (TASK)** — `task.plan.md`: pecah tiap story jadi task mikro siap eksekusi.
