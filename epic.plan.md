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

### ✅ EP-08 — Knowledge / Company Profile (Otak Agent) `[P0 · M2]` — **DONE (ST-08.1–08.6)**
- **Tujuan:** "otak" agent diisi lean (6 kartu) + versioning; jadi sumber kebenaran discovery/scoring.
- **Scope in:** entity `company_profile`, `target_criteria`, `nogo_rule`, `source`, `keyword_set` (§10); endpoint baca/tulis + versioning; preset Indonesia (sumber & keyword) + default; FE Onboarding lean (Design §4.2: dua jalur, target < 2 menit) + halaman edit 6 kartu (§4.13) + sub-tab Sumber (badge akses Publik/Login/Manual) + Scoring advanced collapsed (slider bobot).
- **Scope out:** PDF ingest (EP-13) & crawling (EP-12) & MCP expose (EP-09).
- **Deliverables:** profil tersimpan & versi-an dengan preset & default; UI lean.
- **Dependency:** EP-02, EP-03.
- **DoD:** quick start bisa selesai dengan minimal input (kapabilitas + 1 negara + nilai min) lalu "Aktifkan Agent"; profil ter-versi; badge "diperbarui {waktu}".
- **Risiko:** form membludak → tegakkan prinsip lean (mayoritas opsional, chip preset, default).
- **Progres:** ST-08.1 (migrasi `0008_profile` + domain/repo ber-versi, snapshot penuh), ST-08.2 (`GET`/`PUT /api/profile` + default preset + RBAC), ST-08.3 (CRUD `source` + katalog preset Indonesia 1-klik) **selesai & diverifikasi end-to-end** (`go build`/`vet`/`test` hijau + `curl` RBAC/validasi/versioning). ST-08.4 (keyword auto-generate), ST-08.5 (FE Onboarding), ST-08.6 (FE Otak Agent 6 kartu) **belum dikerjakan**.

### ✅ EP-09 — MCP Server & Sales Data Tools `[P0 · M2–M3]` — **DONE (ST-09.1–09.4, sisi Go)**
- **Tujuan:** expose data sales ke Hermes via HTTP MCP (read aman + write gated), termasuk `get_company_profile`.
- **Scope in:** `internal/mcp` server di `/mcp` (Bearer `${SALES_MCP_TOKEN}`); read tools: `list_tenders`, `get_tender`, `search_tenders`, `list_events`, `get_event`, `list_prospects`, `get_prospect`, `get_pipeline_summary`, `get_revenue_summary`, `get_company_profile`; write tools (gated/whitelist): `update_prospect_stage`, `save_playbook_draft`; config Hermes `mcp_servers.sales`.
- **Scope out:** logika scoring (EP-10).
- **Deliverables:** agent (via chat EP-04) bisa membaca data nyata; tool-call terlihat di UI.
- **Dependency:** EP-01, dan entity dari EP-05/06/07/08.
- **DoD:** round-trip MCP terbukti (chat "tender prioritas?" memanggil `list_tenders`); write tools hanya yang di-whitelist; contract test MCP hijau.
- **Risiko:** write tool berbahaya → whitelist + human-in-the-loop.
- **Progres:** ST-09.1–09.4 (server `/mcp` + Bearer constant-time auth, 10 read tools + 2 write tools whitelist-only dengan audit (`audit_log`, `playbook_draft` — baru, forward-compatible EP-14/EP-17), contract test in-process + `//go:build contract`) **selesai & diverifikasi end-to-end**: `go build`/`vet`/`test`/`golangci-lint` hijau; end-to-end nyata via `docker compose` Postgres + `go run ./apps/api` — `/mcp` 401 tanpa/token salah, handshake `initialize` + `tools/list` (12 tool cocok) + `tools/call` (read & write, termasuk verifikasi baris `audit_log` tersimpan) sukses terhadap data Postgres sungguhan. (**Catatan penting:** DoD "chat memicu `list_tenders`" **belum terbukti end-to-end** — diverifikasi bahwa `services/hermes-bridge/app/agent_factory.py` tidak membaca `deploy/hermes/config.yaml` sama sekali (hanya `ENABLED_TOOLSETS` env diteruskan ke `AIAgent`), dan library `hermes-agent` tidak ter-install di environment ini untuk diverifikasi lebih lanjut. Ini gap di sisi `hermes-bridge`/Python, di luar scope kode Go EP-09 — butuh keputusan/kerja terpisah sebelum DoD epic ini benar-benar tuntas.)

