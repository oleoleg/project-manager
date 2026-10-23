package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oleoleg/project-manager/internal/models"
)

type BudgetRepo struct {
	pool *pgxpool.Pool
}

func NewBudgetRepo(pool *pgxpool.Pool) *BudgetRepo {
	return &BudgetRepo{pool: pool}
}

// GetSettings получает настройки бюджета по ID договора.
func (r *BudgetRepo) GetSettings(ctx context.Context, contractID string) (*models.BudgetSettings, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT contract_id, project_start_date, project_end_date, payment_delay_days, 
		       COALESCE(equip_payment_percent, 0), COALESCE(equip_payment_delay, 0), overhead_rate,
		       created_at, updated_at
		FROM budget_settings WHERE contract_id = $1
	`, contractID)

	var s models.BudgetSettings
	err := row.Scan(
		&s.ContractID, &s.ProjectStartDate, &s.ProjectEndDate, &s.PaymentDelayDays,
		&s.EquipPaymentPercent, &s.EquipPaymentDelay, &s.OverheadRate,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get budget settings: %w", err)
	}
	return &s, nil
}

// SaveSettings сохраняет или обновляет настройки бюджета.
func (r *BudgetRepo) SaveSettings(ctx context.Context, s *models.BudgetSettings) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO budget_settings (
			contract_id, project_start_date, project_end_date, payment_delay_days,
			equip_payment_percent, equip_payment_delay, overhead_rate, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (contract_id) DO UPDATE SET
			project_start_date = EXCLUDED.project_start_date,
			project_end_date = EXCLUDED.project_end_date,
			payment_delay_days = EXCLUDED.payment_delay_days,
			equip_payment_percent = EXCLUDED.equip_payment_percent,
			equip_payment_delay = EXCLUDED.equip_payment_delay,
			overhead_rate = EXCLUDED.overhead_rate,
			updated_at = NOW()
	`, s.ContractID, s.ProjectStartDate, s.ProjectEndDate, s.PaymentDelayDays,
		s.EquipPaymentPercent, s.EquipPaymentDelay, s.OverheadRate)

	if err != nil {
		return fmt.Errorf("save budget settings: %w", err)
	}
	return nil
}

// GetLatestSnapshot получает ID последнего снимка плана для договора.
func (r *BudgetRepo) GetLatestSnapshot(ctx context.Context, contractID string) (*models.BudgetPlanSnapshot, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT s.id, s.contract_id, s.created_at, s.created_by, COALESCE(u.full_name, '—'), 
		       COALESCE(s.note, ''), COALESCE(s.is_active, false), COALESCE(s.settings, '{}'::jsonb), 
		       s.approved_by, s.approved_at, COALESCE(a.full_name, '—')
		FROM budget_plan_snapshots s
		LEFT JOIN users u ON u.id = s.created_by
		LEFT JOIN users a ON a.id = s.approved_by
		WHERE s.contract_id = $1 
		ORDER BY s.created_at DESC LIMIT 1
	`, contractID)

	var s models.BudgetPlanSnapshot
	var settingsBytes []byte
	var approvedAt *time.Time
	var approvedBy *int64

	err := row.Scan(
		&s.ID, &s.ContractID, &s.CreatedAt, &s.CreatedBy, &s.CreatorName,
		&s.Note, &s.IsActive, &settingsBytes, &approvedBy, &approvedAt, &s.ApproverName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest snapshot: %w", err)
	}
	s.Settings = settingsBytes
	s.ApprovedBy = approvedBy
	s.ApprovedAt = approvedAt
	return &s, nil
}

// GetPlanLines получает строки плана для конкретного снимка.
func (r *BudgetRepo) GetPlanLines(ctx context.Context, snapshotID string) ([]models.BudgetPlanLine, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, snapshot_id, stage_guid, work_kind_code, name, operation_date, 
		       delay_days, payment_date, income, expense, operation_type, note
		FROM budget_plan_lines 
		WHERE snapshot_id = $1
		ORDER BY operation_date, id
	`, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("get plan lines: %w", err)
	}
	defer rows.Close()

	var lines []models.BudgetPlanLine
	for rows.Next() {
		var l models.BudgetPlanLine
		if err := rows.Scan(
			&l.ID, &l.SnapshotID, &l.StageGUID, &l.WorkKindCode, &l.Name,
			&l.OperationDate, &l.DelayDays, &l.PaymentDate, &l.Income, &l.Expense,
			&l.OperationType, &l.Note,
		); err != nil {
			return nil, fmt.Errorf("scan plan line: %w", err)
		}
		l.Balance = l.Income - l.Expense
		lines = append(lines, l)
	}
	return lines, rows.Err()
}

