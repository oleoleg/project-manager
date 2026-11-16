package services

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/oleoleg/project-manager/internal/db"
	"github.com/oleoleg/project-manager/internal/models"
)

var (
	ErrContractIDFormat     = errors.New("ID договора должен начинаться с 4 цифр номера проекта (например 6666 или 6666-D2)")
	ErrContractCounterparty = errors.New("контрагент обязателен")
	ErrContractType         = errors.New("тип договора обязателен")
	ErrContractExists       = errors.New("договор с таким ID уже существует")
	ErrContractIDRange      = errors.New("ID договора должен начинаться с цифр от 2000 до 9999")
)

// Минимум: 4 цифры. Дальше может быть суффикс через дефис:
// 6666, 6666-D, 6666-D2, 6666-D и т.п. (буквы только латинские)
var contractIDRe = regexp.MustCompile(`^[0-9]{4}(-[A-Za-z0-9]{1,5})?$`)

type ContractService struct {
	repo *db.ContractRepo
}

func NewContractService(repo *db.ContractRepo) *ContractService {
	return &ContractService{repo: repo}
}

var ErrContractProjectURLRequired = errors.New("путь к корню проекта обязателен")

func (s *ContractService) Validate(c *models.Contract) error {
	c.ID = strings.TrimSpace(c.ID)
	c.ContractNumber = strings.TrimSpace(c.ContractNumber)
	c.ProjectURL = strings.TrimSpace(c.ProjectURL)

	if !contractIDRe.MatchString(c.ID) {
		return ErrContractIDFormat
	}

	// Проверка диапазона: первые 4 цифры должны быть от 2000 до 9999
	if len(c.ID) >= 4 {
		num, err := strconv.Atoi(c.ID[:4])
		if err != nil || num < 2000 || num > 9999 {
			return ErrContractIDRange
		}
	}

	if c.ProjectURL == "" {
		return ErrContractProjectURLRequired
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

func (s *ContractService) ListActiveUsers(ctx context.Context) ([]models.User, error) {
	return s.repo.ListActiveUsers(ctx)
}

func (s *ContractService) ListAssignees(ctx context.Context, contractID string) ([]models.User, error) {
	return s.repo.ListAssignees(ctx, contractID)
}

func (s *ContractService) SetAssignees(ctx context.Context, contractID string, userIDs []int64) error {
	return s.repo.SetAssignees(ctx, contractID, userIDs)
}

func (s *ContractService) GetResponsible(ctx context.Context, userID int64) (*models.User, error) {
	return s.repo.GetResponsible(ctx, userID)
}

func (s *ContractService) GetResponsibleUserID(ctx context.Context, contractID string) (int64, bool) {
	return s.repo.GetResponsibleUserID(ctx, contractID)
}