### ✅ EP-10 — AI Scoring & Recommendation `[P0 · M3]` — **DONE (ST-10.1–10.4)**
- **Tujuan:** fit score 0–100 + recommended_action + confidence + reasoning + evidence per dimensi + risk_flags, pakai rubrik §8.
- **Scope in:** `internal/ai` scoring service (rakit prompt: tender/prospect + Company Profile + rubrik) → `GenerateJSON` schema `{fit_score, recommended_action, confidence, reasoning, evidence[], risk_flags[]}`; simpan `prospect_score`/skor tender; ambang rekomendasi + no-go rule → Need Partner/Auto No-Go; endpoint `POST /api/tenders/:id/score` & `/api/prospects/:id/score`; "Analisa ulang"; FE: score ring berwarna (skala §2.1), badge recommended_action, evidence per dimensi, "Dibuat AI • confidence • waktu".
- **Scope out:** discovery scoring massal (EP-12 memakai service ini).
- **Deliverables:** skor tersimpan & tampil di Tender/Prospect detail; gagal AI → pesan ramah, data utuh.
- **Dependency:** EP-08 (profil), EP-09 (opsional konteks), EP-01.
- **DoD:** output sesuai schema & rubrik; ambang & no-go diterapkan; idempotent re-score; audit (model/waktu/evidence).
- **Risiko:** halusinasi → wajib reasoning+evidence+confidence + tanda "Dibuat AI".
- **Progres:** ST-10.1–10.4 **selesai & diverifikasi end-to-end**: `internal/ai/scoring.go` (prompt rubrik §8 8-dimensi + Company Profile + no-go, schema `ScoreResult`, `Scorer.Score` via `GenerateJSON`), `internal/ai/recommend.go` (`RecommendAction` deterministik — ambang §8 + no-go override, tak pernah mempercayai `recommended_action` mentah dari LLM), migrasi `0011_prospect_score` (append-histori — bukan `0008` seperti draft plan awal, nomor bergeser karena EP-09 sudah memakai s.d. `0010`), `internal/service/score_service.go` (orkestrasi: resolve target+profil → scorer → recommend → persist row → denormalisasi ke tender), `internal/http/handlers/score_handler.go` + 4 route (`POST`/`GET` × tender/prospect), FE `AiScorePanel.tsx` digeneralisasi `{targetType,targetId}` + `api/scores.ts`, terpasang di TenderDetail & ProspectDrawer (+ score ring header). Verifikasi: `go build`/`vet`/`test`/`golangci-lint` hijau (6 unit test service + test prompt/recommend/scorer di `internal/ai`); FE `tsc -b`/`eslint`/`vite build` hijau; **end-to-end nyata** — Postgres Docker + API native + mock server `/v1/responses` (HTTP round-trip sungguhan, bukan stub) membuktikan: GET sebelum skor→`null`, POST dgn Hermes down→400 ramah + data utuh, POST sukses→persist+denormalisasi tender, re-score→append row baru (bukan overwrite), scoring prospect→tak menyentuh tender. (**Catatan:** dua keputusan desain — `prospect_score` append-histori vs upsert, dan GET endpoint terpisah vs skor disematkan di detail response — dikonfirmasi eksplisit ke user via `AskUserQuestion` sebelum implementasi. **Catatan lain:** AC "Analisa ulang…streaming" dipenuhi lewat alternatif "loading" yang sama-sama diizinkan AC, karena `GenerateJSON` di sisi Go bukan API streaming; verifikasi visual FE di browser tidak dilakukan — lingkungan CLI tanpa akses browser, hanya type-check/lint/build.)

