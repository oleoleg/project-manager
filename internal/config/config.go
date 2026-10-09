package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv   string
	HTTPPort string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	SessionSecret string

	BootstrapAdminUsername string
	BootstrapAdminPassword string
	BootstrapAdminFullName string
}

// Load читает .env (если есть) и переменные окружения.
// Приоритет: реальные env > .env > дефолт.
func Load() (*Config, error) {
	// .env не обязателен — в продакшене переменные придут из окружения
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:        getEnv("APP_ENV", "local"),
		HTTPPort:      getEnv("HTTP_PORT", "8080"),
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "postgres"),
		DBPassword:    getEnv("DB_PASSWORD", "postgres"),
		DBName:        getEnv("DB_NAME", "project_manager"),
		DBSSLMode:     getEnv("DB_SSLMODE", "disable"),
		SessionSecret: getEnv("SESSION_SECRET", ""),

		BootstrapAdminUsername: getEnv("BOOTSTRAP_ADMIN_USERNAME", ""),
		BootstrapAdminPassword: getEnv("BOOTSTRAP_ADMIN_PASSWORD", ""),
		BootstrapAdminFullName: getEnv("BOOTSTRAP_ADMIN_FULLNAME", "Администратор"),
	}

	if _, err := strconv.Atoi(cfg.HTTPPort); err != nil {
		return nil, fmt.Errorf("HTTP_PORT must be a number: %w", err)
	}

	return cfg, nil
}

// DSN возвращает строку подключения к PostgreSQL.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
