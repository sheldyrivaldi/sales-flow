package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"gorm.io/gorm"

	"salespilot/internal/domain"
	"salespilot/internal/http/dto"
	"salespilot/internal/http/httperr"
	"salespilot/internal/pagination"
)

// EventService handles business logic for the event entity.
type EventService struct {
	repo         domain.EventRepository
	prospectRepo domain.ProspectRepository
}

func NewEventService(repo domain.EventRepository, prospectRepo domain.ProspectRepository) *EventService {
	return &EventService{repo: repo, prospectRepo: prospectRepo}
}

// Create creates a new event.
func (s *EventService) Create(ctx context.Context, req *dto.EventCreateRequest) (*domain.Event, error) {
	eventType := domain.EventType(req.Type)
	if !eventType.Valid() {
		return nil, httperr.NewBadRequest("INVALID_TYPE", "tipe event tidak valid")
	}

	status := domain.EventStatusPlanned
	if req.Status != nil {
		s := domain.EventStatus(*req.Status)
		if !s.Valid() {
			return nil, httperr.NewBadRequest("INVALID_STATUS", "status event tidak valid")
		}
		status = s
	}

	emails, err := normalizeEmails(req.ParticipantEmails)
	if err != nil {
		return nil, err
	}

	e := &domain.Event{
		Name:              req.Name,
		Type:              eventType,
		Status:            status,
		Location:          req.Location,
		Organizer:         req.Organizer,
		Notes:             req.Notes,
		ParticipantEmails: emails,
		Attachments:       toAttachments(req.Attachments),
	}

	if req.Date != nil {
		parsed, err := time.Parse(time.RFC3339, *req.Date)
		if err != nil {
			return nil, httperr.NewBadRequest("INVALID_DATE", "format date tidak valid, gunakan RFC3339")
		}
		e.Date = &parsed
	}

	if err := s.repo.Create(ctx, e); err != nil {
		return nil, fmt.Errorf("event.Create: %w", err)
	}
	return e, nil
}

// Get returns an event by ID.
func (s *EventService) Get(ctx context.Context, id string) (*domain.Event, error) {
	e, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httperr.NewNotFound("event tidak ditemukan")
		}
		return nil, fmt.Errorf("event.Get: %w", err)
	}
	return e, nil
}

// List returns paginated events matching the filter.
func (s *EventService) List(ctx context.Context, f domain.EventFilter, page, pageSize int) ([]domain.Event, int64, error) {
	page, pageSize = pagination.Normalize(page, pageSize)
	return s.repo.List(ctx, f, page, pageSize)
}

// Update applies a partial update to an event.
func (s *EventService) Update(ctx context.Context, id string, req *dto.EventUpdateRequest) (*domain.Event, error) {
	e, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		e.Name = *req.Name
	}
	if req.Type != nil {
		t := domain.EventType(*req.Type)
		if !t.Valid() {
			return nil, httperr.NewBadRequest("INVALID_TYPE", "tipe event tidak valid")
		}
		e.Type = t
	}
	if req.Status != nil {
		st := domain.EventStatus(*req.Status)
		if !st.Valid() {
			return nil, httperr.NewBadRequest("INVALID_STATUS", "status event tidak valid")
		}
		e.Status = st
	}
	if req.Date != nil {
		parsed, perr := time.Parse(time.RFC3339, *req.Date)
		if perr != nil {
			return nil, httperr.NewBadRequest("INVALID_DATE", "format date tidak valid, gunakan RFC3339")
		}
		e.Date = &parsed
	}
	if req.Location != nil {
		e.Location = req.Location
	}
	if req.Organizer != nil {
		e.Organizer = req.Organizer
	}
	if req.Notes != nil {
		e.Notes = req.Notes
	}
	// Pointer nil = field tidak disinggung; slice kosong = sengaja dikosongkan.
	if req.ParticipantEmails != nil {
		emails, eerr := normalizeEmails(*req.ParticipantEmails)
		if eerr != nil {
			return nil, eerr
		}
		e.ParticipantEmails = emails
	}
	if req.Attachments != nil {
		e.Attachments = toAttachments(*req.Attachments)
	}

	if err := s.repo.Update(ctx, e); err != nil {
		return nil, fmt.Errorf("event.Update: %w", err)
	}
	return e, nil
}