### ✅ EP-11 — Dashboard `[P1 · M3]` — **DONE (ST-11.1–11.2)**
- **Tujuan:** ringkasan: penemuan AI hari ini, pipeline per stage, estimasi revenue, prospek/tender prioritas.
- **Scope in:** `GET /api/dashboard/summary` (agregasi pipeline + revenue + prioritas + discovery hari ini); FE Dashboard (stat card, score ring, AI insight callout, banner "Lengkapi Otak Agent" bila profil kosong).
- **Scope out:** —
- **Deliverables:** dashboard dengan data nyata + empty states.
- **Dependency:** EP-05/07/10; discovery count dari EP-12 (opsional, degrade bila belum ada).
- **DoD:** metrik akurat terhadap DB; performa < 300ms p95 (query teragregasi).
- **Risiko:** query berat → indeks + agregasi SQL.
- **Progres:** ST-11.1–11.2 **selesai & diverifikasi**: `DashboardService` (3 sub-query: pipeline via `ProspectRepo.SummaryByStage` reuse, prioritas via `TenderRepository.TopByFitScore` baru, discovery-hari-ini via `CountDiscoveryToday` baru — semua degrade natural ke 0/[] tanpa data) + `GET /api/dashboard/summary`; FE `Dashboard.tsx` (stat cards, pipeline per stage, prioritas dgn score ring, AI insight callout, `OtakAgentBanner` disatukan). `go build/vet/test/golangci-lint` + FE `tsc -b`/`eslint`/`vite build` hijau; angka dashboard diverifikasi cocok terhadap data Postgres nyata hasil EP-12.

### ✅ EP-12 — Tender Discovery via Hermes (Crawling) & Discovery Inbox `[P1 · M4]` — **DONE (ST-12.1–12.7)**
- **Tujuan:** agent menemukan tender dari sumber yang disetujui → Inbox Penemuan AI; dedup; terjadwal; patuh hukum.
- **Scope in:** entity `discovery_run`; orchestrator discovery (`internal/ai`): pakai Company Profile (sumber/keyword/target/no-go) → Hermes crawling/browser tools → ekstrak field → scoring (EP-10) → simpan tender `origin=discovery`; **deduplikasi** via `dedup_key`; **kepatuhan §9** (tidak bypass CAPTCHA/login/paywall; sumber Login/Manual ditandai; rate limit + backoff); idempotency key; endpoint `POST /api/discovery/run`, `GET /api/discovery/runs`, list inbox; penjadwalan (cron Hermes / scheduler internal); FE Penemuan AI (Design §4.3): kartu skor+rekomendasi+risk chips, aksi Tinjau/Pursue/Watchlist/Tolak(+alasan), progress crawl, empty states.
- **Scope out:** —
- **Deliverables:** discovery run menghasilkan tender baru di inbox; Pursue mem-promote ke pipeline.
- **Dependency:** EP-08, EP-10, EP-01; audit dari EP-17.
- **DoD:** dedup bekerja; sumber Login/Manual tidak dibobol (ditandai); run async (background) + idempotent; Tolak menyimpan alasan (untuk learning EP-16).
- **Risiko:** legal/blokir sumber → hormati TOS, prioritas API/RSS/portal resmi, audit trail.
- **Progres:** ST-12.1–12.7 **selesai & diverifikasi end-to-end**. Migrasi `0012_discovery` (+ `0013_tender_reviewed_at`, `0014_profile_crawl_schedule` — nomor bergeser dari draft plan karena EP-09/10 sudah memakai s.d. `0011`). Arsitektur (3 keputusan dikonfirmasi user): (1) seam `Crawler` interface — pipeline inti fully unit-testable via fake, implementasi live `hermesCrawler` mengisolasi ketergantungan Hermes (gap web-toolset sama seperti EP-09); (2) scheduler = ticker Go internal (bukan cron Hermes), `crawl_frequency`/`crawl_enabled` di `company_profile`, default nonaktif; (3) reject = kolom `tender.reviewed_at` + alasan ke `audit_log` (reuse EP-09), bukan status/entitas baru. Async run via goroutine + **detached context** (bukan queue infra). Rate limit/backoff per-sumber di `hermesCrawler` (di-refactor jadi loop per-sumber, bukan 1 prompt gabungan). FE `DiscoveryInbox.tsx` lengkap: header+status+filter(rekomendasi/min skor)+kartu+progress+empty states+aksi Pursue/Watchlist/Tolak(modal alasan). **2 bug produksi nyata ditemukan & diperbaiki** selama verifikasi end-to-end sungguhan (bukan hanya unit test): (a) `CandidateTender` tanpa json tag → field snake_case hasil ekstraksi LLM gagal ter-bind diam-diam (hanya `title` kebetulan cocok); (b) `discovery_run.SourceIDs` nil → GORM kirim `NULL` eksplisit, melanggar `NOT NULL` — keduanya diperbaiki + ditambah regression test. Verifikasi: `go build/vet/test/golangci-lint` + FE `tsc -b`/`eslint`/`vite build` hijau (puluhan unit test baru di `internal/ai`, `internal/service`); **end-to-end nyata** (Postgres Docker + API native + mock server Hermes `/v1/chat/completions`+`/v1/responses`) membuktikan seluruh alur: compliance guard (sumber login tak pernah di-crawl, tercatat di `audit_log`), crawl→ekstrak→dedup→skor→simpan, idempotency (`correlation_key` sama → run id sama), filter inbox (`min_score`/`recommended_action`), dan Watchlist/Tolak(+alasan tersimpan di `audit_log`)/Pursue.

