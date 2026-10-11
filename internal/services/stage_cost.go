package services

import (
	"context"
	"errors"
	"strings"

	"github.com/oleoleg/project-manager/internal/db"
	"github.com/oleoleg/project-manager/internal/models"
)

var (
	ErrStageCostWorkKindEmpty = errors.New("вид работы в затратах обязателен")
	ErrStageCostComplex       = errors.New("вид работы «Комплексный» нельзя выбрать внутри затрат")
	ErrStageCostNegative      = errors.New("суммы не могут быть отрицательными")
	ErrStageCostRatioRange    = errors.New("коэффициент должен быть от 0 до 1")
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
	c.WorkKindCode = strings.TrimSpace(c.WorkKindCode)
	if c.WorkKindCode == "" {
		return ErrStageCostWorkKindEmpty
	}
	if c.WorkKindCode == "complex" {
		return ErrStageCostComplex
	}
	if c.Cost < 0 || c.FOTPlan < 0 || c.LaborDays < 0 || c.RatePerDay < 0 ||
		c.TravelExpenses < 0 || c.EquipmentSale < 0 || c.EquipmentCost < 0 ||
		c.Subcontract < 0 || c.Commission < 0 || c.OtherExpenses < 0 {
		return ErrStageCostNegative
	}
	if c.FOTRatio < 0 || c.FOTRatio > 1 {
		return ErrStageCostRatioRange
	}
	return nil
}

// Recalculate пересчитывает ФОТп и коэф, если они не заданы вручную.
//   - если rate > 0 и days > 0 → ФОТп = rate * days
//   - если cost > 0 → коэф = ФОТп / cost
func (s *StageCostService) Recalculate(c *models.StageCost) {
	if c.RatePerDay > 0 && c.LaborDays > 0 {
		c.FOTPlan = c.RatePerDay * c.LaborDays
	}
	if c.Cost > 0 {
		c.FOTRatio = c.FOTPlan / c.Cost
		if c.FOTRatio > 1 {
			c.FOTRatio = 1
		}
	}
}

func (s *StageCostService) ReplaceForStage(ctx context.Context, stageGUID string, costs []models.StageCost) error {
	// валидация + пересчёт
	for i := range costs {
		s.Recalculate(&costs[i])
		if err := s.Validate(&costs[i]); err != nil {
			return err
		}
	}
	return s.repo.ReplaceForStage(ctx, stageGUID, costs)
}