// maxEventParticipants membatasi undangan per event agar payload tetap wajar.
const maxEventParticipants = 200

// normalizeEmails merapikan daftar undangan: dipangkas, dijadikan huruf kecil,
// divalidasi, dan dibuang duplikatnya. Email peserta TIDAK harus terdaftar
// sebagai user aplikasi — yang divalidasi hanya bentuk alamatnya.
func normalizeEmails(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return []string{}, nil
	}
	seen := make(map[string]bool, len(raw))
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		e := strings.ToLower(strings.TrimSpace(r))
		if e == "" {
			continue
		}
		// mail.ParseAddress saja tidak cukup: ia menerima "Nama <a@b>" dan
		// domain tanpa titik seperti "a@b" (sah untuk host lokal, tapi tidak
		// pernah sah untuk undangan ke peserta luar). Syarat tambahan di sini
		// menyamakan aturan dengan validasi di frontend, supaya alamat yang
		// ditolak di form tidak diam-diam lolos lewat API.
		addr, err := mail.ParseAddress(e)
		if err != nil || addr.Address != e || !hasDottedDomain(e) {
			return nil, httperr.NewBadRequest("INVALID_EMAIL", fmt.Sprintf("email peserta tidak valid: %s", r))
		}
		if seen[e] {
			continue
		}
		seen[e] = true
		out = append(out, e)
	}
	if len(out) > maxEventParticipants {
		return nil, httperr.NewBadRequest("TOO_MANY_PARTICIPANTS",
			fmt.Sprintf("maksimal %d peserta per event", maxEventParticipants))
	}
	return out, nil
}

// hasDottedDomain memastikan bagian setelah "@" punya titik dengan label tak
// kosong di kedua sisinya (mis. "contoh.com", bukan "b", ".com", atau "b.").
func hasDottedDomain(email string) bool {
	at := strings.LastIndex(email, "@")
	if at < 0 {
		return false
	}
	domain := email[at+1:]
	dot := strings.LastIndex(domain, ".")
	return dot > 0 && dot < len(domain)-1
}

func toAttachments(in []dto.EventAttachmentDTO) []domain.EventAttachment {
	out := make([]domain.EventAttachment, 0, len(in))
	for _, a := range in {
		out = append(out, domain.EventAttachment{Name: a.Name, URL: a.URL, Mime: a.Mime, Size: a.Size})
	}
	return out
}

// SaveAnalysisState menyimpan perubahan pada field analisa (hasil, status,
// error, waktu) apa adanya. Dipakai EventAnalysisService yang sudah menyiapkan
// nilai-nilainya.
func (s *EventService) SaveAnalysisState(ctx context.Context, e *domain.Event) error {
	if err := s.repo.Update(ctx, e); err != nil {
		return fmt.Errorf("event.SaveAnalysisState: %w", err)
	}
	return nil
}

// Delete removes an event by ID.
func (s *EventService) Delete(ctx context.Context, id string) error {
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("event.Delete: %w", err)
	}
	return nil
}

// Convert creates a prospect from an event. Returns error if already converted.
func (s *EventService) Convert(ctx context.Context, eventID, ownerUserID string) (*domain.Prospect, error) {
	e, err := s.Get(ctx, eventID)
	if err != nil {
		return nil, err
	}

	existing, err := s.prospectRepo.GetBySource(ctx, domain.ProspectSourceEvent, eventID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("event.Convert check existing: %w", err)
	}
	if existing != nil {
		return nil, httperr.NewBadRequest("ALREADY_CONVERTED", "event ini sudah dikonversi ke prospek")
	}

	p := &domain.Prospect{
		Name:        e.Name,
		Company:     e.Organizer,
		SourceType:  domain.ProspectSourceEvent,
		SourceID:    &eventID,
		Stage:       domain.ProspectStageNew,
		OwnerUserID: &ownerUserID,
	}

	if err := s.prospectRepo.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("event.Convert create prospect: %w", err)
	}
	return p, nil
}
