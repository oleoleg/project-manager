package services

import (
	"context"
	"errors"
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

func (s *StageCostService) Recalculate(c *models.StageCost) {
	locks := parseLocks(c.LockedField)

	hasFOT := locks["fot_plan"]
	hasDays := locks["labor_days"]
	hasRate := locks["rate_per_day"]

	switch {
	case hasFOT && hasDays:
		if c.LaborDays > 0 {
			c.RatePerDay = c.FOTPlan / c.LaborDays
		}
	case hasFOT && hasRate:
		if c.RatePerDay > 0 {
			c.LaborDays = c.FOTPlan / c.RatePerDay
		}
	case hasDays && hasRate:
		c.FOTPlan = c.LaborDays * c.RatePerDay
	case hasFOT:
		if c.LaborDays > 0 {
			c.RatePerDay = c.FOTPlan / c.LaborDays
		}
	case hasDays:
		if c.LaborDays > 0 {
			c.RatePerDay = c.FOTPlan / c.LaborDays
		}
	case hasRate:
		if c.LaborDays > 0 {
			c.FOTPlan = c.RatePerDay * c.LaborDays
		}
	default:
		if c.RatePerDay > 0 && c.LaborDays > 0 {
			c.FOTPlan = c.RatePerDay * c.LaborDays
		}
	}
}

func parseLocks(s string) map[string]bool {
	res := map[string]bool{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			res[part] = true
		}
	}
	return res
}

func (s *StageCostService) ReplaceForStage(ctx context.Context, stageGUID string, costs []models.StageCost) error {
	seen := map[string]bool{}
	for _, c := range costs {
		if seen[c.WorkKind] {
			return ErrStageCostDuplicateWorkKind
		}
		seen[c.WorkKind] = true
	}

	for i := range costs {
		s.Recalculate(&costs[i])
		if err := s.Validate(&costs[i]); err != nil {
			return err
		}
	}
	return s.repo.ReplaceForStage(ctx, stageGUID, costs)
}

func (s *StageCostService) GetAllCostsForContract(ctx context.Context, contractID string) ([]models.StageCost, error) {
	return s.repo.GetAllCostsForContract(ctx, contractID)
}
