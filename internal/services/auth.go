package services

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"github.com/oleoleg/project-manager/internal/db"
	"github.com/oleoleg/project-manager/internal/models"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserInactive       = errors.New("user is inactive")
)

type AuthService struct {
	users *db.UserRepo
}

func NewAuthService(users *db.UserRepo) *AuthService {
	return &AuthService{users: users}
}

// Authenticate проверяет логин/пароль и возвращает пользователя.
func (s *AuthService) Authenticate(ctx context.Context, username, password string) (*models.User, error) {
	u, err := s.users.GetByUsername(ctx, username)
	if errors.Is(err, db.ErrUserNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("auth: %w", err)
	}
	if !u.IsActive {
		return nil, ErrUserInactive
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	return u, nil
}

func (s *AuthService) GetByID(ctx context.Context, id int64) (*models.User, error) {
	return s.users.GetByID(ctx, id)
}
