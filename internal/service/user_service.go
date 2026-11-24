package service

import (
	"advanced-blog-management-system/internal/errors/apperrors"
	"advanced-blog-management-system/internal/model"
	"advanced-blog-management-system/internal/repository"
	"advanced-blog-management-system/pkg/auth"
	"context"
	"fmt"
)

type UserService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Register(ctx context.Context, req *model.UserCreateRequest) (*model.User, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", apperrors.ErrValidation, err)
	}

	existsEmail, err := s.repo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if existsEmail {
		return nil, apperrors.ErrUserAlreadyExists
	}

	existsUsername, err := s.repo.ExistsByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if existsUsername {
		return nil, apperrors.ErrUserAlreadyExists
	}

	hashed, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: hashed,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) Login(ctx context.Context, req *model.UserLoginRequest) (*model.User, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", apperrors.ErrValidation, err)
	}

	user, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	if !auth.VerifyPassword(user.Password, req.Password) {
		return nil, apperrors.ErrInvalidCredentials
	}

	return user, nil
}

func (s *UserService) GetByID(ctx context.Context, id int) (*model.User, error) {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, apperrors.ErrUserNotFound
	}
	return user, nil
}

func (s *UserService) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, apperrors.ErrUserNotFound
	}
	return user, nil
}
