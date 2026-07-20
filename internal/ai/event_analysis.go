// event_analysis.go — penyusun instruksi Analisa AI pasca-event.
//
// Tiga prinsip yang membentuk berkas ini:
//
//  1. SUMBER DATANYA SELURUH EVENT — identitas, catatan tim, daftar undangan,
//     dan SEMUA lampiran. Lampiran event adalah satu-satunya tempat berkas;
//     tidak ada jalur unggah kedua yang terpisah.
//
//  2. STRUKTUR HASILNYA TIDAK DIPATOK. Expo, seminar teknis, dan forum
//     regulasi menghasilkan bentuk temuan yang berbeda; memaksakan satu
//     kerangka membuat sebagian seksi terisi basa-basi. Bagian analisis datang
//     sebagai "sections" yang judul DAN isinya ditentukan AI.
//
//  3. ISINYA MARKDOWN. Analisa yang berguna panjang dan bertingkat — butuh
//     bullet, penomoran, tebal, dan miring. Menyimpannya sebagai teks polos
//     memaksa struktur itu hilang.
//
// Berkas ini hanya MENYUSUN instruksi. Eksekusinya lewat jalur titip-tugas
// (agent-task) yang asinkron, karena analisa serius bisa makan belasan menit.
package ai

import (
	"fmt"
	"strings"

	"salespilot/internal/domain"
)

// TextFile adalah lampiran berbasis teks yang isinya ikut ke prompt.
type TextFile struct {
	Name    string
	Content string
}

// AnalysisInput adalah bahan mentah yang dikumpulkan pemanggil dari event.
type AnalysisInput struct {
	Event   domain.Event
	Profile *domain.ProfileAggregate
	// TextFiles: isi lampiran berbasis teks, disisipkan langsung ke prompt.
	TextFiles []TextFile
	// AllAttachmentNames: nama SEMUA lampiran, termasuk yang tidak terbaca,
	// supaya AI tahu apa saja yang ada dan bisa menyebutnya di data_gaps.
	AllAttachmentNames []string
}

// maxTextFileChars membatasi isi satu lampiran teks agar prompt tidak meledak.
const maxTextFileChars = 20000

const analysisSchema = `{"summary":"markdown",` +
	`"sections":[{"title":"judul yang kamu tentukan sendiri","body":"markdown"}],` +
	`"internal_opportunities":"markdown","client_opportunities":"markdown",` +
	`"data_gaps":["..."]}`

