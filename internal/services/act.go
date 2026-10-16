package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/oleoleg/project-manager/internal/db"
	"github.com/oleoleg/project-manager/internal/models"
)

var (
	ErrActNameRequired    = errors.New("наименование акта обязательно")
	ErrActKindRequired    = errors.New("вид акта обязателен")
	ErrActKindCustomName  = errors.New("укажите название своего типа акта")
	ErrActNegative        = errors.New("суммы не могут быть отрицательными")
	ErrActOverDistribute  = errors.New("распределение по этапам превышает сумму акта")
	ErrAttachmentRequired = errors.New("приложение обязательно должно быть привязано к акту")
)

type ActService struct {
	repo      *db.ActRepo
	stageRepo *db.StageRepo
}

func NewActService(repo *db.ActRepo, stageRepo *db.StageRepo) *ActService {
	return &ActService{repo: repo, stageRepo: stageRepo}
}

// ======================= Валидация =======================

func (s *ActService) Validate(a *models.Act) error {
	a.Name = strings.TrimSpace(a.Name)
	a.Kind = strings.TrimSpace(a.Kind)
	a.KindCustom = strings.TrimSpace(a.KindCustom)

	if a.Name == "" {
		return ErrActNameRequired
	}
	if a.Kind == "" {
		return ErrActKindRequired
	}
	if a.Kind == models.ActKindCustom && a.KindCustom == "" {
		return ErrActKindCustomName
	}
	if a.Amount < 0 {
		return ErrActNegative
	}
	return nil
}

func (s *ActService) ValidateAttachment(a *models.ActAttachment) error {
	a.Name = strings.TrimSpace(a.Name)
	a.Kind = strings.TrimSpace(a.Kind)
	a.KindCustom = strings.TrimSpace(a.KindCustom)

	if a.Name == "" {
		return ErrActNameRequired
	}
	if a.Kind == "" {
		return ErrActKindRequired
	}
	if a.Kind == models.ActKindCustom && a.KindCustom == "" {
		return ErrActKindCustomName
	}
	if a.ActGUID == "" {
		return ErrAttachmentRequired
	}
	return nil
}

// ======================= Главные акты =======================

func (s *ActService) GetByID(ctx context.Context, guid string) (*models.Act, error) {
	return s.repo.GetByID(ctx, guid)
}

func (s *ActService) ListByStage(ctx context.Context, stageGUID string) ([]models.Act, error) {
	return s.repo.ListByStage(ctx, stageGUID)
}

// Create создаёт акт и привязывает его к этапам (links).
func (s *ActService) Create(ctx context.Context, a *models.Act, links []models.ActStageLink) error {
	if err := s.Validate(a); err != nil {
		return err
	}

	// Сумма распределения не должна превышать сумму акта.
	if len(links) > 0 {
		var total float64
		for _, l := range links {
			total += l.Amount
		}
		if total > a.Amount+0.01 {
			return ErrActOverDistribute
		}
	}

	if err := s.repo.Create(ctx, a); err != nil {
		return err
	}

	// Сохраняем связи и пересчитываем этапы.
	for _, l := range links {
		l.ActGUID = a.GUID
		if err := s.repo.LinkStage(ctx, l); err != nil {
			return err
		}
	}
	s.recalcStagesFromLinks(ctx, links)

	return nil
}

// Update обновляет акт и перезаписывает связи с этапами.
func (s *ActService) Update(ctx context.Context, a *models.Act, links []models.ActStageLink) error {
	if err := s.Validate(a); err != nil {
		return err
	}

	if len(links) > 0 {
		var total float64
		for _, l := range links {
			total += l.Amount
		}
		if total > a.Amount+0.01 {
			return ErrActOverDistribute
		}
	}

	if err := s.repo.Update(ctx, a); err != nil {
		return err
	}

	// Снимаем старые связи.
	oldLinks, _ := s.repo.StagesLinkedToAct(ctx, a.GUID)
	for _, l := range oldLinks {
		_ = s.repo.UnlinkStage(ctx, a.GUID, l.StageGUID)
	}

	// Создаём новые.
	for _, l := range links {
		l.ActGUID = a.GUID
		if err := s.repo.LinkStage(ctx, l); err != nil {
			return err
		}
	}

	// Пересчитываем все затронутые этапы.
	affected := map[string]bool{}
	for _, l := range oldLinks {
		affected[l.StageGUID] = true
	}
	for _, l := range links {
		affected[l.StageGUID] = true
	}
	var affectedLinks []models.ActStageLink
	for g := range affected {
		affectedLinks = append(affectedLinks, models.ActStageLink{StageGUID: g})
	}
	s.recalcStagesFromLinks(ctx, affectedLinks)

	return nil
}

