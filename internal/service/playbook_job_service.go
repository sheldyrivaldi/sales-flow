package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"salespilot/internal/domain"
	"salespilot/internal/hermes"
	"salespilot/internal/http/httperr"
)

// playbookSchemaHint menjelaskan bentuk objek playbook yang harus disusun
// agent dan dikirim balik ke app oleh bridge lewat callback. Field datar
// (summary/value_prop/...) wajib sebagai jaring pengaman; "deck" adalah
// rancangan presentasi sesungguhnya yang dirender jadi slide.
const playbookSchemaHint = `{"title": "judul presentasi yang menjual (bukan menyalin prompt)", "subtitle": "sub-judul singkat", ` +
	`"accent": "emerald|teal|indigo|blue|violet|cyan|amber|rose|slate", ` +
	`"summary": "...", "value_prop": "...", "stakeholders": ["..."], ` +
	`"strategy_checklist": ["..."], "timeline": ["..."], "risks": ["..."], "next_actions": ["..."], ` +
	`"timeline_plan": [{"activity": "...", "start_day": 0, "duration_days": 1}], ` +
	`"metrics": [{"value": "Rp 4,2 M", "label": "...", "caption": "..."}], ` +
	`"deck": [{"layout": "...", "svg": "<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 1280 720'>...</svg>", "eyebrow": "...", "heading": "...", "note": "..."}]}`

// deckGuide menjelaskan pustaka layout slide yang tersedia. Agent MERANCANG
// deck-nya sendiri: memilih slide apa saja yang perlu ada dan layout mana yang
// paling pas untuk tiap isi — supaya deck tiap topik berbeda, bukan template.
const deckGuide = `
## Cara menyusun "deck" (INI YANG JADI SLIDE)
Kamu adalah DESAINER DECK. JUMLAH SLIDE DITENTUKAN OLEH MATERI, BUKAN OLEH ANGKA BAKU.
Tanya dirimu: "berapa slide yang benar-benar dibutuhkan supaya presentasi ini meyakinkan?"
- Topik sempit dengan satu keputusan: mungkin cukup 6 slide yang tajam.
- Tender besar dengan banyak stakeholder, regulasi, dan fase: bisa 14-16 slide.
JANGAN menambah slide sekadar mengejar jumlah, dan JANGAN memampatkan materi penting
supaya muat sedikit. Satu slide = satu gagasan yang layak berdiri sendiri.
Pilih layout per slide dari pustaka di bawah sesuai BENTUK ISINYA. Isi hanya field milik layout itu.

- {"layout":"cover","heading":"judul","body":"subjudul"}  → wajib slide pertama
- {"layout":"closing","heading":"...","body":"ajakan bertindak"}  → wajib slide terakhir
- {"layout":"statement","eyebrow":"...","heading":"...","body":"1 gagasan besar, 1-3 kalimat"}
- {"layout":"quote","heading":"...","body":"kutipan/temuan","attribution":"sumber, jabatan"}
- {"layout":"metrics","heading":"...","metrics":[{"value":"68%","label":"...","caption":"asumsi/sumber"}]}  → 3-4 angka
- {"layout":"pillars","heading":"...","cards":[{"title":"...","detail":"...","tag":"opsional"}]}  → 2-4 kartu
- {"layout":"bullets","heading":"...","bullets":["..."]}  → 4-8 poin
- {"layout":"steps","heading":"...","bullets":["..."]}  → urutan bertahap, 3-6 langkah
- {"layout":"process","heading":"...","cards":[{"title":"Tahap","detail":"..."}]}  → alur 3-5 tahap
- {"layout":"comparison","heading":"...","columns":[{"title":"Kondisi sekarang","items":["..."]},{"title":"Setelah kami","items":["..."]}]}  → tepat 2 kolom
- {"layout":"matrix","heading":"...","quadrants":[{"title":"...","items":["..."]}]}  → tepat 4 kuadran
- {"layout":"people","heading":"...","people":[{"name":"...","role":"jabatan","influence":"tinggi|sedang|rendah","angle":"cara memenangkan orang ini"}]}
- {"layout":"risks","heading":"...","risks":[{"risk":"...","impact":"tinggi|sedang|rendah","mitigation":"aksi konkret + pemilik"}]}
- {"layout":"timeline","heading":"...","timeline_plan":[{"activity":"...","start_day":0,"duration_days":5}]}

Aturan perancangan:
- JANGAN memakai urutan layout yang sama untuk setiap topik. Susun alur yang paling meyakinkan untuk topik INI.
- Variasikan: jangan 3 slide "bullets" berturut-turut. Selingi metrics/comparison/matrix/process/quote.
- LARANGAN KEMONOTONAN (penyebab keluhan "semua mirip"): dalam satu deck TIDAK BOLEH ada
  dua slide isi yang memakai layout sama BERUNTUN, dan kuadran ("matrix") maupun timeline
  masing-masing paling banyak dipakai SEKALI — hanya bila materinya memang menuntut. Jangan
  jadikan keduanya kerangka default. Targetkan minimal 5 jenis layout berbeda dalam satu deck.
- Pakai "comparison" bila ada kondisi sekarang vs sesudah; "matrix" untuk segmentasi/prioritisasi;
  "process" untuk metodologi; "people" bila ada organisasi yang harus dimenangkan; "quote" bila ada
  temuan riset atau regulasi yang kuat.
- Tiap slide isi juga "note": satu kalimat "so what" — implikasi bisnisnya, bukan mengulang isi slide.
- "accent" HANYA dipakai bila SVG-mu gagal dipakai (cadangan). Pilih yang paling dekat dengan
  industri topik (mis. siber=indigo, keuangan=blue, kesehatan=teal, energi=amber,
  pendidikan=violet, ritel=rose, logistik=cyan, manufaktur=slate).
  WARNA SEBENARNYA kamu tentukan sendiri di dalam "svg" — kamu TIDAK terikat 9 pilihan ini.
- "layout" juga sekadar penanda cadangan. Desain di "svg" TIDAK harus mengikuti bentuk baku
  layout itu; pakai komposisi apa pun yang paling pas untuk materi slide tersebut.`

