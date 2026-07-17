package service

import (
	"context"
	"fmt"
	"log"
)

// CronJobUpserter is the slice of *hermestui.CronClient this package needs —
// an interface (rather than importing the concrete type directly) so tests
// can stub it, and so CrawlAutomation.Client can be a true nil interface
// when unconfigured. Router wiring must NOT assign a nil *hermestui.CronClient
// straight into this field: an interface holding a nil concrete pointer is
// itself non-nil, which would defeat every `s.crawl.Client == nil` check
// below — see internal/http/router.go's construction for the guard.
type CronJobUpserter interface {
	UpsertJob(ctx context.Context, jobName, schedule, prompt string) error
	PauseJobByName(ctx context.Context, jobName string) error
}

// discoveryCronJobName is the fixed identity key for the one Hermes cron job
// this workspace ever upserts — a stable name (not a stored ID) so
// "update if it exists, create if it doesn't" can be computed by name lookup
// alone, with nothing else to persist on the Go side.
const discoveryCronJobName = "salespilot-discovery-crawl"

// CrawlAutomation bundles what ProfileService needs to keep the Company
// Profile's crawl_enabled/crawl_frequency in sync with a job on Hermes's own
// cron scheduler (EP-12 — the actual crawl/score pipeline still runs in Go,
// triggered by a callback the Hermes job hits; see internal/http/handlers/
// internal_handler.go). Client is nil-able: without HERMES_DASHBOARD_SESSION_TOKEN
// configured, upsert/pause calls are no-ops (logged), same non-blocking
// philosophy as ai.Extractor/hermes.Client elsewhere — a profile save must
// never fail just because Hermes's scheduler couldn't be reached.
type CrawlAutomation struct {
	Client             CronJobUpserter
	TriggerSecret      string
	InternalAPIBaseURL string
}

// cronScheduleForFrequency maps the Company Profile's crawl_frequency values
// (PRD §10 Kartu 5) to a 5-field cron expression, all at 08:00 workspace
// time. Mirrors ai.crawlFrequencyIntervals' set of recognized values; an
// unrecognized one falls back to daily, same as that map's default there.
func cronScheduleForFrequency(freq string) string {
	switch freq {
	case "2-3x":
		return "0 8 * * 1,3,5" // Mon/Wed/Fri
	case "mingguan":
		return "0 8 * * 1" // Monday
	default: // "harian" and any unrecognized value
		return "0 8 * * *"
	}
}

// triggerPrompt is the Hermes cron job's instruction — deliberately a single
// mechanical action (fetch one URL) rather than "go find tenders yourself":
// the actual discovery/scoring/compliance pipeline stays deterministic Go
// code (ai.Scheduler.TriggerIfDue -> DiscoveryService.RunAsync), reached via
// this callback, not reimplemented as something the LLM does per tick.
func triggerPrompt(callbackURL string) string {
	return fmt.Sprintf(
		"Tugas cron otomatis SalesPilot. SATU langkah saja: lakukan HTTP GET ke URL berikut untuk memicu "+
			"pencarian tender terjadwal: %s — Jangan mencari tender secara manual, jangan menjelaskan apa pun, "+
			"jangan melakukan langkah lain. Cukup panggil URL tersebut, lalu selesai.",
		callbackURL,
	)
}

// syncCrawlAutomation upserts or pauses the Hermes cron job to match p's
// current crawl_enabled/crawl_frequency. Best-effort: any failure is logged
// and swallowed — called after the profile version is already persisted, so
// it must never turn a successful save into an error response.
func (s *ProfileService) syncCrawlAutomation(ctx context.Context, companyName string, crawlEnabled bool, crawlFrequency string) {
	if s.crawl == nil || s.crawl.Client == nil {
		return
	}

	if !crawlEnabled {
		if err := s.crawl.Client.PauseJobByName(ctx, discoveryCronJobName); err != nil {
			log.Printf("profile: syncCrawlAutomation: gagal pause job Hermes (%s): %v", discoveryCronJobName, err)
		}
		return
	}

	if s.crawl.TriggerSecret == "" {
		log.Printf("profile: syncCrawlAutomation: CRON_TRIGGER_SECRET belum diset, lewati upsert job Hermes")
		return
	}

	callbackURL := fmt.Sprintf("%s/internal/discovery/trigger?secret=%s", s.crawl.InternalAPIBaseURL, s.crawl.TriggerSecret)
	schedule := cronScheduleForFrequency(crawlFrequency)
	prompt := triggerPrompt(callbackURL)

	if err := s.crawl.Client.UpsertJob(ctx, discoveryCronJobName, schedule, prompt); err != nil {
		log.Printf("profile: syncCrawlAutomation: gagal upsert job Hermes (%s, jadwal=%s) untuk %q: %v", discoveryCronJobName, schedule, companyName, err)
	}
}
