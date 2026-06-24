# PRD — Sales AI Tool ("SalesPilot")

> **Status:** Draft v1.3 (MVP) · **Tanggal:** 2026-06-17
> **Bahasa:** Prosa Bahasa Indonesia, istilah teknis & nama field English.
> **Dokumen terkait:** [Design.md](./Design.md) (spesifikasi UI/desain detail untuk Claude design) · Plan teknis: `C:\Users\sheld\.claude\plans\saya-ingin-membuat-tools-effervescent-kahan.md`.

> **🏢 Sifat aplikasi:** SalesPilot adalah **tool internal untuk SATU perusahaan** (dipakai sendiri oleh tim sales/operasional kita), **bukan produk SaaS** yang dijual ke banyak perusahaan. Tidak ada signup publik, tidak ada billing/multi-tenant. Akun dibuat oleh Admin internal. **MVP single-organization murni** — tidak ada `organization_id` di schema; multi-tenant dipikirkan nanti hanya jika benar-benar dibutuhkan.

**Legenda prioritas:** **P0** = wajib MVP · **P1** = sebaiknya ada di MVP · **P2** = pasca-MVP.

**Perubahan v1.3:** Penegasan bahwa aplikasi ini **internal untuk satu perusahaan** (bukan SaaS multi-tenant) — disesuaikan di Ringkasan Produk, Persona, Auth, Non-goals, Scope, dan Keamanan. **`organization_id` dihapus dari model data** (single-org murni untuk MVP; multi-tenant ditunda sampai benar-benar perlu).
**Perubahan v1.2:** Hermes kini juga **menemukan tender via crawling**. Ditambah **Knowledge / Company Profile ("otak agent")** yang bisa diisi via **form lean atau upload PDF**, plus rubrik scoring & alur review (selaras kebutuhan RFI Procurement Intelligence Moonlay).

---

## 1. Ringkasan Produk

**SalesPilot** adalah **aplikasi internal** untuk tim **sales/operasional B2B di satu perusahaan** (dipakai sendiri, bukan dijual) yang membantu mereka **menemukan, menilai, dan memenangkan** peluang project (tender) — serta mengelola peluang dari **event** — sehingga fokus ke peluang dengan revenue tertinggi. Aplikasi menggabungkan:
- **CRUD sederhana** sebagai system of record (tender, event, prospek).
- **AI agent "otak"** (Hermes Agent) yang: **mencari tender otomatis** dari sumber yang ditentukan, **menilai/scoring** kelayakan, **menyusun playbook**, **membuat laporan**, dan menjawab via **chat** — serta **terus belajar** dari konteks & hasil (won/lost) seiring waktu.

Agar AI tidak mencari peluang asal-asalan, agent dipandu oleh **Knowledge / Company Profile**: kapabilitas perusahaan, kriteria target, no-go, sumber, dan keyword. Profil ini bisa diisi cepat (form lean) atau di-ekstrak dari PDF.

### Masalah yang dipecahkan
- Tender/peluang banyak & tersebar di banyak portal → **sulit & lama menemukannya**, lalu sulit menilai **mana yang layak dikejar**.
- Tidak ada **strategi/playbook** terstruktur per peluang.
- Pelaporan manual memakan waktu.
- Pengetahuan menang/kalah & kriteria perusahaan tidak terdokumentasi & tidak "dipelajari".

### Solusi
Satu workspace: isi **profil perusahaan sekali** → agent **mencari & menyaring tender** dari sumber yang disetujui → **scoring + rekomendasi** (Pursue/Review/Watchlist/Reject/Need Partner) → **review manusia** → playbook & laporan → chat asisten yang makin pintar.

### Value proposition
> "Berhenti mencari tender manual. Biarkan AI menemukan & menilai peluang terbaik untuk perusahaan kita — dengan strategi siap pakai dan asisten yang makin paham bisnis kita."

### Non-goals (sengaja BUKAN tujuan)
- **Bukan produk SaaS multi-tenant**: hanya untuk internal satu perusahaan; tanpa signup publik, tanpa billing, tanpa onboarding pelanggan eksternal.
- Bukan CRM penuh (tidak menggantikan email/telepon/kalender).
- Bukan sumber kebenaran legal kontrak (tanpa e-signature/manajemen dokumen hukum).
- **Tidak melakukan scraping ilegal**: tidak bypass CAPTCHA/paywall/login tanpa otorisasi (lihat §9).
- Tidak mengambil keputusan bid final tanpa manusia (human-in-the-loop).

