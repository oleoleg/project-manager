package db_test

import (
	"context"
	"testing"

	"github.com/oleoleg/project-manager/internal/db"
	"github.com/oleoleg/project-manager/internal/models"
)

func TestContractRepo(t *testing.T) {
	t.Run("CreateAndGet", func(t *testing.T) {
		clearTestDB(t)
		ctx := context.Background()

		// Нам нужен контрагент, так как на него ссылается договор
		cpRepo := db.NewCounterpartyRepo(testPool)
		_ = cpRepo.Create(ctx, &models.Counterparty{ID: "1234", Name: "Заказчик"})

		repo := db.NewContractRepo(testPool)

		c := &models.Contract{
			ID:             "1234-01",
			ContractNumber: "Д-123",
			StatusCode:     1,
			CounterpartyID: "1234",
			ContractType:   "contract",
		}

		// 1. Создаем
		err := repo.Create(ctx, c)
		if err != nil {
			t.Fatalf("ошибка создания договора: %v", err)
		}

		// 2. Получаем
		got, err := repo.GetByID(ctx, "1234-01")
		if err != nil {
			t.Fatalf("ошибка получения договора: %v", err)
		}

		if got.ContractNumber != c.ContractNumber {
			t.Errorf("ожидался номер %q, получено %q", c.ContractNumber, got.ContractNumber)
		}
	})
}
