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
	id, contract_number, contract_date, work_name, responsible_person,
	status_code, counterparty_id, workshop, project_url,
	procurement_card_status, extra_details, finplan_status, plan_schedule_status,
	short_name, contract_type, created_at, updated_at
`

func scanContract(row pgx.Row, c *models.Contract) error {
	return row.Scan(
		&c.ID, &c.ContractNumber, &c.ContractDate, &c.WorkName, &c.ResponsiblePerson,
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
			status_code, counterparty_id, workshop, project_url,
			procurement_card_status, extra_details, finplan_status, plan_schedule_status,
			short_name, contract_type
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
	`,
		c.ID, c.ContractNumber, c.ContractDate, c.WorkName, c.ResponsiblePerson,
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
			status_code = $6, counterparty_id = $7, workshop = $8, project_url = $9,
			procurement_card_status = $10, extra_details = $11, finplan_status = $12,
			plan_schedule_status = $13, short_name = $14, contract_type = $15,
			updated_at = NOW()
		WHERE id = $1
	`,
		c.ID, c.ContractNumber, c.ContractDate, c.WorkName, c.ResponsiblePerson,
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