// CreateSnapshotWithLines создает снимок и сохраняет все его строки в транзакции.
func (r *BudgetRepo) CreateSnapshotWithLines(ctx context.Context, contractID string, userID int64, note string, settings models.BudgetSettings, lines []models.BudgetPlanLine) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var snapshotID string
	settingsJSON, _ := json.Marshal(settings)

	err = tx.QueryRow(ctx, `
		INSERT INTO budget_plan_snapshots (contract_id, created_by, note, settings) 
		VALUES ($1, $2, $3, $4) RETURNING id
	`, contractID, userID, note, settingsJSON).Scan(&snapshotID)
	if err != nil {
		return "", fmt.Errorf("create snapshot: %w", err)
	}

	insertQuery := `
		INSERT INTO budget_plan_lines (
			snapshot_id, stage_guid, work_kind_code, name, operation_date, delay_days, 
			payment_date, income, expense, operation_type, note
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	for _, l := range lines {
		_, err := tx.Exec(ctx, insertQuery,
			snapshotID, l.StageGUID, l.WorkKindCode, l.Name, l.OperationDate, l.DelayDays,
			l.PaymentDate, l.Income, l.Expense, l.OperationType, l.Note,
		)
		if err != nil {
			return "", fmt.Errorf("insert plan line: %w", err)
		}
	}

	// Новый снимок сразу делаем активным
	_, err = tx.Exec(ctx, `UPDATE budget_plan_snapshots SET is_active = FALSE WHERE contract_id = $1`, contractID)
	if err != nil {
		return "", err
	}
	_, err = tx.Exec(ctx, `UPDATE budget_plan_snapshots SET is_active = TRUE WHERE id = $1`, snapshotID)
	if err != nil {
		return "", err
	}

	return snapshotID, tx.Commit(ctx)
}

// GetStagesForPlan возвращает этапы договора и их затраты.
func (r *BudgetRepo) GetStagesForPlan(ctx context.Context, contractID string) ([]models.Stage, []models.StageCost, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+stageColumns+` FROM stages WHERE contract_id = $1`, contractID)
	if err != nil {
		return nil, nil, fmt.Errorf("get stages: %w", err)
	}
	defer rows.Close()

	var stages []models.Stage
	for rows.Next() {
		var s models.Stage
		if err := scanStage(rows, &s); err != nil {
			return nil, nil, fmt.Errorf("scan stage: %w", err)
		}
		stages = append(stages, s)
	}

	costRows, err := r.pool.Query(ctx, `
		SELECT `+stageCostColumns+` 
		FROM stage_costs 
		WHERE stage_guid IN (SELECT guid FROM stages WHERE contract_id = $1)
	`, contractID)
	if err != nil {
		return nil, nil, fmt.Errorf("get stage costs: %w", err)
	}
	defer costRows.Close()

	var costs []models.StageCost
	for costRows.Next() {
		var c models.StageCost
		if err := scanStageCost(costRows, &c); err != nil {
			return nil, nil, fmt.Errorf("scan stage cost: %w", err)
		}
		costs = append(costs, c)
	}

	return stages, costs, nil
}

// GetAllSnapshots возвращает список всех снимков для договора.
func (r *BudgetRepo) GetAllSnapshots(ctx context.Context, contractID string) ([]models.BudgetPlanSnapshot, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT s.id, s.contract_id, s.created_at, s.created_by, COALESCE(u.full_name, '—'), 
		       COALESCE(s.note, ''), COALESCE(s.is_active, false), COALESCE(s.settings, '{}'::jsonb), 
		       s.approved_by, s.approved_at, COALESCE(a.full_name, '—')
		FROM budget_plan_snapshots s
		LEFT JOIN users u ON u.id = s.created_by
		LEFT JOIN users a ON a.id = s.approved_by
		WHERE s.contract_id = $1 
		ORDER BY s.created_at DESC
	`, contractID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []models.BudgetPlanSnapshot
	for rows.Next() {
		var s models.BudgetPlanSnapshot
		var settingsBytes []byte
		var approvedAt *time.Time
		var approvedBy *int64

		if err := rows.Scan(
			&s.ID, &s.ContractID, &s.CreatedAt, &s.CreatedBy, &s.CreatorName,
			&s.Note, &s.IsActive, &settingsBytes, &approvedBy, &approvedAt, &s.ApproverName,
		); err != nil {
			return nil, err
		}
		s.Settings = settingsBytes
		s.ApprovedBy = approvedBy
		s.ApprovedAt = approvedAt
		res = append(res, s)
	}
	return res, rows.Err()
}

// DeleteSnapshot удаляет снимок и все его строки.
func (r *BudgetRepo) DeleteSnapshot(ctx context.Context, snapshotID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM budget_plan_snapshots WHERE id = $1`, snapshotID)
	return err
}

// GetSnapshotByID получает один снимок по ID.
func (r *BudgetRepo) GetSnapshotByID(ctx context.Context, snapshotID string) (*models.BudgetPlanSnapshot, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT s.id, s.contract_id, s.created_at, s.created_by, COALESCE(u.full_name, '—'), 
		       COALESCE(s.note, ''), COALESCE(s.is_active, false), COALESCE(s.settings, '{}'::jsonb), 
		       s.approved_by, s.approved_at, COALESCE(a.full_name, '—')
		FROM budget_plan_snapshots s
		LEFT JOIN users u ON u.id = s.created_by
		LEFT JOIN users a ON a.id = s.approved_by
		WHERE s.id = $1
	`, snapshotID)

	var s models.BudgetPlanSnapshot
	var settingsBytes []byte
	var approvedAt *time.Time
	var approvedBy *int64

	err := row.Scan(
		&s.ID, &s.ContractID, &s.CreatedAt, &s.CreatedBy, &s.CreatorName,
		&s.Note, &s.IsActive, &settingsBytes, &approvedBy, &approvedAt, &s.ApproverName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get snapshot by id: %w", err)
	}
	s.Settings = settingsBytes
	s.ApprovedBy = approvedBy
	s.ApprovedAt = approvedAt
	return &s, nil
}

// SetActiveSnapshot делает указанный снимок активным.
func (r *BudgetRepo) SetActiveSnapshot(ctx context.Context, contractID, snapshotID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `UPDATE budget_plan_snapshots SET is_active = FALSE WHERE contract_id = $1`, contractID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `UPDATE budget_plan_snapshots SET is_active = TRUE WHERE id = $1`, snapshotID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *BudgetRepo) ApproveSnapshot(ctx context.Context, snapshotID string, userID int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE budget_plan_snapshots 
		SET approved_by = $1, approved_at = NOW() 
		WHERE id = $2
	`, userID, snapshotID)
	return err
}
