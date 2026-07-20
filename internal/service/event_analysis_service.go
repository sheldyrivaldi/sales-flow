package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"salespilot/internal/ai"
	"salespilot/internal/domain"
	"salespilot/internal/hermes"
	"salespilot/internal/http/httperr"
)

// EventAnalysisService menjalankan Analisa AI dengan model TITIP-TUGAS yang
// sama seperti generate playbook: app menandai event "running" lalu menitipkan
// instruksi ke bridge (fire-and-forget), dan Hermes MELAPOR BALIK lewat
// callback saat selesai. App tidak menahan koneksi panjang, sehingga analisa
// boleh berjalan belasan menit tanpa membuat request time out.
type EventAnalysisService struct {
	events         *EventService
	profiles       *ProfileService
	runner         hermes.AgentTaskRunner
	callbackBase   string
	callbackSecret string
	// docs mengambil lampiran event dari disk; disuntik agar service tidak
	// perlu tahu soal direktori unggahan.
	docs AttachmentReader
}

// AttachmentReader membaca lampiran event menjadi bahan analisa.
type AttachmentReader interface {
	ReadEventAttachments(ev *domain.Event) ([]hermes.AgentDocument, []ai.TextFile, []string)
}

func NewEventAnalysisService(
	events *EventService,
	profiles *ProfileService,
	runner hermes.AgentTaskRunner,
	docs AttachmentReader,
	callbackBase, callbackSecret string,
) *EventAnalysisService {
	return &EventAnalysisService{
		events: events, profiles: profiles, runner: runner, docs: docs,
		callbackBase: callbackBase, callbackSecret: callbackSecret,
	}
}

// Start menandai event sedang dianalisa lalu menitipkan tugas ke Hermes.
func (s *EventAnalysisService) Start(ctx context.Context, id string) (*domain.Event, error) {
	ev, err := s.events.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	// Satu analisa per event pada satu waktu: dua proses paralel akan saling
	// menimpa hasil dan membuat status tidak bisa dipercaya.
	if ev.AnalysisStatus == domain.AnalysisRunning {
		return nil, httperr.NewBadRequest("ANALYSIS_RUNNING", "analisa untuk event ini sedang berjalan")
	}
	if s.runner == nil {
		return nil, httperr.NewBadRequest("AI_UNAVAILABLE", "AI agent tidak dikonfigurasi")
	}

	profile, _ := s.profiles.GetCurrent(ctx)
	docs, textFiles, names := s.docs.ReadEventAttachments(ev)

	instruction := ai.BuildEventAnalysisInstruction(ai.AnalysisInput{
		Event:              *ev,
		Profile:            profile,
		TextFiles:          textFiles,
		AllAttachmentNames: names,
	})

	ev.AnalysisStatus = domain.AnalysisRunning
	ev.AnalysisError = nil
	if err := s.events.SaveAnalysisState(ctx, ev); err != nil {
		return nil, err
	}

	go s.dispatch(ev.ID, instruction, docs)
	return ev, nil
}

func (s *EventAnalysisService) dispatch(eventID, instruction string, docs []hermes.AgentDocument) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	task := hermes.AgentTask{
		Instruction:    instruction,
		JobID:          eventID,
		CallbackURL:    fmt.Sprintf("%s/internal/events/%s/analysis-complete", strings.TrimRight(s.callbackBase, "/"), eventID),
		CallbackSecret: s.callbackSecret,
		Documents:      docs,
	}
	if err := s.runner.RunAgentTask(ctx, task); err != nil {
		log.Printf("event analysis %s: gagal menitipkan tugas: %v", eventID, err)
		s.fail(eventID, "Gagal mengirim tugas ke AI. Coba lagi.")
	}
}

// Complete dipanggil callback bridge saat analisa selesai.
func (s *EventAnalysisService) Complete(ctx context.Context, eventID string, content []byte, errMsg string) error {
	ev, err := s.events.Get(ctx, eventID)
	if err != nil {
		return err
	}
	if len(content) > 0 {
		now := time.Now()
		ev.Analysis = content
		ev.AnalyzedAt = &now
		ev.AnalysisStatus = domain.AnalysisSuccess
		ev.AnalysisError = nil
	} else {
		if errMsg == "" {
			errMsg = "Analisa AI gagal."
		}
		ev.AnalysisStatus = domain.AnalysisFailed
		ev.AnalysisError = &errMsg
	}
	return s.events.SaveAnalysisState(ctx, ev)
}

func (s *EventAnalysisService) fail(eventID, reason string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	ev, err := s.events.Get(ctx, eventID)
	if err != nil || ev.AnalysisStatus != domain.AnalysisRunning {
		return
	}
	ev.AnalysisStatus = domain.AnalysisFailed
	ev.AnalysisError = &reason
	_ = s.events.SaveAnalysisState(ctx, ev)
}

// ReapStale menandai analisa yang mandek sebagai gagal — jaring pengaman bila
// Hermes tidak pernah melapor balik.
func (s *EventAnalysisService) ReapStale(ctx context.Context, olderThan time.Duration) error {
	events, _, err := s.events.List(ctx, domain.EventFilter{}, 1, 500)
	if err != nil {
		return err
	}
	cutoff := time.Now().Add(-olderThan)
	for i := range events {
		ev := events[i]
		if ev.AnalysisStatus != domain.AnalysisRunning || ev.UpdatedAt.After(cutoff) {
			continue
		}
		msg := "Waktu habis menunggu AI menyelesaikan analisa. Jalankan ulang."
		ev.AnalysisStatus = domain.AnalysisFailed
		ev.AnalysisError = &msg
		if err := s.events.SaveAnalysisState(ctx, &ev); err != nil {
			log.Printf("event analysis reaper: %s: %v", ev.ID, err)
		}
	}
	return nil
}
