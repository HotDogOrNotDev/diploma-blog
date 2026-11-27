package service

import (
	"context"
	"errors"
	"testing"

	"advanced-blog-management-system/internal/errors/apperrors"
	"advanced-blog-management-system/internal/model"
	"advanced-blog-management-system/pkg/auth"
)

type mockUserRepo struct {
	users map[string]*model.User
	next  int
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*model.User), next: 1}
}

func (m *mockUserRepo) Create(ctx context.Context, user *model.User) error {
	user.ID = m.next
	m.next++
	cp := *user
	m.users[user.Email] = &cp
	return nil
}

func (m *mockUserRepo) GetByID(ctx context.Context, id int) (*model.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			cp := *u
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	u, ok := m.users[email]
	if !ok {
		return nil, nil
	}
	cp := *u
	return &cp, nil
}

func (m *mockUserRepo) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	return nil, nil
}

func (m *mockUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	_, ok := m.users[email]
	return ok, nil
}

func (m *mockUserRepo) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	for _, u := range m.users {
		if u.Username == username {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockUserRepo) Update(ctx context.Context, user *model.User) error { return nil }
func (m *mockUserRepo) Delete(ctx context.Context, id int) error           { return nil }

func TestUserServiceLoginSuccess(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewUserService(repo)
	ctx := context.Background()

	user, err := svc.Register(ctx, &model.UserCreateRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if !auth.VerifyPassword(user.Password, "password123") {
		t.Fatal("password must be hashed")
	}

	logged, err := svc.Login(ctx, &model.UserLoginRequest{
		Email:    "alice@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if logged.ID != user.ID {
		t.Fatalf("unexpected user id %d", logged.ID)
	}
}

func TestUserServiceLoginInvalidCredentials(t *testing.T) {
	repo := newMockUserRepo()
	svc := NewUserService(repo)
	ctx := context.Background()

	_, err := svc.Register(ctx, &model.UserCreateRequest{
		Username: "bob",
		Email:    "bob@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	_, err = svc.Login(ctx, &model.UserLoginRequest{
		Email:    "bob@example.com",
		Password: "wrong",
	})
	if !errors.Is(err, apperrors.ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials, got %v", err)
	}

	_, err = svc.Login(ctx, &model.UserLoginRequest{
		Email:    "missing@example.com",
		Password: "password123",
	})
	if !errors.Is(err, apperrors.ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials for missing user, got %v", err)
	}
}
