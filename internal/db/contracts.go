package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oleoleg/project-manager/internal/models"
)

var ErrContractNotFound = errors.New("contract not found")

type ContractRepo struct {
	pool *pgxpool.Pool
}

func NewContractRepo(pool *pgxpool.Pool) *ContractRepo {
	return &ContractRepo{pool: pool}
}

const contractColumns = `
	id, contract_number, contract_date, work_name, responsible_person, responsible_user_id,
	status_code, counterparty_id, workshop, project_url, 
	procurement_card_status, extra_details, finplan_status, plan_schedule_status,
	short_name, contract_type, created_at, updated_at
`

func scanContract(row pgx.Row, c *models.Contract) error {
	return row.Scan(
		&c.ID, &c.ContractNumber, &c.ContractDate, &c.WorkName, &c.ResponsiblePerson, &c.ResponsibleUserID,
		&c.StatusCode, &c.CounterpartyID, &c.Workshop, &c.ProjectURL,
		&c.ProcurementCardStatus, &c.ExtraDetails, &c.FinplanStatus, &c.PlanScheduleStatus,
		&c.ShortName, &c.ContractType, &c.CreatedAt, &c.UpdatedAt,
	)
}

func (r *ContractRepo) GetByID(ctx context.Context, id string) (*models.Contract, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+contractColumns+` FROM contracts WHERE id = $1`, id)
	var c models.Contract
	if err := scanContract(row, &c); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrContractNotFound
		}
		return nil, fmt.Errorf("get contract: %w", err)
	}
	return &c, nil
}

// ContractFilter — параметры фильтра списка.
type ContractFilter struct {
	Search         string // ищем по id, номеру, названию
	CounterpartyID string
	StatusCode     *int16
}

func (r *ContractRepo) List(ctx context.Context, f ContractFilter) ([]models.Contract, error) {
	query := `SELECT ` + contractColumns + ` FROM contracts WHERE 1=1`
	args := []any{}
	i := 1

	if f.Search != "" {
		pattern := "%" + f.Search + "%"
		query += fmt.Sprintf(` AND (id ILIKE $%d OR contract_number ILIKE $%d OR work_name ILIKE $%d OR short_name ILIKE $%d)`, i, i, i, i)
		args = append(args, pattern)
		i++
	}
	if f.CounterpartyID != "" {
		query += fmt.Sprintf(` AND counterparty_id = $%d`, i)
		args = append(args, f.CounterpartyID)
		i++
	}
	if f.StatusCode != nil {
		query += fmt.Sprintf(` AND status_code = $%d`, i)
		args = append(args, *f.StatusCode)
		i++
	}
	query += ` ORDER BY id`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list contracts: %w", err)
	}
	defer rows.Close()

	var result []models.Contract
	for rows.Next() {
		var c models.Contract
		if err := scanContract(rows, &c); err != nil {
			return nil, fmt.Errorf("scan contract: %w", err)
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (r *ContractRepo) Create(ctx context.Context, c *models.Contract) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO contracts (
			id, contract_number, contract_date, work_name, responsible_person,
			responsible_user_id,
			status_code, counterparty_id, workshop, project_url, 
			procurement_card_status, extra_details, finplan_status, plan_schedule_status,
			short_name, contract_type
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
	`,
		c.ID, c.ContractNumber, c.ContractDate, c.WorkName, c.ResponsiblePerson,
		c.ResponsibleUserID,
		c.StatusCode, c.CounterpartyID, c.Workshop, c.ProjectURL,
		c.ProcurementCardStatus, c.ExtraDetails, c.FinplanStatus, c.PlanScheduleStatus,
		c.ShortName, c.ContractType,
	)
	if err != nil {
		return fmt.Errorf("create contract: %w", err)
	}
	return nil
}

func (r *ContractRepo) Update(ctx context.Context, c *models.Contract) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE contracts SET
			contract_number = $2, contract_date = $3, work_name = $4, responsible_person = $5,
			responsible_user_id = $6,
			status_code = $7, counterparty_id = $8, workshop = $9, project_url = $10,
			procurement_card_status = $11, extra_details = $12, finplan_status = $13,
			plan_schedule_status = $14, short_name = $15, contract_type = $16,
			updated_at = NOW()
		WHERE id = $1
	`,
		c.ID, c.ContractNumber, c.ContractDate, c.WorkName, c.ResponsiblePerson,
		c.ResponsibleUserID,
		c.StatusCode, c.CounterpartyID, c.Workshop, c.ProjectURL,
		c.ProcurementCardStatus, c.ExtraDetails, c.FinplanStatus, c.PlanScheduleStatus,
		c.ShortName, c.ContractType,
	)
	if err != nil {
		return fmt.Errorf("update contract: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrContractNotFound
	}
	return nil
}

func (r *ContractRepo) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM contracts WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete contract: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrContractNotFound
	}
	return nil
}

