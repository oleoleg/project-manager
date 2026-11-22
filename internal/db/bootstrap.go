package db

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/oleoleg/project-manager/internal/config"
	"github.com/oleoleg/project-manager/internal/models"
)

// BootstrapAdmin создаёт первого администратора, если таблица users пуста.
func BootstrapAdmin(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config) error {
	repo := NewUserRepo(pool)

	n, err := repo.Count(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil // пользователи уже есть — ничего не делаем
	}

	if cfg.BootstrapAdminUsername == "" || cfg.BootstrapAdminPassword == "" {
		log.Println("bootstrap: users table is empty and no BOOTSTRAP_ADMIN_* set — skipping admin creation")
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.BootstrapAdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	u := &models.User{
		Username:     cfg.BootstrapAdminUsername,
		FullName:     cfg.BootstrapAdminFullName,
		PasswordHash: string(hash),
		RoleCode:     models.RoleAdmin,
		IsActive:     true,
	}

	if err := repo.Create(ctx, u); err != nil {
		return err
	}

	log.Printf("bootstrap: created admin user %q (id=%d)", u.Username, u.ID)
	return nil
}

// BootstrapRoles заполняет справочник ролей базовыми значениями.
// Вызывается один раз при старте, до BootstrapAdmin.
func BootstrapRoles(ctx context.Context, pool *pgxpool.Pool) error {
	roles := []struct {
		code, name, desc string
	}{
		{"admin", "Администратор", "Создаёт пользователей, полный доступ"},
		{"rp_chief", "Начальник РП", "Полный доступ ко всему"},
		{"director", "Директор", "Строит графики, видит статистику по фирме"},
		{"rp", "Руководитель проектов", "Управляет своими проектами и персоналом"},
		{"accountant", "Бухгалтер", "Доступ на чтение ко всему"},
		{"lawyer", "Юрист", "Ведение истории версий договора и приложений на сервере"},
	}

	for _, r := range roles {
		_, err := pool.Exec(ctx,
			`INSERT INTO roles (code, name, description) VALUES ($1, $2, $3) ON CONFLICT (code) DO NOTHING`,
			r.code, r.name, r.desc,
		)
		if err != nil {
			return err
		}
	}
	return nil
}
