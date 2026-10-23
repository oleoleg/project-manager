package services

import (
	"context"
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

func (s *BudgetService) GetLatestPlan(ctx context.Context, contractID string) (*models.BudgetPlanSnapshot, []models.BudgetPlanLine, error) {
	snapshot, err := s.repo.GetLatestSnapshot(ctx, contractID)
	if err != nil {
		return nil, nil, err
	}
	if snapshot == nil {
		return nil, nil, nil
	}

	lines, err := s.repo.GetPlanLines(ctx, snapshot.ID)
	if err != nil {
		return nil, nil, err
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

	// Считаем общую стоимость всех затрат по этапу (для распределения Аванса/Оплаты)
	stageTotalCost := make(map[string]float64)
	for _, c := range costs {
		stageTotalCost[c.StageGUID] += c.Cost
	}

	// Группируем затраты по этапам
	costsByStage := make(map[string][]models.StageCost)
	for _, c := range costs {
		costsByStage[c.StageGUID] = append(costsByStage[c.StageGUID], c)
	}

	var lines []models.BudgetPlanLine

	// Перебираем ЭТАПЫ, чтобы взять их даты и проценты
	for _, st := range stages {
		stageCosts := costsByStage[st.GUID]
		totalCostForStage := stageTotalCost[st.GUID]

		// Дата закрытия (для операций)
		closeDate := settings.ProjectStartDate
		if st.CloseDatePlan != nil {
			closeDate = *st.CloseDatePlan
		}

		// Если у этапа НЕТ затрат, но есть Аванс/Оплата — выводим их с видом работы самого этапа
		if len(stageCosts) == 0 {
			if st.AdvancePercent > 0 {
				advSum := st.Cost * (st.AdvancePercent / 100.0)
				lines = append(lines, models.BudgetPlanLine{
					StageGUID:     &st.GUID,
					WorkKindCode:  st.WorkKindCode,
					Name:          "Аванс ." + st.Name,
					OperationDate: settings.ProjectStartDate,
					DelayDays:     2,
					PaymentDate:   settings.ProjectStartDate.AddDate(0, 0, 2),
					Income:        advSum,
					OperationType: "Платежи по договору",
				})
			}
			remainSum := st.Cost - (st.Cost * (st.AdvancePercent / 100.0))
			if remainSum > 0 {
				lines = append(lines, models.BudgetPlanLine{
					StageGUID:     &st.GUID,
					WorkKindCode:  st.WorkKindCode,
					Name:          "Оплата по договору. " + st.Name,
					OperationDate: closeDate,
					DelayDays:     settings.PaymentDelayDays,
					PaymentDate:   closeDate.AddDate(0, 0, settings.PaymentDelayDays),
					Income:        remainSum,
					OperationType: "Платежи по договору",
				})
			}
			continue
		}

		// Если затраты ЕСТЬ, перебираем каждую строку затрат
		for _, c := range stageCosts {
			// Считаем долю этой строки в общей стоимости работ этапа
			ratio := 0.0
			if totalCostForStage > 0 {
				ratio = c.Cost / totalCostForStage
			}

			// 1. АВАНС (распределяем по строкам пропорционально стоимости)
			if st.AdvancePercent > 0 {
				advSum := (st.Cost * (st.AdvancePercent / 100.0)) * ratio
				lines = append(lines, models.BudgetPlanLine{
					StageGUID:     &st.GUID,
					WorkKindCode:  c.WorkKindCode,
					Name:          fmt.Sprintf("Аванс. %s (%.1f%%)", st.Name, ratio*100),
					OperationDate: settings.ProjectStartDate,
					DelayDays:     2,
					PaymentDate:   settings.ProjectStartDate.AddDate(0, 0, 2),
					Income:        advSum,
					OperationType: "Платежи по договору",
				})
			}

			// 2. ОПЛАТА ПО ДОГОВОРУ
			remainSum := (st.Cost - (st.Cost * (st.AdvancePercent / 100.0))) * ratio
			if remainSum > 0 {
				lines = append(lines, models.BudgetPlanLine{
					StageGUID:     &st.GUID,
					WorkKindCode:  c.WorkKindCode,
					Name:          fmt.Sprintf("Оплата по договору. %s (%.1f%%)", st.Name, ratio*100),
					OperationDate: closeDate,
					DelayDays:     settings.PaymentDelayDays,
					PaymentDate:   closeDate.AddDate(0, 0, settings.PaymentDelayDays),
					Income:        remainSum,
					OperationType: "Платежи по договору",
				})
			}

			// 3. ЗАРПЛАТА (ФОТп)
			if c.FOTPlan > 0 {
				opDate := closeDate.AddDate(0, 0, -15)
				lines = append(lines, models.BudgetPlanLine{
					StageGUID:     &st.GUID,
					WorkKindCode:  c.WorkKindCode,
					Name:          "Зарплата сотрудникам. " + st.Name,
					OperationDate: opDate,
					DelayDays:     0,
					PaymentDate:   opDate,
					Expense:       c.FOTPlan,
					OperationType: "Зарплата сотрудникам",
				})

				// 4. НАКЛАДНЫЕ
				overheadSum := c.FOTPlan * settings.OverheadRate
				lines = append(lines, models.BudgetPlanLine{
					StageGUID:     &st.GUID,
					WorkKindCode:  c.WorkKindCode,
					Name:          "Накладные. " + st.Name,
					OperationDate: opDate,
					DelayDays:     0,
					PaymentDate:   opDate,
					Expense:       overheadSum,
					OperationType: "Накладные",
				})
			}

			// 5. ОБОРУДОВАНИЕ
			if c.EquipmentCost > 0 {
				prepay := c.EquipmentCost * (settings.EquipPaymentPercent / 100.0)
				if prepay > 0 {
					lines = append(lines, models.BudgetPlanLine{
						StageGUID:     &st.GUID,
						WorkKindCode:  c.WorkKindCode,
						Name:          "Предоплата за оборудование. " + st.Name,
						OperationDate: settings.ProjectStartDate,
						DelayDays:     2,
						PaymentDate:   settings.ProjectStartDate.AddDate(0, 0, 2),
						Expense:       prepay,
						OperationType: "Закупка по договору",
					})
				}
				postpay := c.EquipmentCost * (1 - settings.EquipPaymentPercent/100.0)
				if postpay > 0 {
					postDate := settings.ProjectStartDate.AddDate(0, 0, settings.EquipPaymentDelay)
					lines = append(lines, models.BudgetPlanLine{
						StageGUID:     &st.GUID,
						WorkKindCode:  c.WorkKindCode,
						Name:          "Постоплата за оборудование. " + st.Name,
						OperationDate: postDate,
						DelayDays:     0,
						PaymentDate:   postDate,
						Expense:       postpay,
						OperationType: "Закупка по договору",
					})
				}
			}

			// 6. СУБПОДРЯД
			if c.Subcontract > 0 {
				lines = append(lines, models.BudgetPlanLine{
					StageGUID:     &st.GUID,
					WorkKindCode:  c.WorkKindCode,
					Name:          "Оплата субподрядчику. " + st.Name,
					OperationDate: closeDate,
					DelayDays:     0,
					PaymentDate:   closeDate,
					Expense:       c.Subcontract,
					OperationType: "Субподрядчики",
				})
			}

			// 7. КОМАНДИРОВКИ
			if c.TravelExpenses > 0 {
				opDate := closeDate.AddDate(0, 0, -15)
				lines = append(lines, models.BudgetPlanLine{
					StageGUID:     &st.GUID,
					WorkKindCode:  c.WorkKindCode,
					Name:          "Командировочные. " + st.Name,
					OperationDate: opDate,
					DelayDays:     0,
					PaymentDate:   opDate,
					Expense:       c.TravelExpenses,
					OperationType: "Командировки",
				})
			}

			// 8. КОМИССИЯ
			if c.Commission > 0 {
				lines = append(lines, models.BudgetPlanLine{
					StageGUID:     &st.GUID,
					WorkKindCode:  c.WorkKindCode,
					Name:          "Комиссия. " + st.Name,
					OperationDate: closeDate,
					DelayDays:     0,
					PaymentDate:   closeDate,
					Expense:       c.Commission,
					OperationType: "Комиссия",
				})
			}

			// 9. ДРУГОЕ
			if c.OtherExpenses > 0 {
				lines = append(lines, models.BudgetPlanLine{
					StageGUID:     &st.GUID,
					WorkKindCode:  c.WorkKindCode,
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

	return s.repo.CreateSnapshotWithLines(ctx, contractID, userID, note, settings, lines)
}

func (s *BudgetService) GetAllSnapshots(ctx context.Context, contractID string) ([]models.BudgetPlanSnapshot, error) {
	return s.repo.GetAllSnapshots(ctx, contractID)
}

func (s *BudgetService) DeleteSnapshot(ctx context.Context, snapshotID string) error {
	return s.repo.DeleteSnapshot(ctx, snapshotID)
}

// GetPlanByID возвращает снимок и его строки по ID снимка.
func (s *BudgetService) GetPlanByID(ctx context.Context, snapshotID string) (*models.BudgetPlanSnapshot, []models.BudgetPlanLine, error) {
	snapshot, err := s.repo.GetSnapshotByID(ctx, snapshotID)
	if err != nil || snapshot == nil {
		return nil, nil, err
	}

	lines, err := s.repo.GetPlanLines(ctx, snapshot.ID)
	if err != nil {
		return nil, nil, err
	}
	return snapshot, lines, nil
}

// SetActiveSnapshot делает снимок активным.
func (s *BudgetService) SetActiveSnapshot(ctx context.Context, contractID, snapshotID string) error {
	return s.repo.SetActiveSnapshot(ctx, contractID, snapshotID)
}

func (s *BudgetService) ApproveSnapshot(ctx context.Context, snapshotID string, userID int64) error {
	return s.repo.ApproveSnapshot(ctx, snapshotID, userID)
}
