package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SettingsRepo struct {
	pool *pgxpool.Pool
}

func NewSettingsRepo(pool *pgxpool.Pool) *SettingsRepo {
	return &SettingsRepo{pool: pool}
}

func (r *SettingsRepo) Get(ctx context.Context, key string) (string, error) {
	var val string
	err := r.pool.QueryRow(ctx, `SELECT value FROM app_settings WHERE key = $1`, key).Scan(&val)
	if err != nil {
		return "", err
	}
	return val, nil
}

func (r *SettingsRepo) Set(ctx context.Context, key, value string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO app_settings (key, value) 
		VALUES ($1, $2) 
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()
	`, key, value)
	return err
}