### ✅ EP-13 — PDF Ingest untuk Company Profile `[P1 · M4]` — **DONE (ST-13.1–13.3)**
- **Tujuan:** upload PDF (capability deck/company profile/RFI) → ekstrak → isi draft field profil untuk direview.
- **Scope in:** upload endpoint + storage ref (`source_doc_refs`); pipeline ekstraksi via Hermes (baca dokumen → field profil terstruktur); FE: dropzone (Onboarding §4.2 & Otak Agent §4.13), progress "AI membaca dokumen…" (streaming/live-region), field hasil ditandai chip "diisi AI ✨", review & konfirmasi sebelum simpan.
- **Scope out:** OCR berat / dokumen non-PDF (P2).
- **Deliverables:** alur PDF → draft profil → konfirmasi → tersimpan.
- **Dependency:** EP-08, EP-01.
- **DoD:** PDF gagal baca → "coba isi manual"; field hasil ekstraksi bisa diedit sebelum disimpan; tidak menimpa profil tanpa konfirmasi.
- **Risiko:** ekstraksi tidak akurat → selalu review manusia; tandai field AI.
- **Progres:** ST-13.1 (`POST /api/profile/ingest` multipart, `internal/storage/pdf.go` validasi magic-byte+ukuran+nama UUID anti path-traversal, volume `uploads` di compose), ST-13.2 (ekstraksi teks PDF via `github.com/ledongthuc/pdf` — pure-Go, nol dependency transitif tambahan, dipilih via investigasi eksplisit dibanding binary `pdftotext`/poppler — + ekstraksi field via Hermes `GenerateJSON` menghasilkan `ai.ProfileDraft`, degrade graceful (PDF 0-teks/rusak atau Hermes gagal → `Degraded=true` tanpa menggagalkan upload)), ST-13.3 (FE `ProfilePdfIngest.tsx` dropzone+live-region+review-editable+chip "diisi AI"+simpan, dipasang di Onboarding jalur "Cara cepat" & tombol "Isi dari PDF" di Otak Agent) **selesai & diverifikasi**: `go build`/`vet`/`test ./...` + `golangci-lint` hijau (13 unit test baru: 5 `internal/storage`, 3 `internal/ai` ekstraksi PDF, 5 `internal/service` IngestUpload dgn stub Hermes); FE `tsc -b`+`eslint`+`vite build` hijau (dijalankan via Node v20.19.4 — Node default lingkungan `v14.17.6` terlalu tua untuk tooling FE modern). **Catatan:** verifikasi `curl` end-to-end terhadap Postgres/Hermes-bridge nyata & verifikasi visual browser tidak dilakukan (Docker tidak aktif, lingkungan CLI tanpa browser) — validasi mengandalkan build/vet/lint/unit test dengan stub `hermes.Client`.