// BuildEventAnalysisInstruction merakit instruksi lengkap untuk agent.
func BuildEventAnalysisInstruction(in AnalysisInput) string {
	ev := in.Event

	var b strings.Builder
	b.WriteString("Kamu adalah analis market intelligence sekaligus penasihat strategi untuk perusahaan B2B.\n")
	b.WriteString("Perusahaan kami menghadiri event di bawah untuk menyerap ilmu, membangun relasi, dan memetakan peluang — peluang klien baru MAUPUN perbaikan di dalam perusahaan sendiri.\n")
	b.WriteString("Tugasmu: mengubah seluruh jejak event ini menjadi sesuatu yang bisa dieksekusi minggu ini.\n\n")

	fmt.Fprintf(&b, "## Event\n- Nama: %s\n- Tipe: %s\n", ev.Name, ev.Type)
	if ev.Date != nil {
		fmt.Fprintf(&b, "- Tanggal: %s\n", ev.Date.Format("2006-01-02"))
	}
	if ev.Location != nil && *ev.Location != "" {
		fmt.Fprintf(&b, "- Lokasi: %s\n", *ev.Location)
	}
	if ev.Organizer != nil && *ev.Organizer != "" {
		fmt.Fprintf(&b, "- Penyelenggara: %s\n", *ev.Organizer)
	}
	fmt.Fprintf(&b, "- Status: %s\n", ev.Status)
	if ev.Notes != nil && *ev.Notes != "" {
		fmt.Fprintf(&b, "- Catatan tim: %s\n", *ev.Notes)
	}
	if len(ev.ParticipantEmails) > 0 {
		fmt.Fprintf(&b, "- Peserta dari sisi kami (%d): %s\n",
			len(ev.ParticipantEmails), strings.Join(ev.ParticipantEmails, ", "))
	}

	writeProfileContext(&b, in.Profile)

	if len(in.AllAttachmentNames) > 0 {
		fmt.Fprintf(&b, "\n## Lampiran event (%d)\n", len(in.AllAttachmentNames))
		for _, n := range in.AllAttachmentNames {
			fmt.Fprintf(&b, "- %s\n", n)
		}
		b.WriteString("Lampiran berformat dokumen/gambar ikut terkirim — baca SEMUANYA, bukan hanya yang pertama.\n")
	}

	for _, tf := range in.TextFiles {
		content := tf.Content
		if len(content) > maxTextFileChars {
			content = content[:maxTextFileChars] + "\n…(dipotong)"
		}
		fmt.Fprintf(&b, "\n### Isi lampiran: %s\n%s\n", tf.Name, content)
	}

	b.WriteString("\n## Riset\n")
	b.WriteString("Cari di internet: penyelenggara, tema dan pembicara event ini, perusahaan/organisasi yang terlibat, serta tren atau regulasi industri yang dibahas. Sebutkan bila tidak ketemu.\n")

	b.WriteString("\n## Bentuk keluaran\n")
	b.WriteString("Seluruh isi teks memakai MARKDOWN: pakai bullet (- ), penomoran (1. ), **tebal** untuk angka/nama penting, dan *miring* untuk istilah. Tabel markdown boleh dipakai bila memang membantu.\n")
	b.WriteString("- summary: narasi ringkas — apa sebenarnya yang terjadi dan apa artinya bagi kami. Bukan pengulangan judul dan tanggal.\n")
	b.WriteString("- sections: bagian analisis yang JUDUL DAN ISINYA KAMU TENTUKAN SENDIRI mengikuti materi event. Tidak ada kerangka baku. Buat sebanyak yang materinya memang mendukung; kalau bahannya tipis, sedikit bagian yang tajam lebih baik daripada banyak bagian yang kosong. Panjang tidak dibatasi — tulis selengkap yang materinya izinkan.\n")
	b.WriteString("- internal_opportunities: apa yang bisa DIOLAH UNTUK PERUSAHAAN SENDIRI — perbaikan proses, produk, kapabilitas, cara jualan, atau data yang layak dikumpulkan. Sebut pemilik (jabatan) dan kapan.\n")
	b.WriteString("- client_opportunities: peluang KLIEN BARU — organisasi mana, lewat pintu apa, siapa yang ditemui, langkah pertamanya. Hanya organisasi yang benar-benar muncul di data atau riset.\n")
	b.WriteString("- data_gaps: yang TIDAK bisa kamu simpulkan dan data apa yang perlu dikumpulkan tim.\n")

	b.WriteString("\n## Standar isi — WAJIB, ini yang membedakan analisa berguna dari formalitas\n")
	b.WriteString("- Tiap butir membawa SATU hal spesifik: angka, nama sistem/produk/organisasi, jabatan, regulasi, tenggat, atau mekanisme.\n")
	b.WriteString("- DILARANG kalimat kosong: 'meningkatkan efisiensi', 'solusi inovatif', 'sinergi', 'berpotensi', 'menjanjikan', 'transformasi digital' — kecuali disertai angka atau mekanisme yang menjelaskannya.\n")
	b.WriteString("- JANGAN mengarang organisasi, orang, atau angka yang tidak ada dasarnya. Bila data tipis, katakan begitu di data_gaps.\n")
	b.WriteString("- Bila lampiran memuat daftar peserta/perusahaan, pakai itu sebagai sumber utama client_opportunities.\n")
	b.WriteString("- Bahasa Indonesia profesional dan tegas.\n")

	b.WriteString("\nBalas HANYA satu objek JSON valid dengan schema persis: ")
	b.WriteString(analysisSchema)
	b.WriteString(". Nilai tiap field berisi teks markdown. Tanpa penjelasan di luar JSON, tanpa code fence.")

	return b.String()
}