// Delete удаляет акт и пересчитывает затронутые этапы.
func (s *ActService) Delete(ctx context.Context, guid string) error {
	links, _ := s.repo.StagesLinkedToAct(ctx, guid)
	if err := s.repo.Delete(ctx, guid); err != nil {
		return err
	}
	s.recalcStagesFromLinks(ctx, links)
	return nil
}

// ======================= Суммы =======================

// SumByStage — сумма распределённых по этапу актов.
func (s *ActService) SumByStage(ctx context.Context, stageGUID string) (float64, error) {
	return s.repo.SumByStage(ctx, stageGUID)
}

// SumByAct — сколько уже распределено из акта.
func (s *ActService) SumByAct(ctx context.Context, actGUID string) (float64, error) {
	return s.repo.SumByAct(ctx, actGUID)
}

// StageGUIDByAct — GUID первого этапа, к которому привязан акт.
func (s *ActService) StageGUIDByAct(ctx context.Context, actGUID string) (string, error) {
	return s.repo.StageGUIDByAct(ctx, actGUID)
}

// CountByContract — map stageGUID → количество актов.
func (s *ActService) CountByContract(ctx context.Context, contractID string) (map[string]int, error) {
	return s.repo.CountByContract(ctx, contractID)
}

// ======================= Приложения =======================

// ListAttachmentsByAct — список приложений главного акта.
func (s *ActService) ListAttachmentsByAct(ctx context.Context, actGUID string) ([]models.ActAttachment, error) {
	return s.repo.ListAttachmentsByAct(ctx, actGUID)
}

// GetAttachmentByID — приложение по GUID.
func (s *ActService) GetAttachmentByID(ctx context.Context, guid string) (*models.ActAttachment, error) {
	return s.repo.GetAttachmentByID(ctx, guid)
}

// CreateAttachment — создаёт приложение.
func (s *ActService) CreateAttachment(ctx context.Context, a *models.ActAttachment) error {
	if err := s.ValidateAttachment(a); err != nil {
		return err
	}
	return s.repo.CreateAttachment(ctx, a)
}

// DeleteAttachment — удаляет приложение.
func (s *ActService) DeleteAttachment(ctx context.Context, guid string) error {
	return s.repo.DeleteAttachment(ctx, guid)
}

// ======================= Пересчёт этапов =======================

// recalcStagesFromLinks пересчитывает stages.closed_by_acts для этапов,
// у которых closed_by_acts_manual = false.
func (s *ActService) recalcStagesFromLinks(ctx context.Context, links []models.ActStageLink) {
	seen := map[string]bool{}
	for _, l := range links {
		if seen[l.StageGUID] {
			continue
		}
		seen[l.StageGUID] = true

		stage, err := s.stageRepo.GetByID(ctx, l.StageGUID)
		if err != nil || stage == nil {
			continue
		}
		if stage.ClosedByActsManual {
			continue
		}
		sum, err := s.repo.SumByStage(ctx, l.StageGUID)
		if err != nil {
			continue
		}
		stage.ClosedByActs = sum
		_ = s.stageRepo.Update(ctx, stage)
	}
}

// ======================= Утилиты =======================

// ParseDateSafe — парсинг даты из формы.
func ParseDateSafe(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	return &t
}

// StagesLinkedToAct — список связей акта с этапами.
func (s *ActService) StagesLinkedToAct(ctx context.Context, actGUID string) ([]models.ActStageLink, error) {
	return s.repo.StagesLinkedToAct(ctx, actGUID)
}

func (s *ActService) AmountByActAndStage(ctx context.Context, actGUID, stageGUID string) (float64, error) {
	return s.repo.AmountByActAndStage(ctx, actGUID, stageGUID)
}

// UpdateAttachment обновляет приложение.
func (s *ActService) UpdateAttachment(ctx context.Context, a *models.ActAttachment) error {
	if err := s.ValidateAttachment(a); err != nil {
		return err
	}
	return s.repo.UpdateAttachment(ctx, a)
}