---

## 2. Tujuan & Metrik Sukses

> Target = **hipotesis awal** untuk divalidasi 4–6 minggu. Diukur via **telemetry in-app**.

| Tujuan | Metrik | Baseline | Target MVP | Cara ukur |
|---|---|---|---|---|
| Temukan peluang lebih banyak & cepat | Jumlah tender relevan ditemukan/minggu | manual (sedikit) | ≥ 3× manual | hitung tender hasil discovery |
| Fokus ke peluang terbaik | % tender high-fit (≥80/"Pursue") yang direview ≤2 hari | — | ≥ 70% | event review pada tender Pursue |
| Hemat waktu administrasi | Waktu buat 1 laporan | ~30–60 mnt | < 2 mnt | durasi generate report |
| Strategi lebih siap | % peluang Pursue yang punya playbook | 0% | ≥ 50% | rasio playbook : peluang Pursue |
| Asisten dipakai | WAU yang membuka chat | — | ≥ 70% | event `chat_opened` |
| Win rate naik (jangka panjang) | Win rate | catat di bulan-1 | +relatif | `outcome_event` WON/total close |

**Definisi sukses MVP:** Setelah mengisi profil (atau upload PDF), agent menampilkan daftar tender relevan dengan skor & rekomendasi; sales bisa review → Pursue → generate playbook → buka chat tanya prioritas → generate laporan.

---

## 3. Persona & Peran Pengguna

> Semua persona adalah **karyawan internal** perusahaan yang sama. Tidak ada pengguna eksternal/pelanggan.

| Persona | Kebutuhan utama | Peran |
|---|---|---|
| **Sales / Account Manager** | Lihat tender hasil discovery, skor & playbook, chat | `SALES` |
| **Operasional / Pre-sales** | Atur profil perusahaan, sumber, keyword; review peluang | `OPS` |
| **Manager / Lead** | Pipeline, laporan, keputusan pursue/no-go | `MANAGER` |
| **Admin (IT/internal)** | Setup workspace, buat akun user, integrasi | `ADMIN` |

> **MVP:** **internal satu perusahaan** — single organization (1 workspace), multi-user internal, role sederhana. **Tanpa self-signup**: akun dibuat oleh Admin. **Tanpa `organization_id`/multi-tenant** — schema dibuat sesuai kebutuhan saat ini saja.

### 3.1 Permissions Matrix (ringkas)

| Capability | SALES | OPS | MANAGER | ADMIN |
|---|:--:|:--:|:--:|:--:|
| Lihat tender/event/prospek | ✅ | ✅ | ✅ | ✅ |
| CRUD tender/event/prospek | ✅ | ✅ | ✅ | ✅ |
| Edit Knowledge/Company Profile & Sumber | ❌ | ✅ | ✅ | ✅ |
| Jalankan/atur Discovery (crawling) | ❌ | ✅ | ✅ | ✅ |
| AI score/playbook/report & Chat | ✅ | ✅ | ✅ | ✅ |
| Keputusan Pursue/Reject/Need Partner | ✅ (miliknya) | ✅ | ✅ | ✅ |
| Kelola user & integrasi Hermes | ❌ | ❌ | ❌ | ✅ |

---

## 4. Ruang Lingkup (Scope)

### In-scope (MVP)
- **[P0]** Auth (JWT) + role dasar — **login internal saja**, akun dibuat Admin (tanpa registrasi mandiri).
- **[P0]** **Knowledge / Company Profile** (form lean) — kapabilitas, target, no-go, sumber, keyword.
- **[P1]** **Ingest PDF** profil/capability deck → ekstrak otomatis isi profil.
- **[P0]** Tender management (CRUD + status pipeline) + tampilan **Inbox Penemuan AI**.
- **[P1]** **Tender Discovery via Hermes** (crawling sumber yang disetujui, terjadwal).
- **[P0]** AI: Scoring + rekomendasi (Pursue/Review/Watchlist/Reject/Need Partner) + reasoning + evidence + risk flags.
- **[P0]** Event management (CRUD + konversi event → prospek).
- **[P0]** Prospect management (board/kanban + detail).
- **[P0]** Chat terhubung Hermes (streaming, baca data via tools).
- **[P1]** AI: Playbook generator (versi, immutable).
- **[P1]** AI: Report generator (digest harian, pipeline mingguan, per-peluang).
- **[P1]** Dashboard (pipeline + revenue + penemuan AI + prioritas).
- **[P1]** Continuous learning (memory Hermes + feedback won/lost + alasan reject).

