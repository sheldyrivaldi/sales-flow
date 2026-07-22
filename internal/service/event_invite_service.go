package service

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"salespilot/internal/domain"
	"salespilot/internal/http/httperr"
	"salespilot/internal/mailer"
)

// InviteWhen adalah pilihan jadwal kirim undangan.
type InviteWhen string

const (
	InviteSameDay InviteWhen = "same_day" // hari H, jam 07:00
	InviteH1      InviteWhen = "h1"       // 1 hari sebelum, jam 07:00
	InviteH3      InviteWhen = "h3"       // 3 hari sebelum, jam 07:00
	InviteH7      InviteWhen = "h7"       // 7 hari sebelum, jam 07:00
	InviteCustom  InviteWhen = "custom"   // tanggal + jam pilihan user
)

// EventInviteService menjadwalkan dan mengirim undangan event dari server.
//
// Undangan non-custom otomatis dikirim jam 07:00 waktu lokal pada H / H-1 /
// H-3 / H-7. Custom memakai tanggal+jam yang dipilih user. Scheduler menyapu
// undangan yang jatuh tempo tiap menit dan mengirim ke SELURUH daftar peserta,
// dengan pengirim = user yang menjadwalkan.
type EventInviteService struct {
	repo   domain.EventInviteRepository
	events *EventService
	mailer mailer.Mailer // nil bila SMTP belum dikonfigurasi
	loc    *time.Location
}

func NewEventInviteService(repo domain.EventInviteRepository, events *EventService, m mailer.Mailer, loc *time.Location) *EventInviteService {
	if loc == nil {
		loc = time.Local
	}
	return &EventInviteService{repo: repo, events: events, mailer: m, loc: loc}
}

// ScheduleInput adalah permintaan menjadwalkan undangan.
type ScheduleInput struct {
	When        InviteWhen
	CustomAt    *time.Time // wajib bila When==custom
	SenderName  string
	SenderEmail string
}

// scheduledAt menghitung waktu kirim dari pilihan jadwal + tanggal event.
func (s *EventInviteService) scheduledAt(ev *domain.Event, in ScheduleInput) (time.Time, error) {
	if in.When == InviteCustom {
		if in.CustomAt == nil {
			return time.Time{}, httperr.NewBadRequest("NO_CUSTOM_TIME", "pilih tanggal dan jam kirim untuk jadwal custom")
		}
		return *in.CustomAt, nil
	}

	if ev.Date == nil {
		return time.Time{}, httperr.NewBadRequest("NO_EVENT_DATE", "event belum punya tanggal — pakai jadwal custom atau isi tanggal event dulu")
	}
	offset := map[InviteWhen]int{InviteSameDay: 0, InviteH1: 1, InviteH3: 3, InviteH7: 7}
	days, ok := offset[in.When]
	if !ok {
		return time.Time{}, httperr.NewBadRequest("BAD_WHEN", "pilihan jadwal tidak dikenal")
	}
	// Jam 07:00 waktu lokal pada tanggal event dikurangi offset.
	d := ev.Date.In(s.loc)
	at := time.Date(d.Year(), d.Month(), d.Day(), 7, 0, 0, 0, s.loc).AddDate(0, 0, -days)
	return at, nil
}