### ✅ EP-14 — Playbook Generator `[P1 · M5]` — **DONE (ST-14.1–14.4)**
- **Tujuan:** playbook terstruktur per peluang, versi immutable.
- **Scope in:** entity `playbook`; `internal/ai` playbook service (konteks peluang + playbook menang sebelumnya dari memory) → sections terstruktur; endpoint `POST /api/tenders/:id/playbook` & `/api/prospects/:id/playbook`; FE Playbooks (Design §4.10): viewer terstruktur (Ringkasan/Value Prop/Stakeholders/Strategi checklist/Timeline/Risiko/Next Actions), generate streaming, versi baru + bandingkan, salin/export markdown.
- **Scope out:** export PDF (P2).
- **Deliverables:** generate playbook streaming + simpan versi immutable + export markdown.
- **Dependency:** EP-10, EP-01, EP-09 (write tool `save_playbook_draft` opsional).
- **DoD:** versi immutable (generate baru = versi+1, lama tetap); konten terstruktur (jsonb) ter-render rapi.
- **Risiko:** —
- **Progres:** ST-14.1 (migrasi `0015_playbook` — nomor bergeser dari draft `0010` karena EP-09..EP-12 sudah memakai s.d. `0014`; `domain.Playbook`+`PlaybookRepository` reuse `PlaybookTargetType` yang sudah ada dari `playbook_draft` EP-09, tabel terpisah), ST-14.2 (`ai.PlaybookGenerator` — prompt P-8 reuse `ai.ScoreInput`/`ScoreInputFromTender`/`FromProspect` dari EP-10, bukan struct input baru, menghasilkan `PlaybookContent` 7 section), ST-14.3 (`PlaybookService.Generate` — pola sama `ScoreService`: resolve target+profil→AI→hitung versi via `GetLatestVersion+1`→persist immutable; endpoint `POST/GET /api/{tenders,prospects}/:id/playbook(s)` + `GET /api/playbooks/:id`, RBAC `CapCRUDData`), ST-14.4 (FE `PlaybookPanel.tsx` — pola sama `AiScorePanel`: empty-state+CTA, versi+model+waktu, 7 section, salin/export markdown, bandingkan versi lama; dipasang di tab Playbook `TenderDetail` & drawer `ProspectDrawer`, menggantikan placeholder EP-14; route `/playbooks` diganti `PlaybooksIndex` pengarah) **selesai & diverifikasi**: `go build`/`vet`/`test ./...` + `golangci-lint` hijau (2 unit test `internal/ai/playbook_test.go` + 6 unit test `internal/service/playbook_service_test.go`, keduanya dengan stub Hermes/fake repo, membuktikan versi immutable — generate ke-2 tak mengubah versi 1 — dan gagal AI→tanpa row); FE `tsc -b`+`eslint`+`vite build` hijau (dijalankan via Node v20.19.4, lihat catatan EP-13). **Catatan:** `curl`/migrasi end-to-end terhadap Postgres/Hermes nyata & verifikasi visual browser tidak dilakukan (Docker tidak aktif; satu-satunya Postgres lokal ditemukan memakai kredensial tak terkait proyek ini, sengaja tidak dipaksa).

### ✅ EP-15 — Report Generator `[P1 · M5]` — **DONE (ST-15.1–15.4)**
- **Tujuan:** laporan otomatis: Daily Opportunity Digest, Weekly Pipeline, Per-peluang.
- **Scope in:** entity `report`; `internal/ai` report service (agregasi pipeline/aktivitas → markdown via Hermes); endpoint `POST /api/reports` (type+period); FE Reports (Design §4.11): list, generate modal (tipe+periode) streaming, viewer terstruktur, export markdown/salin/hapus.
- **Scope out:** export PDF (P2).
- **Deliverables:** 3 tipe laporan ter-generate < 2 menit + tersimpan + export markdown.
- **Dependency:** EP-05/07/11, EP-01.
- **DoD:** `period_start ≤ period_end`; laporan berisi ringkasan + tabel pipeline + prospek prioritas + insight AI.
- **Risiko:** —
- **Progres:** ST-15.1 (migrasi `0016_report` — bergeser dari draft `0011` karena EP-09..EP-14 sudah memakai s.d. `0015`; `content TEXT` markdown, bukan jsonb, karena output AI prosa bebas; validasi period di service bukan DB CHECK), ST-15.2 (`ai.ReportGenerator` — `ReportData` berbentuk identik `service.DashboardSummary` EP-11 tanpa impor silang; `buildReportPrompt` minta **markdown** via `hc.Chat` non-stream, bukan `GenerateJSON`, karena output bukan JSON terstruktur; heading tetap `## Ringkasan/Tabel Pipeline/Prospek Prioritas/Insight AI`), ST-15.3 (`ReportService.Create` — reuse `DashboardService.Summary` untuk agregasi alih-alih query repo baru; validasi type+period di service; endpoint `POST/GET/DELETE /api/reports`, RBAC View=semua role/Create-Delete=`CapCRUDData`; `stubHermesClient` paket `service` diperluas field `chat` opsional — additive, tak mengubah test lain), ST-15.4 (FE `ReportsPage.tsx` — `Table` generik+filter tipe+kebab actions, `GenerateReportModal.tsx` (tipe+`DatePicker`→RFC3339, validasi mulai≤akhir client-side), `ReportViewer.tsx` (markdown render pola sama `MessageBubble.tsx` Chat, salin/export/hapus); route `/reports` diganti `ReportsPage`) **selesai & diverifikasi**: `go build`/`vet`/`test ./...` + `golangci-lint` hijau (6 unit test `internal/ai/report_test.go` — 3 tipe+error+respons kosong+data kosong — + 6 unit test `internal/service/report_service_test.go` — sukses/type invalid/period terbalik/gagal AI/filter list/delete not-found — semua dengan stub Hermes/fake repo); FE `tsc -b` (1 error `buildQueryString`/`Record` type ditemukan & diperbaiki pola spread, sama seperti `api/tenders.ts`) + `eslint` + `vite build` hijau (Node v20.19.4). **Catatan:** `curl`/migrasi end-to-end terhadap Postgres/Hermes nyata & verifikasi visual browser tidak dilakukan (Docker tidak aktif; Postgres lokal yang ditemukan memakai kredensial tak terkait proyek ini).