### Out-of-scope (P2)
- Integrasi email/WhatsApp/kalender eksternal & multi-channel notifikasi.
- **Multi-tenant / billing / signup pelanggan eksternal** — aplikasi internal satu perusahaan; tidak akan dijual sebagai SaaS di MVP. Mobile native app.
- E-signature / manajemen dokumen legal; export PDF (MVP cukup markdown/preview).
- Draft proposal lengkap otomatis (MVP cukup playbook + brief).

---

## 5. Arsitektur Singkat (konteks fitur)

- **Frontend:** React + Vite. **Backend:** Go + Echo (CRUD + orkestrasi). **DB:** PostgreSQL.
- **AI "otak":** **Hermes Agent** = service terpisah, diakses lewat **API `/v1` OpenAI-compatible** (stabil). Hermes membaca data sales kita via **MCP tools**, dan menjalankan **crawling/discovery** (web/browser tools + jadwal/cron).
- **Anti-breaking-change:** coupling hanya ke `/v1` + skema MCP kita; anti-corruption layer; versi Hermes di-pin; contract tests.
- **Knowledge sync:** Company Profile = sumber kebenaran di DB kita; di-ekspos ke agent via MCP tool `get_company_profile` (agent selalu baca versi terbaru) + dipakai sebagai konteks discovery/scoring. Plus **memory Hermes** agar agent belajar lintas sesi.

---

## 6. Knowledge / Company Profile — "Otak Agent" (LEAN)

> **Prinsip:** jangan bikin user muak. Mayoritas field **opsional**, pakai **chip preset + default**, dan dukung **isi-otomatis dari PDF**. **Quick start < 2 menit.** Versi lengkap (RFI 22 halaman) di-distilasi jadi **6 kartu** berikut.

**Dua cara mengisi:**
- **A. Upload PDF** (capability deck / company profile / RFI) → Hermes ekstrak → field ter-isi → user **review & konfirmasi**. *(jalur tercepat)*
- **B. Form lean** — isi chip & beberapa angka.

**Kartu 1 — Profil Perusahaan**
- `company_name`* · `one_liner` (1 kalimat, opsional) · area upload PDF.

**Kartu 2 — Kapabilitas (yang dijual)** — chip preset multi-select:
- `service_categories`: Web App, System Integration, AI/Automation, Data/BI, Cloud/DevOps, Maintenance/Support, QA/Testing (+tambah).
- `tech_stack` (tags, opsional). → dasar **capability fit**.

**Kartu 3 — Target Peluang** — sedikit field, ada default:
- `countries` (chip; default: Indonesia) · `industries` (chip, opsional).
- `value_min`* + `value_ideal` (angka + currency; default min Rp 1.000.000.000).
- `deadline_min_days` (default 7) · `procurement_types` (chip: Open tender/RFP/RFQ/Vendor registration/Direct).

**Kartu 4 — Hindari (No-Go)** — toggle preset + free text:
- Toggle: Hardware murni · Embedded/IoT · Onsite penuh luar kota · Payment 100% after delivery · Sertifikasi khusus tak dimiliki · Deadline < minimum · Unpaid PoC besar.
- `nogo_custom` (free text, opsional).

**Kartu 5 — Sumber & Keyword (untuk crawling Hermes)** — preset siap pakai:
- `sources`: preset Indonesia bisa diaktifkan 1-klik (SPSE/Inaproc LKPP, eProc PLN, Pertamina, Telkom/SMILE, PaDi UMKM) + tambah URL. Tiap sumber: `name`, `url`, `country`, `access` (Publik/Login/Manual), `legal_note`, `enabled`.
- `keywords`: di-generate dari kapabilitas (bisa edit) + `negative_keywords` preset.
- `crawl_frequency`: Harian / 2–3x seminggu / Mingguan (default Harian utk prioritas tinggi).

**Kartu 6 — Scoring (Advanced, collapsed; sudah ada default)**
- Bobot default (lihat §8) + threshold rekomendasi. User boleh tweak via slider. Boleh dilewati.

> **Catatan UX:** user bisa selesai hanya dengan: upload PDF **atau** pilih beberapa chip kapabilitas + 1 negara + nilai minimum, lalu "Aktifkan agent". Sisanya default/auto.

---

## 7. Daftar Fitur (Epics, User Stories & Acceptance Criteria)

