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
