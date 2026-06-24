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

	db, err := repository.Open(cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	if serr := repository.SeedAdmin(context.Background(), db, cfg); serr != nil {
		log.Printf("seed: peringatan seed admin gagal: %v", serr)
	}

	e := apphttp.New(cfg, db, hc)

	go func() {
		if err := e.Start(":" + cfg.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			e.Logger.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}