### E11 — Company Knowledge Profile **[P0; PDF ingest P1]**
- Sebagai Ops, saya bisa mengisi profil (form lean) atau **upload PDF** untuk diekstrak agent.
- **AC:** profil tersimpan & versi-an; upload PDF menghasilkan draft field yang bisa direview/diedit sebelum disimpan; agent memakai profil terbaru saat discovery & scoring.

### E12 — Tender Discovery via Hermes (crawling) **[P1]**
- Sebagai Ops, saya bisa menjalankan/menjadwalkan pencarian tender dari sumber yang disetujui; hasil masuk **Inbox Penemuan AI**.
- **AC:**
  - Menjalankan discovery menghasilkan `discovery_run` + daftar tender baru (status awal: hasil discovery, belum direview).
  - Tiap tender hasil punya: source, deadline, buyer, scope_summary, service_category, **fit_score**, **recommended_action**, risk_flags, reasoning.
  - **Deduplikasi** tender yang sama dari beberapa sumber.
  - Discovery **tidak** bypass CAPTCHA/login/paywall; sumber yang butuh login/manual ditandai, bukan dibobol (lihat §9).
  - Bisa dijadwalkan (mis. harian) dan menghormati rate limit.

### E1 — Tender Management **[P0]**
- CRUD tender; ubah status (`IDENTIFIED → QUALIFYING → BIDDING → SUBMITTED → WON/LOST`); picu Analisa AI & Playbook.
- **AC:** buat dgn min `title`; status tersimpan; hapus konfirmasi; tender hasil discovery bisa di-"promote" ke pipeline.

### E4 — AI Scoring & Rekomendasi **[P0]**
- AI memberi **fit score 0–100**, **recommended_action**, **confidence**, **reasoning**, **evidence per dimensi**, dan **risk_flags** — pakai rubrik §8.
- **AC:** hasil tersimpan (`prospect_score`/skor tender) + tampil (score ring, badge rekomendasi, evidence); bisa "Analisa ulang"; gagal AI → pesan ramah, data utuh.

### E2 — Event Management **[P0]** · E3 — Prospect Management **[P0]**
- (Sama seperti v1.1.) Event CRUD + konversi → prospek; Kanban prospek (`NEW→QUALIFIED→ENGAGED→PROPOSAL→WON/LOST`); drag-drop tersimpan.

### E7 — Chat Assistant (Hermes) **[P0]**
- Chat streaming; agent baca data via MCP tools; indikator tool-call; histori tersimpan; degrade graceful bila AI down.

### E5 — Playbook Generator **[P1]** · E6 — Report Generator **[P1]**
- Playbook terstruktur, versi immutable. Report: **Daily Opportunity Digest**, **Weekly Pipeline**, **Per-peluang** (markdown; PDF=P2).

### E8 — Dashboard **[P1]** · E9 — Continuous Learning **[P1]** · E10 — Auth & Settings **[P0]**
- Dashboard: penemuan AI hari ini, pipeline, revenue, prioritas. Learning: WON/LOST + alasan reject → memory Hermes. Settings: profil user, integrasi Hermes, reset memory (admin).

---

## 8. Spesifikasi Fitur AI (perilaku & rubrik)

1. **Discovery (crawling)** — agent memakai Company Profile (sumber, keyword, target, no-go) untuk mencari tender dari sumber yang disetujui; mengekstrak field; menilai; menyimpan ke Inbox. Hanya sumber legal; yang butuh login/manual ditandai.
2. **Scoring & rekomendasi (terstruktur)** — input: tender + Company Profile. Output JSON: `{ fit_score, recommended_action, confidence, reasoning, evidence[], risk_flags[] }`.
3. **Chat (brain + tools)** — agent baca data via MCP, jawab streaming, memory aktif.
4. **Playbook generator** — konteks peluang + playbook menang sebelumnya (memory) → sections terstruktur (versi baru).
5. **Report generator** — agregasi pipeline/aktivitas → markdown.
6. **Continuous-learning** — WON/LOST + alasan reject → memory Hermes (session-key workspace) → memperkaya discovery, scoring, playbook berikutnya.

### Rubrik Scoring (default, bisa di-tweak — selaras RFI)
| Dimensi | Bobot |
|---|---|
| Capability fit (cocok kapabilitas utama) | 20% |
| Portfolio match (ada pengalaman sejenis) | 15% |
| Commercial attractiveness (nilai/margin) | 15% |
| Eligibility fit (syarat legal/sertifikasi/pengalaman) | 15% |
| Deadline feasibility (cukup waktu proposal) | 10% |
| Strategic account value (buyer strategis) | 10% |
| Delivery risk (scope/onsite/dependency) | 10% |
| Competition / win probability | 5% |

