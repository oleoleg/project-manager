package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oleoleg/project-manager/internal/models"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, username, full_name, password_hash, role_code, is_active, created_at, updated_at
		FROM users
		WHERE username = $1
	`, username)

	var u models.User
	err := row.Scan(&u.ID, &u.Username, &u.FullName, &u.PasswordHash, &u.RoleCode, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return &u, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (*models.User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, username, full_name, password_hash, role_code, is_active, created_at, updated_at
		FROM users
		WHERE id = $1
	`, id)

	var u models.User
	err := row.Scan(&u.ID, &u.Username, &u.FullName, &u.PasswordHash, &u.RoleCode, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &u, nil
}

func (r *UserRepo) Create(ctx context.Context, u *models.User) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (username, full_name, password_hash, role_code, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`, u.Username, u.FullName, u.PasswordHash, u.RoleCode, u.IsActive,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *UserRepo) Count(ctx context.Context) (int64, error) {
	var n int64
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return n, nil
}