// --- справочники ---

func (r *ContractRepo) ListStatuses(ctx context.Context) ([]models.ContractStatusRef, error) {
	rows, err := r.pool.Query(ctx, `SELECT code, name FROM contract_statuses ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []models.ContractStatusRef
	for rows.Next() {
		var s models.ContractStatusRef
		if err := rows.Scan(&s.Code, &s.Name); err != nil {
			return nil, err
		}
		res = append(res, s)
	}
	return res, rows.Err()
}

func (r *ContractRepo) ListTypes(ctx context.Context) ([]models.ContractTypeRef, error) {
	rows, err := r.pool.Query(ctx, `SELECT code, name FROM contract_types ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []models.ContractTypeRef
	for rows.Next() {
		var s models.ContractTypeRef
		if err := rows.Scan(&s.Code, &s.Name); err != nil {
			return nil, err
		}
		res = append(res, s)
	}
	return res, rows.Err()
}

func (r *ContractRepo) ListProcurementCardStatuses(ctx context.Context) ([]models.TrackingStatusRef, error) {
	rows, err := r.pool.Query(ctx, `SELECT code, name FROM procurement_card_statuses ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []models.TrackingStatusRef
	for rows.Next() {
		var s models.TrackingStatusRef
		if err := rows.Scan(&s.Code, &s.Name); err != nil {
			return nil, err
		}
		res = append(res, s)
	}
	return res, rows.Err()
}

func (r *ContractRepo) ListTrackingStatuses(ctx context.Context) ([]models.TrackingStatusRef, error) {
	rows, err := r.pool.Query(ctx, `SELECT code, name FROM tracking_statuses ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []models.TrackingStatusRef
	for rows.Next() {
		var s models.TrackingStatusRef
		if err := rows.Scan(&s.Code, &s.Name); err != nil {
			return nil, err
		}
		res = append(res, s)
	}
	return res, rows.Err()
}

// ListAssignees — пользователи, назначенные на договор (кроме главного ответственного).
func (r *ContractRepo) ListAssignees(ctx context.Context, contractID string) ([]models.User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.username, u.full_name, u.password_hash, u.role_code, u.is_active, u.theme, u.created_at, u.updated_at
		FROM contract_assignees ca
		JOIN users u ON u.id = ca.user_id
		WHERE ca.contract_id = $1
		ORDER BY u.full_name
	`, contractID)
	if err != nil {
		return nil, fmt.Errorf("list assignees: %w", err)
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
			return nil, err
		}
		res = append(res, u)
	}
	return res, rows.Err()
}

// SetAssignees — заменить список назначенных пользователей.
func (r *ContractRepo) SetAssignees(ctx context.Context, contractID string, userIDs []int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM contract_assignees WHERE contract_id = $1`, contractID); err != nil {
		return fmt.Errorf("clear assignees: %w", err)
	}

	for _, uid := range userIDs {
		if _, err := tx.Exec(ctx,
			`INSERT INTO contract_assignees (contract_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			contractID, uid,
		); err != nil {
			return fmt.Errorf("insert assignee: %w", err)
		}
	}
	return tx.Commit(ctx)
}

// ListActiveUsers — для выпадающих списков в форме договора.
func (r *ContractRepo) ListActiveUsers(ctx context.Context) ([]models.User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, username, full_name, password_hash, role_code, is_active, theme, created_at, updated_at
		FROM users
		WHERE is_active = TRUE
		ORDER BY full_name
	`)
	if err != nil {
		return nil, fmt.Errorf("list active users: %w", err)
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
			return nil, err
		}
		res = append(res, u)
	}
	return res, rows.Err()
}

// GetResponsible — обёртка для получения пользователя-ответственного.
func (r *ContractRepo) GetResponsible(ctx context.Context, userID int64) (*models.User, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, username, full_name, password_hash, role_code, is_active, theme, created_at, updated_at
		FROM users WHERE id = $1
	`, userID)
	var u models.User
	if err := row.Scan(
		&u.ID, &u.Username, &u.FullName, &u.PasswordHash,
		&u.RoleCode, &u.IsActive, &u.Theme,
		&u.CreatedAt, &u.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

// GetResponsibleUserID возвращает responsible_user_id договора.
// ok = false, если договор не найден или ответственный не назначен.
func (r *ContractRepo) GetResponsibleUserID(ctx context.Context, contractID string) (int64, bool) {
	var id *int64
	err := r.pool.QueryRow(ctx,
		`SELECT responsible_user_id FROM contracts WHERE id = $1`, contractID,
	).Scan(&id)
	if err != nil || id == nil {
		return 0, false
	}
	return *id, true
}
