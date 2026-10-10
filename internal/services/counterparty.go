package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/oleoleg/project-manager/internal/db"
	"github.com/oleoleg/project-manager/internal/models"
)

var (
	ErrCounterpartyIDFormat  = errors.New("ID контрагента должен быть в формате NNNN (4 цифры)")
	ErrCounterpartyNameEmpty = errors.New("наименование обязательно")
	ErrCounterpartyExists    = errors.New("контрагент с таким ID уже существует")
)

var counterpartyIDRe = regexp.MustCompile(`^[0-9]{4}$`)

type CounterpartyService struct {
	repo *db.CounterpartyRepo
}

func NewCounterpartyService(repo *db.CounterpartyRepo) *CounterpartyService {
	return &CounterpartyService{repo: repo}
}

// Validate нормализует и проверяет поля.
func (s *CounterpartyService) Validate(c *models.Counterparty) error {
	c.ID = strings.TrimSpace(c.ID)
	c.Name = strings.TrimSpace(c.Name)

	if !counterpartyIDRe.MatchString(c.ID) {
		return ErrCounterpartyIDFormat
	}
	if c.Name == "" {
		return ErrCounterpartyNameEmpty
	}
	return nil
}

func (s *CounterpartyService) Create(ctx context.Context, c *models.Counterparty) error {
	if err := s.Validate(c); err != nil {
		return err
	}
	err := s.repo.Create(ctx, c)
	if isUniqueViolation(err) {
		return ErrCounterpartyExists
	}
	return err
}

func (s *CounterpartyService) Update(ctx context.Context, c *models.Counterparty) error {
	if err := s.Validate(c); err != nil {
		return err
	}
	return s.repo.Update(ctx, c)
}

func (s *CounterpartyService) GetByID(ctx context.Context, id string) (*models.Counterparty, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *CounterpartyService) List(ctx context.Context, search string) ([]models.Counterparty, error) {
	return s.repo.List(ctx, search)
}

func (s *CounterpartyService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// isUniqueViolation — ошибка нарушения уникальности из PostgreSQL (код 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

var _ = fmt.Sprintf // на будущее