### Ambang Rekomendasi
| Skor | Recommended action | Aksi |
|---|---|---|
| 80–100 | **Pursue** | Notifikasi prioritas tinggi, siap proposal/playbook |
| 65–79 | **Review** | Shortlist, validasi manual risiko/eligibility |
| 50–64 | **Watchlist** | Pantau, jangan keluarkan effort besar dulu |
| < 50 | **Reject** | Arsip + alasan (untuk pembelajaran) |
| (kondisi no-go) | **Need Partner / Auto No-Go** | Reject/tandai butuh partner sesuai no-go rule |

**Penanganan kegagalan AI:** semua fitur AI non-blocking terhadap CRUD; gagal → pesan ramah + retry; data utuh.

---

## 9. Data Privacy, Security & Kepatuhan Scraping

- **Aliran data ke AI:** kirim hanya data relevan. Untuk data sangat sensitif, workspace bisa pakai **model self-hosted** Hermes agar data tidak keluar.
- **Kepatuhan crawling (WAJIB):** prioritaskan API/RSS/export/portal resmi. **Tidak** bypass CAPTCHA/anti-bot/paywall; **tidak** akses akun tanpa otorisasi; hormati TOS & rate limit (frekuensi wajar + backoff). Sumber yang melarang → "Do not scrape"; yang butuh login → tandai "Login/Manual", jangan dibobol.
- **Credential:** tidak disimpan di profil/aplikasi sebagai plaintext; pakai vault/env. Field sumber hanya menyimpan jenis akses + PIC, bukan credential.
- **Human-in-the-loop:** keputusan bid/aksi eksternal selalu lewat review manusia.
- **Audit:** log sumber, waktu akses, data diambil, + reasoning/evidence/model tiap output AI.
- **Akses internal:** aplikasi hanya untuk karyawan perusahaan; tanpa akses publik/pelanggan. Akun dibuat Admin; idealnya di-deploy di jaringan/SSO internal.
- **Isolasi & retensi:** MVP single-org (semua data milik satu perusahaan, tanpa `organization_id`); user bisa hapus data & reset memory Hermes (admin); retensi data discovery dapat dikonfigurasi.
- **Redaction (P2):** filter field sensitif sebelum dikirim model.

---

## 10. Model Data (ringkas) & Validasi

Semua entity: `id`, `created_at`, `updated_at`. (`*` wajib.) — *single-org, tanpa `organization_id`.*

**Knowledge / Profile:**
| Entity | Field kunci |
|---|---|
| `company_profile` | company_name*, one_liner, service_categories[], tech_stack[], source_doc_refs[] |
| `target_criteria` | countries[], industries[], value_min*, value_ideal, value_max, currency, deadline_min_days, procurement_types[] |
| `nogo_rule` | preset_flags[], custom[] |
| `source` | name*, url*, country, access (publik/login/manual), legal_note, enabled, priority |
| `keyword_set` | category, keywords[], negative_keywords[], language |
| `discovery_run` | started_at, source_ids[], status, found_count, summary |

**Peluang & AI:**
| Entity | Field kunci |
|---|---|
| `tender` | title*, buyer_name, buyer_country, buyer_industry, value_estimate, currency, published_date, submission_deadline, source_name, source_url, service_category, scope_summary, eligibility_requirements, technical_requirements, status*, **fit_score**, **recommended_action** (PURSUE/REVIEW/WATCHLIST/REJECT/NEED_PARTNER), risk_flags[], reasoning_summary, dedup_key, origin (manual/discovery) |
| `event` | name*, type*, date, location, organizer, notes, status |
| `prospect` | name*, company, contact_info, source_type, source_id, stage*, est_value, owner_user_id |
| `prospect_score` | target_type*, target_id*, fit_score*, confidence, reasoning, evidence(jsonb), risk_flags(jsonb), model |
| `playbook` | target_type*, target_id*, title*, content(jsonb)*, version* |
| `report` | type*, period_start, period_end, content*, generated_by |
| `conversation`/`message` | session_key*, hermes_session_id / role*, content, tool_calls(jsonb) |
| `outcome_event` | target_type*, target_id*, result* (WON/LOST), notes |

