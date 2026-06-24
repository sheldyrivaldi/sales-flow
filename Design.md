# Design.md — SalesPilot UI/UX Specification

> **Tujuan dokumen:** Spesifikasi desain lengkap untuk men-generate UI (mis. via Claude design). Menjelaskan **semua capability, layar, komponen, state, flow, dan design system** secara detail.
> **Versi:** 1.3 · **Pendamping:** [PRD.md](./PRD.md) (kebutuhan produk). **Bahasa UI:** Bahasa Indonesia.
> **Platform:** Web app, desktop-first, responsive sampai tablet.
> **🏢 Sifat aplikasi:** **tool internal satu perusahaan** (dipakai sendiri oleh tim sales/ops kita), **bukan SaaS**. UI tidak perlu signup publik, halaman pricing/marketing, atau onboarding pelanggan; akun dibuat Admin internal. "Company Profile / Otak Agent" = profil **perusahaan kita sendiri**.
> **Perubahan v1.3:** Penegasan konteks **internal satu perusahaan** — hilangkan self-signup di Login, tegaskan akun dikelola Admin di Settings/Users.
> **Perubahan v1.2:** + layar **Knowledge / Otak Agent** (setup lean + upload PDF), **Inbox Penemuan AI** (hasil crawling Hermes), manajemen **Sumber**, badge **rekomendasi** & **risk flags** di tender, prinsip **lean input**.

---

## 0. Ringkasan Produk (untuk konteks desain)

**SalesPilot** = AI tool **internal** untuk tim sales/operasional B2B di satu perusahaan: **menemukan tender otomatis** (crawling oleh agent), **menilai** (scoring + rekomendasi), mengelola **event** & **prospek**, menyusun **playbook**, membuat **laporan**, dan **chat** dengan AI agent yang **terus belajar**. Nuansa: **profesional, fokus, cerdas, ringan** — bukan CRM berat, bukan produk SaaS yang dijual.

Agent dipandu **Knowledge / Company Profile** ("otak") yang diisi user **secara ringkas** (form lean) atau **upload PDF**. Sales harus merasa "dibantu AI yang paham bisnisnya", bukan dibebani form.

**Tone visual:** modern SaaS, bersih, banyak whitespace, data jelas, AI hadir tapi tidak mengganggu.

### Brand & Logo
- **Nama:** SalesPilot. **Tagline:** "AI copilot untuk sales".
- **Konsep logo:** ikon kompas/navigasi + sparkles (AI), gradient indigo→violet. Wordmark Inter Semibold. Monokrom putih untuk header gelap.

---

## 1. Design Principles

1. **Clarity over density** — info penting dulu; hindari tabel padat.
2. **AI as a quiet copilot** — saran AI ditandai jelas (aksen violet), selalu bisa dilihat alasannya (reasoning/evidence), bukan kotak hitam.
3. **Action-oriented** — tiap layar 1 aksi utama jelas (primary CTA).
4. **Progressive disclosure** — detail/advanced disembunyikan di balik expand.
5. **Trust & transparency** — output AI tampilkan confidence + evidence + waktu/model.
6. **Fast & forgiving** — skeleton, optimistic update, empty state membantu.
7. **Lean input (anti-muak)** — mayoritas field **opsional**, pakai **chip preset + default**, dukung **isi-otomatis dari PDF**. Setup awal harus bisa selesai **< 2 menit**. Jangan pernah menampilkan form panjang sekaligus.

---

## 2. Design System

### 2.1 Color Palette
- **Primary/Brand:** Indigo `#4F46E5` · hover `#4338CA`.
- **Accent/AI:** Violet `#7C3AED` (badge "AI", bubble assistant, panel AI).
- **Success (WON/high-fit/Pursue):** Emerald `#10B981`.
- **Warning (deadline dekat/Review):** Amber `#F59E0B`.
- **Danger (LOST/overdue/Reject):** Rose `#EF4444`.
- **Info/Watchlist:** Sky `#0EA5E9`. **Need Partner:** Violet/Indigo.
- **Neutral text:** `#0F172A`/`#475569`/`#94A3B8`. **Surface:** `#FFFFFF`/`#F8FAFC`/`#F1F5F9`. **Border:** `#E2E8F0`.
- **Dark mode (P2):** bg `#0B1120`, surface `#111827`, border `#1F2937`.

**Score color scale (fit_score):** 0–49 Rose, 50–64 Sky, 65–79 Amber, 80–100 Emerald (selaras ambang rekomendasi).

