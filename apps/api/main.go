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

	"salespilot/internal/config"
	"salespilot/internal/hermes"
	apphttp "salespilot/internal/http"
	"salespilot/internal/repository"
)

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

	// Run schema migrations before anything else touches the database.
	// Migrations are embedded into this binary at build time (see
	// db/migrations/migrations.go), so this works self-contained against
	// any DATABASE_URL — a company-managed external database, a fresh
	// Kubernetes deployment, etc. — with no separate migrate step or
	// init container required. Fatal on failure: without a schema, the
	// admin seed below and the entire app are broken anyway.
	// Set AUTO_MIGRATE=false to opt out (e.g. schema changes are managed
	// out-of-band by a DBA on a company database).
	if cfg.AutoMigrate {
		if err := repository.RunMigrations(cfg); err != nil {
			log.Fatalf("migrate: %v", err)
		}
	} else {
		log.Println("migrate: AUTO_MIGRATE=false, skip (pastikan skema sudah di-apply manual)")
	}

	db, err := repository.Open(cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	if serr := repository.SeedAdmin(context.Background(), db, cfg); serr != nil {
		log.Printf("seed: peringatan seed admin gagal: %v", serr)
	}

	e, _, _, aiSettingSvc, hermesTuiH, scheduler := apphttp.New(cfg, db, hc)

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

	// Discovery scheduler (EP-12 ST-12.5.1) — constructed inside apphttp.New
	// (shared with InternalHandler's Hermes-cron callback, see router.go).
	// Checks company_profile.crawl_enabled/crawl_frequency on a timer and
	// triggers a discovery run when due. Stopped via schedCancel below,
	// before the HTTP server shuts down, so no scheduled run starts
	// mid-shutdown.
	schedCtx, schedCancel := context.WithCancel(context.Background())
	go scheduler.Start(schedCtx)

	// Hermes TUI idle/hard-cap sweeper (nil if HERMES_TUI_BASE_URL was
	// malformed — feature simply unavailable, see router.go). Same
	// stop-before-HTTP-shutdown ordering as the discovery scheduler above.
	tuiSweepCtx, tuiSweepCancel := context.WithCancel(context.Background())
	if hermesTuiH != nil {
		go hermesTuiH.RunSweeper(tuiSweepCtx)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	schedCancel()
	tuiSweepCancel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}