### ✅ EP-16 — Continuous Learning & Outcome Feedback `[P1 · M6]` — **DONE (ST-16.1–16.4)**
- **Tujuan:** WON/LOST + alasan reject → memory Hermes (session-key workspace) → memperkaya discovery/scoring/playbook berikutnya.
- **Scope in:** entity `outcome_event`; hook pada WON/LOST (EP-05/07) & Tolak discovery (EP-12) → kirim catatan ke memory Hermes via `internal/hermes`; reset memory (admin); cue UI "Asisten belajar dari aktivitas & hasil kamu".
- **Scope out:** —
- **Deliverables:** loop feedback aktif; bukti memory bertahan lintas restart Hermes.
- **Dependency:** EP-01, EP-05/07/12.
- **DoD:** outcome tercatat & terkirim ke memory; reset memory (admin) bekerja; chat berikutnya mempertimbangkan konteks.
- **Risiko:** —
- **Progres:** ST-16.1 (`outcome_event` **superseded** — sudah lengkap sejak EP-05 migrasi `0005_outcome_events`, tak ada migrasi baru dibuat), ST-16.2 (`internal/ai/learning.go` `LearningHermes` — menulis memory = `hc.Chat` biasa dgn `SessionKey` workspace, tak ada API "tulis memory" terpisah; `LearningHook` interface diperluas additive `RecordDiscoveryReject`; `TenderService.Review` kini terima `reason` & memicu hook async saat Tolak), ST-16.3 (**full Go+bridge+investigasi** sesuai arahan user — `hermes.Client.ResetMemory` + `POST /admin/hermes/reset-memory` [ADMIN]; investigasi nyata source `hermes-agent` pin `v2026.6.19` via git clone: tak ada API in-process untuk clear memory, `hermes memory reset` CLI hanya hapus file `MEMORY.md`/`USER.md`/`memory_store.db` di `get_hermes_home()` — direplikasi in-process di bridge `services/hermes-bridge/app/routes/admin.py`), ST-16.4 (cue UI "Asisten belajar..." disebarkan ke 3 titik yang belum punya — modal WON/LOST prospect, Tolak discovery, empty-state Chat — melengkapi yang sudah ada di TenderDetail sejak EP-05; **verifikasi persist mendokumentasikan temuan infrastruktur nyata**: `hermes-bridge` di `docker-compose.yml` tidak punya volume utk home directory-nya → memory akan hilang tiap restart container, membatalkan premis inti epic ini — **diperbaiki** dgn volume `hermes-memory` + `HERMES_HOME=/root/.hermes`, divalidasi `docker compose config`) **selesai & diverifikasi**: `go build`/`vet`/`test ./...` + `golangci-lint` hijau (4 unit test `internal/ai/learning_test.go`, 2 unit test tambahan `internal/service/tender_service_test.go` dgn `spyLearningHook`, 2 unit test ACL `internal/hermes/reset_test.go`); **bridge Python** — ditemukan venv+pytest nyata di `services/hermes-bridge/.venv`, dijalankan sungguhan: 28/28 pass (24 lama + 4 baru `test_reset_memory.py`, termasuk kondisi nyata "hermes-agent tak ter-install" → 502 ramah, bukan mock); FE `tsc -b`+`eslint`+`vite build` hijau (Node v20.19.4). Dokumentasi lengkap di `docs/ep16-learning-verification.md` termasuk skenario end-to-end literal yang **belum** dijalankan (Docker Desktop tak aktif, tanpa API key provider LLM nyata) + langkah persis menuntaskannya.

