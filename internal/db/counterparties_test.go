package db_test

import (
	"context"
	"testing"

	"github.com/oleoleg/project-manager/internal/db"
	"github.com/oleoleg/project-manager/internal/models"
)

func TestCounterpartyRepo(t *testing.T) {
	t.Run("CreateAndGet", func(t *testing.T) {
		clearTestDB(t) // Очищаем БД
		ctx := context.Background()
		repo := db.NewCounterpartyRepo(testPool)

		cp := &models.Counterparty{
			ID:        "8888",
			Name:      "ООО Ромашка",
			Partner:   "Иванов",
			ShortName: "Ромашка",
		}

		// 1. Создаем
		err := repo.Create(ctx, cp)
		if err != nil {
			t.Fatalf("ошибка создания контрагента: %v", err)
		}

		// 2. Получаем
		got, err := repo.GetByID(ctx, "8888")
		if err != nil {
			t.Fatalf("ошибка получения контрагента: %v", err)
		}

		if got.Name != cp.Name {
			t.Errorf("ожидалось имя %q, получено %q", cp.Name, got.Name)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		clearTestDB(t)
		ctx := context.Background()
		repo := db.NewCounterpartyRepo(testPool)

		_, err := repo.GetByID(ctx, "0000")
		if err != db.ErrCounterpartyNotFound {
			t.Errorf("ожидалась ошибка ErrCounterpartyNotFound, получено %v", err)
		}
	})
}
