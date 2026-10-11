package services

import (
	"context"
	"errors"

	"github.com/oleoleg/project-manager/internal/db"
	"github.com/oleoleg/project-manager/internal/models"
)

var ErrInvalidTheme = errors.New("недопустимая тема")

type UserService struct {
	repo *db.UserRepo
}

func NewUserService(repo *db.UserRepo) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) SetTheme(ctx context.Context, userID int64, theme string) error {
	if theme != "light" && theme != "dark" {
		return ErrInvalidTheme
	}
	return s.repo.UpdateTheme(ctx, userID, theme)
}

func (s *UserService) GetByID(ctx context.Context, id int64) (*models.User, error) {
	return s.repo.GetByID(ctx, id)
}
