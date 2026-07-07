package service

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"salespilot/internal/auth"
	"salespilot/internal/domain"
	"salespilot/internal/http/httperr"
)

type AuthService struct {
	users     domain.UserRepository
	jwtSecret string
}

func NewAuthService(users domain.UserRepository, jwtSecret string) *AuthService {
	return &AuthService{users: users, jwtSecret: jwtSecret}
}

// Login validates credentials and returns the user plus a fresh token pair.
// Deliberately returns the same error for wrong email or wrong password.
func (s *AuthService) Login(ctx context.Context, email, password string) (*domain.User, string, string, error) {
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", "", httperr.NewUnauthorized("email atau password salah")
		}
		return nil, "", "", fmt.Errorf("auth.Login: %w", err)
	}

	if !u.Active {
		return nil, "", "", httperr.NewUnauthorized("akun tidak aktif")
	}

	if err := auth.Verify(u.PasswordHash, password); err != nil {
		return nil, "", "", httperr.NewUnauthorized("email atau password salah")
	}

	access, refresh, err := auth.Issue(*u, s.jwtSecret)
	if err != nil {
		return nil, "", "", fmt.Errorf("auth.Login issue: %w", err)
	}

	return u, access, refresh, nil
}

// Refresh validates a refresh token and issues a new token pair.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	claims, err := auth.Parse(refreshToken, s.jwtSecret, auth.TokenRefresh)
	if err != nil {
		return "", "", httperr.NewUnauthorized("refresh token tidak valid atau sudah kedaluwarsa")
	}

	u, err := s.users.GetByID(ctx, claims.Subject)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", httperr.NewUnauthorized("pengguna tidak ditemukan")
		}
		return "", "", fmt.Errorf("auth.Refresh: %w", err)
	}

	if !u.Active {
		return "", "", httperr.NewUnauthorized("akun tidak aktif")
	}

	access, refresh, err := auth.Issue(*u, s.jwtSecret)
	if err != nil {
		return "", "", fmt.Errorf("auth.Refresh issue: %w", err)
	}

	return access, refresh, nil
}

// Me returns the user by ID.
func (s *AuthService) Me(ctx context.Context, id string) (*domain.User, error) {
	u, err := s.users.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httperr.NewNotFound("pengguna tidak ditemukan")
		}
		return nil, fmt.Errorf("auth.Me: %w", err)
	}
	return u, nil
}
