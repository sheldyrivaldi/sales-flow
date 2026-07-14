package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"salespilot/internal/ai"
	"salespilot/internal/config"
	"salespilot/internal/hermes"
	apphttp "salespilot/internal/http"
	"salespilot/internal/repository"
)

// schedulerTickInterval is how often the discovery Scheduler wakes up to
// check whether a crawl is due (EP-12 ST-12.5.1) — not the crawl frequency
// itself (that's per-workspace, company_profile.crawl_frequency). Small
// enough that toggling crawl_enabled or changing crawl_frequency from the
// UI takes effect within a few minutes, not a full day.
const schedulerTickInterval = 5 * time.Minute

func main() {
	cfg := config.MustLoad()

	// Probe Hermes saat boot — jangan crash bila tidak tersedia; CRUD tetap jalan.
	hc := hermes.New(cfg)
	hctx, hcancel := context.WithTimeout(context.Background(), 5*time.Second)
	caps, herr := hc.Health(hctx)
	hcancel()
	aiAvailable := herr == nil
	if aiAvailable {
		log.Printf("hermes: connected, version=%s models=%v", caps.Version, caps.Models)
	} else {
		log.Printf("hermes: unavailable, fitur AI degrade (CRUD tetap jalan): %v", herr)
	}
	_ = aiAvailable

	db, err := repository.Open(cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	if serr := repository.SeedAdmin(context.Background(), db, cfg); serr != nil {
		log.Printf("seed: peringatan seed admin gagal: %v", serr)
	}

	e, profileSvc, discoverySvc, aiSettingSvc := apphttp.New(cfg, db, hc)

	// Rehydrate AI Provider Config (EP-18 TK-18.4.4) — hermes-bridge only
	// keeps Configure's payload in memory, so a bridge restart (without a
	// matching API restart) would silently fall back to its env-var
	// defaults. Re-pushing whatever's active in the DB here means a Go API
	// restart always re-syncs the bridge, regardless of why it restarted.
	// Best-effort: an unreachable bridge or unset CONFIG_ENC_KEY must not
	// block boot.
	{
		rctx, rcancel := context.WithTimeout(context.Background(), 5*time.Second)
		pushed, rerr := aiSettingSvc.Rehydrate(rctx)
		rcancel()
		switch {
		case rerr != nil:
			log.Printf("ai_setting: rehydrate gagal (bridge mati atau config error), lanjut boot: %v", rerr)
		case pushed:
			log.Printf("ai_setting: config AI aktif berhasil di-push ulang ke Hermes")
		default:
			log.Printf("ai_setting: tidak ada config AI aktif atau CONFIG_ENC_KEY belum diset, lewati rehydrate")
		}
	}

	go func() {
		if err := e.Start(":" + cfg.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			e.Logger.Fatalf("server: %v", err)
		}
	}()

	// Discovery scheduler (EP-12 ST-12.5.1) — checks company_profile.
	// crawl_enabled/crawl_frequency on a timer and triggers a discovery run
	// when due. Stopped via schedCancel below, before the HTTP server shuts
	// down, so no scheduled run starts mid-shutdown.
	schedCtx, schedCancel := context.WithCancel(context.Background())
	scheduler := ai.NewScheduler(profileSvc, discoverySvc, schedulerTickInterval)
	go scheduler.Start(schedCtx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	schedCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}
