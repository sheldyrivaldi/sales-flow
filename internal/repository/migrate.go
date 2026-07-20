package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver

	dbmigrations "salespilot/db/migrations"
	"salespilot/internal/config"
)

// RunMigrations applies all pending db/migrations (embedded into the binary
// at build time, see db/migrations/migrations.go) against cfg.DatabaseURL.
//
// This exists so the API is self-sufficient on boot regardless of how it is
// deployed: docker-compose's local postgres, a company-managed external
// database, or a Kubernetes Deployment where only the image's own startup
// behavior matters (no docker-compose service, no separate migrate step,
// no init container). Without it, a fresh/company database never gets a
// schema and even the initial admin seed (SeedAdmin) silently fails,
// breaking login.
//
// Idempotent: golang-migrate tracks applied versions in a schema_migrations
// table, so re-running on every boot (including multiple replicas racing
// each other) is safe — "no change" is treated as success.
func RunMigrations(cfg *config.Config) error {
	sqlDB, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("migrate: open db: %w", err)
	}
	defer sqlDB.Close()

	dbDriver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("migrate: postgres driver: %w", err)
	}

	srcDriver, err := iofs.New(dbmigrations.FS, ".")
	if err != nil {
		return fmt.Errorf("migrate: source driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", srcDriver, "postgres", dbDriver)
	if err != nil {
		return fmt.Errorf("migrate: init: %w", err)
	}
	defer m.Close() // releases dbDriver's connection (sqlDB itself is already deferred above)

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate: up: %w", err)
	}

	log.Println("migrate: schema up to date")
	return nil
}
