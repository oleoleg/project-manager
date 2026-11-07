package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/oleoleg/project-manager/internal/db"
	"github.com/oleoleg/project-manager/internal/models"
)

type BudgetService struct {
	repo *db.BudgetRepo
}

func NewBudgetService(repo *db.BudgetRepo) *BudgetService {
	return &BudgetService{repo: repo}
}

func (s *BudgetService) GetSettings(ctx context.Context, contractID string) (models.BudgetSettings, bool, error) {
	set, err := s.repo.GetSettings(ctx, contractID)
	if err != nil {
		return models.BudgetSettings{}, false, err
	}
	if set == nil {
		return models.BudgetSettings{
			ContractID:          contractID,
			ProjectStartDate:    time.Now(),
			PaymentDelayDays:    20,
			EquipPaymentPercent: 0,
			EquipPaymentDelay:   2,
			OverheadRate:        1.2,
		}, false, nil
	}
	return *set, true, nil
}

func (s *BudgetService) SaveSettings(ctx context.Context, settings models.BudgetSettings) error {
	return s.repo.SaveSettings(ctx, &settings)
}

// GetLatestPlan возвращает последний (активный) план и его строки из budget_plan_lines
func (s *BudgetService) GetLatestPlan(ctx context.Context, contractID string) (*models.BudgetPlanSnapshot, []models.BudgetPlanLine, error) {
	snapshot, err := s.repo.GetLatestSnapshot(ctx, contractID)
	if err != nil {
		return nil, nil, err
	}
	if snapshot == nil {
		return nil, nil, nil
	}

	// Строки берем из активной таблицы
	lines, err := s.repo.GetPlanLines(ctx, contractID)
	if err != nil {
		return nil, nil, err
	}
	return snapshot, lines, nil
}

// GetPlanByID возвращает снимок и его строки (если снимок не активен, строки парсятся из JSON)
func (s *BudgetService) GetPlanByID(ctx context.Context, snapshotID string) (*models.BudgetPlanSnapshot, []models.BudgetPlanLine, error) {
	snapshot, err := s.repo.GetSnapshotByID(ctx, snapshotID)
	if err != nil || snapshot == nil {
		return nil, nil, err
	}

	// Если снимок активен, берем строки из бюджетной таблицы
	if snapshot.IsActive {
		lines, err := s.repo.GetPlanLines(ctx, snapshot.ContractID)
		if err != nil {
			return nil, nil, err
		}
		return snapshot, lines, nil
	}

	// Если снимок не активен, достаем строки из plan_data_snapshot (JSON)
	var lines []models.BudgetPlanLine
	if len(snapshot.PlanDataSnapshot) > 0 {
		_ = json.Unmarshal(snapshot.PlanDataSnapshot, &lines)
	}
	// Если JSON пуст, возвращаем пустой массив
	if lines == nil {
		lines = []models.BudgetPlanLine{}
	}
	return snapshot, lines, nil
}

