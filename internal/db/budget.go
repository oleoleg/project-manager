package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
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
		SELECT l.id, l.snapshot_id, l.stage_guid, COALESCE(s.stage_number, ''), l.work_kind_code, l.name, 
		       l.operation_date, l.delay_days, l.payment_date, l.income, l.expense, l.operation_type, l.note
		FROM budget_plan_lines l
		LEFT JOIN stages s ON s.guid = l.stage_guid
		WHERE l.snapshot_id = $1
		ORDER BY l.operation_date, l.id
	`, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("get plan lines: %w", err)
	}
	defer rows.Close()

	var lines []models.BudgetPlanLine
	for rows.Next() {
		var l models.BudgetPlanLine
		if err := rows.Scan(
			&l.ID, &l.SnapshotID, &l.StageGUID, &l.StageNumber, &l.WorkKindCode, &l.Name,
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
func (r *BudgetRepo) CreateSnapshotWithLines(ctx context.Context, contractID string, userID int64, note string, settings models.BudgetSettings, lines []models.BudgetPlanLine, costs []models.StageCost, stages []models.Stage) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var snapshotID string
	settingsJSON, _ := json.Marshal(settings)
	costsJSON, _ := json.Marshal(costs)
	stagesJSON, _ := json.Marshal(stages) // <--- НОВОЕ

	err = tx.QueryRow(ctx, `
		INSERT INTO budget_plan_snapshots (contract_id, created_by, note, settings, stage_costs_snapshot, stage_data_snapshot) 
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id
	`, contractID, userID, note, settingsJSON, costsJSON, stagesJSON).Scan(&snapshotID) // <--- ДОБАВИЛИ $6
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
		FROM stage_costs sc
		WHERE sc.stage_guid IN (SELECT guid FROM stages WHERE contract_id = $1)
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
		       COALESCE(s.stage_costs_snapshot, '[]'::jsonb), COALESCE(s.stage_data_snapshot, '[]'::jsonb),
		       s.approved_by, s.approved_at, COALESCE(a.full_name, '—')
		FROM budget_plan_snapshots s
		LEFT JOIN users u ON u.id = s.created_by
		LEFT JOIN users a ON a.id = s.approved_by
		WHERE s.id = $1
	`, snapshotID)

	var s models.BudgetPlanSnapshot
	var settingsBytes, costsBytes, stagesBytes []byte
	var approvedAt *time.Time
	var approvedBy *int64

	err := row.Scan(
		&s.ID, &s.ContractID, &s.CreatedAt, &s.CreatedBy, &s.CreatorName,
		&s.Note, &s.IsActive, &settingsBytes, &costsBytes, &stagesBytes,
		&approvedBy, &approvedAt, &s.ApproverName,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get snapshot by id: %w", err)
	}
	s.Settings = settingsBytes
	s.StageCostsSnapshot = costsBytes
	s.StageDataSnapshot = stagesBytes
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