**Validasi penting:** value_* ≥ 0; deadline_min_days ≥ 0; `recommended_action`/`status`/`access` ∈ enum; `source.url` valid; period_start ≤ period_end; dedup via `dedup_key` (hash buyer+title+deadline).

---

## 11. Kebutuhan Non-Fungsional

| Aspek | Target MVP |
|---|---|
| Performa | CRUD < 300ms p95; chat mulai tampil < 2s; scoring < 15s; discovery run async (background) |
| Keamanan | JWT; secret via env; MCP di-gate token; write-tools whitelist; kepatuhan scraping §9 |
| Reliabilitas | Health-check Hermes (`/health`,`/v1/capabilities`); degrade graceful; discovery idempotent (correlation/idempotency key) |
| Anti-breaking-change | Coupling hanya `/v1` + MCP kita; anti-corruption layer; pin versi; contract tests |
| Observability | Telemetry event (metrik §2) + structured logging + audit trail crawling |
| i18n & format | UI Bahasa Indonesia; Rupiah & tanggal lokal (Design.md §11) |
| Responsif | Desktop-first, usable di tablet |

---

## 12. Dependencies, Asumsi & Risiko

**Dependencies:** Hermes Agent sehat (gateway `/v1`, crawling/browser tools, cron/jadwal); model provider terkonfigurasi (atau self-hosted); PostgreSQL; memory provider (default Holographic).
**Asumsi:** single-org; volume MVP wajar; sumber utama portal Indonesia (LKPP/BUMN); user mengisi profil minimal sekali.

| Risiko | Mitigasi |
|---|---|
| Hermes upgrade → breaking | Coupling `/v1`; anti-corruption layer; pin versi; contract tests |
| Hermes/crawl gagal/lambat | Health-check + degrade; retry/backoff; CRUD tetap jalan |
| Sumber memblokir/legalitas | Hormati TOS; API/RSS/manual; "Do not scrape"; audit |
| Data sensitif ke LLM | Model self-hosted; minimisasi konteks; redaction (P2) |
| Output AI keliru/halusinasi | Selalu reasoning+evidence+confidence; human review; tandai "Dibuat AI" |
| Biaya/rate-limit LLM | Caching; scoring on-demand; frekuensi crawl wajar; model hemat |
| User muak isi field | **Profil lean + PDF ingest + default/preset** (§6) |

---

## 13. Roadmap / Fasing

| Milestone | Isi |
|---|---|
| **M0** | Scaffolding + jembatan Hermes + Chat passthrough (streaming) |
| **M1** | CRUD Tender/Event/Prospect + Auth + Frontend dasar |
| **M2** | **Knowledge/Company Profile (form lean)** + MCP `get_company_profile` |
| **M3** | MCP tools data + AI Scoring & rekomendasi + Dashboard |
| **M4** | **Tender Discovery via Hermes (crawling) + Inbox Penemuan AI** + **PDF ingest profil** |
| **M5** | Playbook + Report generator |
| **M6** | Continuous learning + telemetry + scheduling discovery + polish + contract tests |

---

## 14. Pertanyaan Terbuka

- Sumber awal yang difokuskan (Indonesia dulu: LKPP/SPSE + BUMN)? Perlu Singapore/SEA di MVP?
- Apakah cukup discovery harian, atau perlu near-real-time alert untuk deadline dekat?
- Template laporan spesifik yang diharapkan manajemen?
- Provider model default: cloud (cepat) vs self-hosted (privasi)?
- Perlu draft email klarifikasi/proposal brief di MVP (RFI menyebut), atau P2?

---

## 15. Glosarium

- **Tender/RFP/RFQ:** permintaan pengadaan dari buyer.
- **Discovery:** proses agent menemukan tender dari sumber yang disetujui.
- **Company/Knowledge Profile:** "otak" agent — kapabilitas, target, no-go, sumber, keyword.
- **Fit score:** skor 0–100 kelayakan peluang. **Recommended action:** Pursue/Review/Watchlist/Reject/Need Partner.
- **No-Go:** kondisi yang membuat peluang ditolak/diturunkan otomatis.
- **Need Partner:** peluang menarik tapi butuh partner (gap legal/lokasi/sertifikasi/kapabilitas).
- **Hermes Agent:** framework AI agent (otak) dengan memory/learning + crawling, diakses via `/v1`.
- **MCP:** protokol agent memanggil tools eksternal (tools data & profil kita).
- **Anti-corruption layer:** lapisan yang mengisolasi aplikasi dari perubahan API Hermes.
