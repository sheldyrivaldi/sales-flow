package service

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"salespilot/internal/auth"
	"salespilot/internal/domain"
	"salespilot/internal/http/httperr"
	"salespilot/internal/pagination"
)

type UserService struct {
	users domain.UserRepository
}

func NewUserService(users domain.UserRepository) *UserService {
	return &UserService{users: users}
}

func (s *UserService) List(ctx context.Context, f domain.UserFilter, page, pageSize int) ([]domain.User, int64, error) {
	page, pageSize = pagination.Normalize(page, pageSize)
	return s.users.List(ctx, f, page, pageSize)
}

func (s *UserService) Create(ctx context.Context, email, name string, role domain.Role, password string) (*domain.User, error) {
	if !role.Valid() {
		return nil, httperr.NewBadRequest("INVALID_ROLE", "role tidak valid")
	}

	_, err := s.users.GetByEmail(ctx, email)
	if err == nil {
		return nil, httperr.NewConflict("EMAIL_EXISTS", "email sudah terpakai")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("user.Create cek email: %w", err)
	}

	hash, err := auth.Hash(password)
	if err != nil {
		return nil, fmt.Errorf("user.Create hash: %w", err)
	}

	u := &domain.User{
		Email:        email,
		PasswordHash: hash,
		Name:         name,
		Role:         role,
		Active:       true,
	}
	if err := s.users.Create(ctx, u); err != nil {
		return nil, fmt.Errorf("user.Create: %w", err)
	}
	return u, nil
}

func (s *UserService) Update(ctx context.Context, id string, name *string, role *string, active *bool) (*domain.User, error) {
	u, err := s.users.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, httperr.NewNotFound("pengguna tidak ditemukan")
		}
		return nil, fmt.Errorf("user.Update get: %w", err)
	}

	// guard admin terakhir (diisi di TK-03.4.2)
	isActiveAdmin := u.Role == domain.RoleAdmin && u.Active
	losesAdmin := (role != nil && domain.Role(*role) != domain.RoleAdmin) ||
		(active != nil && !*active)

	if isActiveAdmin && losesAdmin {
		adminRole := domain.RoleAdmin
		activeTrue := true
		_, total, err := s.users.List(ctx, domain.UserFilter{Role: &adminRole, Active: &activeTrue}, 1, 1)
		if err != nil {
			return nil, fmt.Errorf("user.Update count admin: %w", err)
		}
		if total <= 1 {
			return nil, httperr.NewBadRequest("LAST_ADMIN", "tidak dapat menonaktifkan atau menurunkan admin terakhir")
		}
	}

	if name != nil {
		u.Name = *name
	}
	if role != nil {
		newRole := domain.Role(*role)
		if !newRole.Valid() {
			return nil, httperr.NewBadRequest("INVALID_ROLE", "role tidak valid")
		}
		u.Role = newRole
	}
	if active != nil {
		u.Active = *active
	}

	if err := s.users.Update(ctx, u); err != nil {
		return nil, fmt.Errorf("user.Update: %w", err)
	}
	return u, nil
}

func (s *UserService) ResetPassword(ctx context.Context, id string, newPassword *string) (string, error) {
	u, err := s.users.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", httperr.NewNotFound("pengguna tidak ditemukan")
		}
		return "", fmt.Errorf("user.ResetPassword get: %w", err)
	}

	var plaintext string
	if newPassword == nil {
		plaintext, err = auth.GenerateTempPassword()
		if err != nil {
			return "", fmt.Errorf("user.ResetPassword generate: %w", err)
		}
	} else {
		plaintext = *newPassword
	}

	hash, err := auth.Hash(plaintext)
	if err != nil {
		return "", fmt.Errorf("user.ResetPassword hash: %w", err)
	}
	u.PasswordHash = hash

	if err := s.users.Update(ctx, u); err != nil {
		return "", fmt.Errorf("user.ResetPassword update: %w", err)
	}

	// return plaintext hanya bila server yang generate (newPassword==nil)
	if newPassword != nil {
		return "", nil
	}
	return plaintext, nil
}
