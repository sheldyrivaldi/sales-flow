package service

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"

	"salespilot/internal/domain"
)

// fakeUserRepo is an in-memory implementation of domain.UserRepository for tests.
type fakeUserRepo struct {
	users  map[string]*domain.User
	nextID int
}

func newFakeRepo(initial ...*domain.User) *fakeUserRepo {
	r := &fakeUserRepo{users: make(map[string]*domain.User)}
	for i, u := range initial {
		if u.ID == "" {
			u.ID = string(rune('1' + i))
		}
		r.users[u.ID] = u
	}
	return r
}

func (r *fakeUserRepo) Create(_ context.Context, u *domain.User) error {
	r.nextID++
	u.ID = string(rune('a' + r.nextID))
	r.users[u.ID] = u
	return nil
}

func (r *fakeUserRepo) GetByID(_ context.Context, id string) (*domain.User, error) {
	u, ok := r.users[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	cp := *u
	return &cp, nil
}

func (r *fakeUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	for _, u := range r.users {
		if u.Email == email {
			cp := *u
			return &cp, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeUserRepo) List(_ context.Context, f domain.UserFilter, _, _ int) ([]domain.User, int64, error) {
	var out []domain.User
	for _, u := range r.users {
		if f.Role != nil && u.Role != *f.Role {
			continue
		}
		if f.Active != nil && u.Active != *f.Active {
			continue
		}
		out = append(out, *u)
	}
	return out, int64(len(out)), nil
}

func (r *fakeUserRepo) Update(_ context.Context, u *domain.User) error {
	if _, ok := r.users[u.ID]; !ok {
		return errors.New("not found")
	}
	cp := *u
	r.users[u.ID] = &cp
	return nil
}

func (r *fakeUserRepo) Delete(_ context.Context, id string) error {
	delete(r.users, id)
	return nil
}

func ptr[T any](v T) *T { return &v }

// TestUpdate_LastAdmin_Block verifies that the only active admin cannot be deactivated.
func TestUpdate_LastAdmin_Block_Deactivate(t *testing.T) {
	repo := newFakeRepo(&domain.User{ID: "1", Email: "admin@test.com", Name: "Admin", Role: domain.RoleAdmin, Active: true})
	svc := NewUserService(repo)

	_, err := svc.Update(context.Background(), "1", nil, nil, ptr(false))
	if err == nil {
		t.Fatal("diharapkan error saat menonaktifkan admin terakhir, tapi tidak ada error")
	}
}

// TestUpdate_LastAdmin_Block_Downgrade verifies that the only active admin cannot be downgraded.
func TestUpdate_LastAdmin_Block_Downgrade(t *testing.T) {
	repo := newFakeRepo(&domain.User{ID: "1", Email: "admin@test.com", Name: "Admin", Role: domain.RoleAdmin, Active: true})
	svc := NewUserService(repo)

	salesRole := "SALES"
	_, err := svc.Update(context.Background(), "1", nil, &salesRole, nil)
	if err == nil {
		t.Fatal("diharapkan error saat menurunkan admin terakhir, tapi tidak ada error")
	}
}

// TestUpdate_TwoAdmins_AllowDeactivate verifies that one of two admins can be deactivated.
func TestUpdate_TwoAdmins_AllowDeactivate(t *testing.T) {
	repo := newFakeRepo(
		&domain.User{ID: "1", Email: "admin1@test.com", Name: "Admin1", Role: domain.RoleAdmin, Active: true},
		&domain.User{ID: "2", Email: "admin2@test.com", Name: "Admin2", Role: domain.RoleAdmin, Active: true},
	)
	svc := NewUserService(repo)

	_, err := svc.Update(context.Background(), "1", nil, nil, ptr(false))
	if err != nil {
		t.Fatalf("tidak diharapkan error saat ada 2 admin, dapat: %v", err)
	}
}

// TestUpdate_NonAdmin_NoGuard verifies that updating a non-admin user doesn't trigger the guard.
func TestUpdate_NonAdmin_NoGuard(t *testing.T) {
	repo := newFakeRepo(&domain.User{ID: "1", Email: "sales@test.com", Name: "Sales", Role: domain.RoleSales, Active: true})
	svc := NewUserService(repo)

	_, err := svc.Update(context.Background(), "1", ptr("Sales Baru"), nil, nil)
	if err != nil {
		t.Fatalf("tidak diharapkan error saat update non-admin, dapat: %v", err)
	}
}

// TestUpdate_NotFound verifies 404 for unknown user ID.
func TestUpdate_NotFound(t *testing.T) {
	repo := newFakeRepo()
	svc := NewUserService(repo)

	_, err := svc.Update(context.Background(), "nonexistent", ptr("Name"), nil, nil)
	if err == nil {
		t.Fatal("diharapkan error NOT_FOUND, tapi tidak ada error")
	}
}
