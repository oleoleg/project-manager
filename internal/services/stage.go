package services

import (
	"context"
	"errors"
	"strings"

	"github.com/oleoleg/project-manager/internal/db"
	"github.com/oleoleg/project-manager/internal/models"
)

var (
	ErrStageNameRequired   = errors.New("наименование этапа обязательно")
	ErrStageWorkKindNeeded = errors.New("вид работы обязателен")
)

type StageService struct {
	repo *db.StageRepo
}

func NewStageService(repo *db.StageRepo) *StageService {
	return &StageService{repo: repo}
}

func (s *StageService) Validate(st *models.Stage) error {
	st.Name = strings.TrimSpace(st.Name)
	if st.Name == "" {
		return ErrStageNameRequired
	}
	if st.WorkKindCode == "" {
		return ErrStageWorkKindNeeded
	}
	if st.Cost < 0 || st.ClosedByActs < 0 {
		return ErrStageCostNegative
	}
	return nil
}

func (s *StageService) List(ctx context.Context, contractID string) ([]models.Stage, error) {
	return s.repo.List(ctx, contractID)
}

func (s *StageService) GetByID(ctx context.Context, guid string) (*models.Stage, error) {
	return s.repo.GetByID(ctx, guid)
}

func (s *StageService) Create(ctx context.Context, st *models.Stage) error {
	if err := s.Validate(st); err != nil {
		return err
	}
	if st.RowOrder == 0 {
		n, err := s.repo.NextRowOrder(ctx, st.ContractID)
		if err != nil {
			return err
		}
		st.RowOrder = n
	}
	if err := s.repo.Create(ctx, st); err != nil {
		return err
	}
	// нормализуем порядок после вставки
	return s.repo.NormalizeOrder(ctx, st.ContractID)
}

func (s *StageService) Update(ctx context.Context, st *models.Stage) error {
	if err := s.Validate(st); err != nil {
		return err
	}
	return s.repo.Update(ctx, st)
}

func (s *StageService) Delete(ctx context.Context, contractID, guid string) error {
	if err := s.repo.Delete(ctx, guid); err != nil {
		return err
	}
	// пересобираем порядок
	return s.repo.NormalizeOrder(ctx, contractID)
}

func (s *StageService) Statuses(ctx context.Context) ([]models.StageStatusRef, error) {
	return s.repo.ListStatuses(ctx)
}

func (s *StageService) WorkKinds(ctx context.Context) ([]models.WorkKindRef, error) {
	return s.repo.ListWorkKinds(ctx)
}

func (s *StageService) EnsureCard(ctx context.Context, contractID string) error {
	return s.repo.EnsureCard(ctx, contractID)
}

func (s *StageService) GetCard(ctx context.Context, contractID string) (*models.ContractCard, error) {
	return s.repo.GetCard(ctx, contractID)
}

func (s *StageService) UpdateCardOverhead(ctx context.Context, contractID string, percent float64) error {
	return s.repo.UpdateCardOverhead(ctx, contractID, percent)
}

func (s *StageService) Totals(ctx context.Context, contractID string) (db.StageTotals, error) {
	return s.repo.Totals(ctx, contractID)
}

func (s *StageService) Move(ctx context.Context, contractID, guid string, direction int) error {
	if err := s.repo.SwapOrder(ctx, contractID, guid, direction); err != nil {
		return err
	}
	// пересобираем порядок, чтобы не было дублей
	return s.repo.NormalizeOrder(ctx, contractID)
}

func (s *StageService) ProblemStages(ctx context.Context, contractID string) ([]db.ProblemStage, error) {
	return s.repo.ProblemStages(ctx, contractID)
}

func (s *StageService) StageCostTotals(ctx context.Context, contractID string) (map[string]float64, error) {
	return s.repo.StageCostTotals(ctx, contractID)
}

func (s *StageService) StageActSums(ctx context.Context, contractID string) (map[string]float64, error) {
	return s.repo.StageActSums(ctx, contractID)
}
