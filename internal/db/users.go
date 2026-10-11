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
		SELECT id, username, full_name, password_hash, role_code, is_active, theme, created_at, updated_at
		FROM users
		WHERE username = $1
	`, username)

	var u models.User
	err := row.Scan(&u.ID, &u.Username, &u.FullName, &u.PasswordHash, &u.RoleCode, &u.IsActive, &u.Theme, &u.CreatedAt, &u.UpdatedAt)
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
		SELECT id, username, full_name, password_hash, role_code, is_active, theme, created_at, updated_at
		FROM users
		WHERE id = $1
	`, id)

	var u models.User
	err := row.Scan(&u.ID, &u.Username, &u.FullName, &u.PasswordHash, &u.RoleCode, &u.IsActive, &u.Theme, &u.CreatedAt, &u.UpdatedAt)
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

func (r *UserRepo) UpdateTheme(ctx context.Context, userID int64, theme string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE users SET theme = $2, updated_at = NOW() WHERE id = $1
	`, userID, theme)
	if err != nil {
		return fmt.Errorf("update theme: %w", err)
	}
	return nil
}

func (r *UserRepo) List(ctx context.Context) ([]models.User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, username, full_name, password_hash, role_code, is_active, theme, created_at, updated_at
		FROM users
		ORDER BY is_active DESC, role_code, username
	`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var res []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(
			&u.ID, &u.Username, &u.FullName, &u.PasswordHash,
			&u.RoleCode, &u.IsActive, &u.Theme,
			&u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		res = append(res, u)
	}
	return res, rows.Err()
}

func (r *UserRepo) Update(ctx context.Context, u *models.User) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE users SET
			full_name = $2,
			role_code = $3,
			is_active = $4,
			updated_at = NOW()
		WHERE id = $1
	`, u.ID, u.FullName, u.RoleCode, u.IsActive)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *UserRepo) UpdatePassword(ctx context.Context, userID int64, hash string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1
	`, userID, hash)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *UserRepo) ToggleActive(ctx context.Context, userID int64) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE users SET is_active = NOT is_active, updated_at = NOW() WHERE id = $1
	`, userID)
	if err != nil {
		return fmt.Errorf("toggle active: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *UserRepo) ListRoles(ctx context.Context) ([]models.Role, error) {
	rows, err := r.pool.Query(ctx, `SELECT code, name, COALESCE(description,'') FROM roles ORDER BY code`)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	defer rows.Close()

	var res []models.Role
	for rows.Next() {
		var r models.Role
		if err := rows.Scan(&r.Code, &r.Name, &r.Description); err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, rows.Err()
}