// svgGuide meminta agent MENDESAIN sendiri tiap slide sebagai SVG. Tanpa ini
// agent hanya memilih dari katalog layout yang jumlahnya terbatas, sehingga
// hasilnya selalu terasa template. Aturan teks di bawah sengaja dibuat persis
// mencerminkan yang divalidasi renderer (teks keluar kanvas / tumpang tindih)
// supaya tingkat kelulusannya tinggi.
const svgGuide = `
## DESAIN VISUAL: tulis "svg" sendiri untuk SETIAP slide (INI YANG PALING PENTING)
Selain field di atas, tiap objek slide WAJIB memuat "svg": satu SVG utuh 1280x720
yang KAMU desain khusus untuk topik ini. Di sinilah kualitas presentasi ditentukan.

Kontrak teknis (WAJIB, kalau dilanggar slide-mu dibuang):
- Akar: <svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 1280 720'>...</svg>
- Pakai KUTIP TUNGGAL untuk semua atribut SVG supaya tidak perlu escape di dalam JSON.
- Elemen yang boleh: g, defs, linearGradient, radialGradient, stop, rect, circle, ellipse,
  line, polyline, polygon, path, text, tspan, clipPath, mask, pattern, filter, feGaussianBlur,
  feDropShadow, use, symbol.
- DILARANG: <script>, <foreignObject>, <image>, <a>, atribut on*, url eksternal, @import,
  font dari internet. Semua harus mandiri.
- font-family HANYA 'Arial, Helvetica, sans-serif' (dijamin ada di PowerPoint).

Aturan teks (penyebab nomor satu slide ditolak):
- SVG TIDAK punya word-wrap. Pecah sendiri tiap baris memakai <tspan x='..' dy='..'>.
- Perkiraan lebar Arial: satu karakter kira-kira 0.55 x font-size. Jadi pada font-size 20
  di lebar 600px, maksimal sekitar 54 karakter per baris. HITUNG, jangan menebak.
- Semua teks harus berada di dalam x 64..1216 dan y 56..664. Jangan ada yang keluar.
- Antar blok teks beri jarak vertikal minimal 8px. Blok teks TIDAK BOLEH bertumpuk.
- Judul slide 30-46px, subjudul 18-24px, body 15-18px, caption 12-13px.

Aturan desain (di sinilah kamu harus kreatif):
- Rancang SISTEM VISUAL KHUSUS untuk topik ini: palet 4-6 warna yang mencerminkan industrinya,
  lalu pakai konsisten di semua slide. Deck topik energi tidak boleh terlihat sama dengan deck
  topik perbankan.
- Tiap slide harus punya KOMPOSISI BERBEDA. Variasikan: bidang warna penuh, split asimetris
  (mis. 40/60), diagonal, kartu bertingkat, layout editorial dengan banyak ruang kosong.
  JANGAN pakai kerangka yang sama (header di atas + isi di bawah) untuk semua slide.
- Gambar ILUSTRASI VEKTOR sederhana yang relevan dengan topik memakai path/shape:
  mis. turbin & panel surya untuk energi, perisai & simpul jaringan untuk siber, grafik
  batang/garis untuk performa, ikon gedung untuk institusi. Abstrak dan rapi, bukan clipart.
- Manfaatkan gradient, lingkaran besar transparan, garis tipis, pola titik, dan bayangan halus
  (feDropShadow) untuk kedalaman.
- Kontras teks wajib cukup: teks terang di atas bidang gelap, teks gelap di atas bidang terang.
- Slide cover dan closing boleh paling ekspresif; slide isi tetap rapi dan mudah dibaca.

PENTING: field lain (heading, bullets, metrics, cards, dst) TETAP WAJIB diisi lengkap.
Itu dipakai sebagai cadangan bila "svg" gagal divalidasi, dan sebagai teks yang bisa direvisi.
Kalau ragu pada sebuah slide, buat SVG yang sederhana tapi rapi — jangan yang berantakan.`