**Recommended action badge:** Pursue=Emerald · Review=Amber · Watchlist=Sky · Reject=Rose · Need Partner=Violet.

**Stage pill — tender:** IDENTIFIED slate, QUALIFYING sky, BIDDING indigo, SUBMITTED violet, WON emerald, LOST rose. **prospect:** NEW slate, QUALIFIED sky, ENGAGED indigo, PROPOSAL violet, WON emerald, LOST rose.

### 2.2 Typography
Inter. H1 28–32 bold, H2 22, H3 18, body 14–15, caption 12. Angka (revenue/score): tabular, semibold.

### 2.3 Spacing & Layout
Base 4px; scale 4/8/12/16/24/32/48. Radius card 12, button/input 8, pill 999. Shadow subtle `0 1px 3px rgba(15,23,42,.08)`. Container max 1280, gutter 24.

### 2.4 Komponen dasar (library)
Button (primary/secondary/ghost/danger; sm/md/lg; ikon; loading) · Input/Textarea/Select/DatePicker/Combobox · **Chip/Tag input** (preset + tambah) · Badge/Pill (status, "AI", confidence, recommended_action) · Card · Table (sortable, sticky, pagination, kebab) · Tabs · Breadcrumb · Avatar · Tooltip · Toast · Modal · Drawer (slide-over) · Skeleton · Empty state · Confirmation dialog · Kanban board · **Score ring/gauge** · Stat card · **AI panel/callout** (aksen violet + sparkles + "Lihat alasan") · **Streaming text** · **File dropzone** (upload PDF) · **Toggle/Switch** · **Stepper** (onboarding) · **Risk-flag chip** (amber/rose, ikon ⚠).

**State interaksi (semua kontrol):**
| State | Visual |
|---|---|
| Default / Hover | sesuai palet / sedikit gelap + pointer |
| Active | lebih gelap, skala 0.98 |
| Focus (keyboard) | focus ring indigo 2px offset |
| Disabled | opacity 50%, not-allowed |
| Loading | spinner inline, kontrol non-aktif |
| Error (input) | border rose + helper rose |
| Selected | latar/border indigo |

### 2.5 Iconography
Lucide/Feather (outline): tender (gavel/file-text), discovery (radar/search), event (calendar), prospect (target), playbook (book-open), report (bar-chart), chat (message-square), AI (sparkles), knowledge/otak (brain), sources (globe), dashboard (layout-grid), notifikasi (bell), settings (settings), upload (upload-cloud).

---

## 3. Information Architecture / Navigasi

**Layout global:** Sidebar kiri (collapsible) + Topbar + konten.

```
┌──────────┬───────────────────────────────────────────────┐
│  LOGO    │  Breadcrumb / Judul       [search]  [+New] 🔔 👤 │
│ ──────── ├───────────────────────────────────────────────┤
│ Dashboard│                                                 │
│ Penemuan▴│   (Penemuan AI = inbox hasil crawling)          │
│ Tenders  │              AREA KONTEN                         │
│ Events   │                                                 │
│ Prospects│                                       ┌────────┐│
│ Playbooks│                                       │ Tanya  ││
│ Reports  │                                       │  AI ✨ ││
│ Chat  AI │                                       └────────┘│
│ ──────── │                                                 │
│ Otak Agent│  (Knowledge / Company Profile)                 │
│ Settings │                                                 │
└──────────┴───────────────────────────────────────────────┘
```

**Sidebar (urutan):** Dashboard · **Penemuan AI** (badge jumlah baru) · Tenders · Events · Prospects · Playbooks · Reports · **Chat** (badge "AI") · — divider — · **Otak Agent** (Knowledge) · Settings.

**Topbar:** breadcrumb/judul; **global search** ⌘K [P1]; **"+ New"** kontekstual; **notifikasi** (bell); avatar + menu.
**Floating "Tanya AI"** (kanan bawah): chat slide-over dari mana saja.

### 3.1 Global Search [P1] & 3.2 Notifications
- ⌘K command palette: cari tender/event/prospek; hasil per tipe; Enter → detail.
- Bell popover: "Penemuan AI baru (5)", "Analisa selesai", "Deadline tender besok". Tandai dibaca. Empty → "Belum ada notifikasi".

---

## 4. Spesifikasi Layar (detail)

> Format: **Tujuan → Layout → Komponen → State → Interaksi.**

