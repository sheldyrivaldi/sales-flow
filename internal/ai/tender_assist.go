// tender_assist.go — dua fitur bantuan AI per-tender di halaman detail:
//
//  1. DocChecklist: ceklis kelengkapan dokumen administrasi perusahaan
//     terhadap syarat tender (apa yang diminta tender, mana yang kemungkinan
//     sudah dimiliki berdasar profil, apa yang harus disiapkan + saran).
//  2. ProposalDraft: contoh proposal TERSTANDARISASI (kerangka bagian baku,
//     bukan format bebas per-generate) sebagai panduan menyusun penawaran.
//
// Keduanya on-demand (tidak dipersist) dan non-blocking: kegagalan AI
// dikembalikan sebagai error biasa yang di-handle FE sebagai state degrade.
package ai

import (
	"context"
	"fmt"
	"strings"

	"salespilot/internal/domain"
	"salespilot/internal/hermes"
)

// ── Ceklis Dokumen ───────────────────────────────────────────────────────────

// DocChecklistItem adalah satu dokumen/persyaratan administrasi pada ceklis.
type DocChecklistItem struct {
	Document string `json:"document"`
	// Required: dokumen ini eksplisit diminta tender (true) atau lazim
	// diminta untuk tender sejenis (false).
	Required bool `json:"required"`
	// Status: "tersedia" | "perlu_verifikasi" | "belum_ada"
	Status     string `json:"status"`
	Suggestion string `json:"suggestion"`
}

// DocChecklist adalah hasil pemeriksaan kelengkapan dokumen untuk satu tender.
type DocChecklist struct {
	Items []DocChecklistItem `json:"items"`
	// ReadinessScore 0-100: perkiraan kesiapan administrasi.
	ReadinessScore int    `json:"readiness_score"`
	Summary        string `json:"summary"`
}

// ── Draft Proposal ───────────────────────────────────────────────────────────

// proposalSections adalah kerangka baku proposal — urutan & judul TETAP di
// setiap generate supaya hasilnya terstandarisasi, bukan format acak.
var proposalSections = []string{
	"Ringkasan Eksekutif",
	"Latar Belakang & Pemahaman Kebutuhan",
	"Solusi yang Diusulkan & Ruang Lingkup",
	"Metodologi & Rencana Kerja",
	"Jadwal Pelaksanaan (Timeline)",
	"Tim & Struktur Organisasi Proyek",
	"Pengalaman & Portofolio Relevan",
	"Kepatuhan Administrasi & Legalitas",
	"Asumsi, Batasan, dan Ketergantungan",
	"Penutup & Langkah Selanjutnya",
}

// ProposalSection adalah satu bagian proposal (judul baku + isi hasil AI).
type ProposalSection struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// ProposalDraft adalah contoh proposal untuk satu tender.
type ProposalDraft struct {
	Title    string            `json:"title"`
	Sections []ProposalSection `json:"sections"`
	// Disclaimer selalu ada: ini PANDUAN awal, wajib direview manusia.
	Disclaimer string `json:"disclaimer"`
}

// TenderAssist membangun prompt + memanggil Hermes untuk kedua fitur di atas.
type TenderAssist struct {
	hc hermes.Client
	sk hermes.SessionKey
}

func NewTenderAssist(hc hermes.Client, sk hermes.SessionKey) *TenderAssist {
	return &TenderAssist{hc: hc, sk: sk}
}

// writeProfileContext menulis ringkasan profil perusahaan ke prompt — subset
// buildScoringPrompt yang relevan untuk konteks dokumen/proposal.
func writeProfileContext(b *strings.Builder, profile *domain.ProfileAggregate) {
	if profile == nil {
		return
	}
	p := profile.Profile
	b.WriteString("## Profil Perusahaan\n")
	fmt.Fprintf(b, "- Nama: %s\n", p.CompanyName)
	if p.OneLiner != nil && *p.OneLiner != "" {
		fmt.Fprintf(b, "- Deskripsi: %s\n", *p.OneLiner)
	}
	if len(p.ServiceCategories) > 0 {
		fmt.Fprintf(b, "- Layanan: %s\n", strings.Join(p.ServiceCategories, ", "))
	}
	if len(p.TechStack) > 0 {
		fmt.Fprintf(b, "- Tech stack: %s\n", strings.Join(p.TechStack, ", "))
	}
	if len(p.Products) > 0 {
		fmt.Fprintf(b, "- Produk: %s\n", strings.Join(p.Products, ", "))
	}
	if len(p.PortfolioRefs) > 0 {
		fmt.Fprintf(b, "- Portofolio/bukti pengalaman: %s\n", strings.Join(p.PortfolioRefs, ", "))
	}
}