// PlaybookJobService menjalankan generate playbook lewat model TITIP-TUGAS:
// app hanya membuat baris job (in_progress) lalu menitipkan instruksi ke
// hermes-bridge lewat /v1/agent-task (fire-and-forget). Bridge menyusun
// playbook di background-nya lalu MELAPOR BALIK ke app (POST callback ke
// Complete) — app TIDAK menahan koneksi panjang. Jaring pengaman: reaper
// menandai job yang tak pernah dilaporkan sebagai gagal (lihat ReapStale).
type PlaybookJobService struct {
	repo    domain.PlaybookJobRepository
	runner  hermes.AgentTaskRunner
	profile *ProfileService
	events  *EventService
	// docs membaca lampiran event dari disk agar playbook yang di-generate dari
	// sebuah event otomatis membawa SEMUA berkasnya sebagai konteks (sama seperti
	// Analisa AI). nil-safe: bila tidak disuntik, generate tetap jalan tanpa
	// lampiran event.
	docs AttachmentReader

	// callbackBase + callbackSecret dipakai membangun URL yang di-POST bridge
	// saat playbook selesai (POST {base}/internal/playbook-jobs/{id}/complete).
	callbackBase   string
	callbackSecret string
}

func NewPlaybookJobService(repo domain.PlaybookJobRepository, runner hermes.AgentTaskRunner, profile *ProfileService, events *EventService, docs AttachmentReader, callbackBase, callbackSecret string) *PlaybookJobService {
	return &PlaybookJobService{repo: repo, runner: runner, profile: profile, events: events, docs: docs, callbackBase: callbackBase, callbackSecret: callbackSecret}
}

func (s *PlaybookJobService) List(ctx context.Context) ([]domain.PlaybookJob, error) {
	return s.repo.List(ctx)
}