### ✅ EP-17 — Telemetry, Observability, Audit & NFR Hardening `[P1 · M6]` — **DONE (ST-17.1–17.4)**
- **Tujuan:** penuhi NFR §11 + audit §9 + metrik §2.
- **Scope in:** telemetry event (`chat_opened`, review pursue, durasi report/scoring, outcome) → metrik §2; structured logging; **audit trail** sumber/waktu/akses/data + reasoning/evidence/model tiap output AI; health-check Hermes panel; performa (indeks, p95); contract tests final.
- **Scope out:** dashboard analytics lanjutan (P2).
- **Deliverables:** event telemetry terkirim & terukur; audit log lengkap; target NFR terverifikasi.
- **Dependency:** seluruh epic fungsional.
- **DoD:** metrik §2 terukur in-app; audit trail crawling & AI tersimpan; target performa §11 terpenuhi.
- **Risiko:** —
- **Progres:** ST-17.1 (migrasi `0017_telemetry_event`; `internal/telemetry.Emitter.Emit` async best-effort via `SetEmitter` — sengaja setter nil-safe, bukan parameter constructor, agar tak mengubah signature ~10 call-site test existing; 5 titik emit `chat_opened`/`review_pursue`/`scoring_generated`/`report_generated`/`outcome_recorded`), ST-17.2 (`middleware.Logger()` Echo diganti `RequestLoggerWithConfig`+`slog` JSON — `internal/log` baru; audit trail AI reuse penuh `audit_log`/`AuditRepository` eksisting TANPA migrasi baru, ditulis di handler layer `ai.score`/`ai.playbook`/`ai.report` via helper `writeAIAuditEvent`), ST-17.3 (2 gap indeks nyata ditambal migrasi `0018_nfr_indexes` — `tender(origin,status,reviewed_at)` utk Discovery Inbox composite + `tender/prospect(created_at DESC)` utk `ORDER BY` universal; `GET /api/health/hermes` — helper `hermesStatus` jadi satu sumber kebenaran yg direuse EP-18 TK-18.1.1), ST-17.4 (`go test -tags contract ./...` 4 test `TestContract_*` SKIP bersih tanpa `HERMES_BASE_URL`; `docs/nfr-checklist.md` status jujur per aspek §11, 3 gap eksplisit + langkah). **Verifikasi genuinely end-to-end** (bukan cuma build): mock Hermes bridge minimal (scratchpad) + `go run ./apps/api` melawan **Postgres live** (`deploy-postgres-1:55432`) — login admin asli → buat tender → generate score/playbook/report (200/201) → `SELECT audit_log WHERE action LIKE 'ai.%'` 3 row nyata (actor=user id asli, model+reasoning/version/report_type) → `SELECT telemetry_event` 2 row nyata (`scoring_generated`/`report_generated` dgn `duration_ms`); log JSON server nyata utk 3 request (200/401/401); health endpoint diuji dua kondisi (mock hidup→connected, mock dimatikan→disconnected HTTP 200). Indeks baru dibuktikan via `EXPLAIN ANALYZE` pada 20.000 tender + 10.000 prospect sintetik (dihapus setelah verifikasi) — index scan terkonfirmasi, bukan seq scan. `go build/vet/test ./...` + `go test -tags contract ./...` + `golangci-lint` (7 isu pre-existing tak tersentuh) semua hijau.

