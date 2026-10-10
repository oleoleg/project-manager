package services

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/oleoleg/project-manager/internal/db"
	"github.com/oleoleg/project-manager/internal/models"
)

var (
	ErrContractIDFormat       = errors.New("ID договора должен начинаться с 4 цифр номера проекта (например 6666 или 6666-П2)")
	ErrContractCounterparty   = errors.New("контрагент обязателен")
	ErrContractType           = errors.New("тип договора обязателен")
	ErrContractNumberRequired = errors.New("номер договора обязателен")
	ErrContractExists         = errors.New("договор с таким ID уже существует")
)

// Минимум: 4 цифры. Дальше может быть суффикс через дефис:
// 6666, 6666-П, 6666-П2, 6666-H и т.п.
var contractIDRe = regexp.MustCompile(`^[0-9]{4}(-[A-Za-zА-Яа-я0-9]{1,5})?$`)

type ContractService struct {
	repo *db.ContractRepo
}

func NewContractService(repo *db.ContractRepo) *ContractService {
	return &ContractService{repo: repo}
}

func (s *ContractService) Validate(c *models.Contract) error {
	c.ID = strings.TrimSpace(c.ID)
	c.ContractNumber = strings.TrimSpace(c.ContractNumber)

	if !contractIDRe.MatchString(c.ID) {
		return ErrContractIDFormat
	}
	if c.ContractNumber == "" {
		return ErrContractNumberRequired
	}
	if c.CounterpartyID == "" {
		return ErrContractCounterparty
	}
	if c.ContractType == "" {
		return ErrContractType
	}
	return nil
}

func (s *ContractService) Create(ctx context.Context, c *models.Contract) error {
	if err := s.Validate(c); err != nil {
		return err
	}
	err := s.repo.Create(ctx, c)
	if isUniqueViolation(err) {
		return ErrContractExists
	}
	return err
}

func (s *ContractService) Update(ctx context.Context, c *models.Contract) error {
	if err := s.Validate(c); err != nil {
		return err
	}
	return s.repo.Update(ctx, c)
}

func (s *ContractService) GetByID(ctx context.Context, id string) (*models.Contract, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ContractService) List(ctx context.Context, f db.ContractFilter) ([]models.Contract, error) {
	return s.repo.List(ctx, f)
}

func (s *ContractService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *ContractService) Statuses(ctx context.Context) ([]models.ContractStatusRef, error) {
	return s.repo.ListStatuses(ctx)
}

func (s *ContractService) Types(ctx context.Context) ([]models.ContractTypeRef, error) {
	return s.repo.ListTypes(ctx)
}

func (s *ContractService) ProcurementCardStatuses(ctx context.Context) ([]models.TrackingStatusRef, error) {
	return s.repo.ListProcurementCardStatuses(ctx)
}

func (s *ContractService) TrackingStatuses(ctx context.Context) ([]models.TrackingStatusRef, error) {
	return s.repo.ListTrackingStatuses(ctx)
}

// ParseDate — вспомогательный парсер формата input type="date" (YYYY-MM-DD).
func ParseDate(s string) *time.Time {
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