func (s *PlaybookJobService) Get(ctx context.Context, id string) (*domain.PlaybookJob, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *PlaybookJobService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// derivePlaybookTitle mengambil judul ringkas dari prompt (baris pertama, dipotong).
func derivePlaybookTitle(prompt string) string {
	t := strings.TrimSpace(prompt)
	if i := strings.IndexAny(t, "\n."); i > 0 {
		t = t[:i]
	}
	t = strings.TrimSpace(t)
	if len(t) > 80 {
		t = t[:80] + "…"
	}
	if t == "" {
		return "Playbook custom"
	}
	return t
}

// schemaFooter adalah penutup instruksi: balas HANYA objek JSON playbook
// dengan schema baku (bridge yang mem-parse & melapor balik ke app).
func schemaFooter() string {
	var b strings.Builder

	b.WriteString(deckGuide)
	b.WriteString(svgGuide)

	b.WriteString("\n\n## Standar isi — HARUS \"DAGING\", bukan formalitas\n")
	b.WriteString("Tulis seperti konsultan yang sudah meriset klien ini, bukan seperti template AI generik.\n")
	b.WriteString("- Setiap poin wajib membawa SATU hal spesifik: angka, nama sistem/produk, jabatan, regulasi, tenggat, atau mekanisme teknis.\n")
	b.WriteString("- DILARANG kalimat kosong tanpa isi seperti: \"meningkatkan efisiensi\", \"solusi inovatif\", \"sinergi\", \"holistik\", \"world-class\", \"cutting-edge\", \"mendorong pertumbuhan\", \"transformasi digital\" — kecuali disertai angka atau mekanisme yang menjelaskannya.\n")
	b.WriteString("- Poin list maksimal ~18 kata, langsung ke inti, tanpa basa-basi pembuka.\n")
	b.WriteString("- metrics: angka harus masuk akal dan SELALU beri 'caption' yang menyebut asumsi atau sumbernya (mis. \"asumsi 250 user, benchmark industri 2024\"). Jangan mengarang angka tanpa dasar.\n")
	b.WriteString("- people: pakai jabatan nyata yang lazim di organisasi target (mis. \"Kepala Divisi TI\", \"Procurement Manager\"), plus 'angle' berisi cara konkret memenangkan orang itu.\n")
	b.WriteString("- risks: mitigasi harus aksi nyata beserta pemiliknya, bukan \"melakukan monitoring berkala\".\n")
	b.WriteString("- Sebut regulasi/standar/kompetitor yang relevan bila memang ada di domain topik ini.\n")
	b.WriteString("- Bahasa Indonesia profesional, tegas, berorientasi hasil. Boleh tajam, jangan bertele-tele.\n")

	b.WriteString("\n## Field datar (jaring pengaman, tetap wajib diisi)\n")
	b.WriteString("summary, value_prop, stakeholders, strategy_checklist, timeline, risks, next_actions diisi ringkas — dipakai bila 'deck' gagal terbaca.\n")
	b.WriteString("timeline_plan: rencana bergaya Gantt — tiap aktivitas punya start_day (hari ke-N dari sekarang, mulai 0) dan duration_days; boleh paralel.\n")

	b.WriteString("\nBalas HANYA satu objek JSON valid dengan schema persis: ")
	b.WriteString(playbookSchemaHint)
	b.WriteString(". Tanpa penjelasan, tanpa markdown, tanpa code-fence.")
	return b.String()
}

// profileContext merakit ringkasan profil perusahaan untuk instruksi.
func profileContext(agg *domain.ProfileAggregate) string {
	if agg == nil {
		return ""
	}
	p := agg.Profile
	var b strings.Builder
	b.WriteString("## Profil Perusahaan\n")
	fmt.Fprintf(&b, "- Nama: %s\n", p.CompanyName)
	if len(p.ServiceCategories) > 0 {
		fmt.Fprintf(&b, "- Layanan: %s\n", strings.Join(p.ServiceCategories, ", "))
	}
	if len(p.Products) > 0 {
		fmt.Fprintf(&b, "- Produk: %s\n", strings.Join(p.Products, ", "))
	}
	if len(p.SupportDocuments) > 0 {
		fmt.Fprintf(&b, "- Dokumen pendukung dimiliki: %s\n", strings.Join(p.SupportDocuments, ", "))
	}
	return b.String()
}

// singleDoc membungkus satu berkas menjadi slice AgentDocument (kosong bila
// tidak ada berkas), agar jalur custom/refine tetap seragam dengan jalur event
// yang mengirim BANYAK lampiran.
func singleDoc(filename string, pdf []byte) []hermes.AgentDocument {
	if len(pdf) == 0 {
		return nil
	}
	return []hermes.AgentDocument{{Filename: filename, Bytes: pdf}}
}

// dispatch menitipkan instruksi + lampiran ke Hermes (fire-and-forget). Bila
// gagal dikirim, job langsung ditandai gagal (agent tak akan pernah dipanggil).
func (s *PlaybookJobService) dispatch(jobID, instruction string, docs []hermes.AgentDocument) {
	if s.runner == nil {
		s.markFailed(jobID, "AI agent tidak dikonfigurasi.")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	task := hermes.AgentTask{
		Instruction:    instruction,
		JobID:          jobID,
		CallbackURL:    fmt.Sprintf("%s/internal/playbook-jobs/%s/complete", strings.TrimRight(s.callbackBase, "/"), jobID),
		CallbackSecret: s.callbackSecret,
		Documents:      docs,
	}
	if err := s.runner.RunAgentTask(ctx, task); err != nil {
		log.Printf("playbook job %s: gagal menitipkan tugas ke AI: %v", jobID, err)
		s.markFailed(jobID, "Gagal mengirim tugas ke AI. Coba lagi.")
	}
}

// Complete dipanggil callback bridge saat playbook selesai — tulis hasil
// (success) atau tandai gagal beserta alasan.
func (s *PlaybookJobService) Complete(ctx context.Context, jobID string, content []byte, errMsg string) error {
	job, err := s.repo.GetByID(ctx, jobID)
	if err != nil {
		return err
	}
	if len(content) > 0 {
		job.Status = domain.PlaybookJobSuccess
		job.Content = content
		job.ErrorMessage = nil
		// Judul yang diketik user adalah final — JANGAN pernah ditimpa judul
		// karangan AI. Judul AI hanya dipakai bila user mengosongkannya.
		if !job.UserTitled {
			var parsed struct {
				Title string `json:"title"`
			}
			if err := json.Unmarshal(content, &parsed); err == nil {
				if t := strings.TrimSpace(parsed.Title); t != "" {
					job.Title = t
				}
			}
		}
	} else {
		if errMsg == "" {
			errMsg = "Generate playbook gagal."
		}
		job.Status = domain.PlaybookJobFailed
		job.ErrorMessage = &errMsg
	}
	return s.repo.Update(ctx, job)
}

func (s *PlaybookJobService) markFailed(jobID, reason string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	job, err := s.repo.GetByID(ctx, jobID)
	if err != nil {
		return
	}
	// Hanya turunkan ke failed bila masih berjalan (jangan menimpa hasil yang
	// mungkin sudah dilaporkan agent lebih dulu — race yang sangat jarang).
	if job.Status != domain.PlaybookJobInProgress && job.Status != domain.PlaybookJobUpdating {
		return
	}
	job.Status = domain.PlaybookJobFailed
	job.ErrorMessage = &reason
	_ = s.repo.Update(ctx, job)
}

// userTitle mengembalikan judul HANYA bila diketik user — judul turunan
// prompt tidak boleh dipaksakan ke AI sebagai judul presentasi.
func userTitle(j *domain.PlaybookJob) string {
	if j != nil && j.UserTitled {
		return j.Title
	}
	return ""
}

// customInstruction merakit instruksi untuk playbook custom.
func customInstruction(title, prompt string, profile *domain.ProfileAggregate, hasDoc bool) string {
	var b strings.Builder
	b.WriteString("Kamu adalah asisten strategi sales B2B. Susun PLAYBOOK terstruktur, rapi, dan detail sesuai permintaan di bawah, berdasarkan profil perusahaan.\n\n")
	if pc := profileContext(profile); pc != "" {
		b.WriteString(pc)
	}
	if t := strings.TrimSpace(title); t != "" {
		// Judul ditentukan user — pakai APA ADANYA, jangan dikarang ulang.
		fmt.Fprintf(&b, "\n## Judul (WAJIB dipakai persis)\n%s\n", t)
	}
	fmt.Fprintf(&b, "\n## Permintaan\n%s\n", prompt)
	if hasDoc {
		b.WriteString("\nBaca juga dokumen terlampir sebagai konteks utama.\n")
	}
	b.WriteString(schemaFooter())
	if strings.TrimSpace(title) != "" {
		fmt.Fprintf(&b, "\nField \"title\" HARUS berisi persis: %s", strings.TrimSpace(title))
	}
	return b.String()
}

// eventInstruction merakit instruksi untuk playbook event (riset + konteks).
func eventInstruction(eventCtx string, profile *domain.ProfileAggregate) string {
	var b strings.Builder
	b.WriteString("Kamu adalah asisten strategi sales B2B. Susun PLAYBOOK terstandarisasi, rapi, dan DETAIL untuk memaksimalkan peluang bisnis dari event berikut.\n")
	b.WriteString("Lakukan riset internet TERBARU tentang event & penyelenggaranya (skala, industri fokus, exhibitor/sponsor umum, peluang), lalu gabungkan dengan profil perusahaan. Semua rekomendasi harus konkret.\n\n")
	if pc := profileContext(profile); pc != "" {
		b.WriteString(pc)
	}
	b.WriteString("\n## Konteks Event\n")
	b.WriteString(eventCtx)
	b.WriteString("\ntimeline_plan disusun sepanjang periode sebelum, saat, dan sesudah event.")
	b.WriteString(schemaFooter())
	return b.String()
}

// CreateCustom membuat job (in_progress) lalu menitipkan tugas penyusunan
// playbook ke Hermes. Hasil masuk lewat callback bridge.
func (s *PlaybookJobService) CreateCustom(ctx context.Context, title, prompt string, pdfBytes []byte, filename, attachmentURL string) (*domain.PlaybookJob, error) {
	if strings.TrimSpace(prompt) == "" && len(pdfBytes) == 0 {
		return nil, httperr.NewBadRequest("EMPTY_PROMPT", "isi prompt atau lampirkan dokumen dulu")
	}

	profile, _ := s.profile.GetCurrent(ctx)

	// Judul milik user. Bila diisi, dikunci (user_titled) supaya hasil
	// generate tidak menimpanya; bila kosong, AI yang menyusun judul.
	title = strings.TrimSpace(title)
	userTitled := title != ""
	if !userTitled {
		title = derivePlaybookTitle(prompt)
	}

	job := &domain.PlaybookJob{
		Title:      title,
		UserTitled: userTitled,
		Prompt:     prompt,
		Status:     domain.PlaybookJobInProgress,
		Source:     "custom",
		Revisions:  []domain.PlaybookRevision{},
	}
	if filename != "" {
		job.AttachmentName = &filename
	}
	if attachmentURL != "" {
		job.AttachmentURL = &attachmentURL
	}
	if err := s.repo.Create(ctx, job); err != nil {
		return nil, err
	}

	go s.dispatch(job.ID, customInstruction(userTitle(job), prompt, profile, len(pdfBytes) > 0), singleDoc(filename, pdfBytes))
	return job, nil
}

// CreateForEvent membuat/mengganti playbook yang tertaut ke sebuah event.
//
// Perilaku "SATU event = SATU playbook": bila event sudah punya playbook
// tertaut, tautannya DILEPAS (event_id di-nil-kan, playbook lama tetap ada di
// menu Playbooks) lalu playbook baru mengambil tautannya — jadi generate ulang
// dari detail event selalu menghasilkan satu hasil yang aktif.
//
// title/prompt opsional (modal-nya identik dengan menu Playbooks): title kosong
// → dipakai "Playbook Event: {nama}"; prompt kosong → murni konteks event.
// SELURUH lampiran event otomatis ikut sebagai dokumen konteks, plus satu
// lampiran tambahan opsional (extraDoc).
func (s *PlaybookJobService) CreateForEvent(ctx context.Context, eventID, title, prompt string, extraDoc []byte, extraFilename, extraURL string) (*domain.PlaybookJob, error) {
	if s.events == nil {
		return nil, httperr.NewBadRequest("NO_EVENTS", "modul event belum dikonfigurasi")
	}
	ev, err := s.events.Get(ctx, eventID)
	if err != nil {
		return nil, err
	}
	profile, _ := s.profile.GetCurrent(ctx)

	// Lepas tautan playbook lama lebih dulu — indeks unik parsial melarang dua
	// playbook menunjuk event yang sama, jadi ini WAJIB sebelum membuat yang
	// baru. Playbook lama tidak dihapus, hanya kehilangan tautannya.
	if prev, perr := s.repo.GetByEventID(ctx, eventID); perr == nil && prev != nil {
		prev.EventID = nil
		if uerr := s.repo.Update(ctx, prev); uerr != nil {
			return nil, uerr
		}
	}

	// Kumpulkan lampiran event (semuanya) + lampiran tambahan opsional.
	var docs []hermes.AgentDocument
	if s.docs != nil {
		docs, _, _ = s.docs.ReadEventAttachments(ev)
	}
	docs = append(docs, singleDoc(extraFilename, extraDoc)...)

	// Prompt tersimpan = konteks event + arahan tambahan user (bila ada), supaya
	// Retry bisa merekonstruksi instruksi yang sama.
	storedPrompt := buildEventContext(ev)
	if p := strings.TrimSpace(prompt); p != "" {
		storedPrompt += "\n## Arahan Tambahan dari User\n" + p + "\n"
	}

	title = strings.TrimSpace(title)
	userTitled := title != ""
	if !userTitled {
		title = "Playbook Event: " + ev.Name
	}

	job := &domain.PlaybookJob{
		Title:      title,
		UserTitled: userTitled,
		Prompt:     storedPrompt,
		Status:     domain.PlaybookJobInProgress,
		Source:     "event",
		EventID:    &eventID,
		// WAJIB non-nil: kolom revisions NOT NULL, dan slice nil ditulis GORM
		// sebagai NULL sehingga INSERT ditolak.
		Revisions: []domain.PlaybookRevision{},
	}
	// Lampiran utama untuk ditampilkan di kartu: yang diunggah user, kalau tidak
	// ada pakai lampiran event pertama.
	if extraFilename != "" {
		job.AttachmentName = &extraFilename
		if extraURL != "" {
			job.AttachmentURL = &extraURL
		}
	} else if len(ev.Attachments) > 0 {
		name, url := ev.Attachments[0].Name, ev.Attachments[0].URL
		job.AttachmentName = &name
		job.AttachmentURL = &url
	}
	if err := s.repo.Create(ctx, job); err != nil {
		return nil, err
	}

	go s.dispatch(job.ID, eventInstruction(storedPrompt, profile), docs)
	return job, nil
}

// Retry menjalankan ulang job yang GAGAL dengan konteks yang sama — status
// kembali in_progress, error dibersihkan, lalu dititipkan ulang ke Hermes.
func (s *PlaybookJobService) Retry(ctx context.Context, id string) (*domain.PlaybookJob, error) {
	job, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if job.Status != domain.PlaybookJobFailed {
		return nil, httperr.NewBadRequest("NOT_FAILED", "hanya playbook yang gagal yang bisa dicoba ulang")
	}
	profile, _ := s.profile.GetCurrent(ctx)

	job.Status = domain.PlaybookJobInProgress
	job.ErrorMessage = nil
	if err := s.repo.Update(ctx, job); err != nil {
		return nil, err
	}

	var instruction string
	var docs []hermes.AgentDocument
	if job.Source == "event" {
		instruction = eventInstruction(job.Prompt, profile)
		// Muat ulang lampiran event agar retry membawa konteks yang sama seperti
		// generate pertama (bukan playbook "telanjang").
		if job.EventID != nil && s.events != nil && s.docs != nil {
			if ev, gerr := s.events.Get(ctx, *job.EventID); gerr == nil {
				docs, _, _ = s.docs.ReadEventAttachments(ev)
			}
		}
	} else {
		instruction = customInstruction(userTitle(job), job.Prompt, profile, false)
	}
	go s.dispatch(job.ID, instruction, docs)
	return job, nil
}

// Refine memicu revisi: status menjadi updating, lalu Hermes menyusun ulang
// dan melapor balik lewat callback. Menerima lampiran opsional (pdfBytes +
// url) yang dicatat ke riwayat revisi dan dibaca AI sebagai konteks.
func (s *PlaybookJobService) Refine(ctx context.Context, id, instruction string, pdfBytes []byte, filename, attachmentURL string) (*domain.PlaybookJob, error) {
	if strings.TrimSpace(instruction) == "" {
		return nil, httperr.NewBadRequest("EMPTY_INSTRUCTION", "isi instruksi revisi dulu")
	}
	job, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if job.Status != domain.PlaybookJobSuccess {
		return nil, httperr.NewBadRequest("NOT_READY", "hanya playbook yang sudah selesai yang bisa direvisi")
	}
	if len(job.Content) == 0 {
		return nil, httperr.NewBadRequest("NO_CONTENT", "playbook belum punya hasil untuk direvisi")
	}

	rev := domain.PlaybookRevision{Instruction: instruction, At: time.Now()}
	if filename != "" {
		rev.AttachmentName = &filename
	}
	if attachmentURL != "" {
		rev.AttachmentURL = &attachmentURL
	}
	job.Revisions = append(job.Revisions, rev)
	job.Status = domain.PlaybookJobUpdating
	if err := s.repo.Update(ctx, job); err != nil {
		return nil, err
	}

	var b strings.Builder
	b.WriteString("Kamu adalah asisten strategi sales B2B. Berikut playbook SAAT INI (JSON):\n\n")
	b.Write(job.Content)
	b.WriteString("\n\nRevisi playbook tersebut sesuai instruksi berikut. Pertahankan bagian yang tidak disinggung.\n")
	fmt.Fprintf(&b, "## Instruksi Revisi\n%s\n", instruction)
	if len(pdfBytes) > 0 {
		b.WriteString("\nBaca juga dokumen terlampir sebagai referensi revisi.\n")
	}
	b.WriteString(schemaFooter())

	go s.dispatch(job.ID, b.String(), singleDoc(filename, pdfBytes))
	return job, nil
}

// buildEventContext merakit seluruh field event menjadi blok teks konteks.
func buildEventContext(e *domain.Event) string {
	var b strings.Builder
	fmt.Fprintf(&b, "- Nama event: %s\n", e.Name)
	fmt.Fprintf(&b, "- Tipe: %s\n", e.Type)
	fmt.Fprintf(&b, "- Status: %s\n", e.Status)
	if e.Date != nil {
		fmt.Fprintf(&b, "- Tanggal: %s\n", e.Date.Format("2006-01-02"))
	}
	if e.Location != nil && *e.Location != "" {
		fmt.Fprintf(&b, "- Lokasi: %s\n", *e.Location)
	}
	if e.Organizer != nil && *e.Organizer != "" {
		fmt.Fprintf(&b, "- Penyelenggara: %s\n", *e.Organizer)
	}
	if e.Notes != nil && *e.Notes != "" {
		fmt.Fprintf(&b, "- Catatan: %s\n", *e.Notes)
	}
	return b.String()
}

// ReapStale menandai job yang mandek (in_progress/updating) lebih lama dari
// olderThan sebagai gagal — jaring pengaman bila Hermes tak pernah melapor
// balik (agent crash / lupa memanggil tool callback).
func (s *PlaybookJobService) ReapStale(ctx context.Context, olderThan time.Duration) error {
	jobs, err := s.repo.List(ctx)
	if err != nil {
		return err
	}
	cutoff := time.Now().Add(-olderThan)
	for i := range jobs {
		j := jobs[i]
		if j.Status != domain.PlaybookJobInProgress && j.Status != domain.PlaybookJobUpdating {
			continue
		}
		if j.UpdatedAt.After(cutoff) {
			continue
		}
		msg := "Waktu habis menunggu AI menyelesaikan playbook. Silakan generate ulang."
		j.Status = domain.PlaybookJobFailed
		j.ErrorMessage = &msg
		if err := s.repo.Update(ctx, &j); err != nil {
			log.Printf("playbook reaper: gagal menandai job %s: %v", j.ID, err)
		}
	}
	return nil
}