### 4.1 Login
Centered card di gradient indigo→violet. Logo + tagline. Email, password (toggle), "Masuk" (primary full-width), "Lupa password?" (placeholder). Error inline; loading spinner. **Tanpa link "Daftar/Sign up"** — ini aplikasi internal, akun dibuat oleh Admin (boleh tampilkan catatan kecil "Akun dikelola Admin internal" / opsi SSO perusahaan bila ada).

### 4.2 Onboarding — Otak Agent (first-run, **LEAN**) ⭐
- **Tujuan:** setup "otak" agent secepat mungkin agar discovery bisa mulai. **Target < 2 menit.**
- **Layout:** **Stepper** ringan atau satu halaman kartu. Di atas: pilihan besar **dua jalur**:
```
┌─ Cara cepat ───────────────┐   ┌─ Isi manual ───────────────┐
│  ⬆  Upload PDF              │   │  ✍  Isi beberapa pilihan    │
│  capability deck / company  │   │  (chip & angka, < 2 menit)  │
│  profile → AI isi otomatis  │   │                             │
│  [Pilih file / drop di sini]│   │  [Mulai isi]                │
└─────────────────────────────┘   └─────────────────────────────┘
            "Lewati, atur nanti" (link kecil)
```
- **Jalur PDF:** dropzone → progress "AI membaca dokumen…" (streaming) → **form ter-isi otomatis** (kartu §4.13) untuk **review & konfirmasi**. Field hasil AI ditandai chip "diisi AI ✨" agar user tahu mana yang perlu dicek.
- **Jalur manual:** langsung ke kartu §4.13 dengan default & preset.
- **Akhir:** tombol **"Aktifkan Agent"** → memicu discovery pertama → arahkan ke **Penemuan AI**.
- **State:** PDF gagal baca → "Tidak bisa membaca file, coba isi manual". Skip → profil kosong, banner di Dashboard "Lengkapi Otak Agent agar AI bisa mencari tender".

### 4.13 Otak Agent — Knowledge / Company Profile (halaman edit) ⭐
- **Tujuan:** lihat & ubah "otak" agent. **Prinsip lean:** kartu ringkas, mayoritas chip/optional, advanced di-collapse.
- **Layout:** 6 **kartu** vertikal (atau 2 kolom di desktop), tombol **Simpan** sticky, badge "Profil dipakai agent • diperbarui {waktu}".
```
[Kartu 1] Profil Perusahaan      [⬆ Upload PDF untuk isi otomatis]
  Nama* [__________]   One-liner [____________________]
[Kartu 2] Kapabilitas (yang dijual)
  Layanan:  (Web App)(System Integration)(AI/Automation)(Data/BI)
            (Cloud/DevOps)(Maintenance)(QA) (+ tambah)
  Tech stack: [React]·[Go]·[Node]·[+]
[Kartu 3] Target Peluang
  Negara: (Indonesia)(+)   Industri: (Government)(Finance)(+)
  Nilai min* [Rp 1.000.000.000]  ideal [____]  Deadline min [7] hari
  Procurement: (Open tender)(RFP)(RFQ)(Vendor reg.)(Direct)
[Kartu 4] Hindari (No-Go)   [toggle list]
  ☑ Hardware murni  ☑ Onsite penuh luar kota  ☐ Embedded/IoT …
  Lainnya: [____________________]
[Kartu 5] Sumber & Keyword (untuk pencarian AI)
  Sumber: ☑SPSE/LKPP ☑eProc PLN ☑Pertamina ☑Telkom ☑PaDi (+ URL)
  Keyword: (pengadaan aplikasi)(integrasi sistem)… (auto dari kapabilitas)
  Keyword negatif: (hardware only)(pengadaan laptop)…
  Frekuensi crawl: ( Harian ▾ )
[Kartu 6 ▸ Advanced] Scoring (default Moonlay) — slider bobot + threshold
```
- **Komponen kunci:** chip input preset, toggle, file dropzone (per profil), slider bobot (collapsed). Setiap field punya tooltip singkat.
- **State:** field hasil PDF → chip "diisi AI ✨"; empty → default/preset terisi; simpan → toast "Otak agent diperbarui, discovery berikutnya pakai ini".
- **Sub-tab "Sumber" (atau bagian Kartu 5 diperluas):** tabel sumber (Nama, URL, Negara, Akses [Publik/Login/Manual], Legal note, Aktif). Sumber "Login/Manual" diberi badge — **bukan dibobol** (catatan kepatuhan).

