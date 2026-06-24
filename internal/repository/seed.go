package repository

import (
	"context"
	"errors"
	"fmt"
	"log"

	"gorm.io/gorm"

	"salespilot/internal/auth"
	"salespilot/internal/config"
	"salespilot/internal/domain"
)

// SeedAdmin creates the initial ADMIN user from config if none exists yet.
// Idempotent: safe to call on every boot.
func SeedAdmin(ctx context.Context, db *gorm.DB, cfg *config.Config) error {
	repo := NewUserRepo(db)

	_, err := repo.GetByEmail(ctx, cfg.SeedAdminEmail)
	if err == nil {
		log.Println("seed: admin sudah ada, skip")
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("seed: cek admin: %w", err)
	}

	hash, err := auth.Hash(cfg.SeedAdminPassword)
	if err != nil {
		return fmt.Errorf("seed: hash password: %w", err)
	}

	admin := &domain.User{
		Email:        cfg.SeedAdminEmail,
		PasswordHash: hash,
		Name:         "Administrator",
		Role:         domain.RoleAdmin,
		Active:       true,
	}
	if err := repo.Create(ctx, admin); err != nil {
		return fmt.Errorf("seed: buat admin: %w", err)
	}

	log.Printf("seed: admin ter-seed (email=%s)", cfg.SeedAdminEmail)
	return nil
}
