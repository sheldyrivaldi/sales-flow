package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"salespilot/internal/domain"
)

// compile-time check: *UserRepo implements domain.UserRepository.
var _ domain.UserRepository = (*UserRepo)(nil)

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, u *domain.User) error {
	if err := r.db.WithContext(ctx).Create(u).Error; err != nil {
		return fmt.Errorf("user.Create: %w", err)
	}
	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	var u domain.User
	if err := r.db.WithContext(ctx).First(&u, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("user.GetByID: %w", err)
	}
	return &u, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User
	if err := r.db.WithContext(ctx).First(&u, "email = ?", email).Error; err != nil {
		return nil, fmt.Errorf("user.GetByEmail: %w", err)
	}
	return &u, nil
}

func (r *UserRepo) List(ctx context.Context, f domain.UserFilter, page, pageSize int) ([]domain.User, int64, error) {
	q := r.db.WithContext(ctx).Model(&domain.User{})

	if f.Role != nil {
		q = q.Where("role = ?", *f.Role)
	}
	if f.Active != nil {
		q = q.Where("active = ?", *f.Active)
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("name ILIKE ? OR email ILIKE ?", like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("user.List count: %w", err)
	}

	offset := (page - 1) * pageSize
	var users []domain.User
	if err := q.Limit(pageSize).Offset(offset).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("user.List: %w", err)
	}
	return users, total, nil
}

func (r *UserRepo) Update(ctx context.Context, u *domain.User) error {
	if err := r.db.WithContext(ctx).Save(u).Error; err != nil {
		return fmt.Errorf("user.Update: %w", err)
	}
	return nil
}

func (r *UserRepo) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&domain.User{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("user.Delete: %w", err)
	}
	return nil
}
