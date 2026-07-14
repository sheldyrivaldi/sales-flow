package ai

import (
	"context"
	"fmt"
	"log"
	"time"

	"salespilot/internal/domain"
)

// ProfileFetcher is the minimal slice of ProfileService the scheduler needs
// (mirrors ProfileGetter in discovery.go) — an interface here avoids
// internal/ai importing internal/service, which would create an import
// cycle (internal/service already imports internal/ai for scoring).
type ProfileFetcher interface {
	GetCurrent(ctx context.Context) (*domain.ProfileAggregate, error)
}

// RunTrigger is the minimal slice of DiscoveryService the scheduler needs to
// kick off a run.
type RunTrigger interface {
	RunAsync(ctx context.Context, correlationKey *string) (*domain.DiscoveryRun, error)
}

// crawlFrequencyIntervals maps the Company Profile's crawl_frequency setting
// (PRD §10 Kartu 5) to how often a scheduled run is due. "2-3x" (per week) is
// approximated as once every 3 days.
var crawlFrequencyIntervals = map[string]time.Duration{
	"harian":   24 * time.Hour,
	"2-3x":     3 * 24 * time.Hour,
	"mingguan": 7 * 24 * time.Hour,
}

const defaultSchedulerInterval = 24 * time.Hour

// Scheduler periodically checks the Company Profile's crawl_enabled/
// crawl_frequency setting and triggers a discovery run when due (EP-12
// ST-12.5.1). It never runs the pipeline itself — RunTrigger.RunAsync
// (DiscoveryService) does that, in its own detached goroutine; this type
// only decides *when* to ask for a run.
type Scheduler struct {
	profiles ProfileFetcher
	runner   RunTrigger
	// tickInterval is how often the scheduler wakes up to check whether a
	// run is due — NOT the crawl frequency itself. Small in production (so a
	// frequency/enabled change takes effect promptly) and injectable so
	// tests can use milliseconds instead of waiting on real clock time.
	tickInterval time.Duration
}

func NewScheduler(profiles ProfileFetcher, runner RunTrigger, tickInterval time.Duration) *Scheduler {
	return &Scheduler{profiles: profiles, runner: runner, tickInterval: tickInterval}
}

// Start runs the check loop until ctx is canceled — the only way to stop it.
// Intended to be launched in its own goroutine by apps/api/main.go.
func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(s.tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.tick(ctx, now)
		}
	}
}

// tick checks the current profile and, if crawl is enabled and this period
// hasn't already been triggered, asks RunTrigger for a run. A profile lookup
// failure is logged and the tick skipped — a transient DB blip must not
// crash the scheduler goroutine or spam retries faster than tickInterval.
func (s *Scheduler) tick(ctx context.Context, now time.Time) {
	profile, err := s.profiles.GetCurrent(ctx)
	if err != nil {
		log.Printf("scheduler: gagal ambil profil, lewati tick: %v", err)
		return
	}
	if !profile.Profile.CrawlEnabled {
		return
	}

	interval, ok := crawlFrequencyIntervals[profile.Profile.CrawlFrequency]
	if !ok {
		interval = defaultSchedulerInterval
	}

	// A deterministic, period-bucketed key: every tick within the same
	// interval-sized bucket produces the same key, so DiscoveryService.
	// StartRun's own idempotency (ST-12.3.2) naturally collapses repeated
	// ticks in the same period into a single run — the scheduler doesn't
	// need its own separate "already ran this period" bookkeeping.
	key := fmt.Sprintf("sched-%s-%d", profile.Profile.CrawlFrequency, now.Unix()/int64(interval.Seconds()))
	if _, err := s.runner.RunAsync(ctx, &key); err != nil {
		log.Printf("scheduler: gagal memicu discovery run terjadwal: %v", err)
	}
}
