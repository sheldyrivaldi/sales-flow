package domain

import (
	"context"
	"time"
)

type Role string

const (
	RoleSales   Role = "SALES"
	RoleOps     Role = "OPS"
	RoleManager Role = "MANAGER"
	RoleAdmin   Role = "ADMIN"
)

func (r Role) Valid() bool {
	switch r {
	case RoleSales, RoleOps, RoleManager, RoleAdmin:
		return true
	}
	return false
}

type User struct {
	ID           string    `json:"id"         gorm:"primaryKey"`
	Email        string    `json:"email"      gorm:"uniqueIndex;not null"`
	PasswordHash string    `json:"-"          gorm:"column:password_hash;not null"`
	Name         string    `json:"name"       gorm:"not null"`
	Role         Role      `json:"role"       gorm:"not null"`
	Active       bool      `json:"active"     gorm:"not null;default:true"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (User) TableName() string { return "user" }

type UserFilter struct {
	Role   *Role
	Active *bool
	Search string
}

type UserRepository interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	List(ctx context.Context, f UserFilter, page, pageSize int) ([]User, int64, error)
	Update(ctx context.Context, u *User) error
	Delete(ctx context.Context, id string) error
}