### 4.3 Penemuan AI — Discovery Inbox ⭐
- **Tujuan:** tinjau tender yang **ditemukan agent** sebelum masuk pipeline.
- **Layout:** header (judul + "Jalankan pencarian sekarang" + status "Pencarian terakhir: 2 jam lalu • 12 baru"); filter (rekomendasi, sumber, negara, min skor, deadline); **list kartu/baris**.
```
🔎 Penemuan AI            [Jalankan pencarian]  Terakhir: 2j lalu • 12 baru
Filter: Rekomendasi ▾  Sumber ▾  Min skor ▾  Deadline ▾
┌──────────────────────────────────────────────────────────────┐
│ ◉86 [Pursue]  Pengembangan Portal Vendor — Pemkot Bandung      │
│     Sumber: SPSE • Deadline 24 Jun (7h) ⚠ • Rp 2,5 M           │
│     ⚠ Butuh pengalaman sejenis   [Tinjau ▸] [Pursue] [Tolak]   │
├──────────────────────────────────────────────────────────────┤
│ ◉58 [Watchlist] Pengadaan Lisensi … — Dinas X • SPSE • Rp300jt │
│     "Nilai di bawah ideal"        [Tinjau ▸] [Watchlist][Tolak]│
└──────────────────────────────────────────────────────────────┘
```
- **Kartu hasil:** score ring + **badge recommended_action** (warna), judul, buyer, sumber, deadline (badge bila dekat), nilai, **risk-flag chips**, ringkasan alasan 1 baris, aksi cepat (**Tinjau** → detail; **Pursue** → promote ke pipeline tender; **Watchlist**; **Tolak** → minta alasan singkat untuk pembelajaran).
- **Dedup:** tender sama dari beberapa sumber digabung (tampil "ditemukan di 2 sumber").
- **State:** belum ada profil → empty state "Lengkapi Otak Agent agar AI mulai mencari" + CTA. Sedang crawl → progress/skeleton "AI sedang mencari di 5 sumber…". Tidak ada hasil → "Belum ada peluang baru. Coba longgarkan kriteria atau tambah sumber."

### 4.4 Tenders — List
Header (judul + "+ Tender Baru"); filter (status, buyer, deadline, **rekomendasi**, **origin: manual/AI**, search); table. Kolom: Judul, Buyer, Nilai, Deadline (badge), Status (pill), **Fit Score** (mini ring), **Rekomendasi** (badge), Origin (ikon ✨ bila dari discovery), Aksi (kebab). Empty/loading seperti biasa.

### 4.5 Tender — Detail
```
Tenders / Portal Vendor Pemkot ▾[BIDDING]   [Edit][✨Analisa][✨Playbook][WON/LOST▾]
┌─Ringkasan──────────────────────────┐ ┌─Analisa AI ✨ ──────────────────────┐
│ Buyer: Pemkot Bandung              │ │   ◉ 86   [Pursue]   Confidence: med   │
│ Negara/Industri: ID / Government   │ │ Reasoning: scope cocok web+integrasi… │
│ Nilai: Rp 2.500.000.000            │ │ Evidence per dimensi:                 │
│ Deadline: 24 Jun 2026 (7h) ⚠       │ │  • Capability fit: ✓ web app+API      │
│ Sumber: SPSE (lihat asli ↗)        │ │  • Eligibility: ⚠ butuh pengalaman    │
│ Scope: portal vendor, dashboard…   │ │ Risk flags: ⚠ pengalaman sejenis      │
│ Syarat: NIB, NPWP, pengalaman 3 …  │ │ Dibuat AI • 1j lalu  [Analisa ulang]  │
│ Origin: ✨ Ditemukan AI (SPSE)     │ └───────────────────────────────────────┘
[Tabs: Ringkasan | Analisa AI | Playbook | Timeline]
```
- Tambahan v1.2: tampilkan **recommended_action badge**, **risk_flags chips**, **service_category**, **origin** (manual / ✨ discovery + link sumber asli), `scope_summary`, `eligibility/technical_requirements`. WON/LOST → modal + catatan (memicu learning).

### 4.6 Tender — Form (Create/Edit)
Drawer. Field inti (manual): Judul*, Buyer, Negara, Industri, Nilai+currency, Deadline, Sumber/URL, Service category, Status, Scope/Deskripsi. Opsi "Simpan & Analisa AI". (Field discovery lain auto-terisi bila dari AI.)

### 4.7 Events — List & Detail
List: Nama, Tipe (pill), Tanggal, Lokasi, Organizer, Status, Aksi. Detail: field + "Kontak/Prospek dari event" + **"+ Konversi ke Prospek"** (`source_type=event`). Form: Nama*, Tipe, Tanggal, Lokasi, Organizer, Catatan.

