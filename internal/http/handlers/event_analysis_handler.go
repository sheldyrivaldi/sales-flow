package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"

	"salespilot/internal/ai"
	"salespilot/internal/domain"
	"salespilot/internal/hermes"
	"salespilot/internal/http/dto"
	"salespilot/internal/http/httperr"
	"salespilot/internal/service"
)

// maxAnalysisFileBytes membatasi satu lampiran yang dibaca untuk analisa.
const maxAnalysisFileBytes = 10 << 20

// textAttachmentExts adalah lampiran yang isinya disisipkan ke prompt sebagai
// teks. Sisanya (PDF/gambar) dikirim sebagai dokumen dan dibaca lewat vision.
var textAttachmentExts = map[string]bool{
	".csv": true, ".txt": true, ".md": true, ".json": true, ".log": true,
}

// docAttachmentExts adalah lampiran yang dikirim sebagai DOKUMEN ke agent.
// Bridge merentang tiap PDF jadi gambar per halaman; gambar diteruskan apa
// adanya. SEMUA yang cocok ikut terkirim — bukan cuma yang pertama.
var docAttachmentExts = map[string]bool{
	".pdf": true, ".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true,
}

// EventAttachmentReader membaca lampiran event dari disk menjadi bahan analisa.
// Dipisah dari service agar service tidak perlu tahu soal direktori unggahan.
type EventAttachmentReader struct {
	uploadDir string
}

func NewEventAttachmentReader(uploadDir string) *EventAttachmentReader {
	return &EventAttachmentReader{uploadDir: uploadDir}
}

// localPathOf memetakan URL lampiran ("/uploads/<subdir>/<berkas>") ke jalur
// di disk. Hanya nama dasarnya yang dipakai, sehingga URL yang dimanipulasi
// (mis. berisi "../") tidak bisa keluar dari direktori unggahan.
func (r *EventAttachmentReader) localPathOf(url string) (string, bool) {
	const prefix = "/uploads/"
	if !strings.HasPrefix(url, prefix) {
		return "", false
	}
	subdir, name, ok := strings.Cut(strings.TrimPrefix(url, prefix), "/")
	if !ok || subdir == "" || name == "" {
		return "", false
	}
	subdir = filepath.Base(subdir)
	name = filepath.Base(name)
	for _, part := range []string{subdir, name} {
		if part == "." || part == ".." || part == string(filepath.Separator) {
			return "", false
		}
	}
	return filepath.Join(r.uploadDir, subdir, name), true
}

// ReadEventAttachments mengumpulkan SEMUA lampiran event: berkas teks
// disisipkan isinya, berkas dokumen/gambar dikirim seluruhnya ke agent, dan
// nama semua lampiran tetap dicatat agar AI tahu ada berkas yang tak terbaca.
func (r *EventAttachmentReader) ReadEventAttachments(ev *domain.Event) ([]hermes.AgentDocument, []ai.TextFile, []string) {
	var docs []hermes.AgentDocument
	var texts []ai.TextFile
	var names []string

	for _, att := range ev.Attachments {
		names = append(names, att.Name)

		path, ok := r.localPathOf(att.URL)
		if !ok {
			continue
		}
		info, serr := os.Stat(path)
		if serr != nil || info.Size() > maxAnalysisFileBytes {
			continue
		}

		ext := strings.ToLower(filepath.Ext(att.Name))
		switch {
		case textAttachmentExts[ext]:
			if raw, rerr := os.ReadFile(path); rerr == nil {
				texts = append(texts, ai.TextFile{Name: att.Name, Content: string(raw)})
			}
		case docAttachmentExts[ext]:
			if raw, rerr := os.ReadFile(path); rerr == nil {
				docs = append(docs, hermes.AgentDocument{Filename: att.Name, Bytes: raw})
			}
		}
	}
	return docs, texts, names
}

// EventAnalysisHandler memicu Analisa AI secara ASINKRON.
//
// Polanya sama dengan generate playbook: request ini hanya menandai event
// "running" lalu menitipkan tugas ke Hermes, dan Hermes melapor balik lewat
// callback internal. Analisa serius bisa makan belasan menit — menahannya di
// satu request HTTP akan selalu berujung timeout.
type EventAnalysisHandler struct {
	svc *service.EventAnalysisService
}

func NewEventAnalysisHandler(svc *service.EventAnalysisService) *EventAnalysisHandler {
	return &EventAnalysisHandler{svc: svc}
}

// Analyze handles POST /api/events/:id/analyze. Tanpa body — seluruh bahan
// diambil dari event itu sendiri, termasuk semua lampirannya.
func (h *EventAnalysisHandler) Analyze(c echo.Context) error {
	ev, err := h.svc.Start(c.Request().Context(), c.Param("id"))
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusAccepted, dto.ToEventResponse(*ev))
}