// GeneratePlan генерирует снимок плана, отталкиваясь от СТРОК ЗАТРАТ.
func (s *BudgetService) GeneratePlan(ctx context.Context, contractID string, userID int64, note string) (string, error) {
	settings, hasSettings, err := s.GetSettings(ctx, contractID)
	if err != nil {
		return "", err
	}
	if !hasSettings {
		return "", fmt.Errorf("Сначала сохраните настройки бюджета")
	}

	stages, costs, err := s.repo.GetStagesForPlan(ctx, contractID)
	if err != nil {
		return "", fmt.Errorf("get data for plan: %w", err)
	}

	stageTotalCost := make(map[string]float64)
	for _, c := range costs {
		stageTotalCost[c.StageGUID] += (c.Cost + c.EquipmentSale)
	}

	costsByStage := make(map[string][]models.StageCost)
	for _, c := range costs {
		costsByStage[c.StageGUID] = append(costsByStage[c.StageGUID], c)
	}

	var lines []models.BudgetPlanLine

	for _, st := range stages {
		stageCosts := costsByStage[st.GUID]
		totalCostForStage := stageTotalCost[st.GUID]

		closeDate := settings.ProjectStartDate
		if st.CloseDatePlan != nil {
			closeDate = *st.CloseDatePlan
		}

		if len(stageCosts) == 0 {
			if st.AdvancePercent > 0 {
				advSum := st.Cost * (st.AdvancePercent / 100.0)
				if advSum > 0.01 {
					lines = append(lines, models.BudgetPlanLine{
						StageNumber:   st.StageNumber,
						StageName:     st.Name,
						WorkKind:      st.WorkKind,
						Name:          fmt.Sprintf("Аванс (%.1f%%). %s", st.AdvancePercent, st.Name),
						OperationDate: settings.ProjectStartDate,
						DelayDays:     2,
						PaymentDate:   settings.ProjectStartDate.AddDate(0, 0, 2),
						Income:        advSum,
						OperationType: "Платежи по договору",
					})
				}
			}
			remainSum := st.Cost - (st.Cost * (st.AdvancePercent / 100.0))
			if remainSum > 0.01 {
				remainPercent := 100.0 - st.AdvancePercent
				name := fmt.Sprintf("Оплата по договору. %s", st.Name)
				if remainPercent < 100.0 && remainPercent > 0 {
					name = fmt.Sprintf("Оплата по договору (%.1f%%). %s", remainPercent, st.Name)
				}
				lines = append(lines, models.BudgetPlanLine{
					StageNumber:   st.StageNumber,
					StageName:     st.Name,
					WorkKind:      st.WorkKind,
					Name:          name,
					OperationDate: closeDate,
					DelayDays:     settings.PaymentDelayDays,
					PaymentDate:   closeDate.AddDate(0, 0, settings.PaymentDelayDays),
					Income:        remainSum,
					OperationType: "Платежи по договору",
				})
			}
			continue
		}

		for _, c := range stageCosts {
			rowTotalSale := c.Cost + c.EquipmentSale
			ratio := 0.0
			if totalCostForStage > 0 {
				ratio = rowTotalSale / totalCostForStage
			}

			if st.AdvancePercent > 0 && rowTotalSale > 0 {
				advSum := (st.Cost * (st.AdvancePercent / 100.0)) * ratio
				if advSum > 0.01 {
					lines = append(lines, models.BudgetPlanLine{
						StageNumber:   st.StageNumber,
						StageName:     st.Name,
						WorkKind:      c.WorkKind,
						Name:          fmt.Sprintf("Аванс (%.1f%%). %s", st.AdvancePercent, st.Name),
						OperationDate: settings.ProjectStartDate,
						DelayDays:     2,
						PaymentDate:   settings.ProjectStartDate.AddDate(0, 0, 2),
						Income:        advSum,
						OperationType: "Платежи по договору",
					})
				}
			}

			if rowTotalSale > 0 {
				remainSum := (st.Cost - (st.Cost * (st.AdvancePercent / 100.0))) * ratio
				if remainSum > 0.01 {
					remainPercent := 100.0 - st.AdvancePercent
					name := fmt.Sprintf("Оплата по договору. %s", st.Name)
					if remainPercent < 100.0 && remainPercent > 0 {
						name = fmt.Sprintf("Оплата по договору (%.1f%%). %s", remainPercent, st.Name)
					}

					lines = append(lines, models.BudgetPlanLine{
						StageNumber:   st.StageNumber,
						StageName:     st.Name,
						WorkKind:      c.WorkKind,
						Name:          name,
						OperationDate: closeDate,
						DelayDays:     settings.PaymentDelayDays,
						PaymentDate:   closeDate.AddDate(0, 0, settings.PaymentDelayDays),
						Income:        remainSum,
						OperationType: "Платежи по договору",
					})
				}
			}

			if c.FOTPlan > 0 {
				opDate := closeDate.AddDate(0, 0, -15)
				lines = append(lines, models.BudgetPlanLine{
					StageNumber:   st.StageNumber,
					StageName:     st.Name,
					WorkKind:      c.WorkKind,
					Name:          "Зарплата сотрудникам. " + st.Name,
					OperationDate: opDate,
					DelayDays:     0,
					PaymentDate:   opDate,
					Expense:       c.FOTPlan,
					OperationType: "Зарплата сотрудникам",
				})

				overheadSum := c.FOTPlan * settings.OverheadRate
				lines = append(lines, models.BudgetPlanLine{
					StageNumber:   st.StageNumber,
					StageName:     st.Name,
					WorkKind:      c.WorkKind,
					Name:          "Накладные. " + st.Name,
					OperationDate: opDate,
					DelayDays:     0,
					PaymentDate:   opDate,
					Expense:       overheadSum,
					OperationType: "Накладные",
				})
			}

			if c.EquipmentCost > 0 {
				prepay := c.EquipmentCost * (settings.EquipPaymentPercent / 100.0)
				if prepay > 0.01 {
					lines = append(lines, models.BudgetPlanLine{
						StageNumber:   st.StageNumber,
						StageName:     st.Name,
						WorkKind:      c.WorkKind,
						Name:          fmt.Sprintf("Предоплата за оборудование (%.1f%%). %s", settings.EquipPaymentPercent, st.Name),
						OperationDate: settings.ProjectStartDate,
						DelayDays:     2,
						PaymentDate:   settings.ProjectStartDate.AddDate(0, 0, 2),
						Expense:       prepay,
						OperationType: "Закупка по договору",
					})
				}
				postpay := c.EquipmentCost * (1 - settings.EquipPaymentPercent/100.0)
				if postpay > 0.01 {
					postDate := settings.ProjectStartDate.AddDate(0, 0, settings.EquipPaymentDelay)
					postPercent := 100.0 - settings.EquipPaymentPercent
					lines = append(lines, models.BudgetPlanLine{
						StageNumber:   st.StageNumber,
						StageName:     st.Name,
						WorkKind:      c.WorkKind,
						Name:          fmt.Sprintf("Постоплата за оборудование (%.1f%%). %s", postPercent, st.Name),
						OperationDate: postDate,
						DelayDays:     0,
						PaymentDate:   postDate,
						Expense:       postpay,
						OperationType: "Закупка по договору",
					})
				}
			}

			if c.Subcontract > 0 {
				lines = append(lines, models.BudgetPlanLine{
					StageNumber:   st.StageNumber,
					StageName:     st.Name,
					WorkKind:      c.WorkKind,
					Name:          "Оплата субподрядчику. " + st.Name,
					OperationDate: closeDate,
					DelayDays:     0,
					PaymentDate:   closeDate,
					Expense:       c.Subcontract,
					OperationType: "Субподрядчики",
				})
			}

			if c.TravelExpenses > 0 {
				opDate := closeDate.AddDate(0, 0, -15)
				lines = append(lines, models.BudgetPlanLine{
					StageNumber:   st.StageNumber,
					StageName:     st.Name,
					WorkKind:      c.WorkKind,
					Name:          "Командировочные. " + st.Name,
					OperationDate: opDate,
					DelayDays:     0,
					PaymentDate:   opDate,
					Expense:       c.TravelExpenses,
					OperationType: "Командировки",
				})
			}

			if c.Commission > 0 {
				lines = append(lines, models.BudgetPlanLine{
					StageNumber:   st.StageNumber,
					StageName:     st.Name,
					WorkKind:      c.WorkKind,
					Name:          "Комиссия. " + st.Name,
					OperationDate: closeDate,
					DelayDays:     0,
					PaymentDate:   closeDate,
					Expense:       c.Commission,
					OperationType: "Комиссия",
				})
			}

			if c.OtherExpenses > 0 {
				lines = append(lines, models.BudgetPlanLine{
					StageNumber:   st.StageNumber,
					StageName:     st.Name,
					WorkKind:      c.WorkKind,
					Name:          "Расходы разные. " + st.Name,
					OperationDate: closeDate,
					DelayDays:     0,
					PaymentDate:   closeDate,
					Expense:       c.OtherExpenses,
					OperationType: "Другое (расходы)",
				})
			}
		}
	}

	return s.repo.CreateSnapshotWithLines(ctx, contractID, userID, note, settings, lines, costs, stages)
}

