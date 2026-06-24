package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"salespilot/internal/domain"
	"salespilot/internal/http/dto"
	"salespilot/internal/http/httperr"
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

	e := &domain.Event{
		Name:      req.Name,
		Type:      eventType,
		Status:    status,
		Location:  req.Location,
		Organizer: req.Organizer,
		Notes:     req.Notes,
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
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
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

	if err := s.repo.Update(ctx, e); err != nil {
		return nil, fmt.Errorf("event.Update: %w", err)
	}
	return e, nil
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