// Schedule membekukan isi undangan dan menyimpan jadwalnya. Bila waktunya sudah
// lewat (mis. H-7 padahal event tinggal 3 hari lagi), dikirim pada sapuan
// berikutnya (scheduled_at = now).
func (s *EventInviteService) Schedule(ctx context.Context, eventID string, in ScheduleInput) (*domain.EventInvite, error) {
	ev, err := s.events.Get(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if len(ev.ParticipantEmails) == 0 {
		return nil, httperr.NewBadRequest("NO_RECIPIENTS", "belum ada peserta yang diundang")
	}

	at, err := s.scheduledAt(ev, in)
	if err != nil {
		return nil, err
	}
	if at.Before(time.Now()) {
		at = time.Now()
	}

	subject, body := buildInviteEmail(ev, in.SenderName)
	inv := &domain.EventInvite{
		EventID:     eventID,
		Subject:     subject,
		Body:        body,
		Recipients:  append([]string(nil), ev.ParticipantEmails...),
		ScheduledAt: at,
		Status:      domain.InviteStatusPending,
	}
	if in.SenderName != "" {
		inv.SenderName = &in.SenderName
	}
	if in.SenderEmail != "" {
		inv.SenderEmail = &in.SenderEmail
	}
	if err := s.repo.Create(ctx, inv); err != nil {
		return nil, err
	}
	return inv, nil
}

// List mengembalikan jadwal undangan sebuah event.
func (s *EventInviteService) List(ctx context.Context, eventID string) ([]domain.EventInvite, error) {
	return s.repo.ListByEvent(ctx, eventID)
}

// Cancel menghapus jadwal yang masih pending.
func (s *EventInviteService) Cancel(ctx context.Context, id string) error {
	inv, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if inv.Status != domain.InviteStatusPending {
		return httperr.NewBadRequest("NOT_PENDING", "hanya undangan yang belum terkirim yang bisa dibatalkan")
	}
	return s.repo.Delete(ctx, id)
}

// Dispatch mengirim semua undangan yang jatuh tempo. Dipanggil scheduler.
func (s *EventInviteService) Dispatch(ctx context.Context) {
	due, err := s.repo.ListDue(ctx, time.Now(), 50)
	if err != nil {
		log.Printf("event invite dispatch: list due: %v", err)
		return
	}
	for i := range due {
		s.send(ctx, &due[i])
	}
}

func (s *EventInviteService) send(ctx context.Context, inv *domain.EventInvite) {
	if s.mailer == nil {
		msg := "SMTP belum dikonfigurasi di server (SMTP_HOST/SMTP_FROM). Undangan tidak bisa dikirim."
		inv.Status = domain.InviteStatusFailed
		inv.Error = &msg
		_ = s.repo.Update(ctx, inv)
		return
	}
	var fromName, fromEmail string
	if inv.SenderName != nil {
		fromName = *inv.SenderName
	}
	if inv.SenderEmail != nil {
		fromEmail = *inv.SenderEmail
	}
	err := s.mailer.Send(mailer.Message{
		FromName:  fromName,
		FromEmail: fromEmail,
		To:        inv.Recipients,
		Subject:   inv.Subject,
		Body:      inv.Body,
	})
	if err != nil {
		reason := err.Error()
		inv.Status = domain.InviteStatusFailed
		inv.Error = &reason
		log.Printf("event invite %s: kirim gagal: %v", inv.ID, err)
	} else {
		now := time.Now()
		inv.Status = domain.InviteStatusSent
		inv.SentAt = &now
		inv.Error = nil
	}
	if uerr := s.repo.Update(ctx, inv); uerr != nil {
		log.Printf("event invite %s: gagal update status: %v", inv.ID, uerr)
	}
}

// RunScheduler menyapu undangan jatuh tempo tiap menit sampai ctx dibatalkan.
func (s *EventInviteService) RunScheduler(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	// Sapu sekali di awal agar undangan yang sudah lewat tidak menunggu 1 menit.
	s.Dispatch(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Dispatch(ctx)
		}
	}
}

// buildInviteEmail menyusun subjek + isi undangan yang formal dan jelas.
func buildInviteEmail(ev *domain.Event, senderName string) (subject, body string) {
	tanggal := ""
	if ev.Date != nil {
		tanggal = formatInviteDate(*ev.Date)
	}

	var b strings.Builder
	b.WriteString("Kepada Yth. Bapak/Ibu,\n\n")
	if tanggal != "" {
		fmt.Fprintf(&b, "Dengan hormat, kami mengundang Bapak/Ibu untuk menghadiri %s yang akan diselenggarakan pada %s.\n\n", ev.Name, tanggal)
	} else {
		fmt.Fprintf(&b, "Dengan hormat, kami mengundang Bapak/Ibu untuk menghadiri %s.\n\n", ev.Name)
	}
	b.WriteString("Detail acara:\n")
	fmt.Fprintf(&b, "  Acara         : %s\n", ev.Name)
	if tanggal != "" {
		fmt.Fprintf(&b, "  Tanggal       : %s\n", tanggal)
	}
	if ev.Location != nil && strings.TrimSpace(*ev.Location) != "" {
		fmt.Fprintf(&b, "  Lokasi        : %s\n", *ev.Location)
	}
	if ev.Organizer != nil && strings.TrimSpace(*ev.Organizer) != "" {
		fmt.Fprintf(&b, "  Penyelenggara : %s\n", *ev.Organizer)
	}
	b.WriteString("\n")
	if ev.Notes != nil && strings.TrimSpace(*ev.Notes) != "" {
		b.WriteString("Informasi tambahan:\n")
		b.WriteString(strings.TrimSpace(*ev.Notes))
		b.WriteString("\n\n")
	}
	b.WriteString("Kami sangat mengharapkan kehadiran Bapak/Ibu. Mohon konfirmasi kehadiran dengan membalas email ini.\n\n")
	b.WriteString("Atas perhatian dan kerja samanya, kami ucapkan terima kasih.\n\n")
	b.WriteString("Hormat kami,\n")
	if strings.TrimSpace(senderName) != "" {
		b.WriteString(senderName)
	} else {
		b.WriteString("Panitia " + ev.Name)
	}

	subject = "Undangan: " + ev.Name
	if tanggal != "" {
		subject += " — " + tanggal
	}
	return subject, b.String()
}

// formatInviteDate memformat tanggal Indonesia (mis. "Senin, 20 Juli 2026").
func formatInviteDate(t time.Time) string {
	days := []string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}
	months := []string{"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
		"Juli", "Agustus", "September", "Oktober", "November", "Desember"}
	return fmt.Sprintf("%s, %d %s %d", days[int(t.Weekday())], t.Day(), months[int(t.Month())], t.Year())
}