func (s *BudgetService) GetAllSnapshots(ctx context.Context, contractID string) ([]models.BudgetPlanSnapshot, error) {
	return s.repo.GetAllSnapshots(ctx, contractID)
}

func (s *BudgetService) DeleteSnapshot(ctx context.Context, snapshotID string) error {
	return s.repo.DeleteSnapshot(ctx, snapshotID)
}

func (s *BudgetService) ApproveSnapshot(ctx context.Context, contractID string, snapshotID string, userID int64) error {
	return s.repo.ApproveSnapshot(ctx, contractID, snapshotID, userID)
}

func (s *BudgetService) UnapproveSnapshot(ctx context.Context, snapshotID string) error {
	return s.repo.UnapproveSnapshot(ctx, snapshotID)
}

func (s *BudgetService) GetActivePlanTotals(ctx context.Context, contractID string) (float64, bool, error) {
	return s.repo.GetActivePlanTotals(ctx, contractID)
}

func (s *BudgetService) GenerateFactFromPlan(ctx context.Context, contractID string) error {
	return s.repo.GenerateFactFromPlan(ctx, contractID)
}

func (s *BudgetService) GetFactLines(ctx context.Context, contractID string) ([]models.BudgetFactLine, error) {
	return s.repo.GetFactLines(ctx, contractID)
}

