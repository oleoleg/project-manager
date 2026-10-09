package db

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oleoleg/project-manager/internal/config"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
)

//go:embed all:migrations
var migrationsFS embed.FS

// Connect — точка входа: гарантирует существование БД, накатывает миграции,
// возвращает пул соединений.
func Connect(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	if err := ensureDatabase(ctx, cfg); err != nil {
		return nil, fmt.Errorf("ensure database: %w", err)
	}

	if err := runMigrations(cfg); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	pool, err := pgxpool.New(ctx, cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	return pool, nil
}

// ensureDatabase подключается к служебной БД "postgres" и создаёт целевую,
// если её ещё нет.
func ensureDatabase(ctx context.Context, cfg *config.Config) error {
	adminDSN := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=postgres sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBSSLMode,
	)

	conn, err := pgx.Connect(ctx, adminDSN)
	if err != nil {
		return fmt.Errorf("connect to admin db: %w", err)
	}
	defer conn.Close(ctx)

	var exists bool
	err = conn.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`,
		cfg.DBName,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check database existence: %w", err)
	}

	if exists {
		return nil
	}

	// Имя БД нельзя параметризовать — экранируем кавычками.
	// cfg.DBName берём из env, не из пользовательского ввода — это безопасно.
	stmt := fmt.Sprintf(`CREATE DATABASE %q`, cfg.DBName)
	if _, err := conn.Exec(ctx, stmt); err != nil {
		return fmt.Errorf("create database %q: %w", cfg.DBName, err)
	}

	log.Printf("db: created database %q", cfg.DBName)
	return nil
}

// runMigrations прогоняет миграции из встроенной FS.
func runMigrations(cfg *config.Config) error {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("iofs source: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", src, "postgres://"+buildMigrateURL(cfg))
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}

	version, dirty, _ := m.Version()
	log.Printf("db: migrations up to version=%d dirty=%v", version, dirty)
	return nil
}

func buildMigrateURL(cfg *config.Config) string {
	// migrate/v4 требует URL. Собираем вручную.
	return fmt.Sprintf(
		"%s:%s@%s:%s/%s?sslmode=%s",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBSSLMode,
	)
}
