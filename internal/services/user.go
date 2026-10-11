package services

import (
	"context"
	"errors"
	"strings"

	"github.com/oleoleg/project-manager/internal/db"
	"github.com/oleoleg/project-manager/internal/models"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidTheme     = errors.New("недопустимая тема")
	ErrUsernameRequired = errors.New("логин обязателен")
	ErrFullNameRequired = errors.New("ФИО обязательно")
	ErrPasswordTooShort = errors.New("пароль должен быть не короче 6 символов")
	ErrUsernameExists   = errors.New("пользователь с таким логином уже существует")
	ErrRoleRequired     = errors.New("роль обязательна")
	ErrCannotDeleteSelf = errors.New("нельзя деактивировать самого себя")
)

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

// List возвращает список всех пользователей.
func (s *UserService) List(ctx context.Context) ([]models.User, error) {
	return s.repo.List(ctx)
}

// ListRoles возвращает справочник ролей.
func (s *UserService) ListRoles(ctx context.Context) ([]models.Role, error) {
	return s.repo.ListRoles(ctx)
}

// ValidateNewUser — валидация при создании.
func (s *UserService) ValidateNewUser(u *models.User, password string) error {
	u.Username = strings.TrimSpace(u.Username)
	u.FullName = strings.TrimSpace(u.FullName)

	if u.Username == "" {
		return ErrUsernameRequired
	}
	if u.FullName == "" {
		return ErrFullNameRequired
	}
	if u.RoleCode == "" {
		return ErrRoleRequired
	}
	if len(password) < 6 {
		return ErrPasswordTooShort
	}
	return nil
}

// ValidateEditUser — валидация при редактировании (без пароля).
func (s *UserService) ValidateEditUser(u *models.User) error {
	u.FullName = strings.TrimSpace(u.FullName)

	if u.FullName == "" {
		return ErrFullNameRequired
	}
	if u.RoleCode == "" {
		return ErrRoleRequired
	}
	return nil
}

// Create создаёт пользователя и хеширует пароль.
func (s *UserService) Create(ctx context.Context, u *models.User, password string) error {
	if err := s.ValidateNewUser(u, password); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	if u.Theme == "" {
		u.Theme = "light"
	}
	err = s.repo.Create(ctx, u)
	if isUniqueViolation(err) {
		return ErrUsernameExists
	}
	return err
}

// Update меняет ФИО, роль и активность.
func (s *UserService) Update(ctx context.Context, u *models.User) error {
	if err := s.ValidateEditUser(u); err != nil {
		return err
	}
	return s.repo.Update(ctx, u)
}

// ChangePassword меняет пароль.
func (s *UserService) ChangePassword(ctx context.Context, userID int64, newPassword string) error {
	if len(newPassword) < 6 {
		return ErrPasswordTooShort
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, userID, string(hash))
}

// ToggleActive — переключает активность. Нельзя отключить себя.
func (s *UserService) ToggleActive(ctx context.Context, userID, currentUserID int64) error {
	if userID == currentUserID {
		return ErrCannotDeleteSelf
	}
	return s.repo.ToggleActive(ctx, userID)
}