func (s *BudgetService) UpdateFactLine(ctx context.Context, id string, factIncome, factExpense float64, isConfirmed bool) error {
	return s.repo.UpdateFactLine(ctx, id, factIncome, factExpense, isConfirmed)
}

func (s *BudgetService) DeleteFact(ctx context.Context, contractID string) error {
	return s.repo.DeleteFact(ctx, contractID)
}

func (s *BudgetService) GetFactTotal(ctx context.Context, contractID string) (float64, error) {
	return s.repo.GetFactTotal(ctx, contractID)
}

func (s *BudgetService) GetFactConfirmedTotal(ctx context.Context, contractID string) (float64, error) {
	return s.repo.GetFactConfirmedTotal(ctx, contractID)
}

func (s *BudgetService) GetFactCheckData(ctx context.Context, contractID string) (map[string]float64, error) {
	return s.repo.GetFactCheckData(ctx, contractID)
}

func (s *BudgetService) UpdateFactLinesBatch(ctx context.Context, updates []db.FactUpdateData) error {
	return s.repo.UpdateFactLinesBatch(ctx, updates)
}

func (s *BudgetService) BatchUpdateFactLines(
	ctx context.Context,
	contractID string,
	ids, names, workKindCodes, opTypes, stageNumbers, stageNames []string,
	opDates, delayDays, payDates []string,
	incIncomes, incExpenses, confirms []string,
) error {
	return s.repo.BatchUpdateFactLines(
		ctx,
		contractID,
		ids, names, workKindCodes, opTypes, stageNumbers, stageNames,
		opDates, delayDays, payDates,
		incIncomes, incExpenses, confirms,
	)
}

func (s *BudgetService) RestoreFromSnapshot(ctx context.Context, contractID, snapshotID string) error {
	return s.repo.RestoreFromSnapshot(ctx, contractID, snapshotID)
}

func (s *BudgetService) GetPlanFactSummary(ctx context.Context, contractID string, onlyConfirmed bool) ([]db.PlanFactRow, error) {
	return s.repo.GetPlanFactSummary(ctx, contractID, onlyConfirmed)
}

func (s *BudgetService) AutoConfirmSalaryAndOverhead(ctx context.Context, contractID string) error {
	return s.repo.AutoConfirmSalaryAndOverhead(ctx, contractID)
}

func (s *BudgetService) DeleteFactLine(ctx context.Context, id string) error {
	return s.repo.DeleteFactLine(ctx, id)
}

func (s *BudgetService) AppendMissingFactLinesFromPlan(ctx context.Context, contractID string) error {
	return s.repo.AppendMissingFactLinesFromPlan(ctx, contractID)
}

func (s *BudgetService) GetFactCostTotalsByStage(ctx context.Context, contractID string) (map[string]float64, error) {
	return s.repo.GetFactCostTotalsByStage(ctx, contractID)
}

func (s *BudgetService) GetPlanCostTotalsByStage(ctx context.Context, contractID string) (map[string]float64, error) {
	return s.repo.GetPlanCostTotalsByStage(ctx, contractID)
}

func (s *BudgetService) GetMissingPlanLines(ctx context.Context, contractID string) ([]models.BudgetPlanLine, error) {
	return s.repo.GetMissingPlanLines(ctx, contractID)
}

func (s *BudgetService) AppendSelectedFactLines(ctx context.Context, contractID string, planLineIDs []string) error {
	return s.repo.AppendSelectedFactLines(ctx, contractID, planLineIDs)
}
