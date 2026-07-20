package handlers

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"salespilot/internal/auth"
	"salespilot/internal/domain"
	"salespilot/internal/http/dto"
	"salespilot/internal/http/httperr"
	"salespilot/internal/pagination"
	"salespilot/internal/service"
)

// maxEventAttachmentBytes membatasi satu lampiran event pada 10 MB.
const maxEventAttachmentBytes = 10 << 20

type EventHandler struct {
	svc       *service.EventService
	uploadDir string
}

func NewEventHandler(svc *service.EventService, uploadDir string) *EventHandler {
	return &EventHandler{svc: svc, uploadDir: uploadDir}
}

// UploadAttachment handles POST /api/events/attachments — unggah SATU berkas
// dan kembalikan metadatanya. Sengaja tidak terikat pada :id supaya bisa
// dipakai juga saat event belum dibuat (form create): berkasnya diunggah dulu,
// URL-nya ikut dikirim saat event disimpan.
func (h *EventHandler) UploadAttachment(c echo.Context) error {
	fh, err := c.FormFile("file")
	if err != nil {
		return httperr.Write(c, httperr.NewBadRequest("NO_FILE", "tidak ada berkas yang diunggah"))
	}
	if fh.Size > maxEventAttachmentBytes {
		return httperr.Write(c, httperr.NewBadRequest("FILE_TOO_LARGE", "berkas melebihi batas ukuran 10 MB"))
	}
	f, oerr := fh.Open()
	if oerr != nil {
		return httperr.Write(c, httperr.NewBadRequest("FILE_UNREADABLE", "berkas tidak bisa dibaca"))
	}
	defer func() { _ = f.Close() }()

	raw, rerr := io.ReadAll(io.LimitReader(f, maxEventAttachmentBytes+1))
	if rerr != nil || int64(len(raw)) > maxEventAttachmentBytes {
		return httperr.Write(c, httperr.NewBadRequest("FILE_TOO_LARGE", "berkas melebihi batas ukuran 10 MB"))
	}

	url, mime, serr := saveUploadBytes(h.uploadDir, "event", fh.Filename, raw)
	if serr != nil {
		return httperr.Write(c, serr)
	}
	return c.JSON(http.StatusCreated, dto.EventAttachmentDTO{
		Name: fh.Filename,
		URL:  url,
		Mime: mime,
		Size: int64(len(raw)),
	})
}

// List handles GET /api/events
func (h *EventHandler) List(c echo.Context) error {
	f := domain.EventFilter{}

	// Multi-nilai per kolom dikirim sebagai parameter berulang
	// (?type=EXPO&type=SEMINAR) — bentuk yang sama dipakai Jira dan didukung
	// langsung oleh URLSearchParams di sisi browser. Nilai yang tidak dikenal
	// diabaikan diam-diam supaya URL usang tidak menghasilkan error.
	for _, v := range c.QueryParams()["type"] {
		if t := domain.EventType(v); t.Valid() {
			f.Types = append(f.Types, t)
		}
	}
	for _, v := range c.QueryParams()["status"] {
		if st := domain.EventStatus(v); st.Valid() {
			f.Statuses = append(f.Statuses, st)
		}
	}
	f.Search = strings.TrimSpace(c.QueryParam("search"))
	f.Location = strings.TrimSpace(c.QueryParam("location"))
	f.Organizer = strings.TrimSpace(c.QueryParam("organizer"))

	if v := c.QueryParam("date_from"); v != "" {
		if t, err := parseFilterDate(v, false); err == nil {
			f.DateFrom = &t
		}
	}
	if v := c.QueryParam("date_to"); v != "" {
		// Batas akhir dibuat inklusif sampai akhir hari, kalau tidak event
		// pada tanggal itu sendiri justru tidak ikut terjaring.
		if t, err := parseFilterDate(v, true); err == nil {
			f.DateTo = &t
		}
	}
	if v := c.QueryParam("has_attachment"); v != "" {
		b := v == "true"
		f.HasAttachment = &b
	}
	if v := c.QueryParam("has_participant"); v != "" {
		b := v == "true"
		f.HasParticipant = &b
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	page, pageSize = pagination.Normalize(page, pageSize)

	events, total, err := h.svc.List(c.Request().Context(), f, page, pageSize)
	if err != nil {
		return httperr.Write(c, err)
	}

	items := make([]dto.EventResponse, len(events))
	for i, e := range events {
		items[i] = dto.ToEventResponse(e)
	}

	return c.JSON(http.StatusOK, dto.EventListResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// Get handles GET /api/events/:id
func (h *EventHandler) Get(c echo.Context) error {
	id := c.Param("id")
	e, err := h.svc.Get(c.Request().Context(), id)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, dto.ToEventResponse(*e))
}

// Create handles POST /api/events
func (h *EventHandler) Create(c echo.Context) error {
	var req dto.EventCreateRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	e, err := h.svc.Create(c.Request().Context(), &req)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusCreated, dto.ToEventResponse(*e))
}

// Update handles PUT /api/events/:id
func (h *EventHandler) Update(c echo.Context) error {
	id := c.Param("id")

	var req dto.EventUpdateRequest
	if err := c.Bind(&req); err != nil {
		return httperr.Write(c, httperr.NewBadRequest("BIND_ERROR", "request tidak valid"))
	}
	if err := c.Validate(&req); err != nil {
		return httperr.Write(c, err)
	}

	e, err := h.svc.Update(c.Request().Context(), id, &req)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusOK, dto.ToEventResponse(*e))
}

// Delete handles DELETE /api/events/:id
func (h *EventHandler) Delete(c echo.Context) error {
	id := c.Param("id")
	if err := h.svc.Delete(c.Request().Context(), id); err != nil {
		return httperr.Write(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// Convert handles POST /api/events/:id/convert
func (h *EventHandler) Convert(c echo.Context) error {
	id := c.Param("id")

	user, ok := auth.UserFromContext(c)
	if !ok {
		return httperr.Write(c, httperr.NewUnauthorized("tidak terautentikasi"))
	}

	p, err := h.svc.Convert(c.Request().Context(), id, user.ID)
	if err != nil {
		return httperr.Write(c, err)
	}
	return c.JSON(http.StatusCreated, dto.ToProspectResponse(*p))
}

// parseFilterDate menerima "YYYY-MM-DD" (dari input tanggal) maupun RFC3339.
// endOfDay menggeser ke 23:59:59 agar batas akhir bersifat inklusif.
func parseFilterDate(v string, endOfDay bool) (time.Time, error) {
	t, err := time.Parse("2006-01-02", v)
	if err != nil {
		t, err = time.Parse(time.RFC3339, v)
		if err != nil {
			return time.Time{}, err
		}
	}
	if endOfDay {
		t = t.Add(24*time.Hour - time.Second)
	}
	return t, nil
}