### 4.8 Prospects — Pipeline (Kanban)
Kolom `NEW→QUALIFIED→ENGAGED→PROPOSAL→WON/LOST` (header: nama+jumlah+total nilai). Kartu: nama+company, badge sumber, fit score ring, est value, owner. Drag-drop (optimistic+rollback). Toggle Board↔Table. Filter owner/sumber/min skor.

### 4.9 Prospect — Detail (Drawer)
Header: nama+company+stage+score ring+owner. Section: Info (sumber→link tender/event), Analisa AI, Playbook, Timeline, Aksi cepat (ubah stage, WON/LOST, "Tanya AI tentang prospek ini").

### 4.10 Playbooks
List (target, judul, versi, tanggal). Viewer terstruktur: Ringkasan Peluang · Value Proposition · Stakeholders · Strategi & Langkah (checklist) · Timeline · Risiko & Mitigasi · Next Actions · Footer (versi/model/waktu, "Generate versi baru", "Bandingkan"). Generate = streaming sections. Aksi: Salin/Export (markdown).

### 4.11 Reports
List (tipe, periode, tanggal). Generate (modal): tipe (**Daily Opportunity Digest** / **Weekly Pipeline** / **Per-peluang**) + periode → "Generate" (streaming). Viewer terstruktur (ringkasan + tabel pipeline + prospek prioritas + insight AI). Aksi: Export (markdown; PDF=P2), Salin, Hapus. Empty → "Belum ada laporan."

### 4.12 Chat (AI Assistant) — fitur kunci
```
┌─Percakapan──────┐┌─Prioritas minggu ini            🟢 Terhubung ke AI─┐
│ + Baru          ││ 👤 Tender mana yang prioritas minggu ini?           │
│ • Prioritas… 2j ││ ✨ 🔧 Membaca data tender… (list_tenders) ✓ 12      │
│ • Ringkas pi…1h ││ ✨ Prioritas (skor & deadline):                     │
│ [search]        ││   1. Portal Vendor Pemkot (86, 7 hari)  ▌(stream)   │
│                 ││ [Konteks: Tender — Portal Vendor ✕]                 │
│                 ││ ┌───────────────────────────┐  [Stop] [Kirim ▶]    │
│                 ││ │ Tanya tentang tender/prospek…│                    │
└─────────────────┘└──────────────────────────────────────────────────────┘
```
User (kanan) · Assistant (kiri, violet, sparkles, markdown) · **Tool-call chip** ("🔧 Membaca data tender…" → "✓ 12 dibaca", expandable) · streaming + Stop. Input auto-grow, Enter kirim, placeholder. Suggested chips: "Tender prioritas minggu ini?", "Ringkas pipeline", "Buatkan playbook prospek teratas", "Kenapa tender X skornya rendah?", "Cari tender baru sekarang". Context chip bila dibuka dari detail. Footer cue: "Asisten belajar dari aktivitas & hasil kamu." Error → banner "Agent tidak tersedia" (CRUD tetap jalan).

### 4.14 Settings
Tab: Profil user · Workspace (1 perusahaan, internal) · **Users (admin)** — Admin **menambah/menonaktifkan akun karyawan & atur role** (tidak ada self-signup) · **AI/Hermes** (status "Connected • v0.15.1" dari `/v1/capabilities`, memory "Pembelajaran aktif", **reset memory** [admin], "Test koneksi"). Gagal → badge merah + petunjuk.

---

## 5. User Flows utama

1. **Onboarding (lean):** Login → Otak Agent → **Upload PDF** (atau isi chip) → review → "Aktifkan Agent" → discovery pertama → **Penemuan AI**.
2. **Review penemuan:** Penemuan AI → kartu skor/rekomendasi → **Tinjau** → **Pursue** (promote ke pipeline) / Watchlist / **Tolak** (+alasan→learning).
3. **Analisa AI:** Detail tender → "Analisa AI" → streaming → skor + recommended_action + reasoning + evidence + risk flags.
4. **Playbook:** Detail peluang → "Generate Playbook" → streaming → simpan/export.
5. **Chat berbasis data:** Chat → "Prioritas minggu ini?" → tool-call → jawaban ranking + alasan (atau "Cari tender baru sekarang" → memicu discovery).
6. **Laporan:** Reports → Generate (tipe+periode) → streaming → viewer → export.
7. **Close & learn:** Detail → "Tandai WON/Tolak" → modal catatan/alasan → toast "AI akan belajar dari ini."

