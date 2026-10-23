package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/oleoleg/project-manager/internal/models"
)

type StageCostRepo struct {
	pool *pgxpool.Pool
}

func NewStageCostRepo(pool *pgxpool.Pool) *StageCostRepo {
	return &StageCostRepo{pool: pool}
}

const stageCostColumns = `
	sc.guid, sc.stage_guid, sc.work_kind_code, sc.cost, sc.fot_plan, 
	sc.labor_days, sc.rate_per_day, sc.travel_expenses,
	sc.equipment_sale, sc.equipment_cost, sc.subcontract, sc.commission, sc.other_expenses,
	sc.locked_field, sc.created_at, sc.updated_at
`

func scanStageCost(row interface {
	Scan(dest ...any) error
}, c *models.StageCost) error {
	return row.Scan(
		&c.GUID, &c.StageGUID, &c.WorkKindCode, &c.Cost, &c.FOTPlan,
		&c.LaborDays, &c.RatePerDay, &c.TravelExpenses,
		&c.EquipmentSale, &c.EquipmentCost, &c.Subcontract, &c.Commission, &c.OtherExpenses,
		&c.LockedField, &c.CreatedAt, &c.UpdatedAt,
	)
}

func (r *StageCostRepo) ListByStage(ctx context.Context, stageGUID string) ([]models.StageCost, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+stageCostColumns+`
		FROM stage_costs sc
		WHERE stage_guid = $1
		ORDER BY created_at
	`, stageGUID)
	if err != nil {
		return nil, fmt.Errorf("list stage costs: %w", err)
	}
	defer rows.Close()

	var res []models.StageCost
	for rows.Next() {
		var c models.StageCost
		if err := scanStageCost(rows, &c); err != nil {
			return nil, fmt.Errorf("scan stage cost: %w", err)
		}
		res = append(res, c)
	}
	return res, rows.Err()
}

// ReplaceForStage атомарно перезаписывает все затраты этапа.
func (r *StageCostRepo) ReplaceForStage(ctx context.Context, stageGUID string, costs []models.StageCost) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM stage_costs WHERE stage_guid = $1`, stageGUID); err != nil {
		return fmt.Errorf("clear stage costs: %w", err)
	}

	for _, c := range costs {
		_, err := tx.Exec(ctx, `
			INSERT INTO stage_costs (
				stage_guid, work_kind_code, cost, fot_plan, 
				labor_days, rate_per_day, travel_expenses,
				equipment_sale, equipment_cost, subcontract, commission, other_expenses, locked_field
			) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		`,
			stageGUID, c.WorkKindCode, c.Cost, c.FOTPlan,
			c.LaborDays, c.RatePerDay, c.TravelExpenses,
			c.EquipmentSale, c.EquipmentCost, c.Subcontract, c.Commission, c.OtherExpenses, c.LockedField,
		)
		if err != nil {
			return fmt.Errorf("insert stage cost: %w", err)
		}
	}
	return tx.Commit(ctx)
}

// GetAllCostsForContract возвращает плоский список затрат по всем этапам договора для сводной таблицы.
func (r *StageCostRepo) GetAllCostsForContract(ctx context.Context, contractID string) ([]models.StageCost, error) {
	query := `
		SELECT ` + stageCostColumns + `
		FROM stage_costs sc
		JOIN stages s ON s.guid = sc.stage_guid
		WHERE s.contract_id = $1
		ORDER BY s.row_order, s.stage_number, sc.work_kind_code
	`
	rows, err := r.pool.Query(ctx, query, contractID)
	if err != nil {
		return nil, fmt.Errorf("list all contract costs: %w", err)
	}
	defer rows.Close()

	var res []models.StageCost
	for rows.Next() {
		var c models.StageCost
		if err := scanStageCost(rows, &c); err != nil {
			return nil, fmt.Errorf("scan stage cost: %w", err)
		}
		res = append(res, c)
	}
	return res, rows.Err()
}