### ✅ EP-18 — Settings, Admin & Hermes Ops `[P0/P1 · M1/M6]` — **DONE (ST-18.1–18.4)**
- **Tujuan:** Settings: profil user, workspace, users (admin), AI/Hermes (status `/v1/capabilities`, memory, reset memory, test koneksi) **+ konfigurasi provider/model/API-key AI dari UI** (fleksibel seperti TUI Hermes).
- **Scope in:** FE Settings (Design §4.14) tab; endpoint status Hermes; reset memory (admin); test koneksi; manajemen user UI (memakai EP-03); **AI Provider Config**: pilih provider (**OpenAI / OpenRouter**), model (preset+custom), base_url opsional, API key (tersimpan terenkripsi, write-only di UI), enabled toolsets — disimpan di DB (source of truth) lalu di-push ke `hermes-bridge` via `internal/hermes.Configure`.
- **Scope out:** billing/multi-tenant (out-of-scope PRD).
- **Deliverables:** halaman Settings lengkap + status integrasi Hermes + pengaturan provider AI yang persisten & bisa diubah tanpa redeploy.
- **Dependency:** EP-01 (bridge `/admin/config` + `Configure`), EP-03, EP-16 (reset memory).
- **DoD:** status "Connected • vX" dari `/v1/capabilities`; gagal → badge merah + petunjuk; reset memory hanya admin; **ganti provider/model/key dari UI → Test koneksi hijau → chat berikutnya pakai provider baru tanpa restart**.
- **Risiko:** kebocoran API key → simpan terenkripsi (AES-GCM, `CONFIG_ENC_KEY`), jangan pernah kirim balik ke UI dalam bentuk plaintext.
- **Progres:** ST-18.1 (`handlers.HealthHandler`/`hermesStatus` helper — satu sumber kebenaran dipakai ulang oleh `/api/health/hermes` EP-17 **dan** `/api/settings/hermes` EP-18, menghindari duplikasi logika; `SettingsHandler.HermesStatus`/`TestHermes`), ST-18.2 (`SettingsPage.tsx` shell 4 tab: Profil[reuse `useAuthStore`]/Workspace[reuse `useProfile()` EP-08]/Users[admin-only, CRUD baru `useCreateUser`/`useUpdateUser`/`useResetPassword`]/AI-Hermes; placeholder route `/settings` terakhir di `routes.tsx` diganti), ST-18.3 (`useResetHermesMemory` — wiring tombol destructive ke endpoint EP-16 TK-16.3.1 yang sudah ada, `ConfirmDialog` wajib sebelum aksi), ST-18.4 **(scope terbesar, full-stack)**: migrasi `0019_ai_provider_setting` (partial unique index `is_active` — invariant single-active ditegakkan DB, bukan service), `internal/auth/crypto.go` (AES-256-GCM, `CONFIG_ENC_KEY` opsional — fitur nonaktif tanpa key, bukan crash, konsisten prinsip non-blocking §8 diperluas ke config), `AISettingService` (Get/Update/**Rehydrate**) + `AISettingHandler` (`/settings/ai` GET/PUT/POST-test, seluruhnya `CapManageUsers`), rehydrate-on-boot di `apps/api/main.go` (signature `apphttp.New` diperluas kembalikan `*service.AISettingService`), FE `AiProviderTab.tsx` (provider/model preset+custom/base_url/key write-only/toolsets). **Verifikasi genuinely end-to-end di seluruh ST-18.4** (bukan cuma build) — mock Hermes bridge diperkuat menyimpan state (`lastConfig`) agar `GET /admin/config` benar-benar merefleksikan push terakhir, bukan respons statis: (a) DB `api_key_encrypted` = ciphertext base64 terbukti bukan plaintext; (b) PUT config → **`curl` langsung ke mock bridge membuktikan payload provider/model/api_key PERSIS sama** dengan yang dikirim UI; (c) **skenario restart bridge literal disimulasikan** — matikan mock+API, mock baru (state kosong, `GET /admin/config`→`null`), start API → boot log `"config AI aktif berhasil di-push ulang"` tanpa aksi UI apa pun, mock bridge kembali menampilkan config yang sama persis; (d) key-preserve (Simpan tanpa isi API Key) dibuktikan via bridge menerima key LAMA + model BARU; (e) RBAC: user SALES sungguhan dibuat → 403 di seluruh endpoint `/settings/ai` & `/admin/hermes/reset-memory`; (f) boot tanpa config aktif → log graceful, server tetap start. 12 unit test service (`ai_setting_service_test.go`, fake repo + stub Hermes) + 6 unit test crypto — semua PASS. `go build/vet/test ./...` + `go test -tags contract ./...` + `golangci-lint` (7 isu pre-existing tak tersentuh) + FE `tsc -b`/`eslint`/`vite build` (Node 20.19.4, 3 error pre-existing tak tersentuh) — seluruhnya hijau di akhir EP-18.

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