// writeTenderContext menulis data tender ke prompt.
func writeTenderContext(b *strings.Builder, t domain.Tender) {
	b.WriteString("\n## Data Tender\n")
	fmt.Fprintf(b, "- Judul: %s\n", t.Title)
	if t.BuyerName != nil && *t.BuyerName != "" {
		fmt.Fprintf(b, "- Buyer: %s\n", *t.BuyerName)
	}
	if t.BuyerCountry != nil && *t.BuyerCountry != "" {
		fmt.Fprintf(b, "- Negara: %s\n", *t.BuyerCountry)
	}
	if t.BuyerIndustry != nil && *t.BuyerIndustry != "" {
		fmt.Fprintf(b, "- Industri buyer: %s\n", *t.BuyerIndustry)
	}
	if t.ValueEstimate != nil {
		fmt.Fprintf(b, "- Nilai estimasi: %s %s\n", t.Currency, formatFloatPtr(t.ValueEstimate))
	}
	if t.SubmissionDeadline != nil {
		fmt.Fprintf(b, "- Deadline submission: %s\n", t.SubmissionDeadline.Format("2006-01-02"))
	}
	if t.ServiceCategory != nil && *t.ServiceCategory != "" {
		fmt.Fprintf(b, "- Kategori layanan: %s\n", *t.ServiceCategory)
	}
	if t.ScopeSummary != nil && *t.ScopeSummary != "" {
		fmt.Fprintf(b, "- Ringkasan scope: %s\n", *t.ScopeSummary)
	}
	if t.EligibilityRequirements != nil && *t.EligibilityRequirements != "" {
		fmt.Fprintf(b, "- Syarat eligibilitas/administrasi: %s\n", *t.EligibilityRequirements)
	}
	if t.TechnicalRequirements != nil && *t.TechnicalRequirements != "" {
		fmt.Fprintf(b, "- Syarat teknis: %s\n", *t.TechnicalRequirements)
	}
}

// GenerateDocChecklist memeriksa kelengkapan dokumen perusahaan terhadap
// syarat tender dan memberi saran untuk yang kurang.
func (a *TenderAssist) GenerateDocChecklist(ctx context.Context, t domain.Tender, profile *domain.ProfileAggregate) (*DocChecklist, error) {
	var b strings.Builder
	b.WriteString("Kamu adalah konsultan administrasi tender berpengalaman di Indonesia. ")
	b.WriteString("Buat CEKLIS kelengkapan dokumen untuk mengikuti tender di bawah: gabungkan (1) dokumen yang eksplisit diminta tender, ")
	b.WriteString("dan (2) dokumen administrasi standar tender sejenis di negara/industri tersebut (mis. NIB, NPWP, akta perusahaan, company profile, ")
	b.WriteString("laporan keuangan, surat pernyataan, referensi pengalaman sejenis, sertifikasi terkait).\n")
	b.WriteString("Untuk tiap dokumen tentukan status berdasar profil perusahaan: \"tersedia\" hanya bila profil memberi bukti kuat, ")
	b.WriteString("\"perlu_verifikasi\" bila lazimnya dimiliki tapi tidak terbukti dari profil, \"belum_ada\" bila kemungkinan besar belum dimiliki. ")
	b.WriteString("Isi suggestion dengan langkah konkret menyiapkan/memperolehnya. JANGAN mengarang kepemilikan dokumen.\n\n")

	writeProfileContext(&b, profile)
	writeTenderContext(&b, t)

	b.WriteString("\nBalas HANYA JSON dengan schema persis: ")
	b.WriteString(`{"items": [{"document": "...", "required": true|false, "status": "tersedia|perlu_verifikasi|belum_ada", "suggestion": "..."}], ` +
		`"readiness_score": 0-100, "summary": "..."}`)
	b.WriteString(". Tanpa penjelasan, tanpa markdown, tanpa code fence.")

	var out DocChecklist
	if _, err := a.hc.GenerateJSON(ctx, b.String(), &out, a.sk); err != nil {
		return nil, fmt.Errorf("ai.GenerateDocChecklist: %w", err)
	}
	return &out, nil
}

// GenerateProposalDraft membuat contoh proposal terstandarisasi (kerangka
// bagian baku proposalSections) untuk tender t.
func (a *TenderAssist) GenerateProposalDraft(ctx context.Context, t domain.Tender, profile *domain.ProfileAggregate) (*ProposalDraft, error) {
	var b strings.Builder
	b.WriteString("Kamu adalah bid manager senior. Susun DRAF proposal penawaran untuk tender di bawah, atas nama perusahaan pada profil. ")
	b.WriteString("Draf ini adalah PANDUAN awal bagi tim proposal — konkret, relevan dengan scope tender, dan realistis terhadap kapabilitas profil. ")
	b.WriteString("JANGAN mengarang angka harga (tulis sebagai placeholder '[diisi tim komersial]'), JANGAN mengarang nama orang, ")
	b.WriteString("dan JANGAN mengklaim sertifikasi/pengalaman yang tidak ada di profil.\n\n")

	writeProfileContext(&b, profile)
	writeTenderContext(&b, t)

	b.WriteString("\n## Kerangka WAJIB (urutan dan judul bagian TIDAK boleh diubah)\n")
	for i, s := range proposalSections {
		fmt.Fprintf(&b, "%d. %s\n", i+1, s)
	}

	b.WriteString("\nIsi tiap bagian 1-3 paragraf (boleh bullet) dalam Bahasa Indonesia formal. ")
	b.WriteString("Balas HANYA JSON dengan schema persis: ")
	b.WriteString(`{"title": "...", "sections": [{"title": "...", "content": "..."}], "disclaimer": "..."}`)
	b.WriteString(" — sections HARUS berisi tepat 10 bagian sesuai kerangka di atas, urutan sama. Tanpa markdown fence.")

	var out ProposalDraft
	if _, err := a.hc.GenerateJSON(ctx, b.String(), &out, a.sk); err != nil {
		return nil, fmt.Errorf("ai.GenerateProposalDraft: %w", err)
	}
	return &out, nil
}