// ApproveSnapshot утверждает снимок и делает его единственным утвержденным и активным.
func (r *BudgetRepo) ApproveSnapshot(ctx context.Context, contractID string, snapshotID string, userID int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Снимаем утверждение со всех остальных снимков этого договора
	_, err = tx.Exec(ctx, `
		UPDATE budget_plan_snapshots 
		SET approved_by = NULL, approved_at = NULL 
		WHERE contract_id = $1
	`, contractID)
	if err != nil {
		return err
	}

	// 2. Утверждаем текущий и делаем его активным
	_, err = tx.Exec(ctx, `
		UPDATE budget_plan_snapshots 
		SET approved_by = $1, approved_at = NOW(), is_active = TRUE 
		WHERE id = $2
	`, userID, snapshotID)
	if err != nil {
		return err
	}

	// 3. Снимаем is_active с других снимков
	_, err = tx.Exec(ctx, `
		UPDATE budget_plan_snapshots 
		SET is_active = FALSE 
		WHERE contract_id = $1 AND id != $2
	`, contractID, snapshotID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// UnapproveSnapshot отзывает утверждение снимка.
func (r *BudgetRepo) UnapproveSnapshot(ctx context.Context, snapshotID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE budget_plan_snapshots 
		SET approved_by = NULL, approved_at = NULL 
		WHERE id = $1
	`, snapshotID)
	return err
}

// GetActivePlanTotals возвращает сумму затрат по активному снимку плана и флаг его утверждения.
func (r *BudgetRepo) GetActivePlanTotals(ctx context.Context, contractID string) (float64, bool, error) {
	var total float64
	var isApproved bool
	err := r.pool.QueryRow(ctx, `
		SELECT 
			COALESCE(SUM(l.expense), 0),
			COALESCE(s.approved_at IS NOT NULL, false)
		FROM budget_plan_snapshots s
		LEFT JOIN budget_plan_lines l ON l.snapshot_id = s.id
		WHERE s.contract_id = $1 AND s.is_active = TRUE
		GROUP BY s.id, s.approved_at
	`, contractID).Scan(&total, &isApproved)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil // Активного плана нет
		}
		return 0, false, err
	}
	return total, isApproved, nil
}

// GenerateFactFromPlan создает фактические строки на основе активного плана.
// GenerateFactFromPlan создает фактические строки на основе активного плана.
func (r *BudgetRepo) GenerateFactFromPlan(ctx context.Context, contractID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1. Получаем ID активного снимка плана
	var snapshotID string
	err = tx.QueryRow(ctx, `SELECT id FROM budget_plan_snapshots WHERE contract_id = $1 AND is_active = TRUE LIMIT 1`, contractID).Scan(&snapshotID)
	if err != nil {
		return fmt.Errorf("активный план не найден: %w", err)
	}

	// 2. Очищаем старый факт
	_, err = tx.Exec(ctx, `DELETE FROM budget_fact_lines WHERE contract_id = $1`, contractID)
	if err != nil {
		return err
	}

	// 3. Получаем строки плана
	rows, err := tx.Query(ctx, `
		SELECT id, stage_guid, work_kind_code, name, operation_date, delay_days, payment_date, operation_type, income, expense
		FROM budget_plan_lines WHERE snapshot_id = $1
	`, snapshotID)
	if err != nil {
		return err
	}
	defer rows.Close()

	type PlanLine struct {
		ID           string
		StageGUID    *string
		WorkKindCode string
		Name         string
		OpDate       time.Time
		DelayDays    int
		PayDate      time.Time
		OpType       string
		Income       float64
		Expense      float64
	}
	var lines []PlanLine
	for rows.Next() {
		var l PlanLine
		// ВНИМАТЕЛЬНО: Проверь порядок полей в SELECT и Scan!
		if err := rows.Scan(
			&l.ID,
			&l.StageGUID,
			&l.WorkKindCode,
			&l.Name,
			&l.OpDate,
			&l.DelayDays,
			&l.PayDate,
			&l.OpType,
			&l.Income,
			&l.Expense,
		); err != nil {
			return err
		}
		lines = append(lines, l)
	}
	rows.Close()

	// 4. Вставляем в факт
	now := time.Now()
	insertQuery := `
		INSERT INTO budget_fact_lines (
			contract_id, plan_line_id, stage_guid, work_kind_code, name, operation_date, delay_days, payment_date, 
			operation_type, plan_income, plan_expense, fact_income, fact_expense, is_confirmed, is_manual_edit
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	for _, l := range lines {
		// Логика автоподтверждения для ЗП и Накладных
		isConfirmed := false
		if l.OpType == "Зарплата сотрудникам" || l.OpType == "Накладные" {
			if l.OpDate.Before(now) || l.OpDate.Format("2006-01") == now.Format("2006-01") {
				isConfirmed = true
			}
		}

		_, err := tx.Exec(ctx, insertQuery,
			contractID,     // $1
			l.ID,           // $2
			l.StageGUID,    // $3
			l.WorkKindCode, // $4
			l.Name,         // $5
			l.OpDate,       // $6
			l.DelayDays,    // $7
			l.PayDate,      // $8
			l.OpType,       // $9
			l.Income,       // $10
			l.Expense,      // $11
			l.Income,       // $12
			l.Expense,      // $13
			isConfirmed,    // $14
			false,          // $15
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// Метод получения списка факта
func (r *BudgetRepo) GetFactLines(ctx context.Context, contractID string) ([]models.BudgetFactLine, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT f.id, f.contract_id, f.plan_line_id, f.stage_guid, COALESCE(s.stage_number, ''), COALESCE(s.name, ''), f.work_kind_code, f.name,  
		       f.operation_date, f.delay_days, f.payment_date,
		       TRIM(f.operation_type) as operation_type, f.plan_income, f.plan_expense, f.fact_income, f.fact_expense, 
		       f.is_confirmed, f.is_manual_edit, f.created_at, f.updated_at
		FROM budget_fact_lines f
		LEFT JOIN stages s ON s.guid = f.stage_guid
		WHERE f.contract_id = $1
		ORDER BY 
			CASE TRIM(f.operation_type)
				WHEN 'Платежи по договору' THEN 1
				WHEN 'Закупка по договору' THEN 2
				WHEN 'Зарплата сотрудникам' THEN 3
				WHEN 'Накладные' THEN 4
				WHEN 'Командировки' THEN 5
				WHEN 'Субподрядчики' THEN 6
				WHEN 'Комиссия' THEN 7
				WHEN 'Другое (расходы)' THEN 8
				ELSE 9
			END,
			f.operation_date, f.id
	`, contractID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []models.BudgetFactLine
	for rows.Next() {
		var l models.BudgetFactLine
		var planLineID *string
		var stageGUID *string

		if err := rows.Scan(
			&l.ID, &l.ContractID, &l.PlanLineID, &l.StageGUID, &l.StageNumber, &l.StageName,
			&l.WorkKindCode, &l.Name, &l.OperationDate, &l.DelayDays, &l.PaymentDate,
			&l.OperationType, &l.PlanIncome, &l.PlanExpense, &l.FactIncome, &l.FactExpense,
			&l.IsConfirmed, &l.IsManualEdit, &l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, err
		}
		l.PlanLineID = planLineID
		l.StageGUID = stageGUID
		res = append(res, l)
	}
	return res, rows.Err()
}

// Метод обновления суммы (когда редактируют руками)
func (r *BudgetRepo) UpdateFactLine(ctx context.Context, id string, factIncome, factExpense float64, isConfirmed bool) error {
	// Проверяем, отличается ли новая сумма от плановой
	var planInc, planExp float64
	err := r.pool.QueryRow(ctx, `SELECT plan_income, plan_expense FROM budget_fact_lines WHERE id = $1`, id).Scan(&planInc, &planExp)
	if err != nil {
		return err
	}

	isManual := math.Abs(factIncome-planInc) > 0.01 || math.Abs(factExpense-planExp) > 0.01

	_, err = r.pool.Exec(ctx, `
		UPDATE budget_fact_lines 
		SET fact_income = $2, fact_expense = $3, is_confirmed = $4, is_manual_edit = $5, updated_at = NOW()
		WHERE id = $1
	`, id, factIncome, factExpense, isConfirmed, isManual)
	return err
}

// Метод удаления факта (Опасная зона)
func (r *BudgetRepo) DeleteFact(ctx context.Context, contractID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM budget_fact_lines WHERE contract_id = $1`, contractID)
	return err
}

// GetFactTotal возвращает общую сумму затрат (расходов) по факту.
func (r *BudgetRepo) GetFactTotal(ctx context.Context, contractID string) (float64, error) {
	var total float64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(fact_expense), 0)
		FROM budget_fact_lines
		WHERE contract_id = $1
	`, contractID).Scan(&total)
	return total, err
}

// GetFactConfirmedTotal возвращает сумму подтвержденных (квитированных) затрат.
func (r *BudgetRepo) GetFactConfirmedTotal(ctx context.Context, contractID string) (float64, error) {
	var total float64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(fact_expense), 0)
		FROM budget_fact_lines
		WHERE contract_id = $1 AND is_confirmed = TRUE
	`, contractID).Scan(&total)
	return total, err
}

// GetFactCheckData собирает данные для проверки факта (суммы актов и стоимость этапов).
func (r *BudgetRepo) GetFactCheckData(ctx context.Context, contractID string) (map[string]float64, error) {
	result := make(map[string]float64)

	// 1. Сумма всех актов (Закрыто по актам) по договору
	var totalActs float64
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(closed_by_acts), 0) 
		FROM stages 
		WHERE contract_id = $1
	`, contractID).Scan(&totalActs)
	if err != nil {
		return nil, err
	}
	result["total_acts"] = totalActs

	// 2. Общая стоимость договора (по этапам)
	var totalCost float64
	err = r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(cost), 0) 
		FROM stages 
		WHERE contract_id = $1
	`, contractID).Scan(&totalCost)
	if err != nil {
		return nil, err
	}
	result["total_cost"] = totalCost

	return result, nil
}

type FactUpdateData struct {
	ID           string
	FactIncome   float64
	FactExpense  float64
	IsConfirmed  bool
	IsManualEdit bool
}

// UpdateFactLinesBatch массово обновляет строки факта.
func (r *BudgetRepo) UpdateFactLinesBatch(ctx context.Context, updates []FactUpdateData) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, u := range updates {
		_, err := tx.Exec(ctx, `
			UPDATE budget_fact_lines 
			SET fact_income = $2, fact_expense = $3, is_confirmed = $4, is_manual_edit = $5, updated_at = NOW()
			WHERE id = $1
		`, u.ID, u.FactIncome, u.FactExpense, u.IsConfirmed, u.IsManualEdit)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// BatchUpdateFactLines полностью обновляет строки факта.
func (r *BudgetRepo) BatchUpdateFactLines(
	ctx context.Context,
	contractID string,
	ids, names, workKindCodes, opTypes, stageGUIDs []string,
	opDates, delayDays, payDates []string,
	incIncomes, incExpenses, confirms []string,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	confirmMap := make(map[string]bool)
	for _, id := range confirms {
		if id != "" {
			confirmMap[id] = true
		}
	}

	getStr := func(arr []string, i int) string {
		if i < len(arr) {
			return arr[i]
		}
		return ""
	}

	// Получаем все ID, которые были в базе до обновления
	rows, _ := tx.Query(ctx, `SELECT id FROM budget_fact_lines WHERE contract_id = $1`, contractID)
	existingIDs := make(map[string]bool)
	for rows.Next() {
		var id string
		_ = rows.Scan(&id)
		existingIDs[id] = true
	}
	rows.Close()

	processedIDs := make(map[string]bool)

	for i, id := range ids {
		opType := getStr(opTypes, i)
		workKind := getStr(workKindCodes, i)
		stageGUID := getStr(stageGUIDs, i)
		name := getStr(names, i)
		opDateStr := getStr(opDates, i)
		delayStr := getStr(delayDays, i)
		payDateStr := getStr(payDates, i)
		incStr := getStr(incIncomes, i)
		expStr := getStr(incExpenses, i)

		opDate, _ := time.Parse("2006-01-02", opDateStr)
		payDate, _ := time.Parse("2006-01-02", payDateStr)
		delay, _ := strconv.Atoi(delayStr)
		inc, _ := strconv.ParseFloat(strings.ReplaceAll(incStr, " ", ""), 64)
		exp, _ := strconv.ParseFloat(strings.ReplaceAll(expStr, " ", ""), 64)

		if workKind == "" && id != "" {
			_ = tx.QueryRow(ctx,
				`SELECT work_kind_code FROM budget_fact_lines WHERE id = $1`, id,
			).Scan(&workKind)
		}
		if workKind == "" {
			workKind = "complex"
		}

		// Преобразуем пустой GUID в nil (NULL в БД)
		var stageGUIDPtr *string
		if stageGUID != "" {
			stageGUIDPtr = &stageGUID
		} else if id != "" {
			// Если GUID не пришел с фронта (пустой), берем текущий из базы, чтобы не затереть
			var existingGUID *string
			_ = tx.QueryRow(ctx, `SELECT stage_guid FROM budget_fact_lines WHERE id = $1`, id).Scan(&existingGUID)
			stageGUIDPtr = existingGUID
		}

		if id == "" {
			// НОВАЯ СТРОКА
			_, err := tx.Exec(ctx, `
				INSERT INTO budget_fact_lines 
					(contract_id, stage_guid, work_kind_code, name, operation_date, delay_days, payment_date, operation_type, fact_income, fact_expense, is_confirmed)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			`, contractID, stageGUIDPtr, workKind, name, opDate, delay, payDate, opType, inc, exp, confirmMap[id])
			if err != nil {
				return err
			}
		} else {
			// ОБНОВЛЕНИЕ
			processedIDs[id] = true

			// Обновляем строку
			processedIDs[id] = true
			_, err := tx.Exec(ctx, `
				UPDATE budget_fact_lines 
				SET stage_guid = $2, work_kind_code = $3, name = $4, operation_date = $5, delay_days = $6, payment_date = $7,
				    fact_income = $8, fact_expense = $9, is_confirmed = $10, updated_at = NOW()
				WHERE id = $1
			`, id, stageGUIDPtr, workKind, name, opDate, delay, payDate, inc, exp, confirmMap[id])
			if err != nil {
				return err
			}
		}
	}

	// Удаление
	for oldID := range existingIDs {
		if !processedIDs[oldID] {
			_, err := tx.Exec(ctx, `DELETE FROM budget_fact_lines WHERE id = $1`, oldID)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit(ctx)
}

// RestoreFromSnapshot восстанавливает этапы и затраты из снимка
func (r *BudgetRepo) RestoreFromSnapshot(ctx context.Context, contractID string, snapshotID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var stagesJSON, costsJSON []byte
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(stage_data_snapshot, '[]'::jsonb), COALESCE(stage_costs_snapshot, '[]'::jsonb)
		FROM budget_plan_snapshots WHERE id = $1
	`, snapshotID).Scan(&stagesJSON, &costsJSON)
	if err != nil {
		return err
	}

	var stages []models.Stage
	var costs []models.StageCost
	_ = json.Unmarshal(stagesJSON, &stages)
	_ = json.Unmarshal(costsJSON, &costs)

	// 1. Удаляем текущие затраты и этапы
	_, err = tx.Exec(ctx, `DELETE FROM stage_costs WHERE stage_guid IN (SELECT guid FROM stages WHERE contract_id = $1)`, contractID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `DELETE FROM stages WHERE contract_id = $1`, contractID)
	if err != nil {
		return err
	}

	// 2. Вставляем этапы из снимка
	for _, s := range stages {
		_, err = tx.Exec(ctx, `
			INSERT INTO stages (
				guid, contract_id, row_order, stage_number, name, cost, closed_by_acts, closed_by_acts_manual,
				vat_percent, advance_percent, start_conditions, end_conditions,
				close_date_plan, responsible_person, status_code, work_kind_code, note
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		`, s.GUID, contractID, s.RowOrder, s.StageNumber, s.Name, s.Cost, s.ClosedByActs, s.ClosedByActsManual,
			s.VATPercent, s.AdvancePercent, s.StartConditions, s.EndConditions,
			s.CloseDatePlan, s.ResponsiblePerson, s.StatusCode, s.WorkKindCode, s.Note)
		if err != nil {
			return err
		}
	}

	// 3. Вставляем затраты из снимка
	for _, c := range costs {
		_, err = tx.Exec(ctx, `
			INSERT INTO stage_costs (
				stage_guid, work_kind_code, cost, fot_plan, labor_days, rate_per_day, travel_expenses,
				equipment_sale, equipment_cost, subcontract, commission, other_expenses, locked_field
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		`, c.StageGUID, c.WorkKindCode, c.Cost, c.FOTPlan, c.LaborDays, c.RatePerDay, c.TravelExpenses,
			c.EquipmentSale, c.EquipmentCost, c.Subcontract, c.Commission, c.OtherExpenses, c.LockedField)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

type PlanFactRow struct {
	WorkKindCode  string // код вида работы (pir, ppo и т.д.)
	OperationType string // тип операции
	PlanIncome    float64
	PlanExpense   float64
	PlanBalance   float64
	FactIncome    float64
	FactExpense   float64
	FactBalance   float64
	Deviation     float64 // Факт.Баланс - План.Баланс
}

// GetPlanFactSummary собирает сводку План vs Факт по Видам работ и Типам операций
// GetPlanFactSummary собирает сводку План vs Факт по Видам работ и Типам операций
func (r *BudgetRepo) GetPlanFactSummary(ctx context.Context, contractID string, onlyConfirmed bool) ([]PlanFactRow, error) {

	factFilter := ""
	if onlyConfirmed {
		factFilter = " AND is_confirmed = TRUE "
	}

	query := `
		WITH plan AS (
			SELECT 
				l.work_kind_code, 
				l.operation_type, 
				SUM(l.income) as income, 
				SUM(l.expense) as expense,
				SUM(l.income - l.expense) as balance
			FROM budget_plan_lines l
			JOIN budget_plan_snapshots s ON s.id = l.snapshot_id
			WHERE s.contract_id = $1 AND s.is_active = TRUE
			GROUP BY l.work_kind_code, l.operation_type
		),
		fact AS (
			SELECT 
				work_kind_code, 
				operation_type, 
				SUM(fact_income) as income, 
				SUM(fact_expense) as expense,
				SUM(fact_income - fact_expense) as balance
			FROM budget_fact_lines
			WHERE contract_id = $1 ` + factFilter + `
			GROUP BY work_kind_code, operation_type
		)
		SELECT 
			COALESCE(p.work_kind_code, f.work_kind_code) as work_kind_code,
			COALESCE(p.operation_type, f.operation_type) as operation_type,
			COALESCE(p.income, 0) as plan_income,
			COALESCE(p.expense, 0) as plan_expense,
			COALESCE(p.balance, 0) as plan_balance,
			COALESCE(f.income, 0) as fact_income,
			COALESCE(f.expense, 0) as fact_expense,
			COALESCE(f.balance, 0) as fact_balance,
			(COALESCE(f.balance, 0) - COALESCE(p.balance, 0)) as deviation
		FROM plan p
		FULL OUTER JOIN fact f ON p.work_kind_code = f.work_kind_code AND p.operation_type = f.operation_type
	`

	rows, err := r.pool.Query(ctx, query, contractID)
	if err != nil {
		return nil, fmt.Errorf("get plan fact summary: %w", err)
	}
	defer rows.Close()

	var res []PlanFactRow
	for rows.Next() {
		var row PlanFactRow
		if err := rows.Scan(
			&row.WorkKindCode, &row.OperationType,
			&row.PlanIncome, &row.PlanExpense, &row.PlanBalance,
			&row.FactIncome, &row.FactExpense, &row.FactBalance,
			&row.Deviation,
		); err != nil {
			return nil, err
		}
		res = append(res, row)
	}
	return res, rows.Err()
}