---

## 6. State, Feedback & Microcopy
Empty (ilustrasi + manfaat + CTA) · Loading (skeleton/streaming/spinner) · Error (toast/banner ramah + langkah lanjut) · Konfirmasi destruktif (hapus, WON/LOST, Tolak). Tone ramah-profesional, Bahasa Indonesia; sebut "AI/Asisten/Otak Agent", bukan "Hermes" (kecuali Settings). Tiap output AI: "Dibuat AI • {confidence} • {waktu}" + "Lihat alasan". **Field hasil PDF** ditandai "diisi AI ✨".

## 7. Responsiveness
Desktop ≥1024 (sidebar penuh, multi-kolom, kanban horizontal). Tablet 768–1023 (sidebar ikon; grid 1–2 kolom; kanban scroll; chat full-height; kartu Otak Agent menumpuk 1 kolom). Mobile (P2): stack, bottom nav, chat full screen.

## 8. Aksesibilitas
Kontras ≥ WCAG AA; focus ring (§2.4); keyboard untuk form/table/chat/⌘K; ikon + label/tooltip; status warna selalu + teks/ikon; **live-region** untuk streaming chat & ekstraksi PDF.

## 9. Catatan untuk Generator UI (Claude design)
- Prioritaskan: **Onboarding/Otak Agent (lean)**, **Penemuan AI (Discovery Inbox)**, **Tender Detail (AI Score + rekomendasi + risk flags)**, **Chat**, **Dashboard**.
- Tonjolkan elemen AI (violet + sparkles), **score ring berwarna**, **badge recommended_action**.
- Tegakkan **lean input**: kartu ringkas, chip preset, default terisi, dropzone PDF menonjol.
- Banyak whitespace, kartu rounded-xl, shadow halus, Inter.
- Sertakan **state populated realistis** (data Indonesia, §10).

## 10. Contoh Data (mockup realistis)
- **Tender (discovery):** "Pengembangan Portal Vendor" · Pemkot Bandung · Government · Rp 2.500.000.000 · Deadline 24 Jun 2026 · Sumber SPSE · **Fit 86 • Pursue** · risk: "butuh pengalaman sejenis".
- **Tender:** "Sistem Antrian RSUD" · Dinkes Jabar · Rp 850.000.000 · **Fit 72 • Review** · QUALIFYING.
- **Tender (low):** "Pengadaan Lisensi Office" · Dinas X · Rp 300.000.000 · **Fit 48 • Reject** · "di luar kapabilitas/nilai kecil".
- **Event:** "Indo Security Expo 2026" · Expo · 12 Jul 2026 · JCC Jakarta.
- **Prospek:** "PT Maju Jaya" (dari tender) · ENGAGED · Rp 1.800.000.000 · Fit 85.
- **Sumber:** SPSE/Inaproc LKPP (spse.inaproc.id, Publik/Login), eProc PLN (eproc.pln.co.id, Login), PaDi UMKM.

## 11. Format Lokal (Indonesia)
- **Rupiah:** `Rp 2.500.000.000` (titik ribuan); singkat: `Rp 2,5 M` / `Rp 300 jt`.
- **Tanggal:** `24 Jun 2026`; relatif untuk aktivitas ("2 jam lalu"). Datepicker `dd MMM yyyy`.
- **Angka:** ribuan titik, desimal koma (id-ID). Bahasa: label Bahasa Indonesia; istilah lazim boleh English.

## 12. Komponen → Layar (matriks)
| Komponen | Dipakai di |
|---|---|
| File dropzone (PDF) | Onboarding, Otak Agent |
| Chip/Tag input + Toggle | Otak Agent (kapabilitas/target/no-go/sumber/keyword) |
| Slider (bobot) | Otak Agent (scoring advanced) |
| Score ring/gauge | Dashboard, Penemuan AI, Tender/Prospect |
| Recommended-action badge + Risk-flag chip | Penemuan AI, Tender list/detail |
| Table (sortable) | Tenders, Events, Reports, Sumber, Prospect table |
| Kanban board | Prospects |
| AI panel/callout (streaming) | Onboarding (ekstraksi PDF), Score, Playbook, Report, Chat, Dashboard insights |
| Drawer/slide-over | Forms, Prospect detail, Chat (floating) |
| Tool-call indicator | Chat |
| Stepper | Onboarding |
| Command palette (⌘K) / Notifications | Topbar |
| Modal konfirmasi | Hapus, WON/LOST, Tolak, Generate, Aktifkan Agent |
