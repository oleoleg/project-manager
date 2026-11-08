package services

import (
	"context"
	"errors"
	"math"
	"strings"

	"github.com/oleoleg/project-manager/internal/db"
	"github.com/oleoleg/project-manager/internal/models"
)

var (
	ErrStageCostWorkKindEmpty     = errors.New("вид работы в затратах обязателен")
	ErrStageCostComplex           = errors.New("вид работы «Комплексный» нельзя выбрать внутри затрат")
	ErrStageCostNegative          = errors.New("суммы не могут быть отрицательными")
	ErrStageCostDuplicateWorkKind = errors.New("вид работы не может повторяться в одном этапе")
)

type StageCostService struct {
	repo *db.StageCostRepo
}

func NewStageCostService(repo *db.StageCostRepo) *StageCostService {
	return &StageCostService{repo: repo}
}

func (s *StageCostService) ListByStage(ctx context.Context, stageGUID string) ([]models.StageCost, error) {
	return s.repo.ListByStage(ctx, stageGUID)
}

func (s *StageCostService) Validate(c *models.StageCost) error {
	c.WorkKind = strings.TrimSpace(c.WorkKind)
	if c.WorkKind == "" {
		return ErrStageCostWorkKindEmpty
	}
	if c.WorkKind == "Комплексный" {
		return ErrStageCostComplex
	}
	if c.Cost < 0 || c.FOTPlan < 0 || c.LaborDays < 0 || c.RatePerDay < 0 ||
		c.TravelExpenses < 0 || c.EquipmentSale < 0 || c.EquipmentCost < 0 ||
		c.Subcontract < 0 || c.Commission < 0 || c.OtherExpenses < 0 {
		return ErrStageCostNegative
	}
	return nil
}

// RecalculateCosts пересчитывает одно из трех полей (ФОТп, дни, ставка)
// на основе двух других, в зависимости от того, какое поле указано в CalcField.
func (s *StageCostService) RecalculateCosts(costs []models.StageCost) {
	for i := range costs {
		c := &costs[i]

		// Если ничего не вычисляется, пропускаем
		if c.CalcField == "" {
			continue
		}

		switch c.CalcField {
		case "fot_plan":
			// Вычисляем ФОТп = Ставка * Дни
			c.FOTPlan = c.RatePerDay * c.LaborDays
		case "labor_days":
			// Вычисляем Дни = ФОТп / Ставка
			if c.RatePerDay > 0 {
				c.LaborDays = math.Round(c.FOTPlan / c.RatePerDay)
			} else {
				c.LaborDays = 0
			}
		case "rate_per_day":
			// Вычисляем Ставку = ФОТп / Дни
			if c.LaborDays > 0 {
				c.RatePerDay = c.FOTPlan / c.LaborDays
			} else {
				c.RatePerDay = 0
			}
		}
	}
}

func (s *StageCostService) ReplaceForStage(ctx context.Context, stageGUID string, costs []models.StageCost) error {
	seen := map[string]bool{}
	for _, c := range costs {
		if seen[c.WorkKind] {
			return ErrStageCostDuplicateWorkKind
		}
		seen[c.WorkKind] = true
	}

	// Вызываем пересчет
	s.RecalculateCosts(costs)

	for i := range costs {
		if err := s.Validate(&costs[i]); err != nil {
			return err
		}
	}
	return s.repo.ReplaceForStage(ctx, stageGUID, costs)
}

func (s *StageCostService) GetAllCostsForContract(ctx context.Context, contractID string) ([]models.StageCost, error) {
	return s.repo.GetAllCostsForContract(ctx, contractID)
}
