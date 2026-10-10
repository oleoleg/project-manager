package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oleoleg/project-manager/internal/models"
)

var ErrStageNotFound = errors.New("stage not found")

type StageRepo struct {
	pool *pgxpool.Pool
}

func NewStageRepo(pool *pgxpool.Pool) *StageRepo {
	return &StageRepo{pool: pool}
}

const stageColumns = `
	guid, contract_id, row_order, stage_number, name, cost, closed_by_acts,
	vat_percent, advance_percent, start_conditions, end_conditions,
	close_date_plan, responsible_person, status_code, work_kind_code, note,
	created_at, updated_at
`

func scanStage(row pgx.Row, s *models.Stage) error {
	return row.Scan(
		&s.GUID, &s.ContractID, &s.RowOrder, &s.StageNumber, &s.Name, &s.Cost, &s.ClosedByActs,
		&s.VATPercent, &s.AdvancePercent, &s.StartConditions, &s.EndConditions,
		&s.CloseDatePlan, &s.ResponsiblePerson, &s.StatusCode, &s.WorkKindCode, &s.Note,
		&s.CreatedAt, &s.UpdatedAt,
	)
}

func (r *StageRepo) List(ctx context.Context, contractID string) ([]models.Stage, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+stageColumns+`
		FROM stages
		WHERE contract_id = $1
		ORDER BY row_order, stage_number, created_at
	`, contractID)
	if err != nil {
		return nil, fmt.Errorf("list stages: %w", err)
	}
	defer rows.Close()

	var result []models.Stage
	for rows.Next() {
		var s models.Stage
		if err := scanStage(rows, &s); err != nil {
			return nil, fmt.Errorf("scan stage: %w", err)
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *StageRepo) GetByID(ctx context.Context, guid string) (*models.Stage, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+stageColumns+` FROM stages WHERE guid = $1`, guid)
	var s models.Stage
	if err := scanStage(row, &s); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStageNotFound
		}
		return nil, fmt.Errorf("get stage: %w", err)
	}
	return &s, nil
}

func (r *StageRepo) Create(ctx context.Context, s *models.Stage) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO stages (
			contract_id, row_order, stage_number, name, cost, closed_by_acts,
			vat_percent, advance_percent, start_conditions, end_conditions,
			close_date_plan, responsible_person, status_code, work_kind_code, note
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		RETURNING guid, created_at, updated_at
	`,
		s.ContractID, s.RowOrder, s.StageNumber, s.Name, s.Cost, s.ClosedByActs,
		s.VATPercent, s.AdvancePercent, s.StartConditions, s.EndConditions,
		s.CloseDatePlan, s.ResponsiblePerson, s.StatusCode, s.WorkKindCode, s.Note,
	).Scan(&s.GUID, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create stage: %w", err)
	}
	return nil
}

func (r *StageRepo) Update(ctx context.Context, s *models.Stage) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE stages SET
			row_order = $2, stage_number = $3, name = $4, cost = $5, closed_by_acts = $6,
			vat_percent = $7, advance_percent = $8, start_conditions = $9, end_conditions = $10,
			close_date_plan = $11, responsible_person = $12, status_code = $13,
			work_kind_code = $14, note = $15, updated_at = NOW()
		WHERE guid = $1
	`,
		s.GUID, s.RowOrder, s.StageNumber, s.Name, s.Cost, s.ClosedByActs,
		s.VATPercent, s.AdvancePercent, s.StartConditions, s.EndConditions,
		s.CloseDatePlan, s.ResponsiblePerson, s.StatusCode, s.WorkKindCode, s.Note,
	)
	if err != nil {
		return fmt.Errorf("update stage: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrStageNotFound
	}
	return nil
}

func (r *StageRepo) Delete(ctx context.Context, guid string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM stages WHERE guid = $1`, guid)
	if err != nil {
		return fmt.Errorf("delete stage: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrStageNotFound
	}
	return nil
}

// --- справочники ---

func (r *StageRepo) ListStatuses(ctx context.Context) ([]models.StageStatusRef, error) {
	rows, err := r.pool.Query(ctx, `SELECT code, name FROM stage_statuses ORDER BY code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []models.StageStatusRef
	for rows.Next() {
		var s models.StageStatusRef
		if err := rows.Scan(&s.Code, &s.Name); err != nil {
			return nil, err
		}
		res = append(res, s)
	}
	return res, rows.Err()
}

func (r *StageRepo) ListWorkKinds(ctx context.Context) ([]models.WorkKindRef, error) {
	rows, err := r.pool.Query(ctx, `SELECT code, name, is_complex FROM work_kinds ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []models.WorkKindRef
	for rows.Next() {
		var w models.WorkKindRef
		if err := rows.Scan(&w.Code, &w.Name, &w.IsComplex); err != nil {
			return nil, err
		}
		res = append(res, w)
	}
	return res, rows.Err()
}

// --- contract_cards ---

// EnsureCard создаёт запись в contract_cards, если её ещё нет.
func (r *StageRepo) EnsureCard(ctx context.Context, contractID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO contract_cards (contract_id)
		VALUES ($1)
		ON CONFLICT (contract_id) DO NOTHING
	`, contractID)
	if err != nil {
		return fmt.Errorf("ensure contract card: %w", err)
	}
	return nil
}

func (r *StageRepo) GetCard(ctx context.Context, contractID string) (*models.ContractCard, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT contract_id, overhead_percent, created_at, updated_at
		FROM contract_cards WHERE contract_id = $1
	`, contractID)
	var c models.ContractCard
	if err := row.Scan(&c.ContractID, &c.OverheadPercent, &c.CreatedAt, &c.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *StageRepo) UpdateCardOverhead(ctx context.Context, contractID string, percent float64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE contract_cards SET overhead_percent = $2, updated_at = NOW()
		WHERE contract_id = $1
	`, contractID, percent)
	return err
}

// NextRowOrder — следующий порядковый номер для нового этапа.
func (r *StageRepo) NextRowOrder(ctx context.Context, contractID string) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(row_order), 0) + 1 FROM stages WHERE contract_id = $1
	`, contractID).Scan(&n)
	if err != nil {
		return 0, err
	}
	return n, nil
}

// StageTotals — итоги по договору.
type StageTotals struct {
	Count       int
	TotalCost   float64
	TotalClosed float64
}

func (r *StageRepo) Totals(ctx context.Context, contractID string) (StageTotals, error) {
	var t StageTotals
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(SUM(cost),0), COALESCE(SUM(closed_by_acts),0)
		FROM stages WHERE contract_id = $1
	`, contractID).Scan(&t.Count, &t.TotalCost, &t.TotalClosed)
	if err != nil {
		return t, fmt.Errorf("stage totals: %w", err)
	}
	return t, nil
}
